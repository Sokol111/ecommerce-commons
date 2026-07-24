package mongo

import (
	"context"
	"fmt"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

// TxManager defines the interface for transaction management.
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

type txManager struct {
	client *mongodriver.Client
	log    *zap.Logger
}

func NewTxManager(client *mongodriver.Client, log *zap.Logger) TxManager {
	return &txManager{
		client: client,
		log:    log,
	}
}

// WithTransaction executes the provided function within a MongoDB transaction.
// MongoDB driver's WithTransaction already handles retries for transient errors internally.
func (t *txManager) WithTransaction(ctx context.Context, fn func(sessCtx context.Context) (any, error)) (any, error) {
	t.log.Debug("starting transaction")

	sess, err := t.client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	defer sess.EndSession(ctx)

	result, err := sess.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		return fn(sessCtx)
	})
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	t.log.Debug("transaction committed successfully")
	return result, nil
}
