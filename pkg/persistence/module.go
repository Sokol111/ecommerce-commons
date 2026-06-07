package persistence

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
)

// persistenceOptions holds internal configuration for the persistence module.
type persistenceOptions struct {
	mongoConfig       *mongo.Config
	disableMigrations bool
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

// WithoutMigrations disables automatic migrations on startup.
// Use when migrations are managed externally (e.g. by tenant.NewModule()).
func WithoutMigrations() Option {
	return func(opts *persistenceOptions) {
		opts.disableMigrations = true
	}
}

// NewPersistenceModule provides persistence layer components for dependency injection.
//
// Options:
//   - WithMongoConfig: provide static Mongo Config (useful for tests)
//   - WithoutMigrations: skip automatic migrations (use with tenant.NewModule())
//
// Example usage:
//
//	// Production - loads config from koanf, runs migrations
//	persistence.NewPersistenceModule()
//
//	// Multi-tenant - migrations managed by tenant.NewModule()
//	persistence.NewPersistenceModule(persistence.WithoutMigrations())
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

	if cfg.disableMigrations {
		modules = append(modules, fx.Provide(provideAdminDatabase))
	}

	return fx.Options(modules...)
}

func mongoModule(cfg *persistenceOptions) fx.Option {
	var mongoOpts []mongo.Option
	if cfg.mongoConfig != nil {
		mongoOpts = append(mongoOpts, mongo.WithMongoConfig(*cfg.mongoConfig))
	}
	if cfg.disableMigrations {
		mongoOpts = append(mongoOpts, mongo.WithoutMigrations())
	}
	return mongo.NewMongoModule(mongoOpts...)
}

func provideAdminDatabase(admin mongo.Admin) *mongodriver.Database {
	return admin.GetDatabase()
}
