package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// recoveryMiddleware handles panics and converts them to 500 errors.
func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				fields := append(requestFields(c),
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
				)
				logger.Get(c).Error("Panic recovered", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// RecoveryModule provides recovery middleware.
func RecoveryModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: priority, Handler: recoveryMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
