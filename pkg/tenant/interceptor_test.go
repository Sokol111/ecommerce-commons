package tenant

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/security/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidatorInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		claims  *validation.Claims
		wantErr bool
	}{
		{name: "matching tenant", claims: &validation.Claims{Tenant: "shop"}},
		{name: "mismatched tenant", claims: &validation.Claims{Tenant: "other"}, wantErr: true},
		{name: "cross-tenant scope", claims: &validation.Claims{Permissions: []string{"cross-tenant"}}},
		{name: "tenantless without cross-tenant scope", claims: &validation.Claims{Role: "super_admin"}, wantErr: true},
		{name: "public request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithSlug(context.Background(), "shop")
			if tt.claims != nil {
				ctx = validation.ContextWithClaims(ctx, tt.claims)
			}
			nextCalled := false
			next := func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
				nextCalled = true
				return nil, nil
			}
			_, err := NewValidatorInterceptor()(next)(ctx, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if nextCalled == tt.wantErr {
				t.Fatalf("nextCalled = %v, want %v", nextCalled, !tt.wantErr)
			}
		})
	}
}

func TestResolverInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headerSlug string
		wantErr    bool
		wantSlug   string
	}{
		{name: "valid tenant header", headerSlug: "shop", wantSlug: "shop"},
		{name: "missing tenant header", headerSlug: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := logger.With(context.Background(), zap.NewNop())
			msg := "ignored"
			req := connect.NewRequest(&msg)
			if tt.headerSlug != "" {
				req.Header().Set(TenantSlugHeader, tt.headerSlug)
			}

			nextCalled := false
			next := func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				nextCalled = true
				slug, ok := SlugFromContext(ctx)
				assert.Equal(t, tt.wantSlug, slug)
				assert.True(t, ok)
				return nil, nil
			}

			_, err := NewResolverInterceptor()(next)(ctx, req)
			if tt.wantErr {
				require.Error(t, err)
				assert.False(t, nextCalled)
				return
			}
			require.NoError(t, err)
			assert.True(t, nextCalled)
		})
	}
}
