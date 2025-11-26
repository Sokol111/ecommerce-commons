package deserialization

import (
	"errors"
	"reflect"
	"testing"

	hambavro "github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type mockWireFormatParser struct {
	mock.Mock
}

func (m *mockWireFormatParser) Parse(data []byte) (int, []byte, error) {
	args := m.Called(data)
	return args.Int(0), args.Get(1).([]byte), args.Error(2)
}

type mockWriterSchemaResolver struct {
	mock.Mock
}

func (m *mockWriterSchemaResolver) Resolve(schemaID int) (hambavro.Schema, string, error) {
	args := m.Called(schemaID)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(hambavro.Schema), args.String(1), args.Error(2)
}

type mockDecoder struct {
	mock.Mock
}

func (m *mockDecoder) Decode(payload []byte, writerSchema hambavro.Schema, schemaName string) (interface{}, error) {
	args := m.Called(payload, writerSchema, schemaName)
	return args.Get(0), args.Error(1)
}

// Test event struct
type TestEvent struct {
	ID   string `avro:"id"`
	Name string `avro:"name"`
}

func TestDeserializer_Deserialize_Success(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	inputData := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03} // wire format + payload
	payload := []byte{0x02, 0x03}
	schemaID := 1
	schemaName := "test.TestEvent"

	testSchema := `{
		"type": "record",
		"name": "TestEvent",
		"namespace": "test",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"}
		]
	}`
	parsedSchema, err := hambavro.Parse(testSchema)
	assert.NoError(t, err)

	expectedEvent := &TestEvent{ID: "123", Name: "Test"}

	// Mock expectations
	parser.On("Parse", inputData).Return(schemaID, payload, nil)
	resolver.On("Resolve", schemaID).Return(parsedSchema, schemaName, nil)
	decoder.On("Decode", payload, parsedSchema, schemaName).Return(expectedEvent, nil)

	// Act
	result, err := deserializer.Deserialize(inputData)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedEvent, result)

	parser.AssertExpectations(t)
	resolver.AssertExpectations(t)
	decoder.AssertExpectations(t)
}

func TestDeserializer_Deserialize_ParseError(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	inputData := []byte{0xFF, 0x00} // invalid wire format
	parseError := errors.New("invalid magic byte")

	// Mock expectations
	parser.On("Parse", inputData).Return(0, []byte(nil), parseError)

	// Act
	result, err := deserializer.Deserialize(inputData)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse wire format")
	assert.Contains(t, err.Error(), "invalid magic byte")

	parser.AssertExpectations(t)
}

func TestDeserializer_Deserialize_ResolverError(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	inputData := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03}
	payload := []byte{0x02, 0x03}
	schemaID := 999
	resolverError := errors.New("schema not found in registry")

	// Mock expectations
	parser.On("Parse", inputData).Return(schemaID, payload, nil)
	resolver.On("Resolve", schemaID).Return(nil, "", resolverError)

	// Act
	result, err := deserializer.Deserialize(inputData)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to resolve schema for ID 999")
	assert.Contains(t, err.Error(), "schema not found in registry")

	parser.AssertExpectations(t)
	resolver.AssertExpectations(t)
}

func TestDeserializer_Deserialize_DecoderError(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	inputData := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03}
	payload := []byte{0x02, 0x03}
	schemaID := 1
	schemaName := "test.TestEvent"

	testSchema := `{
		"type": "record",
		"name": "TestEvent",
		"namespace": "test",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`
	parsedSchema, err := hambavro.Parse(testSchema)
	assert.NoError(t, err)

	decodeError := errors.New("failed to unmarshal avro data")

	// Mock expectations
	parser.On("Parse", inputData).Return(schemaID, payload, nil)
	resolver.On("Resolve", schemaID).Return(parsedSchema, schemaName, nil)
	decoder.On("Decode", payload, parsedSchema, schemaName).Return(nil, decodeError)

	// Act
	result, err := deserializer.Deserialize(inputData)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode avro data")
	assert.Contains(t, err.Error(), "failed to unmarshal avro data")

	parser.AssertExpectations(t)
	resolver.AssertExpectations(t)
	decoder.AssertExpectations(t)
}

func TestDeserializer_Deserialize_EmptyData(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	inputData := []byte{}
	parseError := errors.New("data too short")

	// Mock expectations
	parser.On("Parse", inputData).Return(0, []byte(nil), parseError)

	// Act
	result, err := deserializer.Deserialize(inputData)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse wire format")

	parser.AssertExpectations(t)
}

func TestDeserializer_Deserialize_MultipleEvents(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	testSchema := `{
		"type": "record",
		"name": "TestEvent",
		"namespace": "test",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "name", "type": "string"}
		]
	}`
	parsedSchema, err := hambavro.Parse(testSchema)
	assert.NoError(t, err)

	// Test case 1: Event A
	inputData1 := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x0A, 0x0B}
	payload1 := []byte{0x0A, 0x0B}
	event1 := &TestEvent{ID: "1", Name: "Event A"}

	parser.On("Parse", inputData1).Return(1, payload1, nil)
	resolver.On("Resolve", 1).Return(parsedSchema, "test.TestEvent", nil)
	decoder.On("Decode", payload1, parsedSchema, "test.TestEvent").Return(event1, nil)

	result1, err1 := deserializer.Deserialize(inputData1)
	assert.NoError(t, err1)
	assert.Equal(t, event1, result1)

	// Test case 2: Event B
	inputData2 := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x0C, 0x0D}
	payload2 := []byte{0x0C, 0x0D}
	event2 := &TestEvent{ID: "2", Name: "Event B"}

	parser.On("Parse", inputData2).Return(1, payload2, nil)
	resolver.On("Resolve", 1).Return(parsedSchema, "test.TestEvent", nil)
	decoder.On("Decode", payload2, parsedSchema, "test.TestEvent").Return(event2, nil)

	result2, err2 := deserializer.Deserialize(inputData2)
	assert.NoError(t, err2)
	assert.Equal(t, event2, result2)

	parser.AssertExpectations(t)
	resolver.AssertExpectations(t)
	decoder.AssertExpectations(t)
}

func TestDeserializer_Deserialize_DifferentSchemaIDs(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	schema1 := `{"type": "record", "name": "Event1", "namespace": "test", "fields": [{"name": "id", "type": "string"}]}`
	schema2 := `{"type": "record", "name": "Event2", "namespace": "test", "fields": [{"name": "name", "type": "string"}]}`

	parsedSchema1, _ := hambavro.Parse(schema1)
	parsedSchema2, _ := hambavro.Parse(schema2)

	// Message with schema ID 1
	inputData1 := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x0A}
	payload1 := []byte{0x0A}
	event1 := &TestEvent{ID: "123"}

	parser.On("Parse", inputData1).Return(1, payload1, nil)
	resolver.On("Resolve", 1).Return(parsedSchema1, "test.Event1", nil)
	decoder.On("Decode", payload1, parsedSchema1, "test.Event1").Return(event1, nil)

	result1, err1 := deserializer.Deserialize(inputData1)
	assert.NoError(t, err1)
	assert.Equal(t, event1, result1)

	// Message with schema ID 2
	inputData2 := []byte{0x00, 0x00, 0x00, 0x00, 0x02, 0x0B}
	payload2 := []byte{0x0B}
	event2 := &TestEvent{Name: "Test"}

	parser.On("Parse", inputData2).Return(2, payload2, nil)
	resolver.On("Resolve", 2).Return(parsedSchema2, "test.Event2", nil)
	decoder.On("Decode", payload2, parsedSchema2, "test.Event2").Return(event2, nil)

	result2, err2 := deserializer.Deserialize(inputData2)
	assert.NoError(t, err2)
	assert.Equal(t, event2, result2)

	parser.AssertExpectations(t)
	resolver.AssertExpectations(t)
	decoder.AssertExpectations(t)
}

func TestNewDeserializer(t *testing.T) {
	// Arrange
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	// Act
	deserializer := NewDeserializer(parser, resolver, decoder)

	// Assert
	assert.NotNil(t, deserializer)
	assert.Implements(t, (*Deserializer)(nil), deserializer)

	// Verify internal structure (using type assertion)
	avroDeser, ok := deserializer.(*avroDeserializer)
	assert.True(t, ok)
	assert.Equal(t, parser, avroDeser.parser)
	assert.Equal(t, resolver, avroDeser.resolver)
	assert.Equal(t, decoder, avroDeser.decoder)
}

func TestDeserializer_InterfaceCompliance(t *testing.T) {
	// Verify that avroDeserializer implements Deserializer interface
	var _ Deserializer = (*avroDeserializer)(nil)

	// Create instance
	parser := new(mockWireFormatParser)
	resolver := new(mockWriterSchemaResolver)
	decoder := new(mockDecoder)

	deserializer := NewDeserializer(parser, resolver, decoder)

	// Verify interface methods are available
	deserializerType := reflect.TypeOf(deserializer)
	method, found := deserializerType.MethodByName("Deserialize")
	assert.True(t, found)
	assert.Equal(t, 2, method.Type.NumIn())  // receiver + data parameter
	assert.Equal(t, 2, method.Type.NumOut()) // result + error
}
