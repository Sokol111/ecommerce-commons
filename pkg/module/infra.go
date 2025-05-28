package module

import (
	"github.com/Sokol111/ecommerce-commons/pkg/config"
	"github.com/Sokol111/ecommerce-commons/pkg/gin"
	"github.com/Sokol111/ecommerce-commons/pkg/logging"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"github.com/Sokol111/ecommerce-commons/pkg/server"
	"github.com/Sokol111/ecommerce-commons/pkg/swaggerui"
	"go.uber.org/fx"
)

var InfraModules = fx.Options(
	logging.ZapLoggingModule,
	config.ViperModule,
	mongo.MongoModule,
	gin.GinModule,
	server.HttpServerModule,
	swaggerui.SwaggerModule,
)
