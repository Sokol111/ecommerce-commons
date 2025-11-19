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
	os.Setenv("SERVER_PORT", "9000")
	defer os.Unsetenv("SERVER_PORT")

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
	os.Setenv("MY_SERVICE_SOME_NESTED_VALUE", "from-env")
	defer os.Unsetenv("MY_SERVICE_SOME_NESTED_VALUE")

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
