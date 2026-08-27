package ledger

import "errors"

// Domain errors. Callers should use errors.Is to match.
var (
	// ErrUnbalancedTransaction means sum(debits) != sum(credits) for a transaction.
	ErrUnbalancedTransaction = errors.New("ledger: transaction is not balanced (sum of debits must equal sum of credits)")
	// ErrCurrencyMismatch means entries or accounts mixed currencies (v1 is TZS-only).
	ErrCurrencyMismatch = errors.New("ledger: currency mismatch")
	// ErrUnsupportedCurrency means a currency other than the base currency was used.
	ErrUnsupportedCurrency = errors.New("ledger: unsupported currency")
	// ErrAccountNotFound means a referenced account does not exist in the group.
	ErrAccountNotFound = errors.New("ledger: account not found")
	// ErrAccountClosed means an entry was attempted against a closed account.
	ErrAccountClosed = errors.New("ledger: account is closed")
	// ErrGroupMismatch means an entry referenced an account from another group.
	ErrGroupMismatch = errors.New("ledger: account does not belong to this group")
	// ErrInvalidEntry means a structurally invalid entry (zero/negative amount etc).
	ErrInvalidEntry = errors.New("ledger: invalid entry")
	// ErrTooFewEntries means a transaction had fewer than 2 entries.
	ErrTooFewEntries = errors.New("ledger: transaction requires at least 2 entries")
	// ErrConcurrencyConflict means the expected sequence_no did not match the
	// event store (optimistic concurrency control). Caller may retry.
	ErrConcurrencyConflict = errors.New("ledger: concurrency conflict on event stream")
	// ErrEmptyEvents means Append was called with no events.
	ErrEmptyEvents = errors.New("ledger: append requires at least one event")
	// ErrMixedStreams means events from different streams were appended in one call.
	ErrMixedStreams = errors.New("ledger: all appended events must share one stream_id")
	// ErrTrialBalanceNotZero means the derived trial balance did not net to zero.
	ErrTrialBalanceNotZero = errors.New("ledger: trial balance does not net to zero (critical invariant violation)")
	// ErrUnknownEventType means an unregistered event type was encountered during replay/projection.
	ErrUnknownEventType = errors.New("ledger: unknown event type")
	// ErrTransactionNotFound means no recorded transaction exists under that id.
	ErrTransactionNotFound = errors.New("ledger: transaction not found")
	// ErrAlreadyReversed means this transaction has already been reversed;
	// double reversal is refused (reverse twice == original re-applied).
	ErrAlreadyReversed = errors.New("ledger: transaction already reversed")
)
