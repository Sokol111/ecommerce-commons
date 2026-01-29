package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClaims_HasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		check       string
		want        bool
	}{
		{
			name:        "has permission",
			permissions: []string{"read", "write", "delete"},
			check:       "write",
			want:        true,
		},
		{
			name:        "does not have permission",
			permissions: []string{"read", "write"},
			check:       "delete",
			want:        false,
		},
		{
			name:        "empty permissions",
			permissions: []string{},
			check:       "read",
			want:        false,
		},
		{
			name:        "nil permissions",
			permissions: nil,
			check:       "read",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{Permissions: tt.permissions}
			got := c.HasPermission(tt.check)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaims_HasAnyPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		check       []string
		want        bool
	}{
		{
			name:        "has one of required permissions",
			permissions: []string{"read", "write"},
			check:       []string{"write", "delete"},
			want:        true,
		},
		{
			name:        "has all required permissions",
			permissions: []string{"read", "write", "delete"},
			check:       []string{"read", "write"},
			want:        true,
		},
		{
			name:        "has none of required permissions",
			permissions: []string{"read"},
			check:       []string{"write", "delete"},
			want:        false,
		},
		{
			name:        "empty user permissions",
			permissions: []string{},
			check:       []string{"read"},
			want:        false,
		},
		{
			name:        "empty required permissions",
			permissions: []string{"read", "write"},
			check:       []string{},
			want:        false,
		},
		{
			name:        "both empty",
			permissions: []string{},
			check:       []string{},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{Permissions: tt.permissions}
			got := c.HasAnyPermission(tt.check)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaims_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired - future",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired - past",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "expired - just now",
			expiresAt: time.Now().Add(-1 * time.Millisecond),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{ExpiresAt: tt.expiresAt}
			got := c.IsExpired()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaims_IsAccess(t *testing.T) {
	tests := []struct {
		name      string
		tokenType string
		want      bool
	}{
		{
			name:      "access token",
			tokenType: "access",
			want:      true,
		},
		{
			name:      "refresh token",
			tokenType: "refresh",
			want:      false,
		},
		{
			name:      "empty type",
			tokenType: "",
			want:      false,
		},
		{
			name:      "unknown type",
			tokenType: "unknown",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{Type: tt.tokenType}
			got := c.IsAccess()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaims_IsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		tokenType string
		want      bool
	}{
		{
			name:      "refresh token",
			tokenType: "refresh",
			want:      true,
		},
		{
			name:      "access token",
			tokenType: "access",
			want:      false,
		},
		{
			name:      "empty type",
			tokenType: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{Type: tt.tokenType}
			got := c.IsRefresh()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaims_GetString_NilToken(t *testing.T) {
	c := &Claims{token: nil}
	val, err := c.GetString("key")
	assert.Equal(t, "", val)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestClaims_Get_NilToken(t *testing.T) {
	c := &Claims{token: nil}
	var val string
	err := c.Get("key", &val)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
