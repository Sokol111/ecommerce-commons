package tenant

import (
	"fmt"
	"net/http"
	"strings"
)

// Resolver resolves a tenant slug from an HTTP request.
type Resolver interface {
	// Resolve extracts the tenant slug from the request.
	// Returns the tenant slug and nil error on success.
	// Returns empty string and nil if no tenant is present (optional tenant).
	// Returns error if tenant resolution fails unexpectedly.
	Resolve(r *http.Request) (string, error)
}

// SubdomainResolver extracts the tenant slug from the first subdomain segment of the request host.
// For example:
//   - "acme.sokolshop.com" → "acme"
//   - "acme.sokolshop.com:8080" → "acme"
//   - "sokolshop.com" → "" (no subdomain)
//   - "localhost" → "" (no subdomain)
type SubdomainResolver struct{}

// NewSubdomainResolver creates a new SubdomainResolver.
func NewSubdomainResolver() *SubdomainResolver {
	return &SubdomainResolver{}
}

// Resolve extracts the tenant slug from the request's subdomain.
func (r *SubdomainResolver) Resolve(req *http.Request) (string, error) {
	host := req.Host
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Need at least 2 dots for a subdomain (e.g. "acme.sokolshop.com")
	if strings.Count(host, ".") < 2 {
		return "", nil
	}

	// Everything before the first dot is the subdomain
	slug := host[:strings.Index(host, ".")]
	return strings.ToLower(slug), nil
}

// HeaderResolver extracts the tenant slug from a request header.
type HeaderResolver struct {
	headerName string
}

// NewHeaderResolver creates a new HeaderResolver.
// headerName is the HTTP header key (e.g. "X-Tenant-Slug").
func NewHeaderResolver(headerName string) *HeaderResolver {
	return &HeaderResolver{headerName: headerName}
}

// Resolve extracts the tenant slug from the configured HTTP header.
func (r *HeaderResolver) Resolve(req *http.Request) (string, error) {
	return req.Header.Get(r.headerName), nil
}

// ChainResolver tries multiple resolvers in order and returns the first non-empty result.
type ChainResolver struct {
	resolvers []Resolver
}

// NewChainResolver creates a resolver that tries each resolver in order.
func NewChainResolver(resolvers ...Resolver) *ChainResolver {
	return &ChainResolver{resolvers: resolvers}
}

// Resolve tries each resolver in order and returns the first non-empty slug.
func (r *ChainResolver) Resolve(req *http.Request) (string, error) {
	for _, resolver := range r.resolvers {
		slug, err := resolver.Resolve(req)
		if err != nil {
			return "", fmt.Errorf("chain resolver: %w", err)
		}
		if slug != "" {
			return slug, nil
		}
	}
	return "", nil
}
