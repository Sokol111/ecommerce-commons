package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var GinModule = fx.Options(
	fx.Provide(
		provideNewEngine,
		provideHandler,
	),
)

func provideNewEngine(log *zap.Logger) *gin.Engine {
	return NewEngine(log)
}

func provideHandler(e *gin.Engine) http.Handler {
	return e
}
