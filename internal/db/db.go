// Package db opens the Postgres database and applies its migrations. The
// SQL is plain, versioned in migrations/ and embedded in the binary.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URLEnv is the environment variable with the connection string. It holds
// the database password: it is never printed and never goes in a flag,
// where the shell history and the process list would show it.
const URLEnv = "GARIMPO_DATABASE_URL"

//go:embed migrations/*.sql
var migrations embed.FS

// lockID serializes the writers that need the whole picture (migrations,
// a run, an import). Any fixed number works; this one spells "garimpo".
const lockID = 0x67617269_6d706f

// Lock waits for the exclusive use of the database until the transaction
// ends.
func Lock(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "select pg_advisory_xact_lock($1)", int64(lockID))
	return err
}

// Open connects to the database at url and checks that it answers.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		// The parse error could quote the URL, password included.
		return nil, fmt.Errorf("%s não é uma URL de Postgres válida (o formato é postgres://usuario:senha@host:5432/banco)", URLEnv)
	}
	// The Supabase pooler shares server connections between clients, so
	// prepared statements do not survive from one query to the next. This
	// mode never prepares.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	// A person and a daily job: the free plan has few connections to give.
	cfg.MaxConns = 4
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("não consegui preparar a conexão com o banco: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("o banco não respondeu: %w", err)
	}
	return pool, nil
}

// Migrate applies the migrations that the database does not have yet, in
// file name order, each in its own transaction. It returns the names it
// applied.
func Migrate(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	if _, err := pool.Exec(ctx, `create table if not exists schema_migrations (
		version text primary key,
		applied_at timestamptz not null default now())`); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, "alter table schema_migrations enable row level security"); err != nil {
		return nil, err
	}
	var applied []string
	for _, name := range names {
		version := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".sql")
		sql, err := migrations.ReadFile(name)
		if err != nil {
			return applied, err
		}
		done, err := migrateOne(ctx, pool, version, string(sql))
		if err != nil {
			return applied, fmt.Errorf("migração %s: %w", version, err)
		}
		if done {
			applied = append(applied, version)
		}
	}
	return applied, nil
}

func migrateOne(ctx context.Context, pool *pgxpool.Pool, version, sql string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if err := Lock(ctx, tx); err != nil {
		return false, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, "select exists (select 1 from schema_migrations where version = $1)", version).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	// A migration file has many statements; only the simple protocol
	// accepts them in one call.
	if _, err := tx.Exec(ctx, sql, pgx.QueryExecModeSimpleProtocol); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, "insert into schema_migrations (version) values ($1)", version); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// ErrNotMigrated is returned by Check when the tables are missing.
var ErrNotMigrated = errors.New("o banco ainda não tem as tabelas do Garimpo (rode: garimpo migrate)")

// Check reports whether every embedded migration was applied.
func Check(ctx context.Context, pool *pgxpool.Pool) error {
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	var have int
	err = pool.QueryRow(ctx, "select count(*) from schema_migrations").Scan(&have)
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == "42P01": // undefined_table: nothing applied yet
		return ErrNotMigrated
	case err != nil:
		return err
	case have < len(names):
		return ErrNotMigrated
	}
	return nil
}
