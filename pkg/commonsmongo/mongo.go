package commonsmongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type MongoInterface interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	CreateIndexes(ctx context.Context, collection string, indexes []mongo.IndexModel) error
	CreateSimpleIndex(ctx context.Context, collection string, keys interface{}) error
}

type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
	conf     MongoConf
	log      *zap.Logger
}

func NewMongo(log *zap.Logger, conf MongoConf) (*Mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	return &Mongo{conf: conf, log: log}, nil
}

func validateConfig(conf MongoConf) error {
	if conf.Host == "" || conf.Port == 0 || conf.Database == "" {
		return fmt.Errorf("invalid Mongo configuration")
	}
	return nil
}

func (m *Mongo) Connect(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	uri := buildURI(m.conf)
	client, err := mongo.Connect(c, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}
	m.client = client

	if err := client.Ping(c, nil); err != nil {
		return fmt.Errorf("failed to ping mongo: %w", err)
	}

	m.database = client.Database(m.conf.Database)
	m.log.Info("connected to mongo", zap.String("host", m.conf.Host), zap.Int("port", m.conf.Port))
	return nil
}

func buildURI(conf MongoConf) string {
	if conf.Username != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?replicaSet=%s", conf.Username, conf.Password, conf.Host, conf.Port, conf.Database, conf.ReplicaSet)
	}
	return fmt.Sprintf("mongodb://%s:%d/%s?replicaSet=%s", conf.Host, conf.Port, conf.Database, conf.ReplicaSet)
}

func (m *Mongo) GetCollection(collection string) *mongo.Collection {
	return m.database.Collection(collection)
}

func (m *Mongo) Disconnect(ctx context.Context) error {
	if m.client == nil {
		return nil
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := m.client.Disconnect(c); err != nil {
		return fmt.Errorf("failed to disconnect from mongo: %w", err)
	}
	return nil
}

func (m *Mongo) CreateIndexes(ctx context.Context, collection string, indexes []mongo.IndexModel) error {
	names, err := m.database.Collection(collection).Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	for _, name := range names {
		m.log.Info("index created", zap.String("name", name), zap.String("collection", collection))
	}
	return nil
}

func (m *Mongo) CreateSimpleIndex(ctx context.Context, collection string, keys interface{}) error {
	indexModel := mongo.IndexModel{Keys: keys}
	return m.CreateIndexes(ctx, collection, []mongo.IndexModel{indexModel})
}
