package internal

import (
	"context"

	"github.com/jackc/pgx"
)

type PgxDBExecutor struct {
	pool *pgx.ConnPool
	ctx  context.Context
}

func NewPgxDBExecutor(pool *pgx.ConnPool) *PgxDBExecutor {
	return &PgxDBExecutor{pool: pool, ctx: context.Background()}
}

func (e *PgxDBExecutor) Exec(query string, args ...interface{}) (int64, error) {
	tag, err := e.pool.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (e *PgxDBExecutor) Query(query string, args ...interface{}) (Rows, error) {
	rows, err := e.pool.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRowsWrapper{rows: rows}, nil
}

func (e *PgxDBExecutor) QueryRow(query string, args ...interface{}) Row {
	return e.pool.QueryRow(query, args...)
}
