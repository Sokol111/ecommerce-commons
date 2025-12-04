package mongo

import (
	"context"
	"fmt"
	"net/url"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.uber.org/zap"
)

// Mongo is the public interface for repository access.
type Mongo interface {
	GetCollection(collection string) Collection
	GetCollectionWithOptions(collection string, opts ...WrapperOption) Collection
}

// MongoAdmin is the internal interface for infrastructure components (migrations, transactions).
type MongoAdmin interface {
	Mongo
	GetDatabase() *mongodriver.Database
	StartSession(ctx context.Context) (Session, error)
}

type mongo struct {
	client        *mongodriver.Client
	database      *mongodriver.Database
	clientOptions *options.ClientOptions
	databaseName  string
	conf          Config
	log           *zap.Logger
}

func newMongo(log *zap.Logger, conf Config) (*mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	// Build URI and create client options
	uri := buildURI(conf)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(conf.MaxPoolSize).
		SetMinPoolSize(conf.MinPoolSize).
		SetMaxConnIdleTime(conf.MaxConnIdleTime).
		SetServerSelectionTimeout(conf.ServerSelectTimeout).
		SetMonitor(otelmongo.NewMonitor()) // OpenTelemetry tracing

	// Create client and database reference
	// Client is initialized here to avoid nil pointer errors in GetCollection* methods
	// Actual connection validation happens in connect() via Ping
	client, err := mongodriver.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongo client: %w", err)
	}

	return &mongo{
		client:        client,
		database:      client.Database(conf.Database),
		clientOptions: clientOptions,
		databaseName:  conf.Database,
		conf:          conf,
		log:           log,
	}, nil
}

func validateConfig(conf Config) error {
	if conf.ConnectionString != "" {
		return nil
	}
	if conf.Host == "" || conf.Port == 0 || conf.Database == "" {
		return fmt.Errorf("invalid Mongo configuration")
	}
	return nil
}

func (m *mongo) connect(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, m.conf.ConnectTimeout)
	defer cancel()

	// Ping to establish actual connection (Connect was already called in newMongo)
	if err := m.client.Ping(c, nil); err != nil {
		return fmt.Errorf("failed to ping mongo: %w", err)
	}

	fields := []zap.Field{
		zap.String("database", m.conf.Database),
		zap.Uint64("max-pool-size", m.conf.MaxPoolSize),
		zap.Uint64("min-pool-size", m.conf.MinPoolSize),
		zap.Duration("max-conn-idle-time", m.conf.MaxConnIdleTime),
		zap.Duration("query-timeout", m.conf.QueryTimeout),
	}
	if m.conf.ConnectionString == "" {
		fields = append(fields,
			zap.String("host", m.conf.Host),
			zap.Int("port", m.conf.Port),
		)
	}

	m.log.Info("connected to mongo", fields...)
	return nil
}

func (m *mongo) StartSession(ctx context.Context) (Session, error) {
	return m.client.StartSession()
}

func buildURI(conf Config) string {
	if conf.ConnectionString != "" {
		return conf.ConnectionString
	}

	u := &url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Path:   "/" + conf.Database,
	}

	if conf.Username != "" {
		u.User = url.UserPassword(conf.Username, conf.Password)
	}

	q := u.Query()
	if conf.ReplicaSet != "" {
		q.Set("replicaSet", conf.ReplicaSet)
	}
	if conf.DirectConnection {
		q.Set("directConnection", "true")
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// GetCollection returns a Collection with automatic query timeout from config.
func (m *mongo) GetCollection(collection string) Collection {
	return m.GetCollectionWithOptions(collection, WithTimeout(m.conf.QueryTimeout))
}

// GetCollectionWithOptions returns a Collection with custom options (timeout, middleware, etc.)
// Use this when you need different timeout or additional middleware for specific operations.
func (m *mongo) GetCollectionWithOptions(collection string, opts ...WrapperOption) Collection {
	coll := m.database.Collection(collection)
	return newCollectionWrapper(coll, opts...)
}

func (m *mongo) GetDatabase() *mongodriver.Database {
	return m.database
}

func (m *mongo) disconnect(ctx context.Context) error {
	if m.client == nil {
		return nil
	}
	c, cancel := context.WithTimeout(ctx, m.conf.ConnectTimeout)
	defer cancel()
	if err := m.client.Disconnect(c); err != nil {
		return fmt.Errorf("failed to disconnect from mongo: %w", err)
	}
	m.log.Info("disconnected from mongo")
	return nil
}
