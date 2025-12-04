package outbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOutboxEntity(t *testing.T) {
	t.Run("creates entity with all fields", func(t *testing.T) {
		now := time.Now().UTC()
		entity := outboxEntity{
			ID:             "test-id",
			Payload:        []byte("test-payload"),
			Key:            "partition-key",
			Topic:          "test-topic",
			Headers:        map[string]string{"key": "value"},
			Status:         StatusProcessing,
			CreatedAt:      now,
			SentAt:         now.Add(time.Minute),
			LockExpiresAt:  now.Add(30 * time.Second),
			AttemptsToSend: 1,
		}

		assert.Equal(t, "test-id", entity.ID)
		assert.Equal(t, []byte("test-payload"), entity.Payload)
		assert.Equal(t, "partition-key", entity.Key)
		assert.Equal(t, "test-topic", entity.Topic)
		assert.Equal(t, map[string]string{"key": "value"}, entity.Headers)
		assert.Equal(t, StatusProcessing, entity.Status)
		assert.Equal(t, now, entity.CreatedAt)
		assert.Equal(t, now.Add(time.Minute), entity.SentAt)
		assert.Equal(t, now.Add(30*time.Second), entity.LockExpiresAt)
		assert.Equal(t, int32(1), entity.AttemptsToSend)
	})

	t.Run("entity with nil headers", func(t *testing.T) {
		entity := outboxEntity{
			ID:      "test-id",
			Headers: nil,
		}

		assert.Equal(t, "test-id", entity.ID)
		assert.Nil(t, entity.Headers)
	})
}

func TestOutboxStatus(t *testing.T) {
	t.Run("status constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, "PROCESSING", StatusProcessing)
		assert.Equal(t, "SENT", StatusSent)
	})
}
