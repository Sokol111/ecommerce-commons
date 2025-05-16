package commonsserver

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"net/http"
)

var GinModule = fx.Options(
	fx.Provide(
		NewEngine,
		func(engine *gin.Engine) http.Handler {
			return engine
		},
	),
)

func NewEngine(log *zap.Logger) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(ErrorLoggerMiddleware(log))
	return engine
}
