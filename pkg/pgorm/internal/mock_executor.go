package internal

import "database/sql"

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

func (r *sqlRowsWrapper) Values() ([]interface{}, error) {
	columns, err := r.rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	if err := r.rows.Scan(pointers...); err != nil {
		return nil, err
	}

	return values, nil
}
