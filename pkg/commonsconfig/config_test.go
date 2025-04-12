package commonsconfig

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type TestStruct struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temporary config file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temporary config file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temporary config file: %v", err)
	}
	return tmpFile.Name()
}

func TestLoadConfig_ValidConfig(t *testing.T) {
	configContent := `
port: 8080
host: localhost
`
	configFile := createTempConfigFile(t, configContent)
	defer os.Remove(configFile)

	conf, err := LoadConfig[TestStruct](configFile)

	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 8080, conf.Port)
	assert.Equal(t, "localhost", conf.Host)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	conf, err := LoadConfig[TestStruct]("nonexistent.yaml")

	assert.Error(t, err)
	assert.Nil(t, conf)
	assert.Contains(t, err.Error(), "config file [nonexistent.yaml] does not exist")
}

func TestLoadConfig_InvalidFormat(t *testing.T) {
	invalidContent := `
invalid_yaml_content
`
	configFile := createTempConfigFile(t, invalidContent)
	defer os.Remove(configFile)

	conf, err := LoadConfig[TestStruct](configFile)

	assert.Error(t, err)
	assert.Nil(t, conf)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	configFile := createTempConfigFile(t, "")
	defer os.Remove(configFile)

	conf, err := LoadConfig[TestStruct](configFile)

	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 0, conf.Port)
	assert.Equal(t, "", conf.Host)
}

func TestLoadConfig_EnvVariableOverride(t *testing.T) {
	configContent := `
port: 8080
host: localhost
`
	configFile := createTempConfigFile(t, configContent)
	defer os.Remove(configFile)

	os.Setenv("HOST", "127.0.0.1")
	defer os.Unsetenv("HOST")

	conf, err := LoadConfig[TestStruct](configFile)

	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 8080, conf.Port)
	assert.Equal(t, "127.0.0.1", conf.Host)
}
