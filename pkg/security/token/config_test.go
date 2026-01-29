package token

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("valid config with all fields", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(`
security:
  token:
    public-key: "abc123def456"
    service-token: "v4.public.token"
`))
		require.NoError(t, err)

		cfg, err := newConfig(v)

		assert.NoError(t, err)
		assert.Equal(t, "abc123def456", cfg.PublicKey)
		assert.Equal(t, "v4.public.token", cfg.ServiceToken)
	})

	t.Run("valid config with only required fields", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(`
security:
  token:
    public-key: "abc123def456"
`))
		require.NoError(t, err)

		cfg, err := newConfig(v)

		assert.NoError(t, err)
		assert.Equal(t, "abc123def456", cfg.PublicKey)
		assert.Empty(t, cfg.ServiceToken)
	})

	t.Run("missing security section", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(`
other:
  key: value
`))
		require.NoError(t, err)

		cfg, err := newConfig(v)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security configuration section is required")
		assert.Empty(t, cfg.PublicKey)
	})

	t.Run("missing token section", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(`
security:
  other:
    key: value
`))
		require.NoError(t, err)

		cfg, err := newConfig(v)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.token configuration section is required")
		assert.Empty(t, cfg.PublicKey)
	})

	t.Run("missing public-key", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(`
security:
  token:
    service-token: "v4.public.token"
`))
		require.NoError(t, err)

		cfg, err := newConfig(v)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.token.public-key is required")
		assert.Empty(t, cfg.PublicKey)
	})

	t.Run("empty public-key", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(`
security:
  token:
    public-key: ""
`))
		require.NoError(t, err)

		_, err = newConfig(v)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.token.public-key is required")
	})

	t.Run("empty viper", func(t *testing.T) {
		v := viper.New()

		_, err := newConfig(v)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security configuration section is required")
	})
}
