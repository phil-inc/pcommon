## Introduction

pgorm is a lightweight and extensible PostgreSQL ORM. It provides a clean, chainable API for building and executing SQL queries using Go structs. Under the hood, it uses Go's reflection system to map struct fields to table columns, allowing developers to write less boilerplate and interact directly with PostgreSQL in a type-safe and ergonomic way.

## Goals
- Struct-to-table mapping via reflection
- Support for standard SQL operations (Insert, Select, Update, Delete)
- Support for RETURNING clauses
- Abstraction over multiple DB drivers (e.g., database/sql, pgx)
- Mockable interface for testing

## Key Features

- Works with *sql.DB, pgx.ConnPool, and mock implementations via the DBExecutor interface.
- Automatically extracts column names and values from structs using db tags.
- Clean, chainable query syntax.
- Plug in mock executors for easy unit testing.

## Example Usage

``` go
package main

import (
 "database/sql"
 "log"

 "github.com/phil-inc/pcommon/pkg/pgorm"
)

type User struct {
 ID    int    `db:"uid,omitempty"`
 Name  string `db:"name,omitempty"`
 Email string `db:"email,omitempty"`
}

func (User) TableName() string {
 return "user"
}

func main() {
 // Open a database connection
 db, err := sql.Open("postgres", "user=***** password=***** dbname=***** sslmode=disable")
 if err != nil {
  log.Fatal(err)
 }

 // Initialize the ORM with the database connection
 	qb := pgorm.NewQueryBuilderFromSQL(db)

 // Example: INSERT query
 result, err := qb.Table(User{}).Returning(User{},"id").Insert(User{
  Name:  "John Doe",
  Email: "johndoe@email.com",
 })

 if err != nil {
  log.Fatal(err)
 }

 log.Println("Rows affected:", result.RowsAffected)

 // Example: UPDATE query
 result, err = qb.Table(User{}).Where("name = 'John Doe'").Set(User{
  Email: "johndoe@email.com",
 }).Update()

 if err != nil {
  log.Fatal(err)
 }

 log.Println("Rows affected:", result.RowsAffected)

 // Example: SELECT query
 u, err := qb.Table(User{}).Returning().Where("name = 'John Doe'").Select()
 if err != nil {
  log.Fatal(err)
 }

 log.Println("User:", u)

 // Example: DELETE query
 result, err = qb.Table(User{}).Where("name = 'John Doe'").Delete()

 if err != nil {
  log.Fatal(err)
 }
 
 log.Println("Rows affected:", result.RowsAffected)

}

```
