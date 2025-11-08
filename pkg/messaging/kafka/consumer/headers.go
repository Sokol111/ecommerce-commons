package consumer

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

// GetEventType extracts the event-type header value from Kafka message headers
func GetEventType(headers []kafka.Header) string {
	for _, header := range headers {
		if header.Key == "event-type" {
			return string(header.Value)
		}
	}
	return ""
}
