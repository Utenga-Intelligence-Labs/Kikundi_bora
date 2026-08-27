package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is the common query surface shared by *pgxpool.Pool, pgx.Tx and
// pool.Acquire()-conns. It lets the same code run inside or outside a tx.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store is the append-only PostgreSQL event store. It issues INSERT-only SQL
// against the events table — no UPDATE or DELETE path exists anywhere in this
// file (spec §2 invariant 1). Raw pgx, not GORM, deliberately.
type Store struct {
	pool PgxPool
}

// PgxPool is the minimal pgx surface the store needs (*pgxpool.Pool satisfies it).
type PgxPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// NewStore returns an event store backed by the given pool.
func NewStore(pool PgxPool) *Store { return &Store{pool: pool} }

// streamHeadTx reads MAX(sequence_no) inside the append transaction.
func streamHeadTx(ctx context.Context, q Querier, streamID uuid.UUID) (int64, error) {
	var cur int64
	err := q.QueryRow(ctx,
		`SELECT COALESCE(MAX(sequence_no),0) FROM ledger_events WHERE stream_id=$1`, streamID,
	).Scan(&cur)
	return cur, err
}

// StreamHead returns the highest sequence_no currently stored for a stream
// (0 when unknown/empty). Useful for callers building expectedSequence.
func (s *Store) StreamHead(ctx context.Context, q Querier, streamID uuid.UUID) (int64, error) {
	return streamHeadTx(ctx, q, streamID)
}

// Append appends events to a single stream with optimistic concurrency:
// expectedSequence asserts the caller's view of the stream head — use 0 for a
// brand-new stream. If the store's actual head differs from what the caller
// observed, ErrConcurrencyConflict is returned and nothing is written.
//
// Sequence numbers start at 1 and are strictly gapless per stream: they are
// allocated from MAX(sequence_no)+1 observed under the insert transaction's
// snapshot, with UNIQUE(stream_id, sequence_no) as the final arbiter.
//
// recorded_at is always server-generated here (spec §2 invariant 5).
func (s *Store) Append(ctx context.Context, expectedSequence int64, events []NewEvent) ([]Envelope, error) {
	if len(events) == 0 {
		return nil, ErrEmptyEvents
	}
	for _, e := range events {
		if e.StreamID != events[0].StreamID {
			return nil, ErrMixedStreams
		}
		if e.GroupID != events[0].GroupID {
			return nil, fmt.Errorf("ledger: all appended events must share one group_id")
		}
	}
	streamID := events[0].StreamID

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ledger: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cur, err := streamHeadTx(ctx, tx, streamID)
	if err != nil {
		return nil, fmt.Errorf("ledger: read stream head: %w", err)
	}
	if expectedSequence > 0 && expectedSequence != cur {
		return nil, fmt.Errorf("%w: caller expected seq %d, store at %d",
			ErrConcurrencyConflict, expectedSequence, cur)
	}
	if expectedSequence == 0 && cur > 0 {
		return nil, fmt.Errorf("%w: caller assumed new stream but store is at seq %d",
			ErrConcurrencyConflict, cur)
	}

	out := make([]Envelope, 0, len(events))
	now := time.Now().UTC()
	for i, e := range events {
		env := Envelope{
			EventID:       uuid.New(),
			GroupID:       e.GroupID,
			StreamID:      e.StreamID,
			SequenceNo:    cur + int64(i+1),
			GlobalSeq:     0, // assigned by bigserial on insert, read back below
			EventType:     e.EventType,
			EventVersion:  EventVersion,
			Payload:       e.Payload,
			ActorID:       e.ActorID,
			OccurredAt:    e.OccurredAt.UTC(),
			RecordedAt:    now,
			CausationID:   e.CausationID,
			CorrelationID: e.CorrelationID,
		}
		err := tx.QueryRow(ctx,
			`INSERT INTO ledger_events
			   (event_id, group_id, stream_id, sequence_no, event_type, event_version,
			    payload, actor_id, occurred_at, recorded_at, causation_id, correlation_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			 RETURNING global_seq`,
			env.EventID, env.GroupID, env.StreamID, env.SequenceNo,
			string(env.EventType), env.EventVersion, []byte(env.Payload),
			env.ActorID, env.OccurredAt, env.RecordedAt, env.CausationID, env.CorrelationID,
		).Scan(&env.GlobalSeq)
		if err != nil {
			// Unique violation on (stream_id, sequence_no) => lost race => OCC conflict.
			var pgErr *pgconn.PgError
			if errorsAs(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
				return nil, ErrConcurrencyConflict
			}
			return nil, fmt.Errorf("ledger: append event %s: %w", env.EventType, err)
		}
		out = append(out, env)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ledger: commit: %w", err)
	}
	return out, nil
}

const pgErrUniqueViolation = "23505"

func errorsAs(err error, target **pgconn.PgError) bool {
	for err != nil {
		if e, ok := err.(*pgconn.PgError); ok {
			*target = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
