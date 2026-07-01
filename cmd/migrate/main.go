// Command migrate applies (or rolls back) the SQL files in migrations/ in
// lexical order, tracking what's applied in a schema_migrations table.
// Usage: go run ./cmd/migrate [up|down]
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"reporting-service/configs"
)

func main() {
	_ = godotenv.Load()

	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if direction != "up" && direction != "down" {
		fmt.Fprintln(os.Stderr, "usage: migrate [up|down]")
		os.Exit(1)
	}

	cfg, err := configs.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect error:", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap error:", err)
		os.Exit(1)
	}

	files, err := migrationFiles(direction)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read migrations error:", err)
		os.Exit(1)
	}

	applied, err := appliedVersions(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read applied versions error:", err)
		os.Exit(1)
	}

	for _, f := range files {
		version := versionOf(f)
		alreadyApplied := applied[version]

		if direction == "up" && alreadyApplied {
			continue
		}
		if direction == "down" && !alreadyApplied {
			continue
		}

		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read file error:", err)
			os.Exit(1)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "begin tx error:", err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			fmt.Fprintln(os.Stderr, "apply", f, "error:", err)
			os.Exit(1)
		}

		if direction == "up" {
			_, err = tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version)
		} else {
			_, err = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			fmt.Fprintln(os.Stderr, "record migration error:", err)
			os.Exit(1)
		}

		if err := tx.Commit(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "commit error:", err)
			os.Exit(1)
		}
		fmt.Println(direction, version)
	}
}

func migrationFiles(direction string) ([]string, error) {
	entries, err := filepath.Glob(filepath.Join("migrations", "*."+direction+".sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	if direction == "down" {
		sort.Sort(sort.Reverse(sort.StringSlice(entries)))
	}
	return entries, nil
}

func appliedVersions(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func versionOf(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")
	return base
}
