package event

import "time"

type BaseEvent struct {
	EventID       string    `json:"event_id"`
	CreatedAt     time.Time `json:"created_at"`
	Type          string    `json:"type"`
	Source        string    `json:"source"`
	Version       int       `json:"version"`
	TraceId       string    `json:"trace_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
}

type Event[T any] struct {
	BaseEvent
	Payload *T `json:"payload"`
}
