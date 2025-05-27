package mongo

import (
	"context"
	"fmt"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
)

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(sessCtx mongodriver.SessionContext) (any, error)) (any, error)
	BeginTx(ctx context.Context) (mongodriver.SessionContext, error)
	Commit(sessionCtx mongodriver.SessionContext) error
	Rollback(sessionCtx mongodriver.SessionContext) error
}

type txManager struct {
	mongo Mongo
}

func NewTxManager(mongo Mongo) TxManager {
	return &txManager{mongo: mongo}
}

func (t *txManager) WithTransaction(ctx context.Context, fn func(sessCtx mongodriver.SessionContext) (any, error)) (any, error) {
	session, err := t.mongo.StartSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, fn)
	if err != nil {
		return result, fmt.Errorf("transaction failed: %w", err)
	}
	return result, nil
}

func (t *txManager) BeginTx(ctx context.Context) (mongodriver.SessionContext, error) {
	session, err := t.mongo.StartSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	err = session.StartTransaction()

	if err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	sessionCtx := mongodriver.NewSessionContext(ctx, session)

	return sessionCtx, nil
}

func (t *txManager) Commit(sessionCtx mongodriver.SessionContext) error {
	defer sessionCtx.EndSession(sessionCtx)
	return sessionCtx.CommitTransaction(sessionCtx)
}

func (t *txManager) Rollback(sessionCtx mongodriver.SessionContext) error {
	defer sessionCtx.EndSession(sessionCtx)
	return sessionCtx.AbortTransaction(sessionCtx)
}
