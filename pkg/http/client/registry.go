package client

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
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

// registryParams holds the optional dependencies for creating a Registry.
type registryParams struct {
	fx.In
	K           *koanf.Koanf
	TokenSource oauth2.TokenSource `optional:"true"`
}

// newRegistry creates a Registry by scanning all keys under "clients.*" in koanf,
// loading each config and creating an HTTP client for it.
// When tokenSource is provided, all clients are wrapped with OAuth2 Bearer token
// injection for M2M service calls.
func newRegistry(p registryParams) (*Registry, error) {
	r := &Registry{clients: make(map[string]registryEntry)}

	clientsKoanf := p.K.Cut("clients")
	seen := make(map[string]bool)

	for _, key := range clientsKoanf.Keys() {
		name := strings.SplitN(key, ".", 2)[0]
		if seen[name] {
			continue
		}
		seen[name] = true

		cfg, err := config.Load[Config](p.K, "clients."+name, nil)
		if err != nil {
			return nil, fmt.Errorf("http client %q: %w", name, err)
		}

		client := newHTTPClient(cfg, p.TokenSource)

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
	return fx.Provide(newRegistry)
}
