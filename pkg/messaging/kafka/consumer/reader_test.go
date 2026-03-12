package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
)

func TestNewReader(t *testing.T) {
	t.Run("creates reader with dependencies", func(t *testing.T) {
		messagesChan := make(chan *kgo.Record, 10)
		log := zap.NewNop()
		consumerConf := config.ConsumerConfig{MaxPollRecords: 500}

		// newReader takes *kgo.Client which requires real brokers for full testing
		r := newReader(nil, messagesChan, consumerConf, log)

		assert.NotNil(t, r)
		assert.Equal(t, log, r.log)
		assert.Equal(t, 500, r.maxPollRecords)
		assert.NotNil(t, r.throttler)
	})
}
