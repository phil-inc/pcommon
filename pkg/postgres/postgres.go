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
var Tx *pgx.Tx

type Config struct {
	URL         string
	DBName      string
	SSLMode     string
	SSLRootCert string
}

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

// SetupPostgres - creates connection to Postgres database
func SetupPostgres(connConfig *Config) error {
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

func ReportDB() *pgx.ConnPool {
	if pool == nil {
		err := SetupPostgres(config)
		if err != nil {
			logger.Panicf("Error connecting to report db. Error message: %s", err)
			return nil
		}
	}

	return pool
}
