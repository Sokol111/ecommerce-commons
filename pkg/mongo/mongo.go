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
	Host     string `mapstructure:"mongo.host"`
	Port     int    `mapstructure:"mongo.port"`
	Username string `mapstructure:"mongo.username"`
	Password string `mapstructure:"mongo.password"`
	Database string `mapstructure:"mongo.database"`
}

type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
	conf     *MongoConf
}

func NewMongo(conf *MongoConf) *Mongo {
	return &Mongo{conf: conf}
}

func (m *Mongo) Connect(ctx context.Context, timeout time.Duration) *mongo.Database {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%d", m.conf.Username, m.conf.Password, m.conf.Host, m.conf.Port)))
	if err != nil {
		panic(fmt.Errorf("failed to connect to mongo: %v", err))
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = client.Ping(ctxTimeout, nil)

	if err != nil {
		panic(fmt.Errorf("failed to ping mongo with timeout [%v] err: %v", timeout, err))
	}

	return client.Database(m.conf.Database)
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
