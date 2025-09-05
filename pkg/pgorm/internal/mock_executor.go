package internal

import (
	"database/sql"
)

type MockDBExecutor struct {
	DB *sql.DB
}

func (m *MockDBExecutor) Exec(query string, args ...interface{}) (int64, error) {
	res, err := m.DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (m *MockDBExecutor) Query(query string, args ...interface{}) (Rows, error) {
	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsWrapper{rows}, nil
}

type sqlRowsWrapper struct {
	rows *sql.Rows
}

func (r *sqlRowsWrapper) Next() bool {
	return r.rows.Next()
}

func (r *sqlRowsWrapper) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRowsWrapper) Columns() ([]string, error) {
	return r.rows.Columns()
}

func (r *sqlRowsWrapper) Close() error {
	return r.rows.Close()
}

func (r *sqlRowsWrapper) Err() error {
	return r.rows.Err()
}

func (m *MockDBExecutor) QueryRow(query string, args ...interface{}) Row {
	return m.DB.QueryRow(query, args...)
}
