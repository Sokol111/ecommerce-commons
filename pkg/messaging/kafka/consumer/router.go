package consumer

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"
)

// Router dispatches events to registered typed handler functions.
// It implements the Handler interface.
type Router struct {
	handlers map[reflect.Type]func(ctx context.Context, event any) error
	log      *zap.Logger
}

// NewRouter creates a new Router instance.
func NewRouter(log *zap.Logger) *Router {
	return &Router{
		handlers: make(map[reflect.Type]func(ctx context.Context, event any) error),
		log:      log,
	}
}

// Register adds a typed handler function for event type E.
// The handler will be called when an event of type *E is dispatched.
func Register[E any](r *Router, fn func(context.Context, *E) error) {
	eventType := reflect.TypeOf((*E)(nil))
	r.handlers[eventType] = func(ctx context.Context, event any) error {
		return fn(ctx, event.(*E)) //nolint:errcheck // type guaranteed by reflect lookup in Process
	}
}

// Process implements Handler. It dispatches the event to the registered handler
// based on its type. Returns ErrSkipMessage if no handler is registered.
func (r *Router) Process(ctx context.Context, event any) error {
	eventType := reflect.TypeOf(event)
	handler, ok := r.handlers[eventType]
	if !ok {
		r.log.Warn("no handler registered for event type, skipping",
			zap.String("type", fmt.Sprintf("%T", event)))
		return fmt.Errorf("no handler for event type %T: %w", event, ErrSkipMessage)
	}
	return handler(ctx, event)
}
