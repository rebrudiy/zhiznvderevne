package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".sql" {
			files = append(files, filepath.Join(dir, name))
		}
	}
	if len(files) == 0 {
		return nil
	}
	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)
		var exists bool
		if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("exec migration %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}
	return nil
}
