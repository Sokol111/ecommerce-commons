package fxconfig

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func provideMongoClient(log *zap.Logger, conf mongo.Config, appConf config.AppConfig, tp trace.TracerProvider, mp metric.MeterProvider) (*mongodriver.Client, error) {
	// Build URI and create client options
	uri := conf.BuildURI()
	clientOptions := options.Client().
		ApplyURI(uri).
		SetAppName(appConf.ServiceName).
		SetMaxPoolSize(conf.MaxPoolSize).
		SetMinPoolSize(conf.MinPoolSize).
		SetMaxConnIdleTime(conf.MaxConnIdleTime).
		SetServerSelectionTimeout(conf.ServerSelectTimeout).
		SetTimeout(conf.QueryTimeout).
		SetMonitor(otelmongo.NewMonitor(
			otelmongo.WithTracerProvider(tp),
			otelmongo.WithMeterProvider(mp),
		))

	if wc := conf.WriteConcern.BuildWriteConcern(); wc != nil {
		clientOptions.SetWriteConcern(wc)
	}
	if rc := conf.ReadConcern.BuildReadConcern(); rc != nil {
		clientOptions.SetReadConcern(rc)
	}
	if rp := conf.ReadPreference.BuildReadPreference(); rp != nil {
		clientOptions.SetReadPreference(rp)
	}

	// Create client and database reference
	// Client is initialized here to avoid nil pointer errors in GetCollection* methods
	// Actual connection validation happens in connect() via Ping
	client, err := mongodriver.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongo client: %w", err)
	}

	return client, nil
}

func provideDatabase(client *mongodriver.Client, conf mongo.Config) *mongodriver.Database {
	return client.Database(conf.Database)
}
