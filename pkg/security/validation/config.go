package validation

import "fmt"

// Config holds the configuration for JWT token validation (incoming requests).
type Config struct {
	Enabled *bool `koanf:"enabled"`
	// JwksURL is the URL to fetch the JSON Web Key Set for verifying tokens.
	// Example: "http://logto:3001/oidc/jwks"
	JwksURL string `koanf:"jwks-url"`

	// Issuer is the expected issuer (iss) claim in JWT tokens.
	// Example: "https://auth.sokolshop.com/oidc"
	Issuer string `koanf:"issuer"`

	// Audience is the expected audience (aud) claim in JWT tokens.
	// This should match the API resource indicator registered in the OIDC provider.
	// Example: "https://api.sokolshop.com"
	Audience string `koanf:"audience"`
}

// ApplyDefaults is a no-op for validation config (no defaults to set).
func (c *Config) ApplyDefaults() {
	if c.Enabled == nil {
		if c.JwksURL == "" && c.Issuer == "" && c.Audience == "" {
			c.Enabled = new(false)
		} else {
			c.Enabled = new(true)
		}
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// If nothing is configured, the module is optional — skip validation.
	if c.Enabled != nil && !*c.Enabled {
		return nil
	}
	if c.JwksURL == "" {
		return fmt.Errorf("security.jwks.jwks-url is required")
	}
	if c.Issuer == "" {
		return fmt.Errorf("security.jwks.issuer is required")
	}
	if c.Audience == "" {
		return fmt.Errorf("security.jwks.audience is required")
	}
	return nil
}
