## Introduction

goal is to create an easy-to-use, flexible, and extensible ORM that allows developers to interact with PostgreSQL tables through Go structs. The framework will support basic operations like Insert, Update, Update, and Delete while ensuring compatibility with PostgreSQL.

## Key Features

- Use of reflection for automatic table/column mapping.
- Chaining of functions for a clean, easy-to-read API.
- Struct tags for column mapping.
- Simple and clear syntax to interact with the database.

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
 qb := pgorm.NewQueryBuilder(db)

 // Example: INSERT query
 result, err := qb.Table(User{}).Returning("id").Insert(User{
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
