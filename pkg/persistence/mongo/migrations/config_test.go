package migrations

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GetLockingTimeoutDuration(t *testing.T) {
	tests := []struct {
		name           string
		lockingTimeout int
		expected       time.Duration
	}{
		{
			name:           "5 minutes",
			lockingTimeout: 5,
			expected:       5 * time.Minute,
		},
		{
			name:           "0 minutes",
			lockingTimeout: 0,
			expected:       0,
		},
		{
			name:           "10 minutes",
			lockingTimeout: 10,
			expected:       10 * time.Minute,
		},
		{
			name:           "1 minute",
			lockingTimeout: 1,
			expected:       1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{LockingTimeout: tt.lockingTimeout}
			assert.Equal(t, tt.expected, cfg.GetLockingTimeoutDuration())
		})
	}
}

func TestNewConfig_FullConfig(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
    migrations-path: "./custom/migrations"
    collection-name: "custom_migrations"
    auto-migrate: true
    locking-timeout: 10
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "./custom/migrations", cfg.MigrationsPath)
	assert.Equal(t, "custom_migrations", cfg.CollectionName)
	assert.True(t, cfg.AutoMigrate)
	assert.Equal(t, 10, cfg.LockingTimeout)
}

func TestNewConfig_WithDefaults(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
    auto-migrate: false
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "./db/migrations", cfg.MigrationsPath)
	assert.Equal(t, "schema_migrations", cfg.CollectionName)
	assert.False(t, cfg.AutoMigrate)
	assert.Equal(t, 5, cfg.LockingTimeout) // default value
}

func TestNewConfig_NoMigrationsSection(t *testing.T) {
	yamlConfig := `
mongo:
  uri: "mongodb://localhost:27017"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.MigrationsPath)
	assert.Empty(t, cfg.CollectionName)
	assert.False(t, cfg.AutoMigrate)
	assert.Equal(t, 0, cfg.LockingTimeout)
}

func TestNewConfig_EmptyViper(t *testing.T) {
	v := viper.New()

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
}

func TestNewConfig_OnlyEnabledField(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	// defaults should be applied
	assert.Equal(t, "./db/migrations", cfg.MigrationsPath)
	assert.Equal(t, "schema_migrations", cfg.CollectionName)
	assert.Equal(t, 5, cfg.LockingTimeout)
}

func TestNewConfig_CustomCollectionNameOnly(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
    collection-name: "my_schema_versions"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "my_schema_versions", cfg.CollectionName)
	// other defaults
	assert.Equal(t, "./db/migrations", cfg.MigrationsPath)
	assert.Equal(t, 5, cfg.LockingTimeout)
}

func TestNewConfig_CustomMigrationsPathOnly(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
    migrations-path: "/opt/migrations"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "/opt/migrations", cfg.MigrationsPath)
	// other defaults
	assert.Equal(t, "schema_migrations", cfg.CollectionName)
	assert.Equal(t, 5, cfg.LockingTimeout)
}

func TestNewConfig_ZeroLockingTimeout(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
    locking-timeout: 0
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	// zero should be replaced with default 5
	assert.Equal(t, 5, cfg.LockingTimeout)
}

func TestNewConfig_LargeLockingTimeout(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: true
    locking-timeout: 60
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.Equal(t, 60, cfg.LockingTimeout)
	assert.Equal(t, 60*time.Minute, cfg.GetLockingTimeoutDuration())
}

func TestNewConfig_DisabledMigrations(t *testing.T) {
	yamlConfig := `
mongo:
  migrations:
    enabled: false
    migrations-path: "./custom/migrations"
    collection-name: "custom_migrations"
    auto-migrate: true
    locking-timeout: 10
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	cfg, err := newConfig(v)

	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	// Other fields should still be loaded
	assert.Equal(t, "./custom/migrations", cfg.MigrationsPath)
	assert.Equal(t, "custom_migrations", cfg.CollectionName)
	assert.True(t, cfg.AutoMigrate)
	assert.Equal(t, 10, cfg.LockingTimeout)
}
