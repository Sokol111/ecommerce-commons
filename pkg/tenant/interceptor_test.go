package tenant

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/security/validation"
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
		{name: "service account", claims: &validation.Claims{Role: validation.CrossTenantServiceRole}},
		{name: "tenantless user", claims: &validation.Claims{Role: "super_admin"}, wantErr: true},
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
