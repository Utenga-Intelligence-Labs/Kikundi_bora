package handlers

import (
	"errors"
	"time"

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
	Name           string `json:"name"`
	Type           string `json:"type"`
	OwnerMemberRef string `json:"owner_member_ref,omitempty"`
}

func (h *LedgerHandler) OpenAccount(c *fiber.Ctx) error {
	var req openAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
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
	AccountName string `json:"account_name"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amount_minor"` // whole TZS shillings as integer
}
type recordTxRequest struct {
	Memo       string         `json:"memo"`
	OccurredAt *time.Time     `json:"occurred_at,omitempty"` // backdated corrections allowed
	Entries    []entryRequest `json:"entries"`
}

func (h *LedgerHandler) RecordTransaction(c *fiber.Ctx) error {
	var req recordTxRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
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
	return c.JSON(fiber.Map{"statement": lines})
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
