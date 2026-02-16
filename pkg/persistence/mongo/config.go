package mongo

import (
	"fmt"
	"net/url"
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

	// Migration Settings
	Migrations MigrationConfig `mapstructure:"migrations"`
}

// MigrationConfig holds migration-specific configuration.
type MigrationConfig struct {
	// Enabled controls whether migrations run on startup
	Enabled bool `mapstructure:"enabled"`
	// Path to migrations directory
	Path string `mapstructure:"path"`
}

// BuildURI constructs a MongoDB connection string from Config.
// Returns ConnectionString if set, otherwise builds URI from individual fields.
func (c Config) BuildURI() string {
	if c.ConnectionString != "" {
		return c.ConnectionString
	}

	u := &url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.Database,
	}

	if c.Username != "" {
		u.User = url.UserPassword(c.Username, c.Password)
	}

	q := u.Query()
	if c.ReplicaSet != "" {
		q.Set("replicaSet", c.ReplicaSet)
	}
	if c.DirectConnection {
		q.Set("directConnection", "true")
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// applyDefaults sets default values for unset configuration fields.
func (c *Config) applyDefaults() {
	if c.MaxPoolSize == 0 {
		c.MaxPoolSize = 100 // Default: 100 connections
	}
	if c.MinPoolSize == 0 {
		c.MinPoolSize = 10 // Default: 10 connections
	}
	if c.MaxConnIdleTime == 0 {
		c.MaxConnIdleTime = 5 * time.Minute // Default: 5 minutes
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 5 * time.Second // Default: 5 seconds
	}
	if c.ServerSelectTimeout == 0 {
		c.ServerSelectTimeout = 5 * time.Second // Default: 5 seconds (fast-fail)
	}
	if c.QueryTimeout == 0 {
		c.QueryTimeout = 10 * time.Second // Default: 10 seconds for queries
	}
	// Migration defaults
	if c.Migrations.Path == "" {
		c.Migrations.Path = "/db/migrations"
	}
}

// validate checks if the Config has all required fields set.
func (c Config) validate() error {
	if c.ConnectionString != "" {
		return nil
	}
	if c.Host == "" || c.Port == 0 || c.Database == "" {
		return fmt.Errorf("invalid Mongo configuration: host, port, and database are required")
	}
	return nil
}
