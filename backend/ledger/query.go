package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// StatementLine is one denormalized entry rendered for member statements.
// Note: reversal events appear as their own lines (never an edit of history).
type StatementLine struct {
	TransactionID uuid.UUID
	EventType     EventType
	Direction     Direction
	AmountMinor   int64
	Currency      string
	OccurredAt    time.Time // business-effective ordering key
	Memo          string
}

const sqlAccountState = `
	SELECT COALESCE(b.balance_minor,0), a.type, a.currency
	  FROM ledger_accounts a
	  LEFT JOIN ledger_account_balances b USING (group_id, account_name)
	 WHERE a.group_id=$1 AND a.account_name=$2`

// GetBalance reads the projected balance for one account. With asOf==nil it
// serves the cached value tagged with its event offset. With asOf set, it
// derives the historical balance from statement lines up to that business
// time (v1 substitutes point-in-time replay over denormalized lines until
// snapshots land; the semantics are identical because lines are append-only).
//
// Returned money is normal-side signed: liabilities/income/equity credit
// growth is positive.
func (l *Ledger) GetBalance(
	ctx context.Context,
	groupID uuid.UUID,
	accountName string,
	asOf *time.Time,
) (Money, error) {
	q := poolQuerier(l.pool)
	var balRaw int64
	var typ, cur string

	if asOf == nil {
		err := q.QueryRow(ctx, sqlAccountState, groupID, accountName).Scan(&balRaw, &typ, &cur)
		if err == pgx.ErrNoRows {
			return Money{}, fmt.Errorf("%s: %w", accountName, ErrAccountNotFound)
		}
		if err != nil {
			return Money{}, err
		}
	} else {
		row := q.QueryRow(ctx, `
			SELECT COALESCE(SUM(CASE WHEN sl.direction='debit' THEN sl.amount_minor ELSE -sl.amount_minor END),0)
			  FROM ledger_statement_lines sl
			 WHERE sl.group_id=$1 AND sl.account_name=$2 AND sl.occurred_at <= $3`,
			groupID, accountName, asOf.UTC())
		if err := row.Scan(&balRaw); err != nil {
			return Money{}, err
		}
		row2 := q.QueryRow(ctx,
			`SELECT type, currency FROM ledger_accounts WHERE group_id=$1 AND account_name=$2`,
			groupID, accountName)
		switch err := row2.Scan(&typ, &cur); {
		case err == pgx.ErrNoRows:
			return Money{}, fmt.Errorf("%s: %w", accountName, ErrAccountNotFound)
		case err != nil:
			return Money{}, err
		}
		// Derived raw sum (debits +, credits −) needs normalization to the
		// positive-on-normal-side convention of cached values.
		if !(AccountType(typ)).DebitPositive() {
			balRaw = -balRaw
		}
	}
	return Money{AmountMinor: balRaw, Currency: cur}, nil
}

// GetAsOfGlobalSeq reports the event offset a cached balance was computed at
// (proof hook for spec §2 invariant 3: caches are reconstructible).
func (l *Ledger) GetAsOfGlobalSeq(ctx context.Context, groupID uuid.UUID, accountName string) (int64, error) {
	var seq int64
	err := poolQuerier(l.pool).QueryRow(ctx,
		`SELECT b.as_of_global_seq FROM ledger_account_balances b
		  JOIN ledger_accounts a USING (group_id, account_name)
		 WHERE a.group_id=$1 AND a.account_name=$2`, groupID, accountName,
	).Scan(&seq)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("%s: %w", accountName, ErrAccountNotFound)
	}
	return seq, err
}

// GetLedgerEntries renders statement lines for one account within the
// inclusive business-time window [from, to], chronologically stable.
func (l *Ledger) GetLedgerEntries(
	ctx context.Context,
	groupID uuid.UUID,
	accountName string,
	from, to time.Time,
) ([]StatementLine, error) {
	rows, err := poolQuerier(l.pool).Query(ctx, `
		SELECT transaction_id, event_type, direction, amount_minor, currency, occurred_at, memo
		  FROM ledger_statement_lines
		 WHERE group_id=$1 AND account_name=$2 AND occurred_at >= $3 AND occurred_at <= $4
		 ORDER BY occurred_at, line_id`, groupID, accountName, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]StatementLine, 0)
	for rows.Next() {
		var ln StatementLine
		var et, dir string
		if err := rows.Scan(&ln.TransactionID, &et, &dir, &ln.AmountMinor, &ln.Currency, &ln.OccurredAt, &ln.Memo); err != nil {
			return nil, err
		}
		ln.EventType = EventType(et)
		ln.Direction = Direction(dir)
		lines = append(lines, ln)
	}
	return lines, rows.Err()
}

// TrialBalanceLine aggregates one account's debits/credits up to a point.
type TrialBalanceLine struct {
	AccountName string
	Type        AccountType
	DebitMinor  int64
	CreditMinor int64
}

// TrialBalance is the group-wide audit snapshot proving debits==credits.
type TrialBalance struct {
	GroupID           uuid.UUID
	AsOf              *time.Time
	Lines             []TrialBalanceLine
	TotalDebitMinor   int64
	TotalCreditMinor  int64
	Balanced          bool
}

// GetTrialBalance computes the trial balance from statement lines (asOf=nil
// means all recorded activity). It nets to zero whenever all invariants held;
// a nonzero result is returned WITH ErrTrialBalanceNotZero as a critical bug
// alarm, never silently displayed (spec §6 op 6).
func (l *Ledger) GetTrialBalance(
	ctx context.Context,
	groupID uuid.UUID,
	asOf *time.Time,
) (TrialBalance, error) {
	tb := TrialBalance{GroupID: groupID, AsOf: asOf, Lines: []TrialBalanceLine{}}

	baseSQL := `
		SELECT a.account_name, a.type,
		       COALESCE(SUM(CASE WHEN sl.direction='debit' THEN sl.amount_minor END),0),
		       COALESCE(SUM(CASE WHEN sl.direction='credit' THEN sl.amount_minor END),0)
		  FROM ledger_statement_lines sl
		  JOIN ledger_accounts a USING (group_id, account_name)`
	args := []any{groupID}
	if asOf == nil {
		baseSQL += ` WHERE sl.group_id=$1`
	} else {
		baseSQL += ` WHERE sl.group_id=$1 AND sl.occurred_at <= $2`
		args = append(args, asOf.UTC())
	}
	baseSQL += ` GROUP BY a.account_name, a.type ORDER BY a.account_name`

	rows, err := poolQuerier(l.pool).Query(ctx, baseSQL, args...)
	if err != nil {
		return tb, err
	}
	defer rows.Close()
	for rows.Next() {
		var tl TrialBalanceLine
		var typ string
		if err := rows.Scan(&tl.AccountName, &typ, &tl.DebitMinor, &tl.CreditMinor); err != nil {
			return tb, err
		}
		tl.Type = AccountType(typ)
		tb.TotalDebitMinor += tl.DebitMinor
		tb.TotalCreditMinor += tl.CreditMinor
		tb.Lines = append(tb.Lines, tl)
	}
	if err := rows.Err(); err != nil {
		return tb, err
	}

	tb.Balanced = tb.TotalDebitMinor == tb.TotalCreditMinor && len(tb.Lines) > 0
	if !tb.Balanced {
		return tb, fmt.Errorf("group %s trial balance: debits=%d credits=%d: %w",
			groupID, tb.TotalDebitMinor, tb.TotalCreditMinor, ErrTrialBalanceNotZero)
	}
	return tb, nil
}

// CheckGroupReconciliation verifies the weighted-balance invariant across ALL
// projected balances of a group (Σ w·balance == 0). Zero means the chart of
// accounts still reconciles; anything else signals projection drift or a
// broken write path.
func (l *Ledger) CheckGroupReconciliation(ctx context.Context, groupID uuid.UUID) error {
	var wsum int64
	err := poolQuerier(l.pool).QueryRow(ctx, `
		SELECT COALESCE(SUM(b.balance_minor *
		            CASE WHEN a.type IN ('asset','expense') THEN 1 ELSE -1 END),0)
		  FROM ledger_account_balances b
		  JOIN ledger_accounts a USING (group_id, account_name)
		 WHERE b.group_id=$1`, groupID).Scan(&wsum)
	if err != nil {
		return err
	}
	if wsum != 0 {
		return fmt.Errorf("weighted balances sum=%d: %w", wsum, ErrTrialBalanceNotZero)
	}
	return nil
}

func poolQuerier(p PgxPool) RowsQuerier {
	type qr interface {
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
		Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	}
	return p.(qr)
}
