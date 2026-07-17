package validation

import "testing"

func TestClaimsCanAccessAnyTenant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		claims Claims
		want   bool
	}{
		{name: "service account", claims: Claims{Role: CrossTenantServiceRole}, want: true},
		{name: "tenant service account", claims: Claims{Role: CrossTenantServiceRole, Tenant: "shop"}},
		{name: "tenantless user", claims: Claims{Role: "super_admin"}},
		{name: "missing role", claims: Claims{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.claims.CanAccessAnyTenant(); got != tt.want {
				t.Fatalf("CanAccessAnyTenant() = %v, want %v", got, tt.want)
			}
		})
	}
}
