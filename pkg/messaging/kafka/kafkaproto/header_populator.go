package kafkaproto

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// HeaderPopulator populates Kafka message headers with event metadata.
type HeaderPopulator interface {
	// PopulateHeaders writes event_id, event_type, source, and timestamp
	// into the provided headers map. Returns the generated event_id.
	// Trace context is propagated separately via W3C traceparent header.
	PopulateHeaders(event proto.Message, headers map[string]string) string
}

type headerPopulator struct {
	source string
}

// NewHeaderPopulator creates a new HeaderPopulator with the given source service name.
func NewHeaderPopulator(source string) HeaderPopulator {
	return &headerPopulator{source: source}
}

func (p *headerPopulator) PopulateHeaders(event proto.Message, headers map[string]string) string {
	eventID := uuid.New().String()
	headers["event_id"] = eventID
	headers["event_type"] = string(event.ProtoReflect().Descriptor().FullName())
	headers["source"] = p.source
	headers["timestamp"] = strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)

	return eventID
}
