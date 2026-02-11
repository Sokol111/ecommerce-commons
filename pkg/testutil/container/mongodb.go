package container

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDBContainer wraps the testcontainers MongoDB container with a client
type MongoDBContainer struct {
	Container        *mongodb.MongoDBContainer
	Client           *mongo.Client
	ConnectionString string
}

// MongoDBContainerOption configures the MongoDB container
type MongoDBContainerOption func(*mongoDBContainerOptions)

type mongoDBContainerOptions struct {
	image      string
	replicaSet string
}

// WithImage sets the MongoDB image to use
func WithImage(image string) MongoDBContainerOption {
	return func(o *mongoDBContainerOptions) {
		o.image = image
	}
}

// WithReplicaSet enables replica set with the given name
func WithReplicaSet(name string) MongoDBContainerOption {
	return func(o *mongoDBContainerOptions) {
		o.replicaSet = name
	}
}

// StartMongoDBContainer starts a MongoDB container and returns a wrapper with a connected client
func StartMongoDBContainer(ctx context.Context, opts ...MongoDBContainerOption) (*MongoDBContainer, error) {
	options := &mongoDBContainerOptions{
		image: "mongo:7",
	}
	for _, opt := range opts {
		opt(options)
	}

	// Build testcontainers options
	var tcOpts []testcontainers.ContainerCustomizer
	if options.replicaSet != "" {
		tcOpts = append(tcOpts, mongodb.WithReplicaSet(options.replicaSet))
	}

	// Start MongoDB container
	mongoContainer, err := mongodb.Run(ctx, options.image, tcOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start mongodb container: %w", err)
	}

	// Get connection string
	connectionString, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(mongoContainer)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Connect to MongoDB
	clientOpts := mongooptions.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		_ = testcontainers.TerminateContainer(mongoContainer)
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		_ = testcontainers.TerminateContainer(mongoContainer)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &MongoDBContainer{
		Container:        mongoContainer,
		Client:           client,
		ConnectionString: connectionString,
	}, nil
}

// Database returns a database handle for the given name
func (m *MongoDBContainer) Database(name string) *mongo.Database {
	return m.Client.Database(name)
}

// Terminate disconnects the client and terminates the container
func (m *MongoDBContainer) Terminate(ctx context.Context) error {
	var errs []error

	if m.Client != nil {
		if err := m.Client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to disconnect from mongodb: %w", err))
		}
	}

	if m.Container != nil {
		if err := testcontainers.TerminateContainer(m.Container); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate mongodb container: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during termination: %v", errs)
	}
	return nil
}
