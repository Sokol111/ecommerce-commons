package mongo

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// Config holds the MongoDB connection configuration.
type Config struct {
	ConnectionString string `koanf:"connection-string"`
	Host             string `koanf:"host"`
	Port             int    `koanf:"port"`
	ReplicaSet       string `koanf:"replica-set"`
	Username         string `koanf:"username"`
	Password         string `koanf:"password"`
	Database         string `koanf:"database"`
	DirectConnection bool   `koanf:"direct-connection"`

	// Connection Pool Settings
	MaxPoolSize         uint64        `koanf:"max-pool-size"`         // Максимальна кількість з'єднань у пулі
	MinPoolSize         uint64        `koanf:"min-pool-size"`         // Мінімальна кількість з'єднань у пулі
	MaxConnIdleTime     time.Duration `koanf:"max-conn-idle-time"`    // Час простою з'єднання перед закриттям
	ConnectTimeout      time.Duration `koanf:"connect-timeout"`       // Таймаут підключення
	ServerSelectTimeout time.Duration `koanf:"server-select-timeout"` // Таймаут вибору сервера

	// Query Timeout Settings
	QueryTimeout time.Duration `koanf:"query-timeout"` // Максимальний час виконання запиту до БД

	// Read/Write Concern and Read Preference
	WriteConcern   WriteConcernConfig   `koanf:"write-concern"`
	ReadConcern    ReadConcernConfig    `koanf:"read-concern"`
	ReadPreference ReadPreferenceConfig `koanf:"read-preference"`

	// Migration Settings
	Migrations MigrationConfig `koanf:"migrations"`
}

// WriteConcernConfig holds write concern settings.
type WriteConcernConfig struct {
	// W specifies the write concern level: "majority", or a number (e.g. "1", "2").
	W string `koanf:"w"`
	// Journal requests acknowledgment that the write operation has been written to the journal.
	Journal *bool `koanf:"journal"`
}

// ReadConcernConfig holds read concern settings.
type ReadConcernConfig struct {
	// Level specifies the read concern level: "local", "majority", "linearizable", "snapshot", "available".
	Level string `koanf:"level"`
}

// ReadPreferenceConfig holds read preference settings.
type ReadPreferenceConfig struct {
	// Mode specifies the read preference mode: "primary", "primaryPreferred", "secondary",
	// "secondaryPreferred", "nearest".
	Mode string `koanf:"mode"`
	// MaxStaleness is the maximum replication lag for a secondary to be considered for read operations.
	MaxStaleness time.Duration `koanf:"max-staleness"`
}

// MigrationConfig holds migration-specific configuration.
type MigrationConfig struct {
	// Disabled controls whether migrations are skipped on startup
	Disabled bool `koanf:"disabled"`
	// Path to migrations directory
	Path string `koanf:"path"`
}

// BuildURI constructs a MongoDB connection string from Config.
// Returns ConnectionString if set (with Database injected into path if missing),
// otherwise builds URI from individual fields.
func (c Config) BuildURI() string {
	if c.ConnectionString != "" {
		if c.Database != "" {
			u, err := url.Parse(c.ConnectionString)
			if err == nil && (u.Path == "" || u.Path == "/") {
				u.Path = "/" + c.Database
				return u.String()
			}
		}
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
	if c.Port == 0 {
		c.Port = 27017 // Default MongoDB port
	}
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
	if c.ConnectionString == "" {
		if c.Host == "" || c.Port == 0 || c.Database == "" {
			return fmt.Errorf("invalid Mongo configuration: host, port, and database are required")
		}
	}
	if err := c.WriteConcern.validate(); err != nil {
		return err
	}
	if err := c.ReadConcern.validate(); err != nil {
		return err
	}
	if err := c.ReadPreference.validate(); err != nil {
		return err
	}
	return nil
}

var validReadPreferenceModes = map[string]readpref.Mode{
	"primary":            readpref.PrimaryMode,
	"primaryPreferred":   readpref.PrimaryPreferredMode,
	"secondary":          readpref.SecondaryMode,
	"secondaryPreferred": readpref.SecondaryPreferredMode,
	"nearest":            readpref.NearestMode,
}

func (c WriteConcernConfig) validate() error {
	if c.W == "" {
		return nil
	}
	if c.W == "majority" {
		return nil
	}
	if n, err := strconv.Atoi(c.W); err != nil || n < 0 {
		return fmt.Errorf("invalid write concern w %q: must be \"majority\" or a non-negative integer", c.W)
	}
	return nil
}

var validReadConcernLevels = map[string]struct{}{
	"local":        {},
	"majority":     {},
	"linearizable": {},
	"available":    {},
	"snapshot":     {},
}

func (c ReadConcernConfig) validate() error {
	if c.Level == "" {
		return nil
	}
	if _, ok := validReadConcernLevels[c.Level]; !ok {
		return fmt.Errorf("invalid read concern level %q: must be one of local, majority, linearizable, available, snapshot", c.Level)
	}
	return nil
}

func (c ReadPreferenceConfig) validate() error {
	if c.Mode == "" {
		return nil
	}
	if _, ok := validReadPreferenceModes[c.Mode]; !ok {
		return fmt.Errorf("invalid read preference mode %q: must be one of primary, primaryPreferred, secondary, secondaryPreferred, nearest", c.Mode)
	}
	return nil
}

// buildWriteConcern constructs a WriteConcern from config.
// Returns nil if no write concern is configured.
func (c WriteConcernConfig) buildWriteConcern() *writeconcern.WriteConcern {
	if c.W == "" && c.Journal == nil {
		return nil
	}

	wc := &writeconcern.WriteConcern{}

	if c.W != "" {
		if n, err := strconv.Atoi(c.W); err == nil {
			wc.W = n
		} else {
			wc.W = c.W // e.g. "majority"
		}
	}
	if c.Journal != nil {
		wc.Journal = c.Journal
	}

	return wc
}

// buildReadConcern constructs a ReadConcern from config.
// Returns nil if no read concern level is configured.
// Assumes validate() has been called to ensure Level is valid.
func (c ReadConcernConfig) buildReadConcern() *readconcern.ReadConcern {
	switch c.Level {
	case "local":
		return readconcern.Local()
	case "majority":
		return readconcern.Majority()
	case "linearizable":
		return readconcern.Linearizable()
	case "available":
		return readconcern.Available()
	case "snapshot":
		return readconcern.Snapshot()
	default:
		return nil
	}
}

// buildReadPreference constructs a ReadPreference from config.
// Returns nil if no read preference mode is configured.
// Assumes validate() has been called to ensure Mode is valid.
func (c ReadPreferenceConfig) buildReadPreference() *readpref.ReadPref {
	mode, ok := validReadPreferenceModes[c.Mode]
	if !ok {
		return nil
	}

	var opts []readpref.Option
	if c.MaxStaleness > 0 {
		opts = append(opts, readpref.WithMaxStaleness(c.MaxStaleness))
	}

	rp, _ := readpref.New(mode, opts...)
	return rp
}
