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

// testClaimsValidator is a validator that returns predefined claims.
// Useful for tests that need specific claims.
type testClaimsValidator struct {
	claims *Claims
}

// newTestClaimsValidator creates a validator that always returns the given claims.
func newTestClaimsValidator(claims Claims) Validator {
	return &testClaimsValidator{claims: &claims}
}

// ValidateToken always returns the predefined claims.
func (v *testClaimsValidator) ValidateToken(token string) (*Claims, error) {
	return v.claims, nil
}
