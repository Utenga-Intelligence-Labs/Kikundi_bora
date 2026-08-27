package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Ledger is the command/query façade over the event store + projections.
// All writes go through commands that validate FIRST, then append events and
// update projections atomically in one transaction (strongly consistent read
// models; nothing can be observed half-applied).
type Ledger struct {
	pool PgxPool
}

// New constructs a Ledger over the given pgx pool.
func New(pool PgxPool) (*Ledger, error) {
	return &Ledger{pool: pool}, nil
}

// Account mirrors the projected account entity returned by lookups.
type Account struct {
	Name         string
	GroupID      uuid.UUID
	Type         AccountType
	Currency     string
	OwnerRef     string
	Open         bool
	BalanceMinor int64
}

const sqlInsertEvent = `INSERT INTO ledger_events
	   (event_id, group_id, stream_id, sequence_no, event_type, event_version,
	    payload, actor_id, occurred_at, recorded_at, causation_id, correlation_id)
	 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	 RETURNING global_seq`

// insertEvent appends one event inside an open transaction at a given seq.
// recorded_at is server-generated here — clients never supply it
// (spec §2 invariant 5).
func insertEvent(ctx context.Context, tx pgx.Tx, e NewEvent, seq int64) (Envelope, error) {
	env := Envelope{
		EventID:       uuid.New(),
		GroupID:       e.GroupID,
		StreamID:      e.StreamID,
		SequenceNo:    seq,
		GlobalSeq:     0,
		EventType:     e.EventType,
		EventVersion:  EventVersion,
		Payload:       e.Payload,
		ActorID:       e.ActorID,
		OccurredAt:    e.OccurredAt.UTC(),
		RecordedAt:    time.Now().UTC(),
		CausationID:   e.CausationID,
		CorrelationID: e.CorrelationID,
	}
	err := tx.QueryRow(ctx, sqlInsertEvent,
		env.EventID, env.GroupID, env.StreamID, env.SequenceNo,
		string(env.EventType), env.EventVersion, []byte(env.Payload),
		env.ActorID, env.OccurredAt, env.RecordedAt, env.CausationID, env.CorrelationID,
	).Scan(&env.GlobalSeq)
	if err != nil {
		return Envelope{}, err
	}
	return env, nil
}

// streamHeadQ reads MAX(sequence_no) for a stream using any Querier (tx/pool).
func streamHeadQ(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, streamID uuid.UUID) (int64, error) {
	var cur int64
	err := q.QueryRow(ctx,
		`SELECT COALESCE(MAX(sequence_no),0) FROM ledger_events WHERE stream_id=$1`, streamID,
	).Scan(&cur)
	return cur, err
}

// checkOCC applies the optimistic-concurrency rules shared by all commands.
func checkOCC(expectedSequence, current int64) error {
	if expectedSequence > 0 && expectedSequence != current {
		return fmt.Errorf("%w: caller expected seq %d, store at %d",
			ErrConcurrencyConflict, expectedSequence, current)
	}
	if expectedSequence == 0 && current > 0 {
		return fmt.Errorf("%w: assumed new stream but store is at seq %d",
			ErrConcurrencyConflict, current)
	}
	return nil
}

// requireAccounts loads and validates every entry's projected account state:
// exists, belongs to groupID, single currency TZS, still open.
func requireAccounts(ctx context.Context, tx pgx.Tx, groupID uuid.UUID, entries []Entry) ([]Account, error) {
	accounts := make([]Account, 0, len(entries))
	for i, e := range entries {
		a, err := getAccountTx(ctx, tx, groupID, e.AccountName)
		if err == ErrAccountNotFound {
			return nil, fmt.Errorf("entry %d (%s): %w", i, e.AccountName, err)
		}
		if err != nil {
			return nil, err
		}
		if a.GroupID != groupID {
			return nil, fmt.Errorf("entry %d (%s): %w", i, e.AccountName, ErrGroupMismatch)
		}
		if !a.Open {
			return nil, fmt.Errorf("entry %d (%s): %w", i, e.AccountName, ErrAccountClosed)
		}
		if a.Currency != CurrencyTZS {
			return nil, fmt.Errorf("entry %d (%s): %w", i, e.AccountName, ErrCurrencyMismatch)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func getAccountTx(ctx context.Context, tx pgx.Tx, groupID uuid.UUID, name string) (Account, error) {
	var a Account
	var closedSeq *int64
	err := tx.QueryRow(ctx,
		`SELECT account_name, group_id, type, currency, COALESCE(owner_member_ref,''), closed_at_seq
		   FROM ledger_accounts WHERE group_id=$1 AND account_name=$2`, groupID, name,
	).Scan(&a.Name, &a.GroupID, &a.Type, &a.Currency, &a.OwnerRef, &closedSeq)
	switch {
	case err == pgx.ErrNoRows:
		return Account{}, ErrAccountNotFound
	case err != nil:
		return Account{}, err
	}
	a.Open = closedSeq == nil
	return a, nil
}

// ---- Commands --------------------------------------------------------------

// OpenAccount emits an AccountOpened event and projects the new account.
// Account identity is deterministic (UUIDv5 of group+name), so re-opening an
// existing name hits the uniqueness guard instead of creating a shadow.
func (l *Ledger) OpenAccount(
	ctx context.Context,
	groupID, actorID uuid.UUID,
	occurredAt time.Time,
	name string,
	typ AccountType,
	ownerMemberRef string,
	expectedSequence int64,
) (uuid.UUID, error) {
	if !typ.Valid() {
		return uuid.Nil, fmt.Errorf("%w: unknown account type %q", ErrInvalidEntry, typ)
	}
	if name == "" {
		return uuid.Nil, fmt.Errorf("%w: empty account name", ErrInvalidEntry)
	}
	accountID := DeterministicAccountID(groupID, name)

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cur, err := streamHeadQ(ctx, tx, accountID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := checkOCC(expectedSequence, cur); err != nil {
		return uuid.Nil, err
	}

	payload, err := json.Marshal(AccountOpenedPayload{
		Name:          name,
		Type:          typ,
		Currency:      CurrencyTZS,
		OwnerMemberID: ownerMemberRef,
	})
	if err != nil {
		return uuid.Nil, err
	}
	env, err := insertEvent(ctx, tx, NewEvent{
		GroupID:    groupID,
		StreamID:   accountID,
		EventType:  EventAccountOpened,
		Payload:    payload,
		ActorID:    actorID,
		OccurredAt: occurredAt,
	}, cur+1)
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, ErrConcurrencyConflict
		}
		return uuid.Nil, err
	}
	if err := applyToProjections(ctx, tx, env); err != nil {
		return uuid.Nil, fmt.Errorf("project AccountOpened: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return accountID, nil
}

// TransactionRecordResult reports what RecordTransaction produced.
type TransactionRecordResult struct {
	TransactionID uuid.UUID
	EventID       uuid.UUID
	StreamHead    int64 // final sequence_no on the transaction stream (=1 for v1)
}

// RecordTransaction validates a fully-formed double-entry transaction and
// appends it as one immutable TransactionRecorded event (spec §6 op 2).
//
// Validation happens BEFORE any append: ≥2 entries, balanced debits==credits,
// uniform supported currency, accounts exist / in-group / open.
func (l *Ledger) RecordTransaction(
	ctx context.Context,
	groupID, actorID uuid.UUID,
	occurredAt time.Time,
	memo string,
	entries []Entry,
) (uuid.UUID, error) {
	res, err := l.recordTransactionFull(ctx, recordInput{
		GroupID: groupID, ActorID: actorID, OccurredAt: occurredAt,
		Memo: memo, Entries: entries,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return res.TransactionID, nil
}

type recordInput struct {
	GroupID       uuid.UUID
	ActorID       uuid.UUID
	OccurredAt    time.Time
	Memo          string
	Entries       []Entry
	CausationID   *uuid.UUID // set by reversal flow (stage 5)
	CorrelationID *uuid.UUID
	EventType     EventType // TransactionRecorded or TransactionReversed
}

func (l *Ledger) recordTransactionFull(ctx context.Context, in recordInput) (TransactionRecordResult, error) {
	// Domain validation first — reject, don't record (spec §2 invariant 2).
	if err := ValidateEntries(in.Entries); err != nil {
		return TransactionRecordResult{}, err
	}
	txID := uuid.New()
	payload, err := json.Marshal(TransactionRecordedPayload{
		Memo:    in.Memo,
		Entries: entriesToData(in.Entries),
	})
	if err != nil {
		return TransactionRecordResult{}, err
	}
	eventType := in.EventType
	if eventType == "" {
		eventType = EventTransactionRecorded
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return TransactionRecordResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cur, err := streamHeadQ(ctx, tx, txID)
	if err != nil {
		return TransactionRecordResult{}, err
	}
	if cur != 0 {
		return TransactionRecordResult{}, fmt.Errorf("%w: fresh transaction id collided", ErrConcurrencyConflict)
	}

	// Referential validation against projected account state.
	if _, err := requireAccounts(ctx, tx, in.GroupID, in.Entries); err != nil {
		return TransactionRecordResult{}, err
	}

	env, err := insertEvent(ctx, tx, NewEvent{
		GroupID:       in.GroupID,
		StreamID:      txID,
		EventType:     eventType,
		Payload:       payload,
		ActorID:       in.ActorID,
		OccurredAt:    in.OccurredAt,
		CausationID:   in.CausationID,
		CorrelationID: in.CorrelationID,
	}, 1)
	if err != nil {
		return TransactionRecordResult{}, err
	}
	if err := applyToProjections(ctx, tx, env); err != nil {
		return TransactionRecordResult{}, fmt.Errorf("project %s: %w", eventType, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return TransactionRecordResult{}, err
	}
	return TransactionRecordResult{TransactionID: txID, EventID: env.EventID, StreamHead: 1}, nil
}

func entriesToData(entries []Entry) []EntryData {
	data := make([]EntryData, 0, len(entries))
	for _, e := range entries {
		data = append(data, EntryData{
			AccountName: e.AccountName,
			Direction:   e.Direction,
			AmountMinor: e.Amount.AmountMinor,
			Currency:    e.Amount.Currency,
		})
	}
	return data
}

func dataToEntries(data []EntryData) []Entry {
	entries := make([]Entry, 0, len(data))
	for _, d := range data {
		entries = append(entries, Entry{
			AccountName: d.AccountName,
			Direction:   d.Direction,
			Amount:      Money{AmountMinor: d.AmountMinor, Currency: d.Currency},
		})
	}
	return entries
}

// DeterministicAccountID derives the stable account id for a group+name pair.
func DeterministicAccountID(groupID uuid.UUID, name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(groupID.String() + "|" + name))
}
