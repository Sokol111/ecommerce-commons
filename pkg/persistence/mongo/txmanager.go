package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type mongoTxManager struct {
	admin MongoAdmin
	log   *zap.Logger
}

func newTxManager(admin MongoAdmin, log *zap.Logger) persistence.TxManager {
	return &mongoTxManager{
		admin: admin,
		log:   log,
	}
}

// WithTransaction executes the provided function within a MongoDB transaction.
// MongoDB driver's WithTransaction already handles retries for transient errors internally.
func (t *mongoTxManager) WithTransaction(ctx context.Context, fn func(sessCtx context.Context) (any, error)) (any, error) {
	t.log.Debug("starting transaction")

	sess, err := t.admin.StartSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	defer sess.EndSession(ctx)

	result, err := sess.WithTransaction(ctx, func(sessCtx mongodriver.SessionContext) (any, error) {
		return fn(sessCtx)
	})
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	t.log.Debug("transaction committed successfully")
	return result, nil
}
