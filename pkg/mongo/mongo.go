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
	return &Mongo{conf: conf}
}

func (m *Mongo) Connect(ctx context.Context, timeout time.Duration) {
	var uri string
	if m.conf.Username != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?replicaSet=%s", m.conf.Username, m.conf.Password, m.conf.Host, m.conf.Port, m.conf.Database, m.conf.ReplicaSet)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/%s?replicaSet=%s", m.conf.Host, m.conf.Port, m.conf.Database, m.conf.ReplicaSet)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(fmt.Errorf("failed to connect to mongo: %v", err))
	}
	m.client = client

	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = client.Ping(ctxTimeout, nil)

	if err != nil {
		panic(fmt.Errorf("failed to ping mongo with timeout [%v] err: %v", timeout, err))
	}

	m.database = client.Database(m.conf.Database)
}

func (m *Mongo) GetDatabase() *mongo.Database {
	return m.database
}

func (m *Mongo) Disconnect() {
	if err := m.client.Disconnect(context.Background()); err != nil {
		panic(fmt.Errorf("failed to disconnect from mongo: %v", err))
	}
}

func (m *Mongo) CreateSimpleIndex(ctx context.Context, collection string, keys interface{}) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	indexModel := mongo.IndexModel{
		Keys: keys,
	}
	name, err := m.database.Collection(collection).Indexes().CreateOne(ctxTimeout, indexModel)
	if err != nil {
		panic(fmt.Errorf("failed to create indexes: %v", err))
	}
	slog.Info("index created: " + name)
}
