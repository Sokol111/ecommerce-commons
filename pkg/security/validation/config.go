package validation

import "fmt"

// Config holds the configuration for JWT token validation (incoming requests).
type Config struct {
	// JwksURL is the URL to fetch the JSON Web Key Set for verifying tokens.
	// Example: "http://logto:3001/oidc/jwks"
	JwksURL string `koanf:"jwks-url"`

	// Audience is the expected audience (aud) claim in JWT tokens.
	// This should match the API resource indicator registered in the OIDC provider.
	// Example: "https://api.sokolshop.com"
	Audience string `koanf:"audience"`
}

// ApplyDefaults is a no-op for validation config (no defaults to set).
func (c *Config) ApplyDefaults() {}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.JwksURL == "" {
		return fmt.Errorf("security.jwks.jwks-url is required")
	}
	return nil
}
