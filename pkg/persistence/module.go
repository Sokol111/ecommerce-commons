package persistence

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

// persistenceOptions holds internal configuration for the persistence module.
type persistenceOptions struct {
	mongoConfig *mongo.Config
}

// Option is a functional option for configuring the persistence module.
type Option func(*persistenceOptions)

// WithMongoConfig provides a static Mongo Config (useful for tests).
// When set, the Mongo configuration will not be loaded from viper.
func WithMongoConfig(cfg mongo.Config) Option {
	return func(opts *persistenceOptions) {
		opts.mongoConfig = &cfg
	}
}

// NewPersistenceModule provides persistence layer components for dependency injection.
//
// Options:
//   - WithMongoConfig: provide static Mongo Config (useful for tests)
//
// Example usage:
//
//	// Production - loads config from viper
//	persistence.NewPersistenceModule()
//
//	// Testing - with static config
//	persistence.NewPersistenceModule(
//	    persistence.WithMongoConfig(mongo.Config{...}),
//	)
func NewPersistenceModule(opts ...Option) fx.Option {
	cfg := &persistenceOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		mongoModule(cfg),
	)
}

func mongoModule(cfg *persistenceOptions) fx.Option {
	if cfg.mongoConfig != nil {
		return mongo.NewMongoModule(mongo.WithMongoConfig(*cfg.mongoConfig))
	}
	return mongo.NewMongoModule()
}
