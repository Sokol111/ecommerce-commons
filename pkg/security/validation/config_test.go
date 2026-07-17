package validation

import "testing"

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := Config{
		JwksURL:  "https://auth.example/oidc/jwks",
		Issuer:   "https://auth.example/oidc",
		Audience: "https://api.example",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config returned error: %v", err)
	}

	tests := []struct {
		name   string
		config Config
	}{
		{name: "missing JWKS URL", config: Config{Issuer: valid.Issuer, Audience: valid.Audience}},
		{name: "missing issuer", config: Config{JwksURL: valid.JwksURL, Audience: valid.Audience}},
		{name: "missing audience", config: Config{JwksURL: valid.JwksURL, Issuer: valid.Issuer}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.config.Validate(); err == nil {
				t.Fatal("Validate() returned nil")
			}
		})
	}
}
