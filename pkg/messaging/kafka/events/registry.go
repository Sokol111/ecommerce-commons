package events

import (
	"fmt"
	"sync"
)

// EventFactory creates a new instance of an event.
type EventFactory func() Event

// EventRegistry stores event factories by schema name for deserialization.
// Use this to register events from multiple API packages and look them up by schema name.
type EventRegistry interface {
	// Register adds an event factory for the given schema name.
	Register(schemaName string, factory EventFactory)
	// NewEvent creates a new event instance by its Avro schema name.
	// Returns an error if the schema name is not registered.
	NewEvent(schemaName string) (Event, error)
	// HasSchema checks if a schema is registered.
	HasSchema(schemaName string) bool
	// All returns one instance of each registered event.
	// Useful for schema registration on startup.
	All() []Event
}

type eventRegistry struct {
	mu        sync.RWMutex
	factories map[string]EventFactory
}

// NewEventRegistry creates a new event registry.
func NewEventRegistry() EventRegistry {
	return &eventRegistry{
		factories: make(map[string]EventFactory),
	}
}

// Register adds an event factory for the given schema name.
// If the schema name is already registered, it will be overwritten.
func (r *eventRegistry) Register(schemaName string, factory EventFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[schemaName] = factory
}

// NewEvent creates a new event instance by its Avro schema name.
// Returns an error if the schema name is not registered.
func (r *eventRegistry) NewEvent(schemaName string) (Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.factories[schemaName]
	if !ok {
		return nil, fmt.Errorf("unknown event schema: %s", schemaName)
	}
	return factory(), nil
}

// HasSchema checks if a schema is registered.
func (r *eventRegistry) HasSchema(schemaName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.factories[schemaName]
	return ok
}

// All returns one instance of each registered event.
func (r *eventRegistry) All() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Event, 0, len(r.factories))
	for _, factory := range r.factories {
		result = append(result, factory())
	}
	return result
}
