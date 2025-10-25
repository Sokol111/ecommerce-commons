package messaging

import "time"

type BaseEvent struct {
	EventID       string    `json:"event_id"`
	CreatedAt     time.Time `json:"created_at"`
	Type          string    `json:"type"`
	Source        string    `json:"source"`
	Topic         string    `json:"topic"`
	Version       int       `json:"version"`
	TraceId       string    `json:"trace_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
}

type Event[T any] struct {
	BaseEvent
	Payload *T `json:"payload"`
}

type ProductCreated struct {
	ProductID   string    `json:"product_id"`
	Name        string    `json:"name"`
	Price       float32   `json:"price"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
	Quantity    int       `json:"quantity"`
	ImageId     *string   `json:"image_id,omitempty"`
	Enabled     bool      `json:"enabled"`
}

type ProductUpdated struct {
	ProductID   string    `json:"product_id"`
	Name        string    `json:"name"`
	Price       float32   `json:"price"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
	Quantity    int       `json:"quantity"`
	ImageId     *string   `json:"image_id,omitempty"`
	Enabled     bool      `json:"enabled"`
}

type CategoryCreated struct {
	CategoryID string    `json:"product_id"`
	Name       string    `json:"name"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Enabled    bool      `json:"enabled"`
}

type CategoryUpdated struct {
	CategoryID string    `json:"product_id"`
	Name       string    `json:"name"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Enabled    bool      `json:"enabled"`
}
