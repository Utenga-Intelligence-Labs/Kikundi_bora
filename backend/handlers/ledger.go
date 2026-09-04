package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"kikundibora/ledger"
)

// LedgerHandler exposes the ledger core over the existing Fiber API.
// The ledger package itself has zero HTTP dependencies; this file is a thin
// translation layer (parse -> call -> map error -> JSON).
type LedgerHandler struct {
	lg      *ledger.Ledger
	groupID uuid.UUID
}

// NewLedgerHandler wires a handler to one group scope.
func NewLedgerHandler(lg *ledger.Ledger, groupID uuid.UUID) *LedgerHandler {
	return &LedgerHandler{lg: lg, groupID: groupID}
}

// actorUUID resolves the authenticated user to a stable UUID for event
// attribution. Session user ids are expected to be UUIDs; any legacy format
// is mapped deterministically so attribution never blocks posting.
func actorUUID(c *fiber.Ctx) uuid.UUID {
	raw := middleware.GetUserID(c)
	if id, err := uuid.Parse(raw); err == nil {
		return id
	}
	if raw == "" {
		return uuid.NameSpaceOID // system/unknown — only possible in tests without auth
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("actor|"+raw))
}

func ledgerError(c *fiber.Ctx, err error) error {
	var code int
	switch {
	case errors.Is(err, ledger.ErrUnbalancedTransaction),
		errors.Is(err, ledger.ErrTooFewEntries),
		errors.Is(err, ledger.ErrInvalidEntry),
		errors.Is(err, ledger.ErrCurrencyMismatch),
		errors.Is(err, ledger.ErrUnsupportedCurrency):
		code = fiber.StatusUnprocessableEntity
	case errors.Is(err, ledger.ErrAccountNotFound),
		errors.Is(err, ledger.ErrTransactionNotFound):
		code = fiber.StatusNotFound
	case errors.Is(err, ledger.ErrAccountClosed), errors.Is(err, ledger.ErrAlreadyReversed):
		code = fiber.StatusConflict
	case errors.Is(err, ledger.ErrConcurrencyConflict):
		code = fiber.StatusConflict
	default:
		code = fiber.StatusInternalServerError
	}
	return c.Status(code).JSON(fiber.Map{"message": err.Error()})
}

// OpenAccount godoc: POST /admin/ledger/accounts
type openAccountRequest struct {
	Name           string `json:"name" validate:"required,min=2,max=100"`
	Type           string `json:"type" validate:"required,oneof=asset liability income expense equity"`
	OwnerMemberRef string `json:"owner_member_ref,omitempty" validate:"omitempty,max=100"`
}

func (h *LedgerHandler) OpenAccount(c *fiber.Ctx) error {
	var req openAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}
	typ := ledger.AccountType(req.Type)
	id, err := h.lg.OpenAccount(c.Context(), h.groupID, actorUUID(c), time.Now().UTC(),
		req.Name, typ, req.OwnerMemberRef, 0)
	if err != nil {
		return ledgerError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"account_name": req.Name, "id": id})
}

// RecordTransaction godoc: POST /admin/ledger/transactions
type entryRequest struct {
	AccountName string `json:"account_name" validate:"required,max=100"`
	Direction   string `json:"direction" validate:"required,oneof=debit credit"`
	AmountMinor int64  `json:"amount_minor" validate:"required,gt=0"` // whole TZS shillings as integer
}
type recordTxRequest struct {
	Memo       string         `json:"memo" validate:"required,max=500"`
	OccurredAt *time.Time     `json:"occurred_at,omitempty"` // backdated corrections allowed
	Entries    []entryRequest `json:"entries" validate:"required,min=1,max=50,dive"`
}

func (h *LedgerHandler) RecordTransaction(c *fiber.Ctx) error {
	var req recordTxRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}
	entries := make([]ledger.Entry, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = ledger.Entry{
			AccountName: e.AccountName,
			Direction:   ledger.Direction(e.Direction),
			Amount:      ledger.NewTZS(e.AmountMinor),
		}
	}
	at := time.Now().UTC()
	if req.OccurredAt != nil {
		at = *req.OccurredAt
	}
	txID, err := h.lg.RecordTransaction(c.Context(), h.groupID, actorUUID(c), at, req.Memo, entries)
	if err != nil {
		return ledgerError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"transaction_id": txID})
}

// ReverseTransaction godoc: POST /admin/ledger/transactions/:id/reverse
type reverseTxRequest struct {
	Reason string `json:"reason"`
}

func (h *LedgerHandler) ReverseTransaction(c *fiber.Ctx) error {
	txID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID si sahihi"})
	}
	var req reverseTxRequest
	_ = c.BodyParser(&req) // body optional
	eventID, err := h.lg.ReverseTransaction(c.Context(), h.groupID, actorUUID(c), txID,
		time.Now().UTC(), req.Reason)
	if err != nil {
		return ledgerError(c, err)
	}
	return c.JSON(fiber.Map{"reversal_event_id": eventID})
}

// GetBalance godoc: GET /admin/ledger/balance?account=akiba_ya_mwanachama:42&as_of=RFC3339
func (h *LedgerHandler) GetBalance(c *fiber.Ctx) error {
	name := c.Query("account")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "account inahitajika"})
	}
	var asOf *time.Time
	if raw := c.Query("as_of"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "as_of lazima iwe RFC3339"})
		}
		asOf = &t
	}
	money, err := h.lg.GetBalance(c.Context(), h.groupID, name, asOf)
	if err != nil {
		return ledgerError(c, err)
	}
	return c.JSON(fiber.Map{
		"account":      name,
		"amount_minor": money.AmountMinor,
		"currency":     money.Currency,
	})
}

// GetStatement godoc: GET /admin/ledger/statement?account=..&from=..&to=..
func (h *LedgerHandler) GetStatement(c *fiber.Ctx) error {
	name := c.Query("account")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "account inahitajika"})
	}
	parse := func(key string, def time.Time) (time.Time, error) {
		raw := c.Query(key)
		if raw == "" {
			return def, nil
		}
		return time.Parse(time.RFC3339, raw)
	}
	from, err := parse("from", time.Now().UTC().AddDate(0, -1, 0))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "from lazima iwe RFC3339"})
	}
	to, err := parse("to", time.Now().UTC())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "to lazima iwe RFC3339"})
	}
	lines, err := h.lg.GetLedgerEntries(c.Context(), h.groupID, name, from, to)
	if err != nil {
		return ledgerError(c, err)
	}
	return c.JSON(fiber.Map{"statement": h.enrichStatementLines(lines)})
}

// enrichStatementLines attaches display names (actor) and, for reversal
// lines, the memo of the transaction they reverse plus the recorded reason.
func (h *LedgerHandler) enrichStatementLines(lines []ledger.StatementLine) []fiber.Map {
	out := make([]fiber.Map, 0, len(lines))
	if len(lines) == 0 {
		return out
	}

	actors := map[string]string{}
	ids := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, ln := range lines {
		if ln.ActorID != "" && !seen[ln.ActorID] {
			seen[ln.ActorID] = true
			ids = append(ids, ln.ActorID)
		}
	}
	if len(ids) > 0 {
		type actorRow struct {
			ID   string
			Name string
		}
		var rows []actorRow
		database.DB.Raw("SELECT id::text AS id, name FROM users WHERE id::text IN ?",
			ids).Scan(&rows)
		for _, r := range rows {
			actors[r.ID] = r.Name
		}
	}

	revMemo := map[string]string{}
	revReason := map[string]string{}
	var revStreams []string
	for _, ln := range lines {
		if ln.EventType == ledger.EventTransactionReversed {
			revStreams = append(revStreams, ln.TransactionID.String())
		}
	}
	if len(revStreams) > 0 {
		type revRow struct {
			Stream   string
			Original string
			Reason   string
		}
		var revs []revRow
		database.DB.Raw(`SELECT stream_id::text AS stream,
			payload->>'original_event_id' AS original,
			COALESCE(payload->>'reason','') AS reason
			FROM ledger_events WHERE group_id = ? AND stream_id::text IN ?
			AND event_type = ?`,
			h.groupID.String(), revStreams, string(ledger.EventTransactionReversed)).Scan(&revs)
		origIDs := make([]string, 0, len(revs))
		for _, r := range revs {
			origIDs = append(origIDs, r.Original)
			revReason[r.Stream] = r.Reason
		}
		if len(origIDs) > 0 {
			type memoRow struct {
				Event string
				Memo  string
			}
			var memos []memoRow
			database.DB.Raw(`SELECT event_id::text AS event,
				COALESCE(payload->>'memo','') AS memo
				FROM ledger_events WHERE event_id::text IN ?`, origIDs).Scan(&memos)
			byEvent := map[string]string{}
			for _, m := range memos {
				byEvent[m.Event] = m.Memo
			}
			for _, r := range revs {
				revMemo[r.Stream] = byEvent[r.Original]
			}
		}
	}

	for _, ln := range lines {
		name := actors[ln.ActorID]
		if name == "" {
			name = ln.ActorID
		}
		m := fiber.Map{
			"TransactionID": ln.TransactionID.String(),
			"EventType":     string(ln.EventType),
			"Direction":     string(ln.Direction),
			"AmountMinor":   ln.AmountMinor,
			"Currency":      ln.Currency,
			"OccurredAt":    ln.OccurredAt,
			"Memo":          ln.Memo,
			"ActorID":       ln.ActorID,
			"ActorName":     name,
		}
		if ln.EventType == ledger.EventTransactionReversed {
			m["ReversesMemo"] = revMemo[ln.TransactionID.String()]
			m["Reason"] = revReason[ln.TransactionID.String()]
		}
		out = append(out, m)
	}
	return out
}

// GetTransactionDetail godoc: GET /admin/ledger/transactions/:id
// Full detail for the click-to-view flow: memo, actor, legs, and — when the
// memo marks a disbursement — the linked loan (who approved, when it left).
func (h *LedgerHandler) GetTransactionDetail(c *fiber.Ctx) error {
	txID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID si sahihi"})
	}

	type eventRow struct {
		EventType  string
		Payload    string
		Actor      string
		OccurredAt time.Time
	}
	var ev eventRow
	if err := database.DB.Raw(`SELECT event_type, payload::text, actor_id AS actor, occurred_at
		FROM ledger_events WHERE group_id = ? AND stream_id = ?
		AND event_type IN (?, ?) ORDER BY sequence_no LIMIT 1`,
		h.groupID.String(), txID.String(),
		string(ledger.EventTransactionRecorded),
		string(ledger.EventTransactionReversed)).Scan(&ev).Error; err != nil ||
		ev.EventType == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Muamala haujapatikana"})
	}

	var payload struct {
		Memo            string             `json:"memo"`
		Entries         []ledger.EntryData `json:"entries"`
		OriginalEventID string             `json:"original_event_id"`
		Reason          string             `json:"reason"`
	}
	_ = json.Unmarshal([]byte(ev.Payload), &payload)

	actorName := ev.Actor
	var nm string
	database.DB.Raw("SELECT name FROM users WHERE id::text = ?", ev.Actor).Scan(&nm)
	if nm != "" {
		actorName = nm
	}

	detail := fiber.Map{
		"transaction_id": txID.String(),
		"event_type":     ev.EventType,
		"memo":           payload.Memo,
		"actor_name":     actorName,
		"occurred_at":    ev.OccurredAt,
		"entries":        payload.Entries,
		"reason":         payload.Reason,
	}

	if ev.EventType == string(ledger.EventTransactionReversed) && payload.OriginalEventID != "" {
		var origMemo string
		database.DB.Raw(`SELECT COALESCE(payload->>'memo','') FROM ledger_events
			WHERE event_id::text = ?`, payload.OriginalEventID).Scan(&origMemo)
		detail["reverses_memo"] = origMemo
		detail["reverses_transaction_id"] = payload.OriginalEventID
	}

	if mno, ok := loanMemberFromMemo(payload.Memo); ok {
		detail["loan"] = h.loanForDisbursement(mno)
	}

	return c.JSON(detail)
}

// loanMemberFromMemo extracts the member ref from auto-posted disbursement
// memos ("Mkopo uliotolewa KKK-0001").
func loanMemberFromMemo(memo string) (string, bool) {
	const prefix = "Mkopo uliotolewa "
	if !strings.HasPrefix(memo, prefix) {
		return "", false
	}
	ref := strings.TrimSpace(strings.TrimPrefix(memo, prefix))
	if ref == "" {
		return "", false
	}
	return ref, true
}

// loanForDisbursement finds the most recent disbursed loan behind a member
// ref, with the humans behind each step resolved to names.
func (h *LedgerHandler) loanForDisbursement(memberNo string) fiber.Map {
	type loanRow struct {
		ID           string
		MemberName   string
		MemberNo     string
		Amount       string
		Status       string
		ReviewedBy   string
		ReviewedAt   string
		DisbursedBy  string
		DisbursedAt  string
	}
	var lr loanRow
	database.DB.Raw(`
		SELECT l.id::text, m.full_name, m.member_no, l.approved_amount::text,
		       l.status,
		       COALESCE(ru.name,'') AS reviewed_by,
		       COALESCE(l.reviewed_at::text,'') AS reviewed_at,
		       COALESCE(du.name,'') AS disbursed_by,
		       COALESCE(l.disbursed_at::text,'') AS disbursed_at
		  FROM loans l
		  JOIN members m ON m.id = l.member_id
		  LEFT JOIN users ru ON ru.id::text = l.reviewed_by::text
		  LEFT JOIN users du ON du.id::text = l.disbursed_by::text
		 WHERE m.member_no = ? AND l.status IN ('OUTSTANDING','CLOSED','APPROVED')
		 ORDER BY l.disbursed_at DESC NULLS LAST, l.created_at DESC LIMIT 1`,
		memberNo).Scan(&lr)
	if lr.ID == "" {
		return nil
	}
	return fiber.Map{
		"id":           lr.ID,
		"member_name":  lr.MemberName,
		"member_no":    lr.MemberNo,
		"amount":       lr.Amount,
		"status":       lr.Status,
		"reviewed_by":  lr.ReviewedBy,
		"reviewed_at":  lr.ReviewedAt,
		"disbursed_by": lr.DisbursedBy,
		"disbursed_at": lr.DisbursedAt,
	}
}

// GetTrialBalance godoc: GET /admin/ledger/trial-balance
func (h *LedgerHandler) GetTrialBalance(c *fiber.Ctx) error {
	tb, err := h.lg.GetTrialBalance(c.Context(), h.groupID, nil)
	if err != nil && !errors.Is(err, ledger.ErrTrialBalanceNotZero) {
		return ledgerError(c, err)
	}
	status := fiber.StatusOK
	if !tb.Balanced {
		// Nonzero net is a critical alarm surfaced loudly to admins.
		status = fiber.StatusExpectationFailed
	}
	return c.Status(status).JSON(tb)
}

// Replay godoc: POST /admin/ledger/replay  (rebuild projections from events)
func (h *LedgerHandler) Replay(c *fiber.Ctx) error {
	var gid *uuid.UUID
	if c.Query("scope") != "all" {
		g := h.groupID
		gid = &g
	}
	if err := h.lg.RebuildProjections(c.Context(), gid); err != nil {
		return ledgerError(c, err)
	}
	return c.JSON(fiber.Map{"status": "rebuilt"})
}
