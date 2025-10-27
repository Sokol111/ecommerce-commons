package migrations

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Enabled        bool   `mapstructure:"enabled"`
	MigrationsPath string `mapstructure:"migrations-path"`
	CollectionName string `mapstructure:"collection-name"`
	AutoMigrate    bool   `mapstructure:"auto-migrate"`
	LockingTimeout int    `mapstructure:"locking-timeout"`
}

func (c Config) GetLockingTimeoutDuration() time.Duration {
	return time.Duration(c.LockingTimeout) * time.Minute
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config

	if !v.IsSet("mongo.migrations") {
		return Config{Enabled: false}, nil
	}

	if err := v.Sub("mongo.migrations").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load mongo migrations config: %w", err)
	}

	// Set defaults
	if cfg.CollectionName == "" {
		cfg.CollectionName = "schema_migrations"
	}
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "./db/migrations"
	}
	if cfg.LockingTimeout == 0 {
		cfg.LockingTimeout = 5
	}

	return cfg, nil
}
