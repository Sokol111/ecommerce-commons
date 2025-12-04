package encoding

import "fmt"

// WireFormatParser parses Confluent wire format messages.
type WireFormatParser interface {
	// Parse extracts schema ID and payload from Confluent wire format
	// Expected format: [0x00][schema_id (4 bytes)][payload]
	Parse(data []byte) (schemaID int, payload []byte, err error)
}

// WireFormatBuilder builds Confluent wire format messages.
type WireFormatBuilder interface {
	// Build creates Confluent wire format from schema ID and payload
	// Returns format: [0x00][schema_id (4 bytes)][payload]
	Build(schemaID int, payload []byte) []byte
}

type confluentWireFormat struct{}

// NewConfluentWireFormatParser creates a parser for Confluent wire format.
func NewConfluentWireFormat() (WireFormatParser, WireFormatBuilder) {
	f := &confluentWireFormat{}
	return f, f
}

func (w *confluentWireFormat) Parse(data []byte) (int, []byte, error) {
	// Validate minimum length
	if len(data) < 5 {
		return 0, nil, fmt.Errorf("data too short: expected at least 5 bytes, got %d", len(data))
	}

	// Check magic byte
	if data[0] != 0x00 {
		return 0, nil, fmt.Errorf("invalid magic byte: expected 0x00, got 0x%02x", data[0])
	}

	// Extract schema ID (big-endian)
	schemaID := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

	// Return schema ID and payload
	return schemaID, data[5:], nil
}

func (w *confluentWireFormat) Build(schemaID int, payload []byte) []byte {
	// Build Confluent wire format: [0x00][schema_id (4 bytes)][payload]
	result := make([]byte, 5+len(payload))
	result[0] = 0x00 // Magic byte
	result[1] = byte(schemaID >> 24)
	result[2] = byte(schemaID >> 16)
	result[3] = byte(schemaID >> 8)
	result[4] = byte(schemaID)
	copy(result[5:], payload)
	return result
}
