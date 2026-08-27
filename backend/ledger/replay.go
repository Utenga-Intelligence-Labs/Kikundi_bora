package ledger

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RebuildProjections wipes derived state for one group (or every group when
// groupID is nil) and re-applies the entire event log from sequence 0 in
// strict global order (spec §4 replay operation). Projections are caches —
// this must always reproduce identical balances (spec §2 invariant 4).
//
// The rebuild runs inside one transaction so readers never observe a
// half-rebuilt read model, and it refuses to commit unless the reconciled
// trial balance nets to zero.
func (l *Ledger) RebuildProjections(ctx context.Context, groupID *uuid.UUID) error {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Wipe derived tables only; the append-only event log is untouched.
	wipes := []string{
		`DELETE FROM ledger_statement_lines`,
		`DELETE FROM ledger_trial_balance`,
		`DELETE FROM ledger_account_balances`,
		`DELETE FROM ledger_accounts`,
	}
	for _, w := range wipes {
		sql := w
		args := []any{}
		if groupID != nil {
			sql += ` WHERE group_id=$1`
			args = append(args, *groupID)
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return fmt.Errorf("replay wipe: %w", err)
		}
	}

	// 2. Stream every event in deterministic global_seq order and re-apply
	//    through the exact same projector path live commands use.
	streamSQL := `
		SELECT event_id, group_id, stream_id, sequence_no, global_seq, event_type,
		       event_version, payload, actor_id, occurred_at, recorded_at,
		       causation_id, correlation_id
		  FROM ledger_events`
	args := []any{}
	if groupID != nil {
		streamSQL += ` WHERE group_id=$1`
		args = append(args, *groupID)
	}
	streamSQL += ` ORDER BY global_seq`

	rows, err := tx.Query(ctx, streamSQL, args...)
	if err != nil {
		return fmt.Errorf("replay stream: %w", err)
	}

	// Buffer all events first — applying via the same tx while the cursor is
	// open would hit "conn busy". Memory cost scales with group history.
	type row struct {
		env Envelope
	}
	events := make([]row, 0, 256)
	err = func() error {
		defer rows.Close()
		for rows.Next() {
			env := Envelope{}
			var et string
			var payload []byte
			if err := rows.Scan(
				&env.EventID, &env.GroupID, &env.StreamID, &env.SequenceNo, &env.GlobalSeq,
				&et, &env.EventVersion, &payload,
				&env.ActorID, &env.OccurredAt, &env.RecordedAt, &env.CausationID, &env.CorrelationID,
			); err != nil {
				return fmt.Errorf("replay scan: %w", err)
			}
			env.EventType = EventType(et)
			env.Payload = payload
			events = append(events, row{env})
		}
		return rows.Err()
	}()
	if err != nil {
		return err
	}

	for _, r := range events {
		if err := applyToProjections(ctx, tx, r.env); err != nil {
			return fmt.Errorf("replay apply event %s/%d: %w", r.env.EventID, r.env.GlobalSeq, err)
		}
	}

	// 3. Assert the rebuilt state reconciles before committing.
	if err := reconcileTx(ctx, tx, groupID); err != nil {
		return fmt.Errorf("replay rejected (would not commit): %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("replay commit: %w", err)
	}
	return nil
}

// reconcileTx asserts Σ w·balance == 0 for the rebuilt scope, in-tx.
func reconcileTx(ctx context.Context, tx pgx.Tx, groupID *uuid.UUID) error {
	sql := `
		SELECT COALESCE(SUM(b.balance_minor *
		            CASE WHEN a.type IN ('asset','expense') THEN 1 ELSE -1 END),0)
		  FROM ledger_account_balances b
		  JOIN ledger_accounts a USING (group_id, account_name)`
	args := []any{}
	if groupID != nil {
		sql += ` WHERE b.group_id=$1`
		args = append(args, *groupID)
	}
	var wsum int64
	if err := tx.QueryRow(ctx, sql, args...).Scan(&wsum); err != nil {
		return err
	}
	if wsum != 0 {
		return fmt.Errorf("weighted balances sum=%d: %w", wsum, ErrTrialBalanceNotZero)
	}
	return nil
}

