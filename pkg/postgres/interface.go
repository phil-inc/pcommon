package postgres

import (
	"context"
	"io"

	"github.com/jackc/pgx"
)

//go:generate mockgen -source=./interface.go -destination=./mocks/PgxConnPool.go -package=mock_postgres PgxConnPool
type PgxConnPool interface {
	Acquire() (*pgx.Conn, error)
	AcquireEx(ctx context.Context) (*pgx.Conn, error)
	Begin() (*pgx.Tx, error)
	BeginBatch() *pgx.Batch
	BeginEx(ctx context.Context, txOptions *pgx.TxOptions) (*pgx.Tx, error)
	Close()
	CopyFrom(tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int, error)
	CopyFromReader(r io.Reader, sql string) (pgx.CommandTag, error)
	CopyToWriter(w io.Writer, sql string, args ...interface{}) (pgx.CommandTag, error)
	Deallocate(name string) (err error)
	Exec(sql string, arguments ...interface{}) (commandTag pgx.CommandTag, err error)
	ExecEx(ctx context.Context, sql string, options *pgx.QueryExOptions, arguments ...interface{}) (commandTag pgx.CommandTag, err error)
	Prepare(name string, sql string) (*pgx.PreparedStatement, error)
	PrepareEx(ctx context.Context, name string, sql string, opts *pgx.PrepareExOptions) (*pgx.PreparedStatement, error)
	Query(sql string, args ...interface{}) (*pgx.Rows, error)
	QueryEx(ctx context.Context, sql string, options *pgx.QueryExOptions, args ...interface{}) (*pgx.Rows, error)
	QueryRow(sql string, args ...interface{}) *pgx.Row
	QueryRowEx(ctx context.Context, sql string, options *pgx.QueryExOptions, args ...interface{}) *pgx.Row
	Release(conn *pgx.Conn)
	Reset()
	Stat() (s pgx.ConnPoolStat)
}
