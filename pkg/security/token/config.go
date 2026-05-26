package token

import "fmt"

// Config holds the configuration for service-to-service authentication (outgoing requests).
type Config struct {
	// ClientID is the OAuth2 client ID for the client_credentials flow.
	ClientID string `koanf:"client-id"`

	// ClientSecret is the OAuth2 client secret for the client_credentials flow.
	ClientSecret string `koanf:"client-secret"`

	// TokenURL is the OAuth2 token endpoint.
	// Example: "http://logto:3001/oidc/token"
	TokenURL string `koanf:"token-url"`

	// Resource is the API resource indicator (RFC 8707) for token requests.
	// When set, it is sent as the "resource" parameter in client_credentials requests.
	// Example: "https://api.sokolshop.com"
	Resource string `koanf:"resource"`

	// Scopes is the list of scopes to request. Defaults to ["openid"] if empty.
	Scopes []string `koanf:"scopes"`
}

// ApplyDefaults sets default values for unset configuration fields.
func (c *Config) ApplyDefaults() {
	if len(c.Scopes) == 0 {
		c.Scopes = []string{"openid"}
	}
}

// Validate validates the configuration.
// Returns nil if the config is completely empty (module not configured).
func (c *Config) Validate() error {
	// If nothing is configured, the module is optional — skip validation.
	if c.ClientID == "" && c.ClientSecret == "" && c.TokenURL == "" {
		return nil
	}
	if c.ClientID == "" {
		return fmt.Errorf("security.client-credentials.client-id is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("security.client-credentials.client-secret is required")
	}
	if c.TokenURL == "" {
		return fmt.Errorf("security.client-credentials.token-url is required")
	}
	return nil
}
