package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent is a mock event type for testing
type testEvent struct {
	Metadata EventMetadata
	Payload  string
}

func (e *testEvent) GetMetadata() *EventMetadata {
	return &e.Metadata
}

func TestMetadataPopulator_PopulateMetadata(t *testing.T) {
	t.Run("populates all metadata fields", func(t *testing.T) {
		populator := NewMetadataPopulator("test-service")
		event := &testEvent{Payload: "test"}

		beforeTime := time.Now().UTC()
		eventID := populator.PopulateMetadata(context.Background(), event)
		afterTime := time.Now().UTC()

		assert.NotEmpty(t, eventID)
		assert.Equal(t, eventID, event.Metadata.EventID)
		assert.Equal(t, "testEvent", event.Metadata.EventType)
		assert.Equal(t, "test-service", event.Metadata.Source)
		assert.True(t, event.Metadata.Timestamp.After(beforeTime) || event.Metadata.Timestamp.Equal(beforeTime))
		assert.True(t, event.Metadata.Timestamp.Before(afterTime) || event.Metadata.Timestamp.Equal(afterTime))
	})

	t.Run("extracts event type from struct name", func(t *testing.T) {
		populator := NewMetadataPopulator("test-service")
		event := &testEvent{}

		populator.PopulateMetadata(context.Background(), event)

		assert.Equal(t, "testEvent", event.Metadata.EventType)
	})

	t.Run("returns generated event ID", func(t *testing.T) {
		populator := NewMetadataPopulator("test-service")
		event := &testEvent{}

		eventID := populator.PopulateMetadata(context.Background(), event)

		require.NotEmpty(t, eventID)
		assert.Equal(t, eventID, event.Metadata.EventID)
	})
}

func TestExtractEventType(t *testing.T) {
	t.Run("extracts type name from pointer", func(t *testing.T) {
		event := &testEvent{}
		typeName := extractEventType(event)
		assert.Equal(t, "testEvent", typeName)
	})
}
