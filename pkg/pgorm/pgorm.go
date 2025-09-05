package pgorm

import (
	"database/sql"

	"github.com/jackc/pgx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/phil-inc/pcommon/pkg/pgorm/internal"
)

func NewQueryBuilderFromSQL(db *sql.DB) *internal.QueryBuilderImpl {
	return internal.NewQueryBuilder(internal.NewStdDBExecutor(db))
}

func NewQueryBuilderFromPgx(pool *pgx.ConnPool) *internal.QueryBuilderImpl {
	return internal.NewQueryBuilder(internal.NewPgxDBExecutor(pool))
}
