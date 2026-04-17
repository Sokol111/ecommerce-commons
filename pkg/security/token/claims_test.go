package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{
			name:        "wildcard grants all permissions",
			permissions: []string{"*"},
			check:       []string{"read", "write", "delete"},
			want:        true,
		},
		{
			name:        "wildcard with other permissions",
			permissions: []string{"read", "*"},
			check:       []string{"admin:delete"},
			want:        true,
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
