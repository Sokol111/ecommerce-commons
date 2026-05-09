package token

import (
	"context"
	"errors"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// newTokenSource creates an oauth2.TokenSource using the client_credentials grant.
func newTokenSource(cfg ClientCredentialsConfig) (oauth2.TokenSource, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TokenURL == "" {
		return nil, errors.New("ClientCredentialsConfig auth not configured: set client-id, client-secret, and token-url in security.client-credentials")
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

// noopTokenSource is a TokenSource that returns an empty token. Used in test/disabled modes.
type noopTokenSource struct{}

func (noopTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "noop"}, nil
}
