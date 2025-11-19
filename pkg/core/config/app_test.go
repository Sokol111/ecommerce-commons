package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppConfig_Success(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "test")
	os.Setenv(envAppServiceName, "test-service")
	os.Setenv(envAppServiceVersion, "1.0.0")

	// Act
	cfg, err := newAppConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "test", cfg.Environment)
	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.Equal(t, filepath.Join(defaultConfigDir, "config.test.yaml"), cfg.ConfigFile)
}

func TestNewAppConfig_MissingAppEnv(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppServiceName, "test-service")
	os.Setenv(envAppServiceVersion, "1.0.0")

	// Act
	_, err := newAppConfig()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), envAppEnv)
}

func TestNewAppConfig_MissingServiceName(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "test")
	os.Setenv(envAppServiceVersion, "1.0.0")

	// Act
	_, err := newAppConfig()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), envAppServiceName)
}

func TestNewAppConfig_MissingServiceVersion(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "test")
	os.Setenv(envAppServiceName, "test-service")

	// Act
	_, err := newAppConfig()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), envAppServiceVersion)
}

func TestNewAppConfig_CustomConfigFile(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "test")
	os.Setenv(envAppServiceName, "test-service")
	os.Setenv(envAppServiceVersion, "1.0.0")
	os.Setenv(envConfigFile, "/custom/path/config.yaml")

	// Act
	cfg, err := newAppConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "/custom/path/config.yaml", cfg.ConfigFile)
}

func TestNewAppConfig_CustomConfigDir(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "staging")
	os.Setenv(envAppServiceName, "test-service")
	os.Setenv(envAppServiceVersion, "1.0.0")
	os.Setenv(envConfigDir, "/etc/myapp")

	// Act
	cfg, err := newAppConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/etc/myapp", "config.staging.yaml"), cfg.ConfigFile)
}

func TestNewAppConfig_CustomConfigName(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "test")
	os.Setenv(envAppServiceName, "test-service")
	os.Setenv(envAppServiceVersion, "1.0.0")
	os.Setenv(envConfigName, "custom-config")

	// Act
	cfg, err := newAppConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(defaultConfigDir, "custom-config.yaml"), cfg.ConfigFile)
}

func TestNewAppConfig_CustomConfigDirAndName(t *testing.T) {
	// Arrange
	os.Clearenv()
	os.Setenv(envAppEnv, "pro")
	os.Setenv(envAppServiceName, "test-service")
	os.Setenv(envAppServiceVersion, "2.1.0")
	os.Setenv(envConfigDir, "/opt/config")
	os.Setenv(envConfigName, "app")

	// Act
	cfg, err := newAppConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "pro", cfg.Environment)
	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "2.1.0", cfg.ServiceVersion)
	assert.Equal(t, filepath.Join("/opt/config", "app.yaml"), cfg.ConfigFile)
}

func TestNewAppConfig_DifferentEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		expectedCfg string
	}{
		{
			name:        "local environment",
			env:         "local",
			expectedCfg: filepath.Join(defaultConfigDir, "config.local.yaml"),
		},
		{
			name:        "staging environment",
			env:         "staging",
			expectedCfg: filepath.Join(defaultConfigDir, "config.staging.yaml"),
		},
		{
			name:        "production environment",
			env:         "pro",
			expectedCfg: filepath.Join(defaultConfigDir, "config.pro.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			os.Clearenv()
			os.Setenv(envAppEnv, tt.env)
			os.Setenv(envAppServiceName, "test-service")
			os.Setenv(envAppServiceVersion, "1.0.0") // Act
			cfg, err := newAppConfig()

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.env, cfg.Environment)
			assert.Equal(t, tt.expectedCfg, cfg.ConfigFile)
		})
	}
}
