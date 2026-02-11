package token

// noopValidator is a validator that always returns valid claims.
// Useful for tests where token validation should be bypassed.
type noopValidator struct{}

// newNoopValidator creates a new noop validator.
func newNoopValidator() Validator {
	return &noopValidator{}
}

// ValidateToken always returns admin claims with wildcard permission.
func (v *noopValidator) ValidateToken(token string) (*Claims, error) {
	return &Claims{
		UserID:      "test-user",
		Role:        "admin",
		Permissions: []string{WildcardPermission},
		Type:        "access",
	}, nil
}
