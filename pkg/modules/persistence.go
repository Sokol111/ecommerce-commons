package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

func NewPersistenceModule() fx.Option {
	return fx.Options(
		mongo.NewMongoModule(),
	)
}
