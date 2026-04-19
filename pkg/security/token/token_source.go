package token

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/jwt"
)

// newTokenSource creates an oauth2.TokenSource based on the S2S configuration.
// - If PrivateKey is set → JWT Profile / private_key_jwt (RFC 7523).
// - If ClientSecret is set → client_credentials.
// - Otherwise → returns error (fail fast).
func newTokenSource(cfg S2SConfig) (oauth2.TokenSource, error) {
	if cfg.ClientID != "" && cfg.PrivateKey != "" && cfg.TokenURL != "" {
		return newJWTTokenSource(cfg)
	}
	if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TokenURL != "" {
		return newClientCredentialsTokenSource(cfg), nil
	}
	return nil, errors.New("S2S auth not configured: set client-id + (private-key or client-secret) + token-url in security.s2s")
}

// newClientCredentialsTokenSource creates a TokenSource using OAuth2 client_credentials grant.
func newClientCredentialsTokenSource(cfg S2SConfig) oauth2.TokenSource {
	cc := &clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
	}
	return cc.TokenSource(context.Background())
}

// newJWTTokenSource creates a TokenSource that uses JWT Profile (RFC 7523) for S2S auth.
func newJWTTokenSource(cfg S2SConfig) (oauth2.TokenSource, error) {
	jwtCfg := &jwt.Config{
		Email:      cfg.ClientID,
		Subject:    cfg.ClientID,
		PrivateKey: []byte(cfg.PrivateKey),
		TokenURL:   cfg.TokenURL,
		Scopes:     []string{"openid"},
	}
	return jwtCfg.TokenSource(context.Background()), nil
}

// noopTokenSource is a TokenSource that returns an empty token. Used in test/disabled modes.
type noopTokenSource struct{}

func (noopTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "noop"}, nil
}
