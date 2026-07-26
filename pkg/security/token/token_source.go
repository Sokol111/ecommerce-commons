package token

import (
	"context"
	"errors"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// NewTokenSource creates an oauth2.TokenSource using the client_credentials grant.
func NewTokenSource(cfg Config) (oauth2.TokenSource, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TokenURL == "" {
		return nil, errors.New("client-credentials not configured: set client-id, client-secret, and token-url in security.client-credentials")
	}

	cc := &clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Scopes:       cfg.Scopes,
	}

	// Logto requires the "resource" parameter (RFC 8707) for audience-restricted tokens.
	if cfg.Resource != "" {
		cc.EndpointParams = url.Values{"resource": {cfg.Resource}}
	}

	return cc.TokenSource(context.Background()), nil
}
