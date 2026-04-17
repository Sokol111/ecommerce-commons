package token

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// testValidator is a validator that decodes tokens from base64-encoded JSON.
// Useful for integration/e2e tests where real PASETO validation is not needed.
// Tokens can be generated using GenerateTestToken.
type testValidator struct{}

// testTokenPayload is the JSON structure for test tokens.
type testTokenPayload struct {
	Tenant      string   `json:"tenant,omitempty"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// newTestValidator creates a new test validator.
func newTestValidator() Validator {
	return &testValidator{}
}

// ValidateToken decodes a base64-encoded JSON token and returns Claims.
func (v *testValidator) ValidateToken(token string) (*Claims, error) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid test token: %w", err)
	}

	var payload testTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("invalid test token payload: %w", err)
	}

	return &Claims{
		Tenant:      payload.Tenant,
		Role:        payload.Role,
		Permissions: payload.Permissions,
	}, nil
}

// GenerateTestToken creates a base64-encoded JSON token from claims.
// Use this in tests to create tokens that testValidator can decode.
func GenerateTestToken(role string, permissions []string) string {
	payload := testTokenPayload{
		Role:        role,
		Permissions: permissions,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal test token: %v", err))
	}
	return base64.StdEncoding.EncodeToString(data)
}

// GenerateAdminTestToken creates a test token with admin role and wildcard permissions.
func GenerateAdminTestToken() string {
	return GenerateTestToken("admin", []string{WildcardPermission})
}
