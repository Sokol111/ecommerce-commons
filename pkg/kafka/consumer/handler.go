package consumer

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/event"
)

type Handler[P any] interface {
	Validate(payload *P) error
	Process(ctx context.Context, e *event.Event[P]) error
}
