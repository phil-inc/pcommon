package internal

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// Mock model for testing
type MockModel struct {
	ID    int    `db:"id"`
	Name  string `db:"name,omitempty"`
	Email string `db:"email,omitempty"`
}

func (m MockModel) TableName() string {
	return "mock_table"
}

func TestExtractColumnsAndValues(t *testing.T) {
	model := MockModel{ID: 1, Name: "", Email: "test@example.com"}
	columns, values, placeholders := extractColumnsAndValues(model)

	expectedColumns := []string{"id", "email"}
	expectedValues := []interface{}{1, "test@example.com"}
	expectedPlaceholders := []string{"$1", "$2"}

	if !reflect.DeepEqual(columns, expectedColumns) {
		t.Errorf("expected columns %v, got %v", expectedColumns, columns)
	}
	if !reflect.DeepEqual(values, expectedValues) {
		t.Errorf("expected values %v, got %v", expectedValues, values)
	}
	if !reflect.DeepEqual(placeholders, expectedPlaceholders) {
		t.Errorf("expected placeholders %v, got %v", expectedPlaceholders, placeholders)
	}
}

func TestReplacePlaceholders(t *testing.T) {
	condition := "name = ? AND email = ?"
	expected := "name = $2 AND email = $3"

	result := replacePlaceholders(condition, 1)

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	executor := &MockDBExecutor{DB: db}
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO mock_table (id, email) VALUES ($1, $2)`)).
		WithArgs(1, "test@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	queryBuilder := NewQueryBuilder(executor)
	model := MockModel{ID: 1, Name: "", Email: "test@example.com"}

	result, err := queryBuilder.Table(model).Insert(model)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.RowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", result.RowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations were not met: %v", err)
	}
}

func TestInsertWithReturning(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	model := MockModel{ID: 1, Name: "john", Email: "test@example.com"}

	// Columns and values for the query
	columns := []string{"id", "name", "email"}
	values := []interface{}{1, "john", "test@example.com"}
	placeholders := []string{"$1", "$2", "$3"}

	// Expected query with RETURNING clause
	query := fmt.Sprintf("INSERT INTO mock_table (%s) VALUES (%s) RETURNING id, name, email",
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Simulating the result to be returned from the query
	rows := sqlmock.NewRows([]string{"id", "name", "email"}).
		AddRow(1, "john", "test@example.com") // Simulated returned row

	driverValues := make([]driver.Value, len(values))
	for i, v := range values {
		driverValues[i] = v
	}

	// Define mock expectation with RETURNING
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(driverValues...).
		WillReturnRows(rows)

	// Initialize the query builder
	executor := &MockDBExecutor{DB: db}
	queryBuilder := NewQueryBuilder(executor)

	// Perform the Insert with returning
	result, err := queryBuilder.Table(model).Returning(model, "*").Insert(model)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Assert that rows were affected
	if result.RowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", result.RowsAffected)
	}

	// Assert the returned data from the RETURNING clause
	if result.Returning["id"].(int64) != 1 {
		t.Errorf("expected id 1, got %v", result.Returning["id"])
	}
	if result.Returning["name"] != "john" {
		t.Errorf("expected name 'john' , got %v", result.Returning["name"])
	}
	if result.Returning["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %v", result.Returning["email"])
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations were not met: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE mock_table SET id = $1 WHERE id = $2`)).
		WithArgs(2, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	executor := &MockDBExecutor{DB: db}
	queryBuilder := NewQueryBuilder(executor)
	model := MockModel{ID: 2}

	queryBuilder.Table(model).Set(model).Where("id = ?", 1)
	result, err := queryBuilder.Update()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.RowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", result.RowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations were not met: %v", err)
	}
}

func TestUpdateWithReturning(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	// Define the expected SQL query for returning columns
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE mock_table SET id = $1, name = $2, email = $3 WHERE id = $1 RETURNING id, name, email`)).
		WithArgs(1, "john", "test@example.com", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
			AddRow(1, "john", "test@example.com")) // Simulating the returned columns

		// Instantiate your QueryBuilder or whatever object you're using
	executor := &MockDBExecutor{DB: db}
	queryBuilder := NewQueryBuilder(executor)
	model := MockModel{ID: 1, Name: "john", Email: "test@example.com"}

	// Perform the update operation
	result, err := queryBuilder.Table(model).Where("id = ?", 1).Set(model).Returning(model, "*").Update()

	// Check for any unexpected errors
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Assert the rows affected
	if result.RowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", result.RowsAffected)
	}

	// Assert the returned values for the updated columns
	if result.Returning["id"].(int64) != 1 {
		t.Errorf("expected id 1, got %v", result.Returning["id"])
	}

	if result.Returning["name"] != "john" {
		t.Errorf("expected name 'john', got %v", result.Returning["name"])
	}

	if result.Returning["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %v", result.Returning["email"])
	}

	// Ensure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations were not met: %v", err)
	}
}

func TestDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM mock_table WHERE id = $1`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	executor := &MockDBExecutor{DB: db}
	queryBuilder := NewQueryBuilder(executor)
	model := MockModel{}

	queryBuilder.Table(model).Where("id = ?", 1)
	result, err := queryBuilder.Delete()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.RowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", result.RowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations were not met: %v", err)
	}
}

func TestSelect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	row := sqlmock.NewRows([]string{"id", "name", "email"}).
		AddRow(1, "John Doe", "john.doe@example.com")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM mock_table WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(row)

	executor := &MockDBExecutor{DB: db}
	queryBuilder := NewQueryBuilder(executor)
	model := MockModel{}

	queryBuilder.Table(model).Where("id = ?", 1)
	result, err := queryBuilder.Select()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := []map[string]interface{}{
		{"id": int64(1), "name": "John Doe", "email": "john.doe@example.com"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations were not met: %v", err)
	}
}

func TestIsZeroValue(t *testing.T) {
	i := new(int)
	*i = 5
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"Nil interface", nil, true},
		{"Nil pointer", (*int)(nil), true},
		{"Non-nil pointer", i, false},
		{"Zero int", 0, true},
		{"Non-zero int", 42, false},
		{"Zero string", "", true},
		{"Non-zero string", "hello", false},
		{"Empty slice", []int{}, true},
		{"Non-empty slice", []int{1, 2, 3}, false},
		{"Empty map", map[string]int{}, true},
		{"Non-empty map", map[string]int{"key": 42}, false},
		{"Nil slice", []int(nil), true},
		{"Empty array", [3]int{}, true},
		{"Non-empty array", [3]int{1, 2, 3}, false},
		{"Zero struct", struct{}{}, true},
		{"Non-zero struct", struct{ A int }{A: 1}, false},
		{"Zero time", time.Time{}, true},
		{"Non-zero time", time.Now(), false},
		{"Empty channel", make(chan int), true},
		{"Non-nil, unused channel", func() chan int { return make(chan int) }(), true},
		{"Non-nil, active channel", func() chan int { ch := make(chan int, 1); ch <- 42; return ch }(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := reflect.ValueOf(tt.value)
			got := isZeroValue(value)
			if got != tt.want {
				t.Errorf("isZeroValue(%v) = %v; want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestReturning(t *testing.T) {
	// Test cases for the Returning method
	tests := []struct {
		name     string
		columns  []string
		model    interface{}
		expected string
	}{
		{
			name:     "No columns passed",
			columns:  []string{},
			model:    MockModel{},
			expected: "",
		},
		{
			name:     "Return all columns with *",
			columns:  []string{"*"},
			model:    MockModel{ID: 1, Name: "John", Email: "john@example.com"},
			expected: "id, name, email",
		},
		{
			name:     "Return specific columns",
			columns:  []string{"id", "name"},
			model:    MockModel{ID: 1, Name: "John", Email: "john@example.com"},
			expected: "id, name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new QueryBuilderImpl instance
			qb := &QueryBuilderImpl{}

			qbResult := qb.Returning(tt.model, tt.columns...)

			// Since Returning returns QueryBuilder (interface), we need to type assert to *QueryBuilderImpl
			if impl, ok := qbResult.(*QueryBuilderImpl); ok {
				// Check if the returning clause is as expected
				if impl.returning != tt.expected {
					t.Errorf("expected returning clause %v, got %v", tt.expected, impl.returning)
				}
			} else {
				t.Errorf("expected *QueryBuilderImpl, got %T", qbResult)
			}
		})
	}
}
