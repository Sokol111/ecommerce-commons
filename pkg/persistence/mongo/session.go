package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Session represents a MongoDB session for transaction support.
// This interface mirrors all public methods of mongo.Session to enable mocking.
// mongo.Session has unexported methods making it impossible to embed directly.
type Session interface {
	// StartTransaction starts a new transaction, configured with the given options, on this
	// session. This method returns an error if there is already a transaction in-progress for this
	// session.
	StartTransaction(opts ...*options.TransactionOptions) error

	// AbortTransaction aborts the active transaction for this session. This method returns an error
	// if there is no active transaction for this session or if the transaction has been committed
	// or aborted.
	AbortTransaction(ctx context.Context) error

	// CommitTransaction commits the active transaction for this session. This method returns an
	// error if there is no active transaction for this session or if the transaction has been
	// aborted.
	CommitTransaction(ctx context.Context) error

	// WithTransaction starts a transaction on this session and runs the fn callback. Errors with
	// the TransientTransactionError and UnknownTransactionCommitResult labels are retried for up to
	// 120 seconds. Inside the callback, the SessionContext must be used as the Context parameter
	// for any operations that should be part of the transaction. If the ctx parameter already has a
	// Session attached to it, it will be replaced by this session. The fn callback may be run
	// multiple times during WithTransaction due to retry attempts, so it must be idempotent.
	// Non-retryable operation errors or any operation errors that occur after the timeout expires
	// will be returned without retrying. If the callback fails, the driver will call
	// AbortTransaction.
	WithTransaction(ctx context.Context, fn func(ctx mongodriver.SessionContext) (interface{}, error), opts ...*options.TransactionOptions) (interface{}, error)

	// EndSession aborts any existing transactions and close the session.
	EndSession(ctx context.Context)

	// ClusterTime returns the current cluster time document associated with the session.
	ClusterTime() bson.Raw

	// OperationTime returns the current operation time document associated with the session.
	OperationTime() *primitive.Timestamp

	// Client returns the Client associated with the session.
	Client() *mongodriver.Client

	// ID returns the current ID document associated with the session. The ID document is in the
	// form {"id": <BSON binary value>}.
	ID() bson.Raw

	// AdvanceClusterTime advances the cluster time for a session. This method returns an error if
	// the session has ended.
	AdvanceClusterTime(bson.Raw) error

	// AdvanceOperationTime advances the operation time for a session. This method returns an error
	// if the session has ended.
	AdvanceOperationTime(*primitive.Timestamp) error
}

// Compile-time check that mongodriver.Session implements our Session interface
var _ Session = (mongodriver.Session)(nil)
