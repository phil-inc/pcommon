package postgres

import (
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx"
	logger "github.com/phil-inc/plog-ng/pkg/core"
)

var pool *pgx.ConnPool
var config *Config

type Config struct {
	URL         string
	DBName      string
	SSLMode     string
	SSLRootCert string
}

type Tx struct {
	pgx.Tx
}

type Rows struct {
	pgx.Rows
}

var ErrNoRows = pgx.ErrNoRows

func connectPostgres(connConfig *Config) (*pgx.ConnPool, error) {
	connURL := fmt.Sprintf("%s/%s?sslmode=%s", connConfig.URL, connConfig.DBName, connConfig.SSLMode)

	if connConfig.SSLMode != "disable" && strings.Trim(connConfig.SSLRootCert, " ") != "" {
		connURL = fmt.Sprintf("%s&sslrootcert=%s", connURL, connConfig.SSLRootCert)
	}

	connectionConfig, err := pgx.ParseURI(connURL)
	if err != nil {
		return nil, err
	}

	maxConnections := 50
	timeOut := 5 * time.Minute

	poolConfig := pgx.ConnPoolConfig{
		ConnConfig:     connectionConfig,
		MaxConnections: maxConnections,
		AfterConnect:   nil,
		AcquireTimeout: timeOut,
	}

	pgxPool, err := pgx.NewConnPool(poolConfig)
	if err != nil {
		return nil, err
	}

	return pgxPool, nil

}

// Setup - creates connection to Postgres database
func Setup(connConfig *Config) error {
	var err error
	pool, err = connectPostgres(connConfig)

	// Fallback to ssl disable
	if err != nil && connConfig.SSLMode != "disable" {
		connConfig.SSLMode = "disable"
		connConfig.SSLRootCert = ""
		logger.Infof("Error connecting to postgres using sslmode=%s. Falling back to sslmode=disable", connConfig.SSLMode)
		pool, err = connectPostgres(connConfig)
		if err != nil {
			return err
		}
	}

	config = connConfig

	logger.Infof("Connected to postgres using sslmode=%s", connConfig.SSLMode)

	return nil
}

// SetupPool - creates connection to Postgres database and returns the pool
func SetupPool(connConfig *Config) (*pgx.ConnPool, error) {
	pool, err := connectPostgres(connConfig)

	// Fallback to ssl disable
	if err != nil && connConfig.SSLMode != "disable" {
		connConfig.SSLMode = "disable"
		connConfig.SSLRootCert = ""
		logger.Infof("Error connecting to postgres using sslmode=%s. Falling back to sslmode=disable", connConfig.SSLMode)
		pool, err = connectPostgres(connConfig)
		if err != nil {
			return nil, err
		}
	}

	logger.Infof("Connected to postgres using sslmode=%s", connConfig.SSLMode)
	return pool, nil
}

// DB - returns the global connection pool
func DB() *pgx.ConnPool {
	if pool == nil {
		err := Setup(config)
		if err != nil {
			logger.Panicf("Error connecting to report db. Error message: %s", err)
			return nil
		}
	}

	return pool
}

// Check rows.Err() when reading a query as it is also possible an error may have occurred after receiving some rows
// but before the query has completed.
// Reference: https://pkg.go.dev/github.com/jackc/pgx/v4#Conn.Query
func (r Rows) NextRow() (bool, error) {
	if r.Next() {
		return true, nil
	}

	if err := r.Err(); err != nil {
		logger.Errorf("Error executing query. Error message: %s", err)
		return false, err
	}

	return false, nil
}

// ExecQuery - executes query using the global pool
func ExecQuery(queryWithNamedParams string, params map[string]interface{}) (*Rows, error) {
	return execQueryWithPool(DB(), queryWithNamedParams, params)
}

// ExecQueryWithPool - executes query using a specific pool
func ExecQueryWithPool(pool *pgx.ConnPool, queryWithNamedParams string, params map[string]interface{}) (*Rows, error) {
	return execQueryWithPool(pool, queryWithNamedParams, params)
}

func execQueryWithPool(pool *pgx.ConnPool, queryWithNamedParams string, params map[string]interface{}) (*Rows, error) {
	paramArr := []interface{}{}
	count := 1

	for k, v := range params {
		queryWithNamedParams = strings.Replace(queryWithNamedParams, ":"+k, fmt.Sprintf("$%d", count), 1)
		paramArr = append(paramArr, v)
		count++
	}

	db := DB()
	pr, err := db.Query(queryWithNamedParams, paramArr...)
	if err != nil {
		return nil, err
	}

	return &Rows{Rows: *pr}, nil
}
