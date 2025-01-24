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

func NewMongo(conf *MongoConf) *Mongo {
	if err := validateConfig(conf); err != nil {
		panic(err)
	}

	return &Mongo{conf: conf}
}

func validateConfig(conf *MongoConf) error {
	if conf.Host == "" || conf.Port == 0 || conf.Database == "" {
		return fmt.Errorf("invalid Mongo configuration")
	}
	return nil
}

func (m *Mongo) Connect(ctx context.Context) {
	uri := buildURI(m.conf)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(fmt.Errorf("failed to connect to mongo: %w", err))
	}
	m.client = client

	if err := client.Ping(ctx, nil); err != nil {
		panic(fmt.Errorf("failed to ping mongo: %w", err))
	}

	m.database = client.Database(m.conf.Database)
	slog.Info("connected to mongo", slog.String("host", m.conf.Host), slog.Int("port", m.conf.Port))
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

func (m *Mongo) Disconnect(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := m.client.Disconnect(ctx); err != nil {
		panic(fmt.Errorf("failed to disconnect from mongo: %w", err))
	}
	slog.Info("disconnected from mongo")
}

func (m *Mongo) CreateIndexes(ctx context.Context, collection string, indexes []mongo.IndexModel) {
	names, err := m.database.Collection(collection).Indexes().CreateMany(ctx, indexes)
	if err != nil {
		panic(fmt.Errorf("failed to create indexes: %w", err))
	}

	for _, name := range names {
		slog.Info("index created", slog.String("name", name), slog.String("collection", collection))
	}
}

func (m *Mongo) CreateSimpleIndex(ctx context.Context, collection string, keys interface{}) {
	indexModel := mongo.IndexModel{Keys: keys}
	m.CreateIndexes(ctx, collection, []mongo.IndexModel{indexModel})
}
