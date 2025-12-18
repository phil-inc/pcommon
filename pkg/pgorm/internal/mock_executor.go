package internal

import "database/sql"

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

func (m *MockDBExecutor) QueryRow(query string, args ...interface{}) Row {
	return m.DB.QueryRow(query, args...)
}
