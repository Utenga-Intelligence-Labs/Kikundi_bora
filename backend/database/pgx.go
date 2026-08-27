package database

import (
	"context"
	"fmt"
	"time"

	"kikundibora/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPool is the connection pool used by packages that need raw SQL
// (the ledger event store) instead of GORM.
var Pgx *pgxpool.Pool

// ConnectPgx opens a pgx pool from the same DB_* configuration GORM uses
// and stores it in Pgx. Safe to call alongside Connect().
func ConnectPgx(ctx context.Context) (*pgxpool.Pool, error) {
	if Pgx != nil {
		return Pgx, nil
	}
	cfg := config.AppConfig
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx parse config: %w", err)
	}
	poolCfg.MaxConns = 20
	poolCfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pgx connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}
	Pgx = pool
	return pool, nil
}

// ClosePgx releases the pgx pool (call on shutdown).
func ClosePgx() {
	if Pgx != nil {
		Pgx.Close()
		Pgx = nil
	}
}
