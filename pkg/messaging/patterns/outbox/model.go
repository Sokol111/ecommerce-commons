package outbox

import (
	"time"
)

const (
	StatusProcessing = "PROCESSING"
	StatusSent       = "SENT"
)

type outboxEntity struct {
	ID             string `bson:"_id"`
	Payload        string
	Key            string
	Topic          string
	Status         string
	CreatedAt      time.Time `bson:"createdAt"`
	SentAt         time.Time `bson:"sentAt,omitempty"`
	LockExpiresAt  time.Time `bson:"lockExpiresAt,omitempty"`
	AttemptsToSend int32     `bson:"attemptsToSend"`
}
