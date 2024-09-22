package dbo

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"
)

//go:generate sqlc generate
//go:embed migrations
var migrations embed.FS

// Dial to database and apply migrations.
func Dial(ctx context.Context, dbURL string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	// apply schema
	_, err = migrate.ExecContext(ctx, db, "sqlite3", &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "migrations",
	}, migrate.Up)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return db, nil
}

func (q *Queries) WithTransaction(ctx context.Context, fn func(*Queries) error) error {
	type transaction interface {
		BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	}
	db, ok := q.db.(transaction)
	if !ok {
		return fmt.Errorf("connection doesn't support transaction, most likely you are trying to execute nested transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	err = fn(q.WithTx(tx))
	if err != nil {
		return errors.Join(
			err,
			tx.Rollback(), // try to rollback; save the error
		)
	}
	return tx.Commit()
}
