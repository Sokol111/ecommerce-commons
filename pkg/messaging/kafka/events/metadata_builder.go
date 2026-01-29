package events

import (
	"context"
	"reflect"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/observability/tracing"
	"github.com/google/uuid"
)

// MetadataPopulator populates event metadata automatically.
// It extracts EventType from the struct name using reflection.
type MetadataPopulator interface {
	// PopulateMetadata fills in the metadata fields of an event.
	// The event must implement the Event interface.
	// Returns the populated EventID for use as outbox key.
	PopulateMetadata(ctx context.Context, event Event) string
}

type metadataPopulator struct {
	source string
}

// NewMetadataPopulator creates a new MetadataPopulator with the given source service name.
func NewMetadataPopulator(source string) MetadataPopulator {
	return &metadataPopulator{source: source}
}

// PopulateMetadata fills in the metadata fields automatically:
// EventID generates new UUID, EventType extracts from struct name (e.g., "AttributeUpdatedEvent"),
// Source uses configured service name, Timestamp is current UTC time, TraceID extracts from context.
func (p *metadataPopulator) PopulateMetadata(ctx context.Context, event Event) string {
	metadata := event.GetMetadata()

	eventID := uuid.New().String()
	metadata.EventID = eventID
	metadata.EventType = extractEventType(event)
	metadata.Source = p.source
	metadata.Timestamp = time.Now().UTC()

	traceID := tracing.GetTraceID(ctx)
	if traceID != "" {
		metadata.TraceID = &traceID
	}

	return eventID
}

// extractEventType gets the struct type name using reflection.
// For *events.AttributeUpdatedEvent it returns "AttributeUpdatedEvent".
func extractEventType(event Event) string {
	t := reflect.TypeOf(event)

	// Dereference pointer if needed
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name()
}
