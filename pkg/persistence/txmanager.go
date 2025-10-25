package persistence

import "context"

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(txCtx context.Context) (any, error)) (any, error)
}
