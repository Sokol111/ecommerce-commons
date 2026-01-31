package mongo

import (
	"context"
	"fmt"
	"net/url"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

// Mongo is the public interface for repository access.
type Mongo interface {
	GetCollection(collection string) *mongodriver.Collection
}

// Admin is the internal interface for infrastructure components (migrations, transactions).
type Admin interface {
	Mongo
	GetDatabase() *mongodriver.Database
	StartSession(ctx context.Context) (*mongodriver.Session, error)
}

type mongo struct {
	client        *mongodriver.Client
	database      *mongodriver.Database
	clientOptions *options.ClientOptions
	connectionURI string
	databaseName  string
	conf          Config
	log           *zap.Logger
}

func newMongo(log *zap.Logger, conf Config, appName string) (*mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	// Build URI and create client options
	uri := buildURI(conf)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetAppName(appName).
		SetMaxPoolSize(conf.MaxPoolSize).
		SetMinPoolSize(conf.MinPoolSize).
		SetMaxConnIdleTime(conf.MaxConnIdleTime).
		SetServerSelectionTimeout(conf.ServerSelectTimeout).
		SetTimeout(conf.QueryTimeout)
	// TODO: Add OpenTelemetry tracing when otelmongo supports mongo-driver v2
	// SetMonitor(otelmongo.NewMonitor())

	// Create client and database reference
	// Client is initialized here to avoid nil pointer errors in GetCollection* methods
	// Actual connection validation happens in connect() via Ping
	client, err := mongodriver.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongo client: %w", err)
	}

	return &mongo{
		client:        client,
		database:      client.Database(conf.Database),
		clientOptions: clientOptions,
		connectionURI: uri,
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

func (m *mongo) StartSession(ctx context.Context) (*mongodriver.Session, error) {
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

// GetCollection returns a MongoDB collection.
// Operations on this collection automatically use the QueryTimeout configured for the client.
func (m *mongo) GetCollection(collection string) *mongodriver.Collection {
	return m.database.Collection(collection)
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
