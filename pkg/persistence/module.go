package persistence

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

// persistenceOptions holds internal configuration for the persistence module.
type persistenceOptions struct {
	mongoConfig      *mongo.Config
	tenantMigrations bool
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

// WithTenantMigrations enables per-tenant database migrations.
// When set, mongo.TenantSlugsProvider must be provided in the fx container.
func WithTenantMigrations() Option {
	return func(opts *persistenceOptions) {
		opts.tenantMigrations = true
	}
}

// NewPersistenceModule provides persistence layer components for dependency injection.
//
// Options:
//   - WithMongoConfig: provide static Mongo Config (useful for tests)
//
// Example usage:
//
//	// Production - loads config from koanf
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
	var mongoOpts []mongo.Option
	if cfg.mongoConfig != nil {
		mongoOpts = append(mongoOpts, mongo.WithMongoConfig(*cfg.mongoConfig))
	}
	if cfg.tenantMigrations {
		mongoOpts = append(mongoOpts, mongo.WithTenantMigrations())
	}
	return mongo.NewMongoModule(mongoOpts...)
}
