package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

func TestNewMigrationsModule(t *testing.T) {
	t.Run("returns valid fx.Option", func(t *testing.T) {
		module := NewMigrationsModule()

		assert.NotNil(t, module)
	})

	t.Run("module can be used with fx.Options", func(t *testing.T) {
		// This test verifies that the module can be combined with other fx.Options
		combinedModule := fx.Options(
			NewMigrationsModule(),
		)

		assert.NotNil(t, combinedModule)
	})
}
