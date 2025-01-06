package internal

import "database/sql"

type QueryBuilder interface {
	Table(model Model) QueryBuilder
	Returning(columns ...string) QueryBuilder
	Set(model interface{}) QueryBuilder
	Where(condition string, args ...interface{}) QueryBuilder
	Insert(model interface{}) (Result, error)
	Update() (Result, error)
	Delete() (Result, error)
	Select() (interface{}, error)
}

type QueryBuilderImpl struct {
	db        *sql.DB
	tableName string
	columns   []string
	values    []interface{}
	operation string
	where     string
	whereArgs []interface{}
}

type Model interface {
	TableName() string
}

type Result struct {
	RowsAffected int64
}
