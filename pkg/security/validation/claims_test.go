package validation

import "testing"

func TestClaimsCanAccessAnyTenant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		claims Claims
		want   bool
	}{
		{name: "with cross-tenant scope", claims: Claims{Permissions: []string{"cross-tenant"}}, want: true},
		{name: "with cross-tenant scope and tenant", claims: Claims{Permissions: []string{"cross-tenant"}, Tenant: "shop"}},
		{name: "without cross-tenant scope", claims: Claims{Permissions: []string{"products:write"}}},
		{name: "empty claims", claims: Claims{}},
		{name: "service account role without scope", claims: Claims{Role: "service_account"}},
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
