package token

import "context"

// claimsKey is the context key for storing claims.
type claimsKey struct{}

// tokenKey is the context key for storing the raw token string.
type tokenKey struct{}

// ContextWithClaims returns a new context with the claims stored.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

// ClaimsFromContext retrieves claims from the context.
// Returns nil if no claims are present.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey{}).(*Claims)
	return claims
}

// ContextWithToken returns a new context with the raw token string stored.
// This is useful for token propagation in service-to-service calls.
func ContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey{}, token)
}

// TokenFromContext retrieves the raw token string from the context.
// Returns empty string if no token is present.
func TokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey{}).(string)
	return token
}
