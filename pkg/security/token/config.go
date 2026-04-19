package token

// Config holds the configuration for JWT token validation (incoming requests).
type Config struct {
	// JwksURL is the URL to fetch the JSON Web Key Set for verifying tokens.
	// Example: "http://zitadel:8080/oauth/v2/keys"
	JwksURL string `koanf:"jwks-url"`
}

// S2SConfig holds the configuration for service-to-service authentication (outgoing requests).
type S2SConfig struct {
	// ClientID is the OAuth2 client ID (Zitadel machine user ID).
	ClientID string `koanf:"client-id"`

	// ClientSecret is the OAuth2 client secret for the client_credentials flow.
	// Mutually exclusive with PrivateKey.
	ClientSecret string `koanf:"client-secret"`

	// PrivateKey is a PEM-encoded RSA private key for private_key_jwt flow (RFC 7523).
	// Mutually exclusive with ClientSecret.
	PrivateKey string `koanf:"private-key"`

	// TokenURL is the OAuth2 token endpoint.
	// Example: "http://zitadel:8080/oauth/v2/token"
	TokenURL string `koanf:"token-url"`
}
