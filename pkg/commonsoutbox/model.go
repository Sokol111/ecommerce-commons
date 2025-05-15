package commonsoutbox

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type OutboxEntity struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	Payload        string
	Key            string
	Topic          string
	Status         string
	CreatedAt      time.Time `bson:"createdAt"`
	LockExpiresAt  time.Time `bson:"lockExpiresAt,omitempty"`
	AttemptsToSend int32     `bson:"attemptsToSend"`
}
