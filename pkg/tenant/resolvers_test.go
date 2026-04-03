package tenant

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubdomainResolver_WithSubdomain(t *testing.T) {
	resolver := NewSubdomainResolver()
	req, _ := http.NewRequest(http.MethodGet, "http://acme.sokolshop.com/api/products", nil)

	slug, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Equal(t, "acme", slug)
}

func TestSubdomainResolver_WithSubdomainAndPort(t *testing.T) {
	resolver := NewSubdomainResolver()
	req, _ := http.NewRequest(http.MethodGet, "http://acme.sokolshop.com:8080/api/products", nil)

	slug, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Equal(t, "acme", slug)
}

func TestSubdomainResolver_NoSubdomain(t *testing.T) {
	resolver := NewSubdomainResolver()
	req, _ := http.NewRequest(http.MethodGet, "http://sokolshop.com/api/products", nil)

	slug, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Empty(t, slug)
}

func TestSubdomainResolver_Localhost(t *testing.T) {
	resolver := NewSubdomainResolver()
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/products", nil)

	slug, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Empty(t, slug)
}

func TestSubdomainResolver_CaseInsensitive(t *testing.T) {
	resolver := NewSubdomainResolver()
	req, _ := http.NewRequest(http.MethodGet, "http://ACME.sokolshop.com/api/products", nil)

	slug, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Equal(t, "acme", slug)
}

func TestHeaderResolver_WithHeader(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/products", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")

	tenantID, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)
}

func TestHeaderResolver_NoHeader(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/products", nil)

	tenantID, err := resolver.Resolve(req)

	require.NoError(t, err)
	assert.Empty(t, tenantID)
}

func TestChainResolver_FirstResolverMatches(t *testing.T) {
	chain := NewChainResolver(
		NewHeaderResolver("X-Tenant-Slug"),
		NewSubdomainResolver(),
	)
	req, _ := http.NewRequest(http.MethodGet, "http://acme.sokolshop.com/api/products", nil)
	req.Header.Set("X-Tenant-Slug", "from-header")

	slug, err := chain.Resolve(req)

	require.NoError(t, err)
	assert.Equal(t, "from-header", slug)
}

func TestChainResolver_FallsBackToSecond(t *testing.T) {
	chain := NewChainResolver(
		NewHeaderResolver("X-Tenant-Slug"),
		NewSubdomainResolver(),
	)
	req, _ := http.NewRequest(http.MethodGet, "http://acme.sokolshop.com/api/products", nil)

	slug, err := chain.Resolve(req)

	require.NoError(t, err)
	assert.Equal(t, "acme", slug)
}

func TestChainResolver_NoMatch(t *testing.T) {
	chain := NewChainResolver(
		NewHeaderResolver("X-Tenant-Slug"),
		NewSubdomainResolver(),
	)
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/products", nil)

	slug, err := chain.Resolve(req)

	require.NoError(t, err)
	assert.Empty(t, slug)
}
