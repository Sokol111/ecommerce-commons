package persistence

import "errors"

var (
	// ErrEntityNotFound is returned when an entity is not found in the repository.
	ErrEntityNotFound = errors.New("entity not found")

	// ErrOptimisticLocking is returned when an optimistic locking conflict occurs.
	ErrOptimisticLocking = errors.New("optimistic locking error")
)
