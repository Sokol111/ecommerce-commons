package gin

import (
	"net/http"
	"runtime/debug"
	"sort"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Middleware struct {
	Priority int
	Handler  gin.HandlerFunc
}

type mwIn struct {
	fx.In
	Middlewares []Middleware `group:"gin_mw"`
}

func NewGinModule() fx.Option {
	return fx.Provide(
		provideGinAndHandler,
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: 100, Handler: recoveryMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: 200, Handler: loggerMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: 300, Handler: errorLoggerMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}

func provideGinAndHandler(in mwIn) (*gin.Engine, http.Handler) {
	e := newEngine(in.Middlewares)
	return e, e
}

func newEngine(mws []Middleware) *gin.Engine {
	engine := gin.New()

	sort.Slice(mws, func(i, j int) bool { return mws[i].Priority < mws[j].Priority })
	for _, m := range mws {
		if m.Handler == nil {
			continue
		}
		engine.Use(m.Handler)
	}

	return engine
}

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health/live" || path == "/health/ready" {
			c.Next()
			return
		}

		start := time.Now()
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.FromContext(c).Debug("Incoming request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", raw),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.FromContext(c).Error("Panic recovered",
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("client_ip", c.ClientIP()),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func errorLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			log := logger.FromContext(c)
			for _, e := range c.Errors {
				log.Error("Request error",
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.Int("status", c.Writer.Status()),
					zap.String("error", e.Error()),
					zap.Any("meta", e.Meta),
				)
			}
		}
	}
}
