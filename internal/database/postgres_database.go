package database

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
)

type Row interface {
	Scan(dest ...any) error
}

type DB interface {
	QueryRowStruct(ctx context.Context, dest any, sql string, args ...any) error
	QueryStruct(ctx context.Context, dest any, sql string, args ...any) error
	Close(ctx context.Context) error
}

type PostgresDB struct {
	conn *pgx.Conn
}

func NewPostgres(ctx context.Context, dsn string) (DB, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &PostgresDB{conn: conn}, nil
}

func (db *PostgresDB) Close(ctx context.Context) error {
	return db.conn.Close(ctx)
}

func (db *PostgresDB) QueryRowStruct(ctx context.Context, dest any, sql string, args ...any) error {
	rows, err := db.conn.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	return pgxscan.ScanRow(dest, rows)
}

func (db *PostgresDB) QueryStruct(ctx context.Context, dest interface{}, sql string, args ...any) error {
	rows, err := db.conn.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return pgxscan.ScanAll(dest, rows)
}