package mongo

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
	"time"
)

type MongoConf struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	ReplicaSet string `mapstructure:"replica-set"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Database   string `mapstructure:"database"`
}

type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
	conf     *MongoConf
}

func NewMongo(conf *MongoConf) (*Mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	return &Mongo{conf: conf}, nil
}

func validateConfig(conf *MongoConf) error {
	if conf.Host == "" || conf.Port == 0 || conf.Database == "" {
		return fmt.Errorf("invalid Mongo configuration")
	}
	return nil
}

func (m *Mongo) Connect(ctx context.Context, timeout time.Duration) error {
	uri := buildURI(m.conf)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}
	m.client = client

	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(ctxTimeout, nil); err != nil {
		return fmt.Errorf("failed to ping mongo: %w", err)
	}

	m.database = client.Database(m.conf.Database)
	slog.Info("connected to mongo", slog.String("host", m.conf.Host), slog.Int("port", m.conf.Port))
	return nil
}

func buildURI(conf *MongoConf) string {
	if conf.Username != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?replicaSet=%s", conf.Username, conf.Password, conf.Host, conf.Port, conf.Database, conf.ReplicaSet)
	}
	return fmt.Sprintf("mongodb://%s:%d/%s?replicaSet=%s", conf.Host, conf.Port, conf.Database, conf.ReplicaSet)
}

func (m *Mongo) GetCollection(collection string) *mongo.Collection {
	return m.database.Collection(collection)
}

func (m *Mongo) Disconnect(ctx context.Context) error {
	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from mongo: %w", err)
	}
	slog.Info("disconnected from mongo")
	return nil
}

func (m *Mongo) CreateIndexes(ctx context.Context, collection string, timeout time.Duration, indexes []mongo.IndexModel) ([]string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	names, err := m.database.Collection(collection).Indexes().CreateMany(ctxTimeout, indexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	slog.Info("indexes created", slog.Any("indexes", names), slog.String("collection", collection))
	return names, nil
}

func (m *Mongo) CreateSimpleIndex(ctx context.Context, collection string, timeout time.Duration, keys interface{}) (string, error) {
	indexModel := mongo.IndexModel{Keys: keys}
	names, err := m.CreateIndexes(ctx, collection, timeout, []mongo.IndexModel{indexModel})
	if err != nil {
		return "", err
	}
	return names[0], nil
}
