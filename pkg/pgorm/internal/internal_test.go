package internal

import (
	"reflect"
	"regexp"
	"testing"

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
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO mock_table (id, email) VALUES ($1, $2)`)).
		WithArgs(1, "test@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	queryBuilder := NewQueryBuilder(db)
	model := MockModel{ID: 1, Name: "", Email: "test@example.com"}

	result, err := queryBuilder.Table(model).Returning("id").Insert(model)
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

func TestUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE mock_table SET id = $1 WHERE id = $2`)).
		WithArgs(2, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	queryBuilder := NewQueryBuilder(db)
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

func TestDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM mock_table WHERE id = $1`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	queryBuilder := NewQueryBuilder(db)
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

	queryBuilder := NewQueryBuilder(db)
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
