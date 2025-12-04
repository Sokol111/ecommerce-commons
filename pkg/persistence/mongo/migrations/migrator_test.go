package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewMigrator(t *testing.T) {
	log := zap.NewNop()

	t.Run("creates migrator with valid parameters", func(t *testing.T) {
		// Note: database is nil but newMigrator doesn't validate it
		m, err := newMigrator(nil, log, 5)

		assert.NoError(t, err)
		assert.NotNil(t, m)
	})

	t.Run("creates migrator with zero locking timeout", func(t *testing.T) {
		m, err := newMigrator(nil, log, 0)

		assert.NoError(t, err)
		assert.NotNil(t, m)
	})

	t.Run("creates migrator with large locking timeout", func(t *testing.T) {
		m, err := newMigrator(nil, log, 60)

		assert.NoError(t, err)
		assert.NotNil(t, m)
	})
}

func TestMigrator_CreateMigrateInstance_Validation(t *testing.T) {
	log := zap.NewNop()
	m, err := newMigrator(nil, log, 5)
	assert.NoError(t, err)

	migrator := m.(*migrator)

	t.Run("returns error when collection name is empty", func(t *testing.T) {
		mi, err := migrator.createMigrateInstance("", "./migrations")

		assert.Nil(t, mi)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection name is required")
	})

	t.Run("returns error when migrations path is empty", func(t *testing.T) {
		mi, err := migrator.createMigrateInstance("schema_migrations", "")

		assert.Nil(t, mi)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "migrations path is required")
	})

	t.Run("returns error when both are empty", func(t *testing.T) {
		mi, err := migrator.createMigrateInstance("", "")

		assert.Nil(t, mi)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection name is required")
	})
}

func TestMigrator_CreateMigrateInstanceFromFS_Validation(t *testing.T) {
	log := zap.NewNop()
	m, err := newMigrator(nil, log, 5)
	assert.NoError(t, err)

	migrator := m.(*migrator)

	t.Run("returns error when collection name is empty", func(t *testing.T) {
		mi, err := migrator.createMigrateInstanceFromFS("", nil, "migrations")

		assert.Nil(t, mi)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection name is required")
	})

	t.Run("returns error when filesystem is nil", func(t *testing.T) {
		mi, err := migrator.createMigrateInstanceFromFS("schema_migrations", nil, "migrations")

		assert.Nil(t, mi)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "filesystem is required")
	})
}

func TestMigrator_InterfaceCompliance(t *testing.T) {
	log := zap.NewNop()
	m, err := newMigrator(nil, log, 5)
	assert.NoError(t, err)

	// Ensure the migrator implements the Migrator interface
	var _ Migrator = m
}
