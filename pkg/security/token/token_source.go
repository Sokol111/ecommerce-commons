package token

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// newTokenSource creates an oauth2.TokenSource using the client_credentials grant.
func newTokenSource(cfg S2SConfig) (oauth2.TokenSource, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TokenURL == "" {
		return nil, errors.New("S2S auth not configured: set client-id, client-secret, and token-url in security.s2s")
	}

	cc := &clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Scopes:       []string{"openid"},
	}

	ctx := context.Background()
	if cfg.HostOverride != "" {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClientWithHostOverride(cfg.HostOverride))
	}
	return cc.TokenSource(ctx), nil
}

// noopTokenSource is a TokenSource that returns an empty token. Used in test/disabled modes.
type noopTokenSource struct{}

func (noopTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "noop"}, nil
}
