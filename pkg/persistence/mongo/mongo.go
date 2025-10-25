package mongo

import (
	"context"
	"fmt"
	"strings"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Mongo interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	GetCollection(collection string) *mongodriver.Collection
	GetCollectionWithTimeout(collection string) Collection // Повертає інтерфейс для мокування
	CreateIndexes(ctx context.Context, collection string, indexes []mongodriver.IndexModel) error
	CreateSimpleIndex(ctx context.Context, collection string, keys interface{}) error
	StartSession(ctx context.Context) (mongodriver.Session, error)
}

type mongo struct {
	client   *mongodriver.Client
	database *mongodriver.Database
	conf     Config
	log      *zap.Logger
}

func newMongo(log *zap.Logger, conf Config) (Mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	return &mongo{conf: conf, log: log}, nil
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

func (m *mongo) Connect(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, m.conf.ConnectTimeout)
	defer cancel()

	uri := buildURI(m.conf)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(m.conf.MaxPoolSize).
		SetMinPoolSize(m.conf.MinPoolSize).
		SetMaxConnIdleTime(m.conf.MaxConnIdleTime).
		SetServerSelectionTimeout(m.conf.ServerSelectTimeout)

	client, err := mongodriver.Connect(c, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}
	m.client = client

	if err := client.Ping(c, nil); err != nil {
		return fmt.Errorf("failed to ping mongo: %w", err)
	}

	m.database = client.Database(m.conf.Database)
	m.log.Info("connected to mongo",
		zap.String("host", m.conf.Host),
		zap.Int("port", m.conf.Port),
		zap.Uint64("max-pool-size", m.conf.MaxPoolSize),
		zap.Uint64("min-pool-size", m.conf.MinPoolSize),
		zap.Duration("max-conn-idle-time", m.conf.MaxConnIdleTime),
	)
	return nil
}

func (m *mongo) StartSession(ctx context.Context) (mongodriver.Session, error) {
	return m.client.StartSession()
}

func buildURI(conf Config) string {
	if conf.ConnectionString != "" {
		return conf.ConnectionString
	}

	auth := ""
	if conf.Username != "" {
		auth = fmt.Sprintf("%s:%s@", conf.Username, conf.Password)
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d/%s", auth, conf.Host, conf.Port, conf.Database)

	params := []string{}
	if conf.ReplicaSet != "" {
		params = append(params, "replicaSet="+conf.ReplicaSet)
	}
	if conf.DirectConnection {
		params = append(params, "directConnection=true")
	}

	if len(params) > 0 {
		uri += "?" + strings.Join(params, "&")
	}

	return uri
}

func (m *mongo) GetCollection(collection string) *mongodriver.Collection {
	return m.database.Collection(collection)
}

// GetCollectionWithTimeout returns a wrapped collection with automatic query timeout
func (m *mongo) GetCollectionWithTimeout(collection string) Collection {
	coll := m.database.Collection(collection)
	return NewCollectionWrapper(coll, m.conf.QueryTimeout)
}

func (m *mongo) Disconnect(ctx context.Context) error {
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

func (m *mongo) CreateIndexes(ctx context.Context, collection string, indexes []mongodriver.IndexModel) error {
	names, err := m.database.Collection(collection).Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	for _, name := range names {
		m.log.Info("index created", zap.String("name", name), zap.String("collection", collection))
	}
	return nil
}

func (m *mongo) CreateSimpleIndex(ctx context.Context, collection string, keys interface{}) error {
	indexModel := mongodriver.IndexModel{Keys: keys}
	return m.CreateIndexes(ctx, collection, []mongodriver.IndexModel{indexModel})
}
