package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

// NewPersistenceModule provides persistence functionality: mongo, txManager
func NewPersistenceModule() fx.Option {
	return fx.Options(
		mongo.NewMongoModule(),
	)
}
