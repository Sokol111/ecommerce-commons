package outbox

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

// channels holds the communication channels between outbox components.
type channels struct {
	entities chan *outboxEntity
	delivery chan kafka.Event
}

// newChannels creates and initializes the channels used for communication
// between the fetcher, sender, and confirmer components.
func newChannels() *channels {
	return &channels{
		entities: make(chan *outboxEntity, 100),
		delivery: make(chan kafka.Event, 1000),
	}
}

// provideChannels provides channels for dependency injection through fx.
func provideChannels() *channels {
	return newChannels()
}
