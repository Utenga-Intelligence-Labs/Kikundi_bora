package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Executor is anything that can run SQL: *pgxpool.Pool, pgx.Tx, pool conn.
type Executor interface {
	pgxQuerier
	pgxExecer
}

type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type pgxExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// applyToProjections folds ONE event into the read models. It must be
// idempotent when replayed onto truncated tables (Replay truncates first),
// and incremental-safe when applied live inside a command transaction.
//
// Balance rows are tagged with as_of_global_seq so cached values are always
// reconstructible/provable (spec §2 invariant 3).
func applyToProjections(ctx context.Context, ex Executor, env Envelope) error {
	switch env.EventType {
	case EventAccountOpened:
		return projectAccountOpened(ctx, ex, env)
	case EventAccountClosed:
		return projectAccountClosed(ctx, ex, env)
	case EventTransactionRecorded, EventTransactionReversed, EventTransactionAdjusted:
		return projectTransaction(ctx, ex, env)
	default:
		return fmt.Errorf("%w: %q", ErrUnknownEventType, env.EventType)
	}
}

func projectAccountOpened(ctx context.Context, ex Executor, env Envelope) error {
	var p AccountOpenedPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("decode AccountOpened payload: %w", err)
	}
	_, err := ex.Exec(ctx,
		`INSERT INTO ledger_accounts
		   (group_id, account_name, type, currency, owner_member_ref, opened_at_seq, opened_at)
		 VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7)
		 ON CONFLICT (group_id, account_name) DO NOTHING`,
		env.GroupID, p.Name, string(p.Type), p.Currency, p.OwnerMemberID, env.GlobalSeq, env.RecordedAt)
	return err
}

func projectAccountClosed(ctx context.Context, ex Executor, env Envelope) error {
	var p AccountClosedPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("decode AccountClosed payload: %w", err)
	}
	name := accountNameFromClose(env)
	if name == "" {
		return fmt.Errorf("AccountClosed payload missing account_name")
	}
	_, err := ex.Exec(ctx,
		`UPDATE ledger_accounts
		    SET closed_at_seq=$2, closed_at=$3
		  WHERE account_name=$1 AND closed_at_seq IS NULL`,
		name, env.GlobalSeq, env.RecordedAt)
	return err
}

func accountNameFromClose(env Envelope) string {
	var p AccountClosedPayload
	_ = json.Unmarshal(env.Payload, &p) // unmarshal already validated above
	return p.AccountName
}

// bumpBalance updates the cached balance. balance_minor is stored positive-on-
// normal-side (assets/expenses grow on debit, others on credit); as_of_global_seq
// tags the exact event offset the cache reflects (spec §2 invariant 3).
func bumpBalance(ctx context.Context, ex Executor, groupID uuid.UUID, name, currency string, deltaMinor int64, globalSeq int64) error {
	const upsert = `INSERT INTO ledger_account_balances
	                    (group_id, account_name, balance_minor, currency, as_of_global_seq)
	                VALUES ($1,$2,$3,$4,$5)
	                ON CONFLICT (group_id, account_name) DO UPDATE SET
	                    balance_minor    = ledger_account_balances.balance_minor + EXCLUDED.balance_minor,
	                    as_of_global_seq = EXCLUDED.as_of_global_seq`
	_, err := ex.Exec(ctx, upsert, groupID, name, deltaMinor, currency, globalSeq)
	return err
}

func projectTransaction(ctx context.Context, ex Executor, env Envelope) error {
	var p TransactionRecordedPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("decode %s payload: %w", env.EventType, err)
	}
	// Resolve each leg's account type for correct weighting + statement line.
	type acctMeta struct{ typ, cur string }
	metas := make(map[string]acctMeta, len(p.Entries))
	for _, e := range p.Entries {
		if _, ok := metas[e.AccountName]; !ok {
			var typ, cur string
			if err := ex.QueryRow(ctx,
				`SELECT type, currency FROM ledger_accounts WHERE group_id=$1 AND account_name=$2`,
				env.GroupID, e.AccountName,
			).Scan(&typ, &cur); err != nil {
				return fmt.Errorf("resolve account %s: %w", e.AccountName, err)
			}
			metas[e.AccountName] = acctMeta{typ, cur}
		}
	}

	for _, e := range p.Entries {
		m := metas[e.AccountName]
		t := AccountType(m.typ)
		signed := NormalizedDelta(t, e.Direction, e.AmountMinor)
		if err := bumpBalance(ctx, ex, env.GroupID, e.AccountName, m.cur, signed, env.GlobalSeq); err != nil {
			return fmt.Errorf("bump balance %s: %w", e.AccountName, err)
		}
		_, err := ex.Exec(ctx,
			`INSERT INTO ledger_statement_lines
			   (group_id, account_name, transaction_id, event_type, direction,
			    amount_minor, currency, occurred_at, memo, actor_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 ON CONFLICT DO NOTHING`, // idempotent under unique(tx,event,acct,direction)
			env.GroupID, e.AccountName, env.StreamID, string(env.EventType),
			string(e.Direction), e.AmountMinor, e.Currency, env.OccurredAt, p.Memo, env.ActorID)
		if err != nil {
			return fmt.Errorf("insert statement line %s: %w", e.AccountName, err)
		}
	}

	// Trial-balance checkpoint per event: total debits vs credits.
	var td, tc int64
	err := ex.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN direction='debit' THEN amount_minor END),0),
		        COALESCE(SUM(CASE WHEN direction='credit' THEN amount_minor END),0)
		   FROM ledger_statement_lines WHERE group_id=$1`, env.GroupID,
	).Scan(&td, &tc)
	if err != nil {
		return fmt.Errorf("trial balance aggregate: %w", err)
	}
	_, err = ex.Exec(ctx,
		`INSERT INTO ledger_trial_balance
		   (group_id, total_debit_minor, total_credit_minor, net_minor, as_of_global_seq)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (group_id, as_of_global_seq) DO NOTHING`,
		env.GroupID, td, tc, td-tc, env.GlobalSeq)
	return err
}
