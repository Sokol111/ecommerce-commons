package payload

import "time"

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
