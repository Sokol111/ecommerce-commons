package token

import (
	"encoding/hex"

	"aidanwoods.dev/go-paseto"
)

// Validator validates tokens and returns claims.
type Validator interface {
	// ValidateToken validates a token and returns the claims.
	ValidateToken(token string) (*Claims, error)
}

// tokenValidator validates PASETO v4 public tokens using a public key.
// This is intended for use in microservices that need to validate tokens
// issued by the auth-service.
type tokenValidator struct {
	publicKey paseto.V4AsymmetricPublicKey
}

// newTokenValidator creates a new token validator with the given public key.
// The publicKey should be a hex-encoded 32-byte Ed25519 public key.
func newTokenValidator(config Config) (Validator, error) {
	keyBytes, err := hex.DecodeString(config.PublicKey)
	if err != nil {
		return nil, ErrInvalidPublicKey
	}

	if len(keyBytes) != 32 {
		return nil, ErrInvalidPublicKey
	}

	publicKey, err := paseto.NewV4AsymmetricPublicKeyFromBytes(keyBytes)
	if err != nil {
		return nil, ErrInvalidPublicKey
	}

	return &tokenValidator{
		publicKey: publicKey,
	}, nil
}

// ValidateToken validates a token and returns the claims.
func (v *tokenValidator) ValidateToken(tokenString string) (*Claims, error) {
	parser := paseto.NewParser()

	token, err := parser.ParseV4Public(v.publicKey, tokenString, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	subject, err := token.GetSubject()
	if err != nil {
		return nil, ErrInvalidToken
	}

	role, _ := token.GetString("role")      //nolint:errcheck // Optional claim, empty string is valid default
	tokenType, _ := token.GetString("type") //nolint:errcheck // Optional claim, empty string is valid default

	// Parse permissions from token
	var permissions []string
	_ = token.Get("permissions", &permissions) //nolint:errcheck // Optional claim, nil slice is valid default

	// Parse time claims
	iat, _ := token.GetIssuedAt()   //nolint:errcheck // Optional claim, zero time is valid default
	exp, _ := token.GetExpiration() //nolint:errcheck // Optional claim, zero time is valid default
	nbf, _ := token.GetNotBefore()  //nolint:errcheck // Optional claim, zero time is valid default

	return &Claims{
		UserID:      subject,
		Role:        role,
		Permissions: permissions,
		Type:        tokenType,
		IssuedAt:    iat,
		ExpiresAt:   exp,
		NotBefore:   nbf,
		token:       token,
	}, nil
}
