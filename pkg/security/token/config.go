package token

import "net/http"

// Config holds the configuration for JWT token validation (incoming requests).
type Config struct {
	// JwksURL is the URL to fetch the JSON Web Key Set for verifying tokens.
	// Example: "http://zitadel:8080/oauth/v2/keys"
	JwksURL string `koanf:"jwks-url"`

	// HostOverride overrides the Host header in HTTP requests to the JWKS endpoint.
	// Use when the server requires a specific Host header (e.g. Zitadel with ExternalDomain).
	HostOverride string `koanf:"host-override"`
}

// S2SConfig holds the configuration for service-to-service authentication (outgoing requests).
type S2SConfig struct {
	// ClientID is the OAuth2 client ID (Zitadel machine user ID).
	ClientID string `koanf:"client-id"`

	// ClientSecret is the OAuth2 client secret for the client_credentials flow.
	ClientSecret string `koanf:"client-secret"`

	// TokenURL is the OAuth2 token endpoint.
	// Example: "http://zitadel:8080/oauth/v2/token"
	TokenURL string `koanf:"token-url"`

	// HostOverride overrides the Host header in HTTP requests to the token endpoint.
	// Use when the server requires a specific Host header (e.g. Zitadel with ExternalDomain).
	HostOverride string `koanf:"host-override"`
}

// hostOverrideTransport wraps an http.RoundTripper and overrides the Host header.
type hostOverrideTransport struct {
	host string
	base http.RoundTripper
}

func (t *hostOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Host = t.host
	return t.base.RoundTrip(req)
}

func httpClientWithHostOverride(host string) *http.Client {
	return &http.Client{
		Transport: &hostOverrideTransport{host: host, base: http.DefaultTransport},
	}
}
