package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ── SendAfrica SMS provider (REST) ───────────────────────────────────────────
//
// Implements SMSProvider against https://api.sendafrica.online (see
// https://docs.sendafrica.online/ai-integration):
//
//   - Auth: header `X-API-Key: SA-...` (env SMS_API_KEY — never in code).
//   - Single JSON POST to /v1/sms/ with { to, message, from? }.
//   - Every response uses the envelope { success, data, error, request_id,
//     timestamp } — request_id is logged for support.
//   - `Success` means provider-accepted submission, not handset delivery.
//   - Retries: 429 honors `Retry-After`, 5xx retries with backoff — both
//     ALWAYS reuse the same Idempotency-Key so replays can never double-send
//     or double-charge. 401/402/403 are never retried.
//
// Choose with SMS_PROVIDER=sendafrica. SMS_BASE_URL can point at the sandbox.

const (
	sendAfricaDefaultBaseURL = "https://api.sendafrica.online"
	sendAfricaMaxAttempts    = 3
)

type SendAfricaProvider struct {
	apiKey   string
	senderID string // optional; empty = platform default "SENDAFRICA"
	baseURL  string
	client   *http.Client
}

func NewSendAfricaProvider(apiKey, senderID, baseURL string) *SendAfricaProvider {
	if baseURL == "" {
		baseURL = sendAfricaDefaultBaseURL
	}
	return &SendAfricaProvider{
		apiKey:   apiKey,
		senderID: senderID,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *SendAfricaProvider) Name() string { return "sendafrica" }

// sendAfricaEnvelope mirrors the API's response envelope.
type sendAfricaEnvelope struct {
	Success bool             `json:"success"`
	Data    sendAfricaResult `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

type sendAfricaResult struct {
	MessageID   string `json:"message_id"`
	Status      string `json:"status"`
	Cost        string `json:"cost"`
	CreditsUsed int    `json:"credits_used"`
}

// SendSMS submits one message. It returns nil only when SendAfrica accepted
// the submission (status "Success"); handset delivery is reconciled later via
// the account message logs, so callers must not treat nil as "delivered".
func (p *SendAfricaProvider) SendSMS(ctx context.Context, phoneE164, message string) error {
	if p.apiKey == "" {
		return errors.New("sendafrica: SMS_API_KEY is empty")
	}

	payload := map[string]interface{}{
		"to":      phoneE164,
		"message": message,
	}
	if p.senderID != "" {
		payload["from"] = p.senderID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sendafrica: encode request: %w", err)
	}

	// One logical message = one idempotency key, reused across ALL internal
	// retries so a replay returns the cached response instead of double-
	// sending or double-charging (per docs/rate-limits).
	idemKey := uuid.NewString()

	var lastErr error
	for attempt := 1; attempt <= sendAfricaMaxAttempts; attempt++ {
		env, retryable, err := p.attempt(ctx, body, idemKey)
		if err == nil {
			log.Printf("SMS[sendafrica] to=%s message_id=%s credits_used=%d",
				phoneE164, env.Data.MessageID, env.Data.CreditsUsed)
			return nil
		}
		lastErr = err
		if !retryable {
			return err
		}
		if attempt == sendAfricaMaxAttempts {
			break
		}
		// Backoff before the next attempt (429 Retry-After already slept
		// inside attempt(); here we use plain exponential backoff for 5xx).
		if !sleepCtx(ctx, time.Duration(attempt)*2*time.Second) {
			return ctx.Err()
		}
	}
	return lastErr
}

// attempt performs one HTTP round-trip. Returns (envelope, retryable, err).
// On success err==nil; on failure retryable says whether a retry with the
// SAME Idempotency-Key is safe (429 / 5xx — everything else is terminal).
func (p *SendAfricaProvider) attempt(ctx context.Context, body []byte, idemKey string) (sendAfricaEnvelope, bool, error) {
	var env sendAfricaEnvelope

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/sms/", bytes.NewReader(body))
	if err != nil {
		return env, false, fmt.Errorf("sendafrica: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.apiKey)
	req.Header.Set("Idempotency-Key", idemKey)

	resp, err := p.client.Do(req)
	if err != nil {
		// Network/timeout errors: the request may or may not have landed —
		// only a retry with the same key is safe.
		return env, true, fmt.Errorf("sendafrica: network: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if uerr := json.Unmarshal(raw, &env); uerr != nil {
		// Non-JSON body (proxy/gateway error page). Retry only on 5xx.
		return env, resp.StatusCode >= 500,
			fmt.Errorf("sendafrica: HTTP %d with non-JSON body (request_id %s)", resp.StatusCode, env.RequestID)
	}

	reqID := env.RequestID
	errText := "unknown error"
	if env.Error != nil && env.Error.Message != "" {
		errText = env.Error.Message
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if !env.Success {
			// Well-formed envelope reporting failure.
			return env, false, fmt.Errorf("sendafrica: %s (request_id %s)", errText, reqID)
		}
		if env.Data.Status != "Success" {
			return env, false, fmt.Errorf("sendafrica: unexpected status %q (request_id %s)", env.Data.Status, reqID)
		}
		return env, false, nil

	case resp.StatusCode == http.StatusUnauthorized:
		return env, false, fmt.Errorf("sendafrica: invalid or missing API key — check SMS_API_KEY (request_id %s)", reqID)

	case resp.StatusCode == http.StatusPaymentRequired:
		return env, false, fmt.Errorf("sendafrica: insufficient credits — top up the SendAfrica account (request_id %s)", reqID)

	case resp.StatusCode == http.StatusForbidden:
		// invalid_phone / unsupported_destination — fix the input, don't retry.
		return env, false, fmt.Errorf("sendafrica: rejected destination %q: %s (request_id %s)",
			jsonFieldString(body, "to"), errText, reqID)

	case resp.StatusCode == http.StatusTooManyRequests:
		// Honor Retry-After, then let the caller retry with the same key.
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, perr := strconv.Atoi(ra); perr == nil && secs > 0 && secs <= 30 {
				if !sleepCtx(ctx, time.Duration(secs)*time.Second) {
					return env, false, ctx.Err()
				}
			}
		}
		return env, true, fmt.Errorf("sendafrica: rate limited (429) (request_id %s)", reqID)

	case resp.StatusCode >= 500:
		return env, true, fmt.Errorf("sendafrica: server error %d (request_id %s)", resp.StatusCode, reqID)

	default:
		return env, false, fmt.Errorf("sendafrica: HTTP %d: %s (request_id %s)", resp.StatusCode, errText, reqID)
	}
}

// sleepCtx sleeps for d, aborting early if ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// jsonFieldString extracts a top-level string field for error messages.
func jsonFieldString(body []byte, field string) string {
	var m map[string]interface{}
	if json.Unmarshal(body, &m) == nil {
		if v, ok := m[field].(string); ok {
			return v
		}
	}
	return "?"
}
