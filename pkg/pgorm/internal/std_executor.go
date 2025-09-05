package internal

import (
	"context"
	"database/sql"
)

type StdDBExecutor struct {
	db  *sql.DB
	ctx context.Context
}

func NewStdDBExecutor(db *sql.DB) *StdDBExecutor {
	return &StdDBExecutor{db: db, ctx: context.Background()}
}

func (e *StdDBExecutor) Exec(query string, args ...interface{}) (int64, error) {
	res, err := e.db.ExecContext(e.ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (e *StdDBExecutor) Query(query string, args ...interface{}) (Rows, error) {
	return e.db.QueryContext(e.ctx, query, args...)
}

func (e *StdDBExecutor) QueryRow(query string, args ...interface{}) Row {
	return e.db.QueryRowContext(e.ctx, query, args...)
}
