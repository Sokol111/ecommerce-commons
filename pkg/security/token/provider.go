package token

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Provider provides bearer tokens for service-to-service communication.
type Provider interface {
	Token(ctx context.Context) (string, error)
}

type oauth2TokenProvider struct {
	tokenSource oauth2.TokenSource
}

func (o *oauth2TokenProvider) Token(_ context.Context) (string, error) {
	t, err := o.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("oauth2 client credentials token: %w", err)
	}
	return t.AccessToken, nil
}

// newTokenProvider creates a TokenProvider based on the S2S configuration.
func newTokenProvider(cfg S2SConfig) Provider {
	if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TokenURL != "" {
		cc := &clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
		}
		return &oauth2TokenProvider{
			tokenSource: cc.TokenSource(context.Background()),
		}
	}
	return &unconfiguredTokenProvider{}
}

type unconfiguredTokenProvider struct{}

func (u *unconfiguredTokenProvider) Token(_ context.Context) (string, error) {
	return "", fmt.Errorf("S2S auth not configured: set client-id/client-secret/token-url in security.token")
}
