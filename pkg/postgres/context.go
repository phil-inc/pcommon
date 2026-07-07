package postgres

import "context"

type contextKey string

const (
	ContextKeyPgxConnPool contextKey = "ctx:PgxConnPool"
)

func SetPgxConnPoolOnContext(ctx context.Context, pool PgxConnPool) context.Context {
	return context.WithValue(ctx, ContextKeyPgxConnPool, pool)
}

func GetPgxConnPoolFromContext(ctx context.Context) PgxConnPool {
	return ctx.Value(ContextKeyPgxConnPool).(PgxConnPool)
}
