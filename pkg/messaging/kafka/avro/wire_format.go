package avro

import "fmt"

// WireFormatParser parses Confluent wire format messages
type WireFormatParser interface {
	// Parse extracts schema ID and payload from Confluent wire format
	// Expected format: [0x00][schema_id (4 bytes)][payload]
	Parse(data []byte) (schemaID int, payload []byte, err error)
}

type confluentWireFormatParser struct{}

// NewConfluentWireFormatParser creates a parser for Confluent wire format
func NewConfluentWireFormatParser() WireFormatParser {
	return &confluentWireFormatParser{}
}

func (p *confluentWireFormatParser) Parse(data []byte) (int, []byte, error) {
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
