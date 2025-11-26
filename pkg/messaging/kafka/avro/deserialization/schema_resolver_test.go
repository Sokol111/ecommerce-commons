package deserialization

import (
	"errors"
	"sync"
	"testing"

	schemaregistry "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSchema1 = `{
		"type": "record",
		"name": "ProductCreated",
		"namespace": "ecommerce.product",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"}
		]
	}`

	testSchema2 = `{
		"type": "record",
		"name": "CategoryCreated",
		"namespace": "ecommerce.category",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`

	testEnumSchema = `{
		"type": "enum",
		"name": "Status",
		"namespace": "ecommerce.common",
		"symbols": ["ACTIVE", "INACTIVE"]
	}`
)

func TestWriterSchemaResolver_Resolve_Success(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	// Register test schema and get actual schema ID
	schemaID := registerTestSchema(t, client, 0, "product-events-value", testSchema1)

	// Act
	schema, schemaName, err := resolver.Resolve(schemaID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "ecommerce.product.ProductCreated", schemaName)

	// Verify it's a valid schema
	namedSchema, ok := schema.(hambavro.NamedSchema)
	assert.True(t, ok)
	assert.Equal(t, "ProductCreated", namedSchema.Name())
	assert.Equal(t, "ecommerce.product", namedSchema.Namespace())
}

func TestWriterSchemaResolver_Resolve_Caching(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	schemaID := registerTestSchema(t, client, 0, "product-events-value", testSchema1)

	// Act - First call (should fetch from registry)
	schema1, schemaName1, err1 := resolver.Resolve(schemaID)
	require.NoError(t, err1)

	// Act - Second call (should use cache)
	schema2, schemaName2, err2 := resolver.Resolve(schemaID)
	require.NoError(t, err2)

	// Assert - Both calls return the same result
	assert.Equal(t, schemaName1, schemaName2)
	assert.Equal(t, "ecommerce.product.ProductCreated", schemaName1)

	// Verify schemas are equal
	assert.Equal(t, schema1.String(), schema2.String())
}

func TestWriterSchemaResolver_Resolve_MultipleSchemasWithCaching(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	schemaID1 := registerTestSchema(t, client, 0, "product-events-value", testSchema1)
	schemaID2 := registerTestSchema(t, client, 0, "category-events-value", testSchema2)

	// Act - Resolve schema 1
	schema1, schemaName1, err1 := resolver.Resolve(schemaID1)
	require.NoError(t, err1)
	assert.Equal(t, "ecommerce.product.ProductCreated", schemaName1)

	// Act - Resolve schema 2
	schema2, schemaName2, err2 := resolver.Resolve(schemaID2)
	require.NoError(t, err2)
	assert.Equal(t, "ecommerce.category.CategoryCreated", schemaName2)

	// Act - Resolve schema 1 again (from cache)
	schema1Again, schemaName1Again, err1Again := resolver.Resolve(schemaID1)
	require.NoError(t, err1Again)
	assert.Equal(t, schemaName1, schemaName1Again)
	assert.Equal(t, schema1.String(), schema1Again.String())

	// Assert - Schemas are different
	assert.NotEqual(t, schema1.String(), schema2.String())
	assert.NotEqual(t, schemaName1, schemaName2)
}

func TestWriterSchemaResolver_Resolve_SchemaNotFound(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	// Act - Try to resolve non-existent schema
	schema, schemaName, err := resolver.Resolve(1)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Empty(t, schemaName)
	assert.Contains(t, err.Error(), "failed to get subjects for schema ID")
}

func TestWriterSchemaResolver_Resolve_NoSubjectsFound(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	// Register schema with ID that returns empty subjects list
	// This simulates a schema that was deleted or has no associated subjects
	schemaID := 100

	// Act
	schema, schemaName, err := resolver.Resolve(schemaID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Empty(t, schemaName)
	assert.Contains(t, err.Error(), "failed to get subjects for schema ID")
}

func TestWriterSchemaResolver_Resolve_EnumSchema(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	schemaID := registerTestSchema(t, client, 0, "status-value", testEnumSchema)

	// Act
	schema, schemaName, err := resolver.Resolve(schemaID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "ecommerce.common.Status", schemaName)

	// Verify it's an enum schema
	namedSchema, ok := schema.(hambavro.NamedSchema)
	assert.True(t, ok)
	assert.Equal(t, "Status", namedSchema.Name())
}

func TestWriterSchemaResolver_Resolve_ConcurrentAccess(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	schemaID := registerTestSchema(t, client, 0, "product-events-value", testSchema1)

	// Act - Multiple goroutines accessing the same schema concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			schema, schemaName, err := resolver.Resolve(schemaID)
			if err != nil {
				results <- err
				return
			}
			if schemaName != "ecommerce.product.ProductCreated" {
				results <- errors.New("unexpected schema name: " + schemaName)
				return
			}
			if schema == nil {
				results <- errors.New("schema is nil")
				return
			}
			results <- nil
		}()
	}

	wg.Wait()
	close(results)

	// Assert - All goroutines should succeed
	for err := range results {
		assert.NoError(t, err)
	}
}

func TestWriterSchemaResolver_Resolve_ConcurrentDifferentSchemas(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)
	resolver := NewWriterSchemaResolver(client)

	schemaID1 := registerTestSchema(t, client, 0, "product-events-value", testSchema1)
	schemaID2 := registerTestSchema(t, client, 0, "category-events-value", testSchema2)

	// Act - Multiple goroutines accessing different schemas
	var wg sync.WaitGroup
	numGoroutines := 20
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		schemaID := schemaID1
		expectedName := "ecommerce.product.ProductCreated"
		if i%2 == 0 {
			schemaID = schemaID2
			expectedName = "ecommerce.category.CategoryCreated"
		}

		go func(id int, expected string) {
			defer wg.Done()
			schema, schemaName, err := resolver.Resolve(id)
			if err != nil {
				results <- err
				return
			}
			if schemaName != expected {
				results <- errors.New("unexpected schema name")
				return
			}
			if schema == nil {
				results <- errors.New("schema is nil")
				return
			}
			results <- nil
		}(schemaID, expectedName)
	}

	wg.Wait()
	close(results)

	// Assert
	for err := range results {
		assert.NoError(t, err)
	}
}

func TestNewWriterSchemaResolver(t *testing.T) {
	// Arrange
	client := createMockSchemaRegistryClient(t)

	// Act
	resolver := NewWriterSchemaResolver(client)

	// Assert
	assert.NotNil(t, resolver)
	assert.Implements(t, (*WriterSchemaResolver)(nil), resolver)

	// Verify internal structure
	registryResolver, ok := resolver.(*registryWriterSchemaResolver)
	assert.True(t, ok)
	assert.NotNil(t, registryResolver.client)
	assert.NotNil(t, registryResolver.cache)
	assert.Empty(t, registryResolver.cache)
}

func TestWriterSchemaResolver_InterfaceCompliance(t *testing.T) {
	// Verify that registryWriterSchemaResolver implements WriterSchemaResolver interface
	var _ WriterSchemaResolver = (*registryWriterSchemaResolver)(nil)
}

// Helper functions

func createMockSchemaRegistryClient(t *testing.T) schemaregistry.Client {
	conf := schemaregistry.NewConfig("mock://")
	client, err := schemaregistry.NewClient(conf)
	require.NoError(t, err)
	return client
}

func registerTestSchema(t *testing.T, client schemaregistry.Client, _ int, subject string, schemaJSON string) int {
	info := schemaregistry.SchemaInfo{
		Schema:     schemaJSON,
		SchemaType: "AVRO",
	}

	id, err := client.Register(subject, info, false)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	return id
}
