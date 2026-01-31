package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

// NewPersistenceModule provides persistence layer components for dependency injection.
func NewPersistenceModule() fx.Option {
	return fx.Options(
		mongo.NewMongoModule(),
	)
}
