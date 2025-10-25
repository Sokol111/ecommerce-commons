package consumer

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging"
)

type Handler[P any] interface {
	Validate(payload *P) error
	Process(ctx context.Context, e *messaging.Event[P]) error
}
