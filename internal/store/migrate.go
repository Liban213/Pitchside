package store

import (
	"context"
	"embed"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate runs every .sql file in migrations/ against pool, in filename
// order, inside a schema_migrations-tracked transaction per file. It's
// idempotent: files already recorded as applied are skipped.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("migrate: read migrations dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&applied)
		if err != nil {
			return fmt.Errorf("migrate: check %s: %w", name, err)
		}
		if applied {
			continue
		}

		sqlBytes, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate: begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migrate: exec %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migrate: record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", name, err)
		}
	}
	return nil
}
