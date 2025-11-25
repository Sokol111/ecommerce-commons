package encoding

import (
	"fmt"

	hambavro "github.com/hamba/avro/v2"
)

// Encoder encodes Go structs to Avro bytes
type Encoder interface {
	// Encode serializes a Go object to Avro bytes based on schema
	Encode(msg interface{}, schema hambavro.Schema) ([]byte, error)
}

type hambaEncoder struct{}

// NewHambaEncoder creates an Avro encoder using hamba/avro library
func NewHambaEncoder() Encoder {
	return &hambaEncoder{}
}

func (e *hambaEncoder) Encode(msg interface{}, schema hambavro.Schema) ([]byte, error) {
	// Marshal message to Avro bytes
	avroData, err := hambavro.Marshal(schema, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal avro data: %w", err)
	}
	return avroData, nil
}
