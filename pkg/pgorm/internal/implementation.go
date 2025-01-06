package internal

import (
	"database/sql"
	"fmt"
	"strings"
)

func NewQueryBuilder(DB *sql.DB) *QueryBuilderImpl {
	return &QueryBuilderImpl{db: DB}
}

func (qb *QueryBuilderImpl) Table(model Model) QueryBuilder {
	qb.tableName = model.TableName()
	return qb
}

func (qb *QueryBuilderImpl) Returning(columns ...string) QueryBuilder {
	qb.columns = append(qb.columns, columns...)
	return qb
}

func (qb *QueryBuilderImpl) Insert(model interface{}) (Result, error) {
	qb.operation = "INSERT"
	columns, values, placeholders := extractColumnsAndValues(model)

	if len(columns) == 0 || len(values) == 0 {
		return Result{}, fmt.Errorf("no valid fields to insert")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := qb.db.Exec(query, values...)
	if err != nil {
		return Result{}, err
	}

	rowsAffected, _ := result.RowsAffected()
	return Result{RowsAffected: rowsAffected}, nil
}

func (qb *QueryBuilderImpl) Set(model interface{}) QueryBuilder {
	columns, values, _ := extractColumnsAndValues(model)
	setClauses := []string{}
	for i, column := range columns {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, i+1))
	}
	qb.columns = setClauses
	qb.values = values
	return qb
}

func (qb *QueryBuilderImpl) Update() (Result, error) {
	qb.operation = "UPDATE"
	query := fmt.Sprintf("UPDATE %s SET %s %s",
		qb.tableName,
		strings.Join(qb.columns, ", "),
		qb.where,
	)

	args := append(qb.values, qb.whereArgs...)
	result, err := qb.db.Exec(query, args...)
	if err != nil {
		return Result{}, err
	}

	rowsAffected, _ := result.RowsAffected()
	return Result{RowsAffected: rowsAffected}, nil
}

func (qb *QueryBuilderImpl) Select() (interface{}, error) {
	qb.operation = "SELECT"

	if len(qb.columns) == 0 {
		qb.columns = append(qb.columns, "*")
	}

	query := fmt.Sprintf("SELECT %s FROM %s %s",
		strings.Join(qb.columns, ", "),
		qb.tableName,
		qb.where,
	)

	rows, err := qb.db.Query(query, qb.whereArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error fetching columns: %w", err)
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		rowMap := make(map[string]interface{})
		columnValues := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		for i, colName := range columns {
			rowMap[colName] = columnValues[i]
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}

func (qb *QueryBuilderImpl) Where(condition string, args ...interface{}) QueryBuilder {
	placeholderIndex := len(qb.whereArgs)
	qb.where = "WHERE " + replacePlaceholders(condition, placeholderIndex)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

func (qb *QueryBuilderImpl) Delete() (Result, error) {
	qb.operation = "DELETE"

	if qb.tableName == "" {
		return Result{}, fmt.Errorf("table name is not specified")
	}

	query := fmt.Sprintf("DELETE FROM %s %s", qb.tableName, qb.where)

	// Execute the query with the `where` arguments
	result, err := qb.db.Exec(query, qb.whereArgs...)
	if err != nil {
		return Result{}, fmt.Errorf("delete operation failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return Result{RowsAffected: rowsAffected}, nil
}
