package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, files fs.FS) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := fs.Glob(files, "*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.TrimSuffix(name, ".up.sql")
		if err := applyMigration(ctx, pool, files, name, version); err != nil {
			return fmt.Errorf("migration %s: %w", version, err)
		}
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, files fs.FS, name, version string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var applied bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
	).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return nil
	}

	sql, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
