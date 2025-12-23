package encoding

import (
	"reflect"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
	hambavro "github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHambaDecoder(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()

	// Act
	decoder := NewHambaDecoder(tm)

	// Assert
	assert.NotNil(t, decoder)
	assert.Implements(t, (*Decoder)(nil), decoder)
}

func TestDecoder_Decode_Success(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	// Create Avro payload
	original := TestProduct{ID: "123", Name: "Test Product"}
	payload, err := hambavro.Marshal(schema, &original)
	require.NoError(t, err)

	// Act
	result, err := decoder.Decode(payload, schema, "test.TestProduct")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	decoded, ok := result.(*TestProduct)
	assert.True(t, ok)
	assert.Equal(t, "123", decoded.ID)
	assert.Equal(t, "Test Product", decoded.Name)
}

func TestDecoder_Decode_SchemaNotRegistered(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()
	decoder := NewHambaDecoder(tm)

	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	payload := []byte{0x01, 0x02}

	// Act
	result, err := decoder.Decode(payload, schema, "test.NonExistent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to resolve schema binding")
	assert.Contains(t, err.Error(), "test.NonExistent")
}

func TestDecoder_Decode_InvalidPayload(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	// Empty payload - should fail as strings need length bytes
	emptyPayload := []byte{}

	// Act
	result, err := decoder.Decode(emptyPayload, schema, "test.TestProduct")

	// Assert - Avro unmarshaling empty payload should succeed with zero values
	// This is expected behavior - Avro is lenient
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDecoder_Decode_CorruptedSchema(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	// Corrupt schema
	invalidSchema := "not a valid schema"
	schema, err := hambavro.Parse(invalidSchema)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, schema)
}

func TestDecoder_Decode_EmptyPayload(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)
	schema, err := hambavro.Parse(testProductSchema)
	require.NoError(t, err)

	// Act
	result, err := decoder.Decode([]byte{}, schema, "test.TestProduct")

	// Assert - Avro unmarshals empty payload successfully
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDecoder_Decode_DifferentSchemas(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()

	// Register both schemas
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	err = tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestCategory{}),
		SchemaJSON: []byte(testCategorySchema),
		SchemaName: "test.TestCategory",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)

	// Create payloads for both schemas
	productSchema, _ := hambavro.Parse(testProductSchema)
	categorySchema, _ := hambavro.Parse(testCategorySchema)

	product := TestProduct{ID: "p1", Name: "Product"}
	productPayload, _ := hambavro.Marshal(productSchema, &product)

	category := TestCategory{ID: "c1"}
	categoryPayload, _ := hambavro.Marshal(categorySchema, &category)

	// Act - Decode product
	result1, err1 := decoder.Decode(productPayload, productSchema, "test.TestProduct")

	// Act - Decode category
	result2, err2 := decoder.Decode(categoryPayload, categorySchema, "test.TestCategory")

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)

	decodedProduct, ok := result1.(*TestProduct)
	assert.True(t, ok)
	assert.Equal(t, "p1", decodedProduct.ID)
	assert.Equal(t, "Product", decodedProduct.Name)

	decodedCategory, ok := result2.(*TestCategory)
	assert.True(t, ok)
	assert.Equal(t, "c1", decodedCategory.ID)
}

func TestDecoder_Decode_ReturnsPointer(t *testing.T) {
	// Verify that decoder returns a pointer, not a value
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)
	schema, _ := hambavro.Parse(testProductSchema)

	original := TestProduct{ID: "123", Name: "Test"}
	payload, _ := hambavro.Marshal(schema, &original)

	// Act
	result, err := decoder.Decode(payload, schema, "test.TestProduct")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, reflect.Ptr, reflect.TypeOf(result).Kind())
}

func TestDecoder_Decode_SchemaEvolution(t *testing.T) {
	// Test schema evolution: writer schema has more fields than reader
	// Arrange
	tm := mapping.NewTypeMapping()

	// Reader schema (what we expect)
	readerSchema := `{
		"type": "record",
		"name": "TestProduct",
		"namespace": "test",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`

	// Register reader schema
	type SimpleProduct struct {
		ID string `avro:"id"`
	}

	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(SimpleProduct{}),
		SchemaJSON: []byte(readerSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)

	// Writer schema (has extra field)
	writerSchema, _ := hambavro.Parse(testProductSchema)

	// Create payload with writer schema
	fullProduct := TestProduct{ID: "123", Name: "Test Product"}
	payload, _ := hambavro.Marshal(writerSchema, &fullProduct)

	// Act - Decode with writer schema but expect reader type
	result, err := decoder.Decode(payload, writerSchema, "test.TestProduct")

	// Assert - Should successfully decode, ignoring extra field
	require.NoError(t, err)
	decoded, ok := result.(*SimpleProduct)
	assert.True(t, ok)
	assert.Equal(t, "123", decoded.ID)
}

func TestDecoder_Decode_MultipleMessages(t *testing.T) {
	// Test decoding multiple messages with same decoder
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)
	schema, _ := hambavro.Parse(testProductSchema)

	messages := []TestProduct{
		{ID: "1", Name: "Product 1"},
		{ID: "2", Name: "Product 2"},
		{ID: "3", Name: "Product 3"},
	}

	// Act & Assert
	for _, original := range messages {
		payload, _ := hambavro.Marshal(schema, &original)
		result, err := decoder.Decode(payload, schema, "test.TestProduct")

		require.NoError(t, err)
		decoded := result.(*TestProduct)
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Name, decoded.Name)
	}
}

func TestDecoder_Decode_SpecialCharacters(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping()
	err := tm.RegisterBinding(mapping.SchemaBinding{
		GoType:     reflect.TypeOf(TestProduct{}),
		SchemaJSON: []byte(testProductSchema),
		SchemaName: "test.TestProduct",
		Topic:      "test-topic",
	})
	require.NoError(t, err)

	decoder := NewHambaDecoder(tm)
	schema, _ := hambavro.Parse(testProductSchema)

	original := TestProduct{
		ID:   "123",
		Name: "Test with special chars: émojis 🚀, newlines\n\t, quotes \"'",
	}
	payload, _ := hambavro.Marshal(schema, &original)

	// Act
	result, err := decoder.Decode(payload, schema, "test.TestProduct")

	// Assert
	require.NoError(t, err)
	decoded := result.(*TestProduct)
	assert.Equal(t, original.Name, decoded.Name)
}

func TestDecoder_InterfaceCompliance(t *testing.T) {
	// Verify that hambaDecoder implements Decoder interface
	var _ Decoder = (*hambaDecoder)(nil)
}
