package internal

import (
	"github.com/jackc/pgx"
)

type pgxRowsWrapper struct {
	rows *pgx.Rows
}

type pgxRowWrapper struct {
	row *pgx.Row
}

func (w *pgxRowsWrapper) Next() bool {
	return w.rows.Next()
}

func (w *pgxRowsWrapper) Scan(dest ...interface{}) error {
	return w.rows.Scan(dest...)
}

func (w *pgxRowsWrapper) Columns() ([]string, error) {
	fields := w.rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, fd := range fields {
		columns[i] = string(fd.Name)
	}
	return columns, nil
}

func (w *pgxRowsWrapper) Close() error {
	w.rows.Close()
	return nil
}

func (w *pgxRowsWrapper) Err() error {
	return w.rows.Err()
}

func (w *pgxRowsWrapper) Values() ([]interface{}, error) {
	return w.rows.Values()
}
func (r *pgxRowWrapper) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}
