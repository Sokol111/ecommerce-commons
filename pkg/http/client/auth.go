package client

import (
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

// authTransport is an http.RoundTripper that injects an OAuth2 Bearer token
// into every outgoing request. It is intended for service-to-service M2M calls
// using client_credentials tokens.
type authTransport struct {
	base  http.RoundTripper
	token oauth2.TokenSource
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.token.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain access token: %w", err)
	}

	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	return t.base.RoundTrip(req)
}
