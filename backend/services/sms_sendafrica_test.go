package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newSendAfricaTestServer spins an httptest server answering count requests
// via handler, and returns a provider pointed at it.
func newSendAfricaTestServer(t *testing.T, handler http.HandlerFunc) (*SendAfricaProvider, *[]sendAfricaCapture) {
	t.Helper()
	captures := &[]sendAfricaCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		from, _ := req["from"].(string)
		*captures = append(*captures, sendAfricaCapture{
			key:            r.Header.Get("X-API-Key"),
			idempotencyKey: r.Header.Get("Idempotency-Key"),
			to:             req["to"].(string),
			message:        req["message"].(string),
			from:           from,
		})
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewSendAfricaProvider("SA-test-key", "", srv.URL), captures
}

type sendAfricaCapture struct {
	key, idempotencyKey, to, message, from string
}

func okEnvelope(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"message_id":   "SA-00000000-0000-0000-0000-000000000000",
			"status":       "Success",
			"credits_used": 1,
		},
		"request_id": "11111111-1111-1111-1111-111111111111",
		"timestamp":  "2025-01-01T00:00:00Z",
	})
}

func TestSendAfricaSendSuccess(t *testing.T) {
	p, captures := newSendAfricaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		okEnvelope(w)
	})
	if err := p.SendSMS(context.Background(), "+255710123456", "Habari"); err != nil {
		t.Fatalf("SendSMS error: %v", err)
	}
	if len(*captures) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captures))
	}
	c := (*captures)[0]
	if c.key != "SA-test-key" || c.to != "+255710123456" || c.message != "Habari" {
		t.Errorf("bad capture: %+v", c)
	}
	if c.idempotencyKey == "" {
		t.Error("expected Idempotency-Key header to be set")
	}
}

func TestSendAfricaSenderIDIncluded(t *testing.T) {
	p, captures := newSendAfricaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		okEnvelope(w)
	})
	p.senderID = "MYGROUP"
	if err := p.SendSMS(context.Background(), "+255710123456", "msg"); err != nil {
		t.Fatalf("SendSMS error: %v", err)
	}
	c := (*captures)[0]
	if c.from != "MYGROUP" {
		t.Errorf("expected from=MYGROUP on the wire, got %q", c.from)
	}
}

func TestSendAfricaQueuedStatusAccepted(t *testing.T) {
	// The live API returns status "queued" (not the documented "Success").
	p, _ := newSendAfricaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"message_id":   "SA-5ddf7956-c26d-451a-ace4-4c83a081a853",
				"status":       "queued",
				"credits_used": 1,
			},
			"timestamp": "2026-09-08T12:13:27Z",
		})
	})
	if err := p.SendSMS(context.Background(), "+255680185784", "msg"); err != nil {
		t.Fatalf("queued status must be accepted, got: %v", err)
	}
}

func TestSendAfricaInsufficientCreditsNotRetried(t *testing.T) {
	var attempts int32
	p, _ := newSendAfricaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    false,
			"error":      map[string]string{"code": "insufficient_credits", "message": "top up"},
			"request_id": "22222222-2222-2222-2222-222222222222",
			"timestamp":  "2025-01-01T00:00:00Z",
		})
	})
	err := p.SendSMS(context.Background(), "+255710123456", "msg")
	if err == nil || !strings.Contains(err.Error(), "insufficient credits") || !strings.Contains(err.Error(), "22222222") {
		t.Fatalf("expected insufficient-credits error with request_id, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("402 must not be retried, attempts=%d", attempts)
	}
}

func TestSendAfricaRateLimitRetriesSameKey(t *testing.T) {
	var attempts int32
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, "error": map[string]string{"code": "rate_limit_exceeded"},
				"request_id": "33333333-3333-3333-3333-333333333333", "timestamp": "2025-01-01T00:00:00Z",
			})
			return
		}
		okEnvelope(w)
	}))
	defer srv.Close()
	p := NewSendAfricaProvider("SA-test-key", "", srv.URL)
	if err := p.SendSMS(context.Background(), "+255710123456", "msg"); err != nil {
		t.Fatalf("expected success after 429 retry, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 2 || keys[0] != keys[1] {
		t.Errorf("expected 2 attempts with the SAME Idempotency-Key, got attempts=%d keys=%v", attempts, keys)
	}
}

func TestSendAfricaServerErrorRetriesWithBackoff(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"success":false,"error":{"code":"server_error"},"request_id":"44444444-4444-4444-4444-444444444444","timestamp":"2025-01-01T00:00:00Z"}`))
			return
		}
		okEnvelope(w)
	}))
	defer srv.Close()
	p := NewSendAfricaProvider("SA-test-key", "", srv.URL)
	// Backoff after attempt 1 is 2s, so attempt 2 fires before the 3s
	// deadline; the 4s backoff after attempt 2 is then cut short by ctx.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := p.SendSMS(ctx, "+255710123456", "msg")
	if err == nil {
		t.Fatal("expected error (ctx deadline during backoff)")
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts before ctx cancel, got %d", attempts)
	}
}

func TestSendAfricaInvalidKeyNotRetried(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": map[string]string{"code": "invalid_api_key", "message": "bad key"},
			"request_id": "55555555-5555-5555-5555-555555555555", "timestamp": "2025-01-01T00:00:00Z",
		})
	}))
	defer srv.Close()
	p := NewSendAfricaProvider("SA-wrong", "", srv.URL)
	err := p.SendSMS(context.Background(), "+255710123456", "msg")
	if err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("expected invalid-key error, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("401 must not be retried, attempts=%d", attempts)
	}
}
