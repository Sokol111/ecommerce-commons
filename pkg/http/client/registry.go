package client

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
)

type registryEntry struct {
	client *http.Client
	config Config
}

// Registry holds pre-built HTTP clients keyed by service name.
// It discovers all clients configured under the "clients.*" koanf prefix.
type Registry struct {
	clients map[string]registryEntry
}

// NewRegistry creates a Registry by scanning all keys under "clients.*" in koanf,
// loading each config and creating an HTTP client for it.
func NewRegistry(k *koanf.Koanf) (*Registry, error) {
	r := &Registry{clients: make(map[string]registryEntry)}

	clientsKoanf := k.Cut("clients")
	seen := make(map[string]bool)

	for _, key := range clientsKoanf.Keys() {
		name := strings.SplitN(key, ".", 2)[0]
		if seen[name] {
			continue
		}
		seen[name] = true

		cfg, err := LoadConfig(k, "clients."+name)
		if err != nil {
			return nil, fmt.Errorf("http client %q: %w", name, err)
		}

		client, err := New(cfg)
		if err != nil {
			return nil, fmt.Errorf("http client %q: %w", name, err)
		}

		r.clients[name] = registryEntry{client: client, config: cfg}
	}

	return r, nil
}

// Client returns the HTTP client for the given service name.
func (r *Registry) Client(name string) (*http.Client, error) {
	e, ok := r.clients[name]
	if !ok {
		return nil, fmt.Errorf("http client %q not found in registry", name)
	}
	return e.client, nil
}

// Config returns the client configuration for the given service name.
func (r *Registry) Config(name string) (Config, error) {
	e, ok := r.clients[name]
	if !ok {
		return Config{}, fmt.Errorf("http client config %q not found in registry", name)
	}
	return e.config, nil
}

// RegistryModule provides the Registry via fx dependency injection.
func RegistryModule() fx.Option {
	return fx.Provide(NewRegistry)
}
