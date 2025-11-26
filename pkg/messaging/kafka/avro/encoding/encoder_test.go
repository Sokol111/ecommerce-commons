package encoding

import (
	"testing"

	hambavro "github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestProduct struct {
	ID   string `avro:"id"`
	Name string `avro:"name"`
}

type TestCategory struct {
	ID string `avro:"id"`
}

const (
	testProductSchema = `{
		"type": "record",
		"name": "TestProduct",
		"namespace": "test",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"}
		]
	}`

	testCategorySchema = `{
		"type": "record",
		"name": "TestCategory",
		"namespace": "test",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`
)

func TestNewHambaEncoder(t *testing.T) {
	// Act
	encoder := NewHambaEncoder()

	// Assert
	assert.NotNil(t, encoder)
	assert.Implements(t, (*Encoder)(nil), encoder)
}

func TestEncoder_Encode_Success(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	msg := TestProduct{
		ID:   "123",
		Name: "Test Product",
	}

	// Act
	result, err := encoder.Encode(&msg, schema)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result)
}

func TestEncoder_Encode_EmptyStruct(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	msg := TestProduct{} // Empty values

	// Act
	result, err := encoder.Encode(&msg, schema)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEncoder_Encode_DifferentSchemas(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()

	productSchema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	categorySchema, err := hambavro.Parse(testCategorySchema)
	require.NoError(t, err)

	// Act - Encode product
	product := TestProduct{ID: "p1", Name: "Product"}
	productBytes, err1 := encoder.Encode(&product, productSchema)

	// Act - Encode category
	category := TestCategory{ID: "c1"}
	categoryBytes, err2 := encoder.Encode(&category, categorySchema)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEmpty(t, productBytes)
	assert.NotEmpty(t, categoryBytes)
	assert.NotEqual(t, productBytes, categoryBytes) // Different schemas produce different bytes
}

func TestEncoder_Encode_RoundTrip(t *testing.T) {
	// Test that encoded data can be decoded back
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	original := TestProduct{
		ID:   "123",
		Name: "Test Product",
	}

	// Act - Encode
	avroBytes, err := encoder.Encode(&original, schema)
	require.NoError(t, err)

	// Act - Decode
	var decoded TestProduct
	err = hambavro.Unmarshal(schema, avroBytes, &decoded)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
}

func TestEncoder_Encode_NilMessage(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	// Act
	result, err := encoder.Encode(nil, schema)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to marshal avro data")
}

func TestEncoder_Encode_TypeMismatch(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	// Wrong type - using Category with Product schema
	msg := TestCategory{ID: "123"}

	// Act
	result, err := encoder.Encode(&msg, schema)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to marshal avro data")
}

func TestEncoder_Encode_MultipleMessages(t *testing.T) {
	// Test encoding multiple messages with same encoder
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	messages := []TestProduct{
		{ID: "1", Name: "Product 1"},
		{ID: "2", Name: "Product 2"},
		{ID: "3", Name: "Product 3"},
	}

	// Act & Assert
	for _, msg := range messages {
		result, err := encoder.Encode(&msg, schema)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	}
}

func TestEncoder_Encode_SpecialCharacters(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	msg := TestProduct{
		ID:   "123",
		Name: "Test with special chars: émojis 🚀, newlines\n\t, quotes \"'",
	}

	// Act
	result, err := encoder.Encode(&msg, schema)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify round-trip
	var decoded TestProduct
	err = hambavro.Unmarshal(schema, result, &decoded)
	require.NoError(t, err)
	assert.Equal(t, msg.Name, decoded.Name)
}

func TestEncoder_Encode_LargeString(t *testing.T) {
	// Arrange
	encoder := NewHambaEncoder()
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	// Create a large string
	largeString := string(make([]byte, 10000))
	msg := TestProduct{
		ID:   "123",
		Name: largeString,
	}

	// Act
	result, err := encoder.Encode(&msg, schema)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestEncoder_InterfaceCompliance(t *testing.T) {
	// Verify that hambaEncoder implements Encoder interface
	var _ Encoder = (*hambaEncoder)(nil)
}
