package outbox

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

type channels struct {
	entities chan *outboxEntity
	delivery chan kafka.Event
}

func newChannels() *channels {
	return &channels{
		entities: make(chan *outboxEntity, 100),
		delivery: make(chan kafka.Event, 1000),
	}
}

func provideChannels() *channels {
	return newChannels()
}
