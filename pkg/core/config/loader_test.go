package config

import (
	"errors"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig is a sample config struct for testing the generic loader.
type testConfig struct {
	Host    string `koanf:"host"`
	Port    int    `koanf:"port"`
	Timeout int    `koanf:"timeout"`
}

func (c *testConfig) ApplyDefaults() {
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.Timeout == 0 {
		c.Timeout = 30
	}
}

func (c *testConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port < 0 || c.Port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	return nil
}

func TestLoad_WithOverride(t *testing.T) {
	k := koanf.New(".")

	override := &testConfig{Host: "override-host", Port: 9090}

	cfg, err := Load[testConfig](k, "test", override)
	require.NoError(t, err)

	assert.Equal(t, "override-host", cfg.Host)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, 30, cfg.Timeout) // default applied
}

func TestLoad_FromKoanf(t *testing.T) {
	k := koanf.New(".")
	_ = k.Load(confmap{"test": map[string]interface{}{"host": "koanf-host", "port": 3000}}, nil)

	cfg, err := Load[testConfig](k, "test", nil)
	require.NoError(t, err)

	assert.Equal(t, "koanf-host", cfg.Host)
	assert.Equal(t, 3000, cfg.Port)
	assert.Equal(t, 30, cfg.Timeout) // default applied
}

func TestLoad_EmptyKoanf_DefaultsApplied(t *testing.T) {
	k := koanf.New(".")

	// No key in koanf, no override → empty struct → defaults applied → validation runs
	cfg, err := Load[testConfig](k, "test", &testConfig{Host: "required"})
	require.NoError(t, err)

	assert.Equal(t, "required", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, 30, cfg.Timeout)
}

func TestLoad_ValidationFails(t *testing.T) {
	k := koanf.New(".")

	// Host is empty → validation should fail
	_, err := Load[testConfig](k, "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid test config")
	assert.Contains(t, err.Error(), "host is required")
}

func TestLoad_DefaultsBeforeValidation(t *testing.T) {
	k := koanf.New(".")
	_ = k.Load(confmap{"test": map[string]interface{}{"host": "localhost", "port": -1}}, nil)

	// Port -1 is set explicitly → defaults won't override it → validation should catch it
	_, err := Load[testConfig](k, "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 0 and 65535")
}

// confmap is a simple koanf provider for testing.
type confmap map[string]interface{}

func (c confmap) ReadBytes() ([]byte, error) { return nil, errors.New("not supported") }
func (c confmap) Read() (map[string]interface{}, error) {
	return c, nil
}
