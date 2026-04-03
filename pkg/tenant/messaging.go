package tenant

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

// HeaderKey is the Kafka header key for tenant slug propagation between services.
const HeaderKey = "x-tenant-slug"

// SaveToHeaders adds the tenant slug from context to the headers map.
// Returns the (possibly modified) headers map.
// If no tenant is in context, returns headers unchanged.
func SaveToHeaders(ctx context.Context, headers map[string]string) map[string]string {
	slug, ok := SlugFromContext(ctx)
	if !ok {
		return headers
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	headers[HeaderKey] = slug
	return headers
}

// ContextFromKafkaHeaders creates a context with the tenant ID extracted from Kafka record headers.
// If no tenant ID is present in headers, returns the context unchanged.
func ContextFromKafkaHeaders(ctx context.Context, headers []kgo.RecordHeader) context.Context {
	for _, h := range headers {
		if h.Key == HeaderKey {
			slug := string(h.Value)
			if slug != "" {
				return ContextWithSlug(ctx, slug)
			}
			return ctx
		}
	}
	return ctx
}
