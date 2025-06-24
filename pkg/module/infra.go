package module

import (
	"github.com/Sokol111/ecommerce-commons/pkg/config"
	"github.com/Sokol111/ecommerce-commons/pkg/gin"
	"github.com/Sokol111/ecommerce-commons/pkg/health"
	"github.com/Sokol111/ecommerce-commons/pkg/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"github.com/Sokol111/ecommerce-commons/pkg/server"
	"github.com/Sokol111/ecommerce-commons/pkg/swaggerui"
	"go.uber.org/fx"
)

func NewInfraModule() fx.Option {
	return fx.Options(
		logger.NewZapLoggingModule(),
		config.NewViperModule(),
		mongo.NewMongoModule(),
		gin.NewGinModule(),
		server.NewHttpServerModule(),
		swaggerui.NewSwaggerModule(),
		health.NewHealthModule(),
	)
}
