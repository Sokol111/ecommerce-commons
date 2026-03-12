package producer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestKgoClientImplementsProducer(t *testing.T) {
	var _ Producer = (*kgo.Client)(nil)

	client, err := kgo.NewClient()
	assert.NoError(t, err)
	defer client.Close()

	var p Producer = client
	assert.NotNil(t, p)
}
