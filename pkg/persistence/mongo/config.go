package mongo

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ConnectionString string `mapstructure:"connection-string"`
	Host             string `mapstructure:"host"`
	Port             int    `mapstructure:"port"`
	ReplicaSet       string `mapstructure:"replica-set"`
	Username         string `mapstructure:"username"`
	Password         string `mapstructure:"password"`
	Database         string `mapstructure:"database"`
	DirectConnection bool   `mapstructure:"direct-connection"`

	// Connection Pool Settings
	MaxPoolSize         uint64        `mapstructure:"max-pool-size"`         // Максимальна кількість з'єднань у пулі
	MinPoolSize         uint64        `mapstructure:"min-pool-size"`         // Мінімальна кількість з'єднань у пулі
	MaxConnIdleTime     time.Duration `mapstructure:"max-conn-idle-time"`    // Час простою з'єднання перед закриттям
	ConnectTimeout      time.Duration `mapstructure:"connect-timeout"`       // Таймаут підключення
	ServerSelectTimeout time.Duration `mapstructure:"server-select-timeout"` // Таймаут вибору сервера

	// Query Timeout Settings
	QueryTimeout time.Duration `mapstructure:"query-timeout"` // Максимальний час виконання запиту до БД

	// Retry Settings
	MaxRetries      int           `mapstructure:"max-retries"`      // Максимальна кількість повторів при помилках
	RetryDelay      time.Duration `mapstructure:"retry-delay"`      // Затримка між повторами
	RetryableErrors bool          `mapstructure:"retryable-errors"` // Чи вмикати автоматичний retry
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("mongo").Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load mongo config: %w", err)
	}

	// Set default values for connection pool if not specified
	if cfg.MaxPoolSize == 0 {
		cfg.MaxPoolSize = 100 // Default: 100 connections
	}
	if cfg.MinPoolSize == 0 {
		cfg.MinPoolSize = 10 // Default: 10 connections
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 5 * time.Minute // Default: 5 minutes
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second // Default: 10 seconds
	}
	if cfg.ServerSelectTimeout == 0 {
		cfg.ServerSelectTimeout = 30 * time.Second // Default: 30 seconds
	}
	if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = 30 * time.Second // Default: 30 seconds for queries
	}

	return cfg, nil
}
