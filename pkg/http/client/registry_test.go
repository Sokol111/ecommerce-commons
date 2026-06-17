package client

import (
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKoanf(t *testing.T, data map[string]any) *koanf.Koanf {
	t.Helper()
	k := koanf.New(".")
	err := k.Load(confmap.Provider(data, "."), nil)
	require.NoError(t, err)
	return k
}

func TestNewRegistry_DiscoverClients(t *testing.T) {
	k := newTestKoanf(t, map[string]any{
		"clients.tenant-service.base-url":  "http://tenant:8080",
		"clients.catalog-service.base-url": "http://catalog:8080",
	})

	r, err := newRegistry(registryParams{K: k})
	require.NoError(t, err)

	client, err := r.Client("tenant-service")
	require.NoError(t, err)
	assert.NotNil(t, client)

	cfg, err := r.Config("tenant-service")
	require.NoError(t, err)
	assert.Equal(t, "http://tenant:8080", cfg.BaseURL)

	client, err = r.Client("catalog-service")
	require.NoError(t, err)
	assert.NotNil(t, client)

	cfg, err = r.Config("catalog-service")
	require.NoError(t, err)
	assert.Equal(t, "http://catalog:8080", cfg.BaseURL)
}

func TestNewRegistry_EmptyClients(t *testing.T) {
	k := newTestKoanf(t, map[string]any{
		"mongo.host": "localhost",
	})

	r, err := newRegistry(registryParams{K: k})
	require.NoError(t, err)
	assert.Empty(t, r.clients)
}

func TestRegistry_ClientNotFound(t *testing.T) {
	k := newTestKoanf(t, map[string]any{
		"clients.tenant-service.base-url": "http://tenant:8080",
	})

	r, err := newRegistry(registryParams{K: k})
	require.NoError(t, err)

	_, err = r.Client("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in registry")
}

func TestRegistry_ConfigNotFound(t *testing.T) {
	k := newTestKoanf(t, map[string]any{
		"clients.tenant-service.base-url": "http://tenant:8080",
	})

	r, err := newRegistry(registryParams{K: k})
	require.NoError(t, err)

	_, err = r.Config("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in registry")
}

func TestNewRegistry_InvalidConfig(t *testing.T) {
	k := newTestKoanf(t, map[string]any{
		"clients.bad-service.timeout": "10s", // missing base-url
	})

	_, err := newRegistry(registryParams{K: k})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad-service")
}
