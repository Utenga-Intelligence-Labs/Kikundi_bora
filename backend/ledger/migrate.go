package ledger

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// Migrate applies the ledger schema idempotently. Safe to call on every boot.
// The guard trigger makes ledger_events truly immutable at the database level:
// any UPDATE or DELETE on that table raises an exception (spec §2 invariant 1).
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("ledger: apply schema: %w", err)
	}
	return nil
}

// CreateGroup registers a savings group scope. Idempotent by name: returns
// the existing group's id when one with this name already exists.
func CreateGroup(ctx context.Context, pool *pgxpool.Pool, name, baseCurrency string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO ledger_groups (name, base_currency) VALUES ($1,$2)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`, name, baseCurrency,
	).Scan(&id)
	return id, err
}
