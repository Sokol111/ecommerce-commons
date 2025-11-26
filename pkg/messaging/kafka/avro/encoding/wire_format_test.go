package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfluentWireFormat(t *testing.T) {
	// Act
	parser, builder := NewConfluentWireFormat()

	// Assert
	assert.NotNil(t, parser)
	assert.NotNil(t, builder)
	assert.Implements(t, (*WireFormatParser)(nil), parser)
	assert.Implements(t, (*WireFormatBuilder)(nil), builder)
}

func TestWireFormatParser_Parse_Success(t *testing.T) {
	// Arrange
	parser, _ := NewConfluentWireFormat()
	schemaID := 123
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	// Build valid wire format: [0x00][schema_id][payload]
	data := []byte{
		0x00,                             // Magic byte
		0x00, 0x00, 0x00, byte(schemaID), // Schema ID (big-endian)
		0x01, 0x02, 0x03, 0x04, // Payload
	}

	// Act
	parsedSchemaID, parsedPayload, err := parser.Parse(data)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, schemaID, parsedSchemaID)
	assert.Equal(t, payload, parsedPayload)
}

func TestWireFormatParser_Parse_LargeSchemaID(t *testing.T) {
	// Arrange
	parser, _ := NewConfluentWireFormat()
	schemaID := 16777215 // 0x00FFFFFF (max 3-byte value)

	data := []byte{
		0x00,                   // Magic byte
		0x00, 0xFF, 0xFF, 0xFF, // Schema ID
		0xAA, 0xBB, // Payload
	}

	// Act
	parsedSchemaID, parsedPayload, err := parser.Parse(data)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, schemaID, parsedSchemaID)
	assert.Equal(t, []byte{0xAA, 0xBB}, parsedPayload)
}

func TestWireFormatParser_Parse_EmptyPayload(t *testing.T) {
	// Arrange
	parser, _ := NewConfluentWireFormat()
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x01} // No payload after schema ID

	// Act
	schemaID, payload, err := parser.Parse(data)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, schemaID)
	assert.Empty(t, payload)
}

func TestWireFormatParser_Parse_DataTooShort(t *testing.T) {
	// Arrange
	parser, _ := NewConfluentWireFormat()

	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty data", []byte{}},
		{"1 byte", []byte{0x00}},
		{"2 bytes", []byte{0x00, 0x01}},
		{"3 bytes", []byte{0x00, 0x01, 0x02}},
		{"4 bytes", []byte{0x00, 0x01, 0x02, 0x03}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			schemaID, payload, err := parser.Parse(tc.data)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "data too short")
			assert.Equal(t, 0, schemaID)
			assert.Nil(t, payload)
		})
	}
}

func TestWireFormatParser_Parse_InvalidMagicByte(t *testing.T) {
	// Arrange
	parser, _ := NewConfluentWireFormat()

	testCases := []struct {
		name      string
		magicByte byte
	}{
		{"0x01", 0x01},
		{"0xFF", 0xFF},
		{"0x10", 0x10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{tc.magicByte, 0x00, 0x00, 0x00, 0x01, 0xAA}

			// Act
			schemaID, payload, err := parser.Parse(data)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid magic byte")
			assert.Contains(t, err.Error(), "expected 0x00")
			assert.Equal(t, 0, schemaID)
			assert.Nil(t, payload)
		})
	}
}

func TestWireFormatBuilder_Build_Success(t *testing.T) {
	// Arrange
	_, builder := NewConfluentWireFormat()
	schemaID := 123
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	// Act
	result := builder.Build(schemaID, payload)

	// Assert
	expected := []byte{
		0x00,                             // Magic byte
		0x00, 0x00, 0x00, byte(schemaID), // Schema ID
		0x01, 0x02, 0x03, 0x04, // Payload
	}
	assert.Equal(t, expected, result)
}

func TestWireFormatBuilder_Build_LargeSchemaID(t *testing.T) {
	// Arrange
	_, builder := NewConfluentWireFormat()
	schemaID := 16777215 // 0x00FFFFFF
	payload := []byte{0xAA, 0xBB}

	// Act
	result := builder.Build(schemaID, payload)

	// Assert
	expected := []byte{
		0x00,                   // Magic byte
		0x00, 0xFF, 0xFF, 0xFF, // Schema ID
		0xAA, 0xBB, // Payload
	}
	assert.Equal(t, expected, result)
}

func TestWireFormatBuilder_Build_EmptyPayload(t *testing.T) {
	// Arrange
	_, builder := NewConfluentWireFormat()
	schemaID := 1

	// Act
	result := builder.Build(schemaID, []byte{})

	// Assert
	expected := []byte{0x00, 0x00, 0x00, 0x00, 0x01}
	assert.Equal(t, expected, result)
}

func TestWireFormatBuilder_Build_SchemaIDZero(t *testing.T) {
	// Arrange
	_, builder := NewConfluentWireFormat()
	schemaID := 0
	payload := []byte{0xFF}

	// Act
	result := builder.Build(schemaID, payload)

	// Assert
	expected := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}
	assert.Equal(t, expected, result)
}

func TestWireFormat_RoundTrip(t *testing.T) {
	// Arrange
	parser, builder := NewConfluentWireFormat()

	testCases := []struct {
		name     string
		schemaID int
		payload  []byte
	}{
		{"Small schema ID", 1, []byte{0x01, 0x02}},
		{"Medium schema ID", 1000, []byte{0xAA, 0xBB, 0xCC}},
		{"Large schema ID", 16777215, []byte{0x01, 0x02, 0x03, 0x04, 0x05}},
		{"Empty payload", 42, []byte{}},
		{"Single byte payload", 99, []byte{0xFF}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act - Build and then parse
			wireData := builder.Build(tc.schemaID, tc.payload)
			parsedSchemaID, parsedPayload, err := parser.Parse(wireData)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tc.schemaID, parsedSchemaID)
			assert.Equal(t, tc.payload, parsedPayload)
		})
	}
}

func TestWireFormat_SchemaIDEncoding(t *testing.T) {
	// Test that schema ID is encoded in big-endian format
	_, builder := NewConfluentWireFormat()

	testCases := []struct {
		schemaID int
		expected []byte // Expected bytes for schema ID (4 bytes)
	}{
		{1, []byte{0x00, 0x00, 0x00, 0x01}},
		{256, []byte{0x00, 0x00, 0x01, 0x00}},
		{65536, []byte{0x00, 0x01, 0x00, 0x00}},
		{16777216, []byte{0x01, 0x00, 0x00, 0x00}},
		{0x12345678, []byte{0x12, 0x34, 0x56, 0x78}},
	}

	for _, tc := range testCases {
		t.Run("Schema ID "+string(rune(tc.schemaID)), func(t *testing.T) {
			// Act
			result := builder.Build(tc.schemaID, []byte{})

			// Assert - Check schema ID bytes (skip magic byte, no payload)
			assert.Equal(t, byte(0x00), result[0]) // Magic byte
			assert.Equal(t, tc.expected, result[1:5])
		})
	}
}

func TestWireFormat_InterfaceCompliance(t *testing.T) {
	// Verify that confluentWireFormat implements both interfaces
	var _ WireFormatParser = (*confluentWireFormat)(nil)
	var _ WireFormatBuilder = (*confluentWireFormat)(nil)
}
