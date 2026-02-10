package mongo

import (
	"time"
)

// Config holds the MongoDB connection configuration.
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
}

func applyDefaults(cfg *Config) {
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
		cfg.ConnectTimeout = 5 * time.Second // Default: 5 seconds
	}
	if cfg.ServerSelectTimeout == 0 {
		cfg.ServerSelectTimeout = 5 * time.Second // Default: 5 seconds (fast-fail)
	}
	if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = 10 * time.Second // Default: 10 seconds for queries
	}
}
