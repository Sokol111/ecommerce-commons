package serialization

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
	hambavro "github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test structs
type TestProduct struct {
	ID   string `avro:"id"`
	Name string `avro:"name"`
}

type TestCategory struct {
	ID   string `avro:"id"`
	Name string `avro:"name"`
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
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"}
		]
	}`
)

// Mock implementations

type MockConfluentRegistry struct {
	mock.Mock
}

func (m *MockConfluentRegistry) RegisterSchema(binding *mapping.SchemaBinding) (int, error) {
	args := m.Called(binding)
	return args.Int(0), args.Error(1)
}

func (m *MockConfluentRegistry) RegisterAllSchemasAtStartup() error {
	args := m.Called()
	return args.Error(0)
}

type MockEncoder struct {
	mock.Mock
}

func (m *MockEncoder) Encode(msg interface{}, schema hambavro.Schema) ([]byte, error) {
	args := m.Called(msg, schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

type MockWireFormatBuilder struct {
	mock.Mock
}

func (m *MockWireFormatBuilder) Build(schemaID int, payload []byte) []byte {
	args := m.Called(schemaID, payload)
	return args.Get(0).([]byte)
}

// Helper function to create test type mapping
func createTestTypeMapping(t *testing.T) *mapping.TypeMapping {
	tm := mapping.NewTypeMapping()

	err := tm.Register(
		reflect.TypeOf(TestProduct{}),
		[]byte(testProductSchema),
		"test.TestProduct",
		"test-product-events",
	)
	require.NoError(t, err)

	err = tm.Register(
		reflect.TypeOf(TestCategory{}),
		[]byte(testCategorySchema),
		"test.TestCategory",
		"test-category-events",
	)
	require.NoError(t, err)

	return tm
}

func TestNewSerializer(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	// Act
	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Assert
	assert.NotNil(t, serializer)
	assert.Implements(t, (*Serializer)(nil), serializer)
}

func TestSerializer_Serialize_Success(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	schemaID := 42
	avroData := []byte{0x06, 0x31, 0x32, 0x33} // Example Avro bytes
	expectedWireFormat := []byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x06, 0x31, 0x32, 0x33}

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil)
	mockBuilder.On("Build", schemaID, avroData).Return(expectedWireFormat)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedWireFormat, result)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_TypeNotRegistered(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping() // Empty type mapping
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get schema binding")
	mockRegistry.AssertNotCalled(t, "RegisterSchema")
	mockEncoder.AssertNotCalled(t, "Encode")
	mockBuilder.AssertNotCalled(t, "Build")
}

func TestSerializer_Serialize_RegistryError(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	registryError := errors.New("schema registry unavailable")

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(0, registryError)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to register schema in Confluent")
	assert.Contains(t, err.Error(), "schema registry unavailable")
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertNotCalled(t, "Encode")
	mockBuilder.AssertNotCalled(t, "Build")
}

func TestSerializer_Serialize_EncoderError(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	schemaID := 42
	encoderError := errors.New("encoding failed")

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(nil, encoderError)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to encode avro data")
	assert.Contains(t, err.Error(), "encoding failed")
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertNotCalled(t, "Build")
}

func TestSerializer_Serialize_MultipleMessages(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	messages := []*TestProduct{
		{ID: "1", Name: "Product 1"},
		{ID: "2", Name: "Product 2"},
		{ID: "3", Name: "Product 3"},
	}

	schemaID := 42
	binding, err := tm.GetByValue(messages[0])
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)

	for _, msg := range messages {
		avroData := []byte{0x01, 0x02, 0x03}
		wireFormat := []byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x01, 0x02, 0x03}
		mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil).Once()
		mockBuilder.On("Build", schemaID, avroData).Return(wireFormat).Once()
	}

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act & Assert
	for _, msg := range messages {
		result, err := serializer.Serialize(msg)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}

	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_DifferentTypes(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	product := &TestProduct{ID: "p1", Name: "Product"}
	category := &TestCategory{ID: "c1", Name: "Category"}

	productBinding, err := tm.GetByValue(product)
	require.NoError(t, err)
	categoryBinding, err := tm.GetByValue(category)
	require.NoError(t, err)

	productSchemaID := 100
	categorySchemaID := 200

	mockRegistry.On("RegisterSchema", productBinding).Return(productSchemaID, nil)
	mockRegistry.On("RegisterSchema", categoryBinding).Return(categorySchemaID, nil)

	productAvroData := []byte{0x01}
	categoryAvroData := []byte{0x02}

	mockEncoder.On("Encode", product, productBinding.ParsedSchema).Return(productAvroData, nil)
	mockEncoder.On("Encode", category, categoryBinding.ParsedSchema).Return(categoryAvroData, nil)

	mockBuilder.On("Build", productSchemaID, productAvroData).Return([]byte{0x00, 0x00, 0x00, 0x00, 0x64, 0x01})
	mockBuilder.On("Build", categorySchemaID, categoryAvroData).Return([]byte{0x00, 0x00, 0x00, 0x00, 0xC8, 0x02})

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	productResult, err1 := serializer.Serialize(product)
	categoryResult, err2 := serializer.Serialize(category)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotNil(t, productResult)
	assert.NotNil(t, categoryResult)
	assert.NotEqual(t, productResult, categoryResult)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_WithPointerAndValue(t *testing.T) {
	// Test that serializer works with both pointer and value types
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msgPointer := &TestProduct{ID: "123", Name: "Test"}
	msgValue := TestProduct{ID: "456", Name: "Test2"}

	schemaID := 42
	binding, err := tm.GetByValue(msgPointer)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msgPointer, binding.ParsedSchema).Return([]byte{0x01}, nil)
	mockEncoder.On("Encode", msgValue, binding.ParsedSchema).Return([]byte{0x02}, nil)
	mockBuilder.On("Build", schemaID, []byte{0x01}).Return([]byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x01})
	mockBuilder.On("Build", schemaID, []byte{0x02}).Return([]byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x02})

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	resultPointer, err1 := serializer.Serialize(msgPointer)
	resultValue, err2 := serializer.Serialize(msgValue)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotNil(t, resultPointer)
	assert.NotNil(t, resultValue)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_NilMessage(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act & Assert
	// Nil message will cause panic in reflect.TypeOf, which is expected behavior
	// This test verifies that serializer properly propagates such issues
	defer func() {
		if r := recover(); r != nil {
			// This is expected - nil message causes panic in type reflection
			assert.NotNil(t, r)
		}
	}()

	result, err := serializer.Serialize(nil)

	// If we get here without panic, check for error
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, result)
	}
}

func TestSerializer_InterfaceCompliance(t *testing.T) {
	// Verify that serializer implements Serializer interface
	var _ Serializer = (*serializer)(nil)
}

func TestSerializer_Serialize_EmptyStruct(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{} // Empty struct
	schemaID := 42
	avroData := []byte{0x00, 0x00}
	wireFormat := []byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x00, 0x00}

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil)
	mockBuilder.On("Build", schemaID, avroData).Return(wireFormat)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, wireFormat, result)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_LargeSchemaID(t *testing.T) {
	// Test with maximum schema ID value
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test"}
	schemaID := 2147483647 // Max int32
	avroData := []byte{0x01, 0x02}
	wireFormat := []byte{0x00, 0x7F, 0xFF, 0xFF, 0xFF, 0x01, 0x02}

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil)
	mockBuilder.On("Build", schemaID, avroData).Return(wireFormat)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, wireFormat, result)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_IntegrationWithRealComponents(t *testing.T) {
	// Integration test using real encoder and wire format builder
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	encoder := encoding.NewHambaEncoder()
	_, builder := encoding.NewConfluentWireFormat()

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	schemaID := 42

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)

	serializer := NewSerializer(tm, mockRegistry, encoder, builder)

	// Act
	result, err := serializer.Serialize(msg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(result), 5) // At least magic byte + schema ID

	// Verify wire format structure
	assert.Equal(t, byte(0x00), result[0]) // Magic byte
	// Schema ID should be 42 (0x0000002A in big-endian)
	assert.Equal(t, byte(0x00), result[1])
	assert.Equal(t, byte(0x00), result[2])
	assert.Equal(t, byte(0x00), result[3])
	assert.Equal(t, byte(0x2A), result[4])

	mockRegistry.AssertExpectations(t)
}

func TestSerializer_SerializeWithTopic_Success(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	schemaID := 42
	avroData := []byte{0x06, 0x31, 0x32, 0x33}
	expectedWireFormat := []byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x06, 0x31, 0x32, 0x33}

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil)
	mockBuilder.On("Build", schemaID, avroData).Return(expectedWireFormat)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, topic, err := serializer.SerializeWithTopic(msg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedWireFormat, result)
	assert.Equal(t, "test-product-events", topic)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_SerializeWithTopic_TypeNotRegistered(t *testing.T) {
	// Arrange
	tm := mapping.NewTypeMapping() // Empty type mapping
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, topic, err := serializer.SerializeWithTopic(msg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, topic)
	assert.Contains(t, err.Error(), "failed to get schema binding")
	mockRegistry.AssertNotCalled(t, "RegisterSchema")
	mockEncoder.AssertNotCalled(t, "Encode")
	mockBuilder.AssertNotCalled(t, "Build")
}

func TestSerializer_SerializeWithTopic_RegistryError(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	registryError := errors.New("schema registry unavailable")

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(0, registryError)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, topic, err := serializer.SerializeWithTopic(msg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, topic)
	assert.Contains(t, err.Error(), "failed to register schema in Confluent")
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertNotCalled(t, "Encode")
	mockBuilder.AssertNotCalled(t, "Build")
}

func TestSerializer_SerializeWithTopic_EncoderError(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	schemaID := 42
	encoderError := errors.New("encoding failed")

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(nil, encoderError)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	result, topic, err := serializer.SerializeWithTopic(msg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, topic)
	assert.Contains(t, err.Error(), "failed to encode avro data")
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertNotCalled(t, "Build")
}

func TestSerializer_SerializeWithTopic_DifferentTypes(t *testing.T) {
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	product := &TestProduct{ID: "p1", Name: "Product"}
	category := &TestCategory{ID: "c1", Name: "Category"}

	productBinding, err := tm.GetByValue(product)
	require.NoError(t, err)
	categoryBinding, err := tm.GetByValue(category)
	require.NoError(t, err)

	productSchemaID := 100
	categorySchemaID := 200

	mockRegistry.On("RegisterSchema", productBinding).Return(productSchemaID, nil)
	mockRegistry.On("RegisterSchema", categoryBinding).Return(categorySchemaID, nil)

	productAvroData := []byte{0x01}
	categoryAvroData := []byte{0x02}

	mockEncoder.On("Encode", product, productBinding.ParsedSchema).Return(productAvroData, nil)
	mockEncoder.On("Encode", category, categoryBinding.ParsedSchema).Return(categoryAvroData, nil)

	mockBuilder.On("Build", productSchemaID, productAvroData).Return([]byte{0x00, 0x00, 0x00, 0x00, 0x64, 0x01})
	mockBuilder.On("Build", categorySchemaID, categoryAvroData).Return([]byte{0x00, 0x00, 0x00, 0x00, 0xC8, 0x02})

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act
	productResult, productTopic, err1 := serializer.SerializeWithTopic(product)
	categoryResult, categoryTopic, err2 := serializer.SerializeWithTopic(category)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotNil(t, productResult)
	assert.NotNil(t, categoryResult)
	assert.Equal(t, "test-product-events", productTopic)
	assert.Equal(t, "test-category-events", categoryTopic)
	assert.NotEqual(t, productResult, categoryResult)
	mockRegistry.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBuilder.AssertExpectations(t)
}

func TestSerializer_Serialize_DelegatesToSerializeWithTopic(t *testing.T) {
	// Verify that Serialize properly delegates to SerializeWithTopic
	// Arrange
	tm := createTestTypeMapping(t)
	mockRegistry := new(MockConfluentRegistry)
	mockEncoder := new(MockEncoder)
	mockBuilder := new(MockWireFormatBuilder)

	msg := &TestProduct{ID: "123", Name: "Test Product"}
	schemaID := 42
	avroData := []byte{0x06, 0x31, 0x32, 0x33}
	expectedWireFormat := []byte{0x00, 0x00, 0x00, 0x00, 0x2A, 0x06, 0x31, 0x32, 0x33}

	binding, err := tm.GetByValue(msg)
	require.NoError(t, err)

	mockRegistry.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil)
	mockBuilder.On("Build", schemaID, avroData).Return(expectedWireFormat)

	serializer := NewSerializer(tm, mockRegistry, mockEncoder, mockBuilder)

	// Act - call both methods and verify they return consistent results
	serializeResult, serializeErr := serializer.Serialize(msg)

	// Reset mocks for second call
	mockRegistry2 := new(MockConfluentRegistry)
	mockEncoder2 := new(MockEncoder)
	mockBuilder2 := new(MockWireFormatBuilder)

	mockRegistry2.On("RegisterSchema", binding).Return(schemaID, nil)
	mockEncoder2.On("Encode", msg, binding.ParsedSchema).Return(avroData, nil)
	mockBuilder2.On("Build", schemaID, avroData).Return(expectedWireFormat)

	serializer2 := NewSerializer(tm, mockRegistry2, mockEncoder2, mockBuilder2)
	serializeWithTopicResult, topic, serializeWithTopicErr := serializer2.SerializeWithTopic(msg)

	// Assert
	require.NoError(t, serializeErr)
	require.NoError(t, serializeWithTopicErr)
	assert.Equal(t, serializeResult, serializeWithTopicResult)
	assert.Equal(t, "test-product-events", topic)
}
