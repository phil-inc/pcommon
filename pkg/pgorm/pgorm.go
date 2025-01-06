package pgorm

import (
	"database/sql"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/phil-inc/pcommon/pkg/pgorm/internal"
)

func NewQueryBuilder(DB *sql.DB) *internal.QueryBuilderImpl {
	return internal.NewQueryBuilder(DB)
}
