package outbox

import (
	"time"
)

type outboxEntity struct {
	ID             string `bson:"_id"`
	Payload        string
	Key            string
	Topic          string
	Status         string
	CreatedAt      time.Time `bson:"createdAt"`
	LockExpiresAt  time.Time `bson:"lockExpiresAt,omitempty"`
	AttemptsToSend int32     `bson:"attemptsToSend"`
}
