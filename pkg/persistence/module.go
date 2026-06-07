package persistence

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

// persistenceOptions holds internal configuration for the persistence module.
type persistenceOptions struct {
	mongoConfig      *mongo.Config
	enableMigrations bool
}

// Option is a functional option for configuring the persistence module.
type Option func(*persistenceOptions)

// WithMongoConfig provides a static Mongo Config (useful for tests).
// When set, the Mongo configuration will not be loaded from koanf.
func WithMongoConfig(cfg mongo.Config) Option {
	return func(opts *persistenceOptions) {
		opts.mongoConfig = &cfg
	}
}

// WithMigrations enables automatic database migrations on startup.
func WithMigrations() Option {
	return func(opts *persistenceOptions) {
		opts.enableMigrations = true
	}
}

// NewPersistenceModule provides persistence layer components for dependency injection.
//
// Options:
//   - WithMongoConfig: provide static Mongo Config (useful for tests)
//   - WithMigrations: enable automatic migrations on startup
//
// Example usage:
//
//	// Multi-tenant - migrations managed by tenant.NewModule()
//	persistence.NewPersistenceModule()
//
//	// Single-tenant - runs migrations on startup
//	persistence.NewPersistenceModule(persistence.WithMigrations())
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

	modules := []fx.Option{
		mongoModule(cfg),
	}

	return fx.Options(modules...)
}

func mongoModule(cfg *persistenceOptions) fx.Option {
	var mongoOpts []mongo.Option
	if cfg.mongoConfig != nil {
		mongoOpts = append(mongoOpts, mongo.WithMongoConfig(*cfg.mongoConfig))
	}
	if cfg.enableMigrations {
		mongoOpts = append(mongoOpts, mongo.WithMigrations())
	}
	return mongo.NewMongoModule(mongoOpts...)
}
