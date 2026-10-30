package internal

import (
	"context"
	"database/sql"
)

type StdDBExecutor struct {
	db  *sql.DB
	ctx context.Context
}

type sqlRowWrapper struct {
	row *sql.Row
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
	rows, err := e.db.QueryContext(e.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsWrapper{rows: rows}, nil
}

func (e *StdDBExecutor) QueryRow(query string, args ...interface{}) Row {
	return &sqlRowWrapper{row: e.db.QueryRowContext(e.ctx, query, args...)}
}

func (r *sqlRowWrapper) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}
