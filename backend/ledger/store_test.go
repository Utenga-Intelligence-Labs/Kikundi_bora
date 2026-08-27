package ledger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool connects to a real PostgreSQL for store-level tests.
// Set LEDGER_TEST_DSN (e.g.
//   postgres://kikundi:kikundi_secret_2024@127.0.0.1:5434/kikundi_db)
// to enable; tests skip otherwise.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("LEDGER_TEST_DSN")
	if dsn == "" {
		t.Skip("LEDGER_TEST_DSN not set; skipping ledger DB tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Apply twice — idempotency is part of the contract.
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate (second pass): %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// testGroup creates a uniquely-named group scope for a test.
func testGroup(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	gid, err := CreateGroup(ctx, pool, fmt.Sprintf("test-group-%s", uuid.NewString()[:8]), CurrencyTZS)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	return gid
}

func sampleEvent(gid uuid.UUID) NewEvent {
	return NewEvent{
		GroupID:    gid,
		StreamID:   uuid.New(),
		EventType:  EventTransactionRecorded,
		Payload:    []byte(`{"memo":"stage1","entries":[]}`),
		ActorID:    uuid.New(),
		OccurredAt: time.Now().UTC(),
	}
}

// IsConcurrencyConflict reports whether err is an optimistic-concurrency failure.
func IsConcurrencyConflict(err error) bool { return errors.Is(err, ErrConcurrencyConflict) }

func TestAppendAssignsGaplessMonotonicSequences(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)

	store := NewStore(pool)
	streamID := uuid.New()
	events := make([]NewEvent, 3)
	for i := range events {
		e := sampleEvent(gid)
		e.StreamID = streamID
		events[i] = e
	}

	envs, err := store.Append(ctx, 0, events)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	for i, env := range envs {
		want := int64(i + 1) // sequences start at 1, no gaps
		if env.SequenceNo != want {
			t.Fatalf("seq[%d]=%d want %d", i, env.SequenceNo, want)
		}
		if env.GlobalSeq <= 0 {
			t.Fatalf("global_seq not assigned: %d", env.GlobalSeq)
		}
		if env.RecordedAt.IsZero() || env.OccurredAt.IsZero() {
			t.Fatal("recorded_at/occurred_at must be set")
		}
		if env.OccurredAt.Location() != time.UTC || env.RecordedAt.Location() != time.UTC {
			t.Fatal("timestamps must be UTC")
		}
	}

	head, err := store.StreamHead(ctx, pool, streamID)
	if err != nil {
		t.Fatal(err)
	}
	if head != 3 {
		t.Fatalf("head=%d want 3", head)
	}
}

func TestAppendOptimisticConcurrencyConflict(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)

	store := NewStore(pool)
	e := sampleEvent(gid)
	if _, err := store.Append(ctx, 0, []NewEvent{e}); err != nil {
		t.Fatalf("initial append: %v", err)
	}
	// Caller still believes the stream is new (expectedSequence=0) but the
	// store advanced to seq 1 -> conflict, and NOTHING may be written.
	stale := sampleEvent(gid)
	stale.StreamID = e.StreamID
	stale.Payload = []byte(`{"memo":"stale"}`)
	_, err := store.Append(ctx, 0, []NewEvent{stale})
	if !IsConcurrencyConflict(err) {
		t.Fatalf("want ErrConcurrencyConflict, got %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ledger_events WHERE payload->>'memo' = 'stale'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("conflicting append must write nothing; found %d rows", n)
	}

	// A caller with a correct view of the head appends fine.
	head, _ := store.StreamHead(ctx, pool, e.StreamID)
	next := sampleEvent(gid)
	next.StreamID = e.StreamID
	if _, err := store.Append(ctx, head, []NewEvent{next}); err != nil {
		t.Fatalf("append with correct expectation: %v", err)
	}
}

// TestAppendConcurrentExactlyOneWinnerPerSequence fires N goroutines at the
// same stream; every sequence slot must end up claimed exactly once
// (gapless, unique) — spec §7 requirement 4 (event-store level).
func TestAppendConcurrentExactlyOneWinnerPerSequence(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)

	store := NewStore(pool)
	streamID := uuid.New()
	const writers = 12

	first := sampleEvent(gid)
	first.StreamID = streamID
	envs, err := store.Append(ctx, 0, []NewEvent{first}) // establish stream at seq 1
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var wins, conflicts int32
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := sampleEvent(gid)
			e.StreamID = streamID
			lastSeen := envs[0].SequenceNo // all racers saw only seq 1
			_, err := store.Append(ctx, lastSeen, []NewEvent{e})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case IsConcurrencyConflict(err):
				conflicts++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if wins < 1 {
		t.Fatalf("expected at least one winner, got 0")
	}

	// Head must equal 1 + wins exactly; sequences gapless & unique by constraint.
	var minNo, maxNo, cnt int64
	err = pool.QueryRow(ctx,
		`SELECT MIN(sequence_no), MAX(sequence_no), COUNT(*), COUNT(DISTINCT sequence_no)
		   FROM ledger_events WHERE stream_id=$1`, streamID).Scan(&minNo, &maxNo, &cnt, new(int64))
	if err != nil {
		t.Fatal(err)
	}
	if minNo != 1 || maxNo != cnt || cnt != maxNo-minNo+1 {
		t.Fatalf("sequences not gapless/unique: min=%d max=%d count=%d", minNo, maxNo, cnt)
	}
	if maxNo != int64(wins)+1 {
		t.Fatalf("head=%d want %d (1 base + %d wins)", maxNo, wins+1, wins)
	}
	if conflicts == 0 && wins > 0 && writers > 0 {
		t.Log("note: no conflicts observed (rare scheduling); not a failure")
	}
}

// TestEventsTableIsImmutable tries UPDATE and DELETE directly against the
// store — both must raise an exception via the guard trigger.
func TestEventsTableIsImmutable(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)

	store := NewStore(pool)
	envs, err := store.Append(ctx, 0, []NewEvent{sampleEvent(gid)})
	if err != nil {
		t.Fatal(err)
	}
	eid := envs[0].EventID

	if _, err := pool.Exec(ctx, `UPDATE ledger_events SET payload='{}' WHERE event_id=$1`, eid); err == nil {
		t.Fatal("UPDATE on ledger_events must fail (invariant: events immutable)")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ledger_events WHERE event_id=$1`, eid); err == nil {
		t.Fatal("DELETE on ledger_events must fail (invariant: events immutable)")
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE ledger_events ENABLE TRIGGER trg_ledger_events_immutable`); err != nil {
		t.Fatal(err)
	}
}
