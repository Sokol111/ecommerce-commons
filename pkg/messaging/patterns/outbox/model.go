package outbox

import (
	"time"
)

const (
	StatusProcessing = "PROCESSING"
	StatusSent       = "SENT"
)

type outboxEntity struct {
	ID             string            `bson:"_id"`
	Payload        []byte            `bson:"payload"`
	Key            string            `bson:"key"`
	Topic          string            `bson:"topic"`
	Headers        map[string]string `bson:"headers,omitempty"`
	Status         string            `bson:"status"`
	CreatedAt      time.Time         `bson:"createdAt"`
	SentAt         time.Time         `bson:"sentAt,omitempty"`
	LockExpiresAt  time.Time         `bson:"lockExpiresAt,omitempty"`
	AttemptsToSend int32             `bson:"attemptsToSend"`
}
