package mapping

import (
	"reflect"
	"testing"

	hambavro "github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test event structs
type ProductCreated struct {
	ID   string `avro:"id"`
	Name string `avro:"name"`
}

type CategoryCreated struct {
	ID string `avro:"id"`
}

type OrderPlaced struct {
	OrderID string `avro:"order_id"`
}

const (
	productCreatedSchema = `{
		"type": "record",
		"name": "ProductCreated",
		"namespace": "ecommerce.product",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"}
		]
	}`

	categoryCreatedSchema = `{
		"type": "record",
		"name": "CategoryCreated",
		"namespace": "ecommerce.category",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`

	orderPlacedSchema = `{
		"type": "record",
		"name": "OrderPlaced",
		"namespace": "ecommerce.order",
		"fields": [
			{"name": "order_id", "type": "string"}
		]
	}`

	invalidSchema = `{invalid json`
)

func TestNewTypeMapping(t *testing.T) {
	// Act
	tm := NewTypeMapping()

	// Assert
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.typeToBinding)
	assert.NotNil(t, tm.nameToBinding)
	assert.Empty(t, tm.typeToBinding)
	assert.Empty(t, tm.nameToBinding)
}

func TestTypeMapping_RegisterBinding_Success(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})

	// Assert
	require.NoError(t, err)

	// Verify binding is stored
	assert.Len(t, tm.typeToBinding, 1)
	assert.Len(t, tm.nameToBinding, 1)

	// Verify binding details
	binding, ok := tm.typeToBinding[goType]
	assert.True(t, ok)
	assert.Equal(t, goType, binding.GoType)
	assert.Equal(t, "ecommerce.product.ProductCreated", binding.SchemaName)
	assert.Equal(t, "product-events", binding.Topic)
	assert.NotNil(t, binding.ParsedSchema())
	assert.Equal(t, []byte(productCreatedSchema), binding.SchemaJSON)
}

func TestTypeMapping_RegisterBinding_NilGoType(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     nil,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "goType cannot be nil")
	assert.Empty(t, tm.typeToBinding)
	assert.Empty(t, tm.nameToBinding)
}

func TestTypeMapping_RegisterBinding_EmptySchemaJSON(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte{},
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schemaJSON cannot be empty")
	assert.Empty(t, tm.typeToBinding)
	assert.Empty(t, tm.nameToBinding)
}

func TestTypeMapping_RegisterBinding_EmptySchemaName(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "",
		Topic:      "product-events",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schemaName cannot be empty")
	assert.Empty(t, tm.typeToBinding)
	assert.Empty(t, tm.nameToBinding)
}

func TestTypeMapping_RegisterBinding_EmptyTopic(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic cannot be empty")
	assert.Empty(t, tm.typeToBinding)
	assert.Empty(t, tm.nameToBinding)
}

func TestTypeMapping_RegisterBinding_InvalidSchemaJSON(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(invalidSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse Avro schema")
	assert.Empty(t, tm.typeToBinding)
	assert.Empty(t, tm.nameToBinding)
}

func TestTypeMapping_RegisterBinding_MultipleSchemas(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()

	// Act - Register multiple schemas
	err1 := tm.RegisterBinding(SchemaBinding{
		GoType:     reflect.TypeOf(ProductCreated{}),
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	err2 := tm.RegisterBinding(SchemaBinding{
		GoType:     reflect.TypeOf(CategoryCreated{}),
		SchemaJSON: []byte(categoryCreatedSchema),
		SchemaName: "ecommerce.category.CategoryCreated",
		Topic:      "category-events",
	})
	err3 := tm.RegisterBinding(SchemaBinding{
		GoType:     reflect.TypeOf(OrderPlaced{}),
		SchemaJSON: []byte(orderPlacedSchema),
		SchemaName: "ecommerce.order.OrderPlaced",
		Topic:      "order-events",
	})

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	assert.Len(t, tm.typeToBinding, 3)
	assert.Len(t, tm.nameToBinding, 3)
}

func TestTypeMapping_GetByType_Success(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	require.NoError(t, err)

	// Act
	binding, err := tm.GetByType(goType)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, goType, binding.GoType)
	assert.Equal(t, "ecommerce.product.ProductCreated", binding.SchemaName)
	assert.Equal(t, "product-events", binding.Topic)
}

func TestTypeMapping_GetByType_NotFound(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})

	// Act
	binding, err := tm.GetByType(goType)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, binding)
	assert.Contains(t, err.Error(), "no schema registered for Go type")
	assert.Contains(t, err.Error(), "ProductCreated")
}

func TestTypeMapping_GetByValue_Success(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	require.NoError(t, err)

	// Act - Test with value (not pointer)
	event := ProductCreated{ID: "123", Name: "Test"}
	binding, err := tm.GetByValue(event)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "ecommerce.product.ProductCreated", binding.SchemaName)
}

func TestTypeMapping_GetByValue_Pointer(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	require.NoError(t, err)

	// Act - Test with pointer
	event := &ProductCreated{ID: "123", Name: "Test"}
	binding, err := tm.GetByValue(event)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "ecommerce.product.ProductCreated", binding.SchemaName)
}

func TestTypeMapping_GetBySchemaName_Success(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	require.NoError(t, err)

	// Act
	binding, err := tm.GetBySchemaName("ecommerce.product.ProductCreated")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, goType, binding.GoType)
	assert.Equal(t, "product-events", binding.Topic)
}

func TestTypeMapping_GetBySchemaName_NotFound(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()

	// Act
	binding, err := tm.GetBySchemaName("ecommerce.product.NonExistent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, binding)
	assert.Contains(t, err.Error(), "no schema registered for schema name")
	assert.Contains(t, err.Error(), "ecommerce.product.NonExistent")
}

func TestTypeMapping_GetAllBindings_Empty(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()

	// Act
	bindings := tm.GetAllBindings()

	// Assert
	assert.NotNil(t, bindings)
	assert.Empty(t, bindings)
}

func TestTypeMapping_GetAllBindings_Multiple(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()

	err1 := tm.RegisterBinding(SchemaBinding{
		GoType:     reflect.TypeOf(ProductCreated{}),
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	err2 := tm.RegisterBinding(SchemaBinding{
		GoType:     reflect.TypeOf(CategoryCreated{}),
		SchemaJSON: []byte(categoryCreatedSchema),
		SchemaName: "ecommerce.category.CategoryCreated",
		Topic:      "category-events",
	})
	require.NoError(t, err1)
	require.NoError(t, err2)

	// Act
	bindings := tm.GetAllBindings()

	// Assert
	assert.Len(t, bindings, 2)

	// Verify both bindings are present
	schemaNames := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		schemaNames = append(schemaNames, binding.SchemaName)
	}
	assert.Contains(t, schemaNames, "ecommerce.product.ProductCreated")
	assert.Contains(t, schemaNames, "ecommerce.category.CategoryCreated")
}

func TestSchemaBinding_ParsedSchemaCaching(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: []byte(productCreatedSchema),
		SchemaName: "ecommerce.product.ProductCreated",
		Topic:      "product-events",
	})
	require.NoError(t, err)

	// Act
	binding1, _ := tm.GetByType(goType)
	binding2, _ := tm.GetBySchemaName("ecommerce.product.ProductCreated")

	// Assert - Both should return the same binding with cached parsed schema
	assert.Equal(t, binding1, binding2)
	assert.NotNil(t, binding1.ParsedSchema())
	assert.IsType(t, (*hambavro.RecordSchema)(nil), binding1.ParsedSchema())
}

func TestTypeMapping_SchemaBinding_AllFields(t *testing.T) {
	// Arrange
	tm := NewTypeMapping()
	goType := reflect.TypeOf(ProductCreated{})
	schemaJSON := []byte(productCreatedSchema)
	schemaName := "ecommerce.product.ProductCreated"
	topic := "product-events"

	// Act
	err := tm.RegisterBinding(SchemaBinding{
		GoType:     goType,
		SchemaJSON: schemaJSON,
		SchemaName: schemaName,
		Topic:      topic,
	})
	require.NoError(t, err)

	binding, err := tm.GetByType(goType)
	require.NoError(t, err)

	// Assert - Verify all fields are populated correctly
	assert.Equal(t, goType, binding.GoType)
	assert.Equal(t, schemaJSON, binding.SchemaJSON)
	assert.Equal(t, schemaName, binding.SchemaName)
	assert.Equal(t, topic, binding.Topic)
	assert.NotNil(t, binding.ParsedSchema())

	// Verify parsed schema details
	namedSchema, ok := binding.ParsedSchema().(hambavro.NamedSchema)
	assert.True(t, ok)
	assert.Equal(t, "ProductCreated", namedSchema.Name())
	assert.Equal(t, "ecommerce.product", namedSchema.Namespace())
}
