package token

import "errors"

var (
	// ErrInvalidToken is returned when the token cannot be parsed or verified.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired.
	ErrExpiredToken = errors.New("token expired")
	// ErrInvalidPublicKey is returned when the public key is invalid.
	ErrInvalidPublicKey = errors.New("invalid public key")
)
