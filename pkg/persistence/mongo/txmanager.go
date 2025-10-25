package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type mongoTxManager struct {
	mongo Mongo
	log   *zap.Logger
}

func newTxManager(mongo Mongo, log *zap.Logger) persistence.TxManager {
	return &mongoTxManager{
		mongo: mongo,
		log:   log,
	}
}

// isTransientError checks if the error is a transient MongoDB error that can be retried
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	// MongoDB transient transaction errors have specific labels
	cmdErr, ok := err.(mongodriver.CommandError)
	if !ok {
		return false
	}
	return cmdErr.HasErrorLabel("TransientTransactionError")
}

func (t *mongoTxManager) WithTransaction(ctx context.Context, fn func(sessCtx context.Context) (any, error)) (any, error) {
	const maxRetries = 3
	var result any
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			t.log.Warn("retrying transaction", zap.Int("attempt", attempt))
		} else {
			t.log.Debug("starting transaction")
		}

		session, err := t.mongo.StartSession(ctx)
		if err != nil {
			if isTransientError(err) && attempt < maxRetries {
				continue
			}
			t.log.Error("failed to start session", zap.Error(err), zap.Int("attempt", attempt))
			return nil, fmt.Errorf("failed to start session: %w", err)
		}

		result, err = session.WithTransaction(ctx, func(sessCtx mongodriver.SessionContext) (any, error) {
			return fn(sessCtx)
		})
		session.EndSession(ctx)

		if err == nil {
			t.log.Debug("transaction committed successfully", zap.Int("attempts", attempt))
			return result, nil
		}

		// Retry on transient errors
		if isTransientError(err) && attempt < maxRetries {
			t.log.Warn("transient transaction error, will retry",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries))
			continue
		}

		t.log.Error("transaction failed", zap.Error(err), zap.Int("attempt", attempt))
		break
	}

	return result, fmt.Errorf("transaction failed after %d attempts: %w", maxRetries, err)
}
