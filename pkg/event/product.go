package event

import "time"

type ProductCreatedPayload struct {
	ProductID   string    `json:"product_id"`
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}
