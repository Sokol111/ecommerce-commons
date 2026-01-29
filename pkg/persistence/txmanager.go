package persistence

import "context"

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(txCtx context.Context) (any, error)) (any, error)
}

// WithTransaction is a generic wrapper around TxManager.WithTransaction
// that provides type-safe transaction handling without manual type assertions.
func WithTransaction[T any](ctx context.Context, tm TxManager, fn func(txCtx context.Context) (T, error)) (T, error) {
	var zero T
	result, err := tm.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		return fn(txCtx)
	})
	if err != nil {
		return zero, err
	}
	return result.(T), nil //nolint:errcheck // Type assertion is safe here since fn returns T
}
