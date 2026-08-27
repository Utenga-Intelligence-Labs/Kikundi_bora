package ledger

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType enumerates every event type the ledger emits.
type EventType string

const (
	EventAccountOpened      EventType = "AccountOpened"
	EventAccountClosed      EventType = "AccountClosed"
	EventTransactionRecorded EventType = "TransactionRecorded"
	EventTransactionReversed EventType = "TransactionReversed"
	EventTransactionAdjusted EventType = "TransactionAdjusted"
)

// EventVersion is the schema version of payloads emitted by this codebase.
const EventVersion = 1

// Envelope is the append-only fact stored in the event store. Every field in
// it is mandatory (causation/correlation nullable per spec §3.3).
type Envelope struct {
	EventID       uuid.UUID
	GroupID       uuid.UUID
	StreamID      uuid.UUID // transaction id or account id
	SequenceNo    int64     // monotonic per stream_id, assigned by Append
	GlobalSeq     int64     // server-assigned bigserial; total order for replay
	EventType     EventType
	EventVersion  int
	Payload       json.RawMessage
	ActorID       uuid.UUID
	OccurredAt    time.Time // business-effective time
	RecordedAt    time.Time // system append time, always server-generated
	CausationID   *uuid.UUID
	CorrelationID *uuid.UUID
}

// NewEvent is an event to be appended: identical to Envelope except that
// SequenceNo, GlobalSeq and RecordedAt are assigned by the store — a client
// can never supply them (spec §2 invariant 5).
type NewEvent struct {
	GroupID       uuid.UUID
	StreamID      uuid.UUID
	EventType     EventType
	Payload       json.RawMessage
	ActorID       uuid.UUID
	OccurredAt    time.Time
	CausationID   *uuid.UUID
	CorrelationID *uuid.UUID
}

// ---- Payload schemas -------------------------------------------------------
// Each payload has its own version for future migrations. v1 = 1.

// AccountOpenedPayload accompanies EventAccountOpened.
type AccountOpenedPayload struct {
	Name          string `json:"name"`
	Type          AccountType `json:"type"`
	Currency      string `json:"currency"`
	OwnerMemberID string `json:"owner_member_id,omitempty"` // bigint member id as string; empty if group-owned
}

// AccountClosedPayload accompanies EventAccountClosed.
type AccountClosedPayload struct {
	AccountName string `json:"account_name"`
	Reason      string `json:"reason,omitempty"`
}

// EntryData is one leg of a transaction as persisted inside a payload.
type EntryData struct {
	AccountName  string `json:"account_name"` // chart-of-accounts name, e.g. "akiba_ya_mwanachama:42"
	Direction    Direction `json:"direction"` // debit | credit
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
}

// TransactionRecordedPayload accompanies EventTransactionRecorded.
type TransactionRecordedPayload struct {
	Memo    string      `json:"memo,omitempty"`
	Entries []EntryData `json:"entries"`
}

// TransactionReversedPayload accompanies EventTransactionReversed.
// The mirrored entries are embedded so the reversal alone reconstructs its legs.
type TransactionReversedPayload struct {
	OriginalEventID uuid.UUID   `json:"original_event_id"`
	Reason          string      `json:"reason,omitempty"`
	Entries         []EntryData `json:"entries"` // mirror of the original entries (debits<->credits swapped)
}
