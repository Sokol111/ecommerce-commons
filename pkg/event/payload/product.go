package payload

import "time"

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
