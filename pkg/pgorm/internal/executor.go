package internal

type Row interface {
	Scan(dest ...interface{}) error
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
	Close() error
	Err() error
	Values() ([]interface{}, error)
}

type DBExecutor interface {
	Exec(query string, args ...interface{}) (int64, error)
	Query(query string, args ...interface{}) (Rows, error)
	QueryRow(query string, args ...interface{}) Row
}
