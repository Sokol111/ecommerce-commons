package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewViper_Success(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  host: localhost

database:
  host: localhost
  port: 5432
  name: testdb
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 8080, v.GetInt("server.port"))
	assert.Equal(t, "localhost", v.GetString("server.host"))
	assert.Equal(t, "localhost", v.GetString("database.host"))
	assert.Equal(t, 5432, v.GetInt("database.port"))
	assert.Equal(t, "testdb", v.GetString("database.name"))
}

func TestNewViper_FileNotFound(t *testing.T) {
	// Arrange
	appCfg := AppConfig{
		ConfigFile:     "/nonexistent/path/config.yaml",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.Error(t, err)
	assert.Nil(t, v)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestNewViper_InvalidYAML(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
server:
  port: 8080
  host: localhost
invalid yaml syntax here: [[[
`
	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.Error(t, err)
	assert.Nil(t, v)
}

func TestNewViper_EnvVarOverride(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  host: localhost
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable to override config file value
	t.Setenv("SERVER_PORT", "9000")

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.NoError(t, err)
	// Environment variable should override config file value
	assert.Equal(t, 9000, v.GetInt("server.port"))
}

func TestNewViper_EnvKeyReplacer(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
my-service:
  some.nested.value: "from-file"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test that dots and dashes in env var names are replaced with underscores
	t.Setenv("MY_SERVICE_SOME_NESTED_VALUE", "from-env")

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.NoError(t, err)
	// my-service.some.nested.value should be overridden by MY_SERVICE_SOME_NESTED_VALUE
	assert.Equal(t, "from-env", v.GetString("my-service.some.nested.value"))
}

func TestNewViper_JSONFormat(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	configContent := `{
  "server": {
    "port": 8080,
    "host": "localhost"
  },
  "enabled": true
}`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 8080, v.GetInt("server.port"))
	assert.Equal(t, "localhost", v.GetString("server.host"))
	assert.True(t, v.GetBool("enabled"))
}

func TestNewViper_AllSettings(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
database:
  host: localhost
logging:
  level: info
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.NoError(t, err)

	allSettings := v.AllSettings()
	assert.NotEmpty(t, allSettings)
	assert.Contains(t, allSettings, "server")
	assert.Contains(t, allSettings, "database")
	assert.Contains(t, allSettings, "logging")

	allKeys := v.AllKeys()
	assert.Contains(t, allKeys, "server.port")
	assert.Contains(t, allKeys, "database.host")
	assert.Contains(t, allKeys, "logging.level")
}

func TestNewViper_EmptyConfigFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create empty file
	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	appCfg := AppConfig{
		ConfigFile:     configFile,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	// Act
	v, err := newViper(appCfg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Empty(t, v.AllSettings())
}

func TestNewViper_SubWithEnvOverride(t *testing.T) {
	t.Run("BasicSubConfig", func(t *testing.T) {
		type DatabaseConfig struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
		}

		// Arrange
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		configContent := `
database:
  host: localhost
  port: 5432
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		appCfg := AppConfig{
			ConfigFile:     configFile,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		}

		// Act
		v, err := newViper(appCfg)
		require.NoError(t, err)

		dbSub := v.Sub("database")
		require.NotNil(t, dbSub, "database sub-config should not be nil")

		var dbConfig DatabaseConfig
		err = dbSub.Unmarshal(&dbConfig)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", dbConfig.Host)
		assert.Equal(t, 5432, dbConfig.Port)
	})

	t.Run("SubConfigWithEnvOverride", func(t *testing.T) {
		type CacheConfig struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
			TTL  int    `mapstructure:"ttl"`
		}

		// Arrange
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		configContent := `
cache:
  host: localhost
  port: 6379
  ttl: 300
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Set environment variable to override cache port
		t.Setenv("CACHE_PORT", "6380")

		appCfg := AppConfig{
			ConfigFile:     configFile,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		}

		// Act
		v, err := newViper(appCfg)
		require.NoError(t, err)

		cacheSub := v.Sub("cache")
		require.NotNil(t, cacheSub, "cache sub-config should not be nil")

		var cacheConfig CacheConfig
		err = cacheSub.Unmarshal(&cacheConfig)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", cacheConfig.Host)
		assert.Equal(t, 6380, cacheConfig.Port) // overridden by env var
		assert.Equal(t, 300, cacheConfig.TTL)
	})

	t.Run("NestedSubConfigWithEnvOverride", func(t *testing.T) {
		type Credentials struct {
			Username     string `mapstructure:"user-name"`
			Password     string `mapstructure:"password"`
			DatabaseName string `mapstructure:"database-name"`
		}

		type DatabaseConfig struct {
			Host        string      `mapstructure:"host"`
			Port        int         `mapstructure:"port"`
			Credentials Credentials `mapstructure:"credentials"`
		}

		// Arrange
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		configContent := `
database:
  host: localhost
  port: 5432
  credentials:
    user-name: admin
    password: secret
    database-name: testdb
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Set environment variables to override nested credentials with dashes
		t.Setenv("DATABASE_CREDENTIALS_USER_NAME", "produser")
		t.Setenv("DATABASE_CREDENTIALS_PASSWORD", "prodpass")
		t.Setenv("DATABASE_CREDENTIALS_DATABASE_NAME", "productiondb")

		appCfg := AppConfig{
			ConfigFile:     configFile,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		}

		// Act
		v, err := newViper(appCfg)
		require.NoError(t, err)

		dbSub := v.Sub("database")
		require.NotNil(t, dbSub)

		var dbConfig DatabaseConfig
		err = dbSub.Unmarshal(&dbConfig)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", dbConfig.Host)
		assert.Equal(t, 5432, dbConfig.Port)
		assert.Equal(t, "produser", dbConfig.Credentials.Username) // user-name overridden by USER_NAME
		assert.Equal(t, "prodpass", dbConfig.Credentials.Password)
		assert.Equal(t, "productiondb", dbConfig.Credentials.DatabaseName) // database-name overridden by DATABASE_NAME
	})

	t.Run("MultiLevelNestedSubConfigWithEnvOverride", func(t *testing.T) {
		type Pool struct {
			Min int `mapstructure:"min"`
			Max int `mapstructure:"max"`
		}

		type DatabaseConfig struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
			Pool Pool   `mapstructure:"pool"`
		}

		// Arrange
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		configContent := `
database:
  host: localhost
  port: 5432
  pool:
    min: 5
    max: 20
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Set environment variable to override pool max value
		t.Setenv("DATABASE_POOL_MAX", "50")

		appCfg := AppConfig{
			ConfigFile:     configFile,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		}

		// Act
		v, err := newViper(appCfg)
		require.NoError(t, err)

		dbSub := v.Sub("database")
		require.NotNil(t, dbSub)

		var dbConfig DatabaseConfig
		err = dbSub.Unmarshal(&dbConfig)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", dbConfig.Host)
		assert.Equal(t, 5432, dbConfig.Port)
		assert.Equal(t, 5, dbConfig.Pool.Min)
		assert.Equal(t, 50, dbConfig.Pool.Max) // overridden by env var
	})

	t.Run("SubConfigForMissingSection", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		appCfg := AppConfig{
			ConfigFile:     configFile,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		}

		// Act
		v, err := newViper(appCfg)
		require.NoError(t, err)

		// Try to get non-existent sub-config
		dbSub := v.Sub("database")

		// Assert
		assert.Nil(t, dbSub, "sub-config for missing section should be nil")
	})
}
