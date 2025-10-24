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

// Problem represents RFC7807 Problem Details for HTTP APIs
type Problem struct {
	Type     string       `json:"type,omitempty"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	TraceID  string       `json:"traceId,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
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
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: 400, Handler: problemMiddleware()}
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
	engine := gin.New(func(e *gin.Engine) {
		e.ContextWithFallback = true
	})

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

func problemMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only handle if there are errors and response hasn't been written yet
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		// Get status code, default to 500 if not set
		status := c.Writer.Status()
		if status == 0 || status == http.StatusOK {
			status = http.StatusInternalServerError
		}

		// Build Problem Details from the first error
		firstErr := c.Errors[0]
		problem := Problem{
			Type:     "about:blank",
			Title:    http.StatusText(status),
			Status:   status,
			Detail:   firstErr.Error(),
			Instance: c.Request.URL.Path,
		}

		// Try to extract trace ID from context if available
		if traceID, exists := c.Get(string(logger.LoggerCtxKey)); exists {
			if tid, ok := traceID.(string); ok {
				problem.TraceID = tid
			}
		}

		// If meta contains field errors, add them
		if meta, ok := firstErr.Meta.(map[string]string); ok {
			for field, msg := range meta {
				problem.Errors = append(problem.Errors, FieldError{
					Field:   field,
					Message: msg,
				})
			}
		}

		// If meta is already a Problem, use it
		if existingProblem, ok := firstErr.Meta.(*Problem); ok {
			problem = *existingProblem
		}

		c.JSON(status, problem)
	}
}
