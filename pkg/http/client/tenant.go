package client

import (
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
)

// tenantTransport is an http.RoundTripper that propagates the tenant slug
// from the request context to the outgoing HTTP request as X-Tenant-Slug header.
// Used for service-to-service calls to ensure tenant context is preserved.
type tenantTransport struct {
	base http.RoundTripper
}

func (t *tenantTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if slug, ok := tenant.SlugFromContext(req.Context()); ok {
		req = req.Clone(req.Context())
		req.Header.Set(tenant.TenantSlugHeader, slug)
	}
	return t.base.RoundTrip(req)
}
