package middleware

import (
	"sort"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// NewGinModule provides all Gin middleware modules.
// Middleware execution order (by priority, lower = earlier):
//
//	10 - Recovery         - catches panics (must be first)
//	20 - Logger           - logs all requests
//	30 - ErrorLogger      - logs errors from ALL middleware and handlers
//	40 - Problem          - converts errors to RFC 7807 (must wrap all error sources)
//	50 - Timeout          - kills hanging requests
//	60 - CircuitBreaker   - protects against cascading failures
//	70 - RateLimit        - limits requests/second
//	80 - HTTPBulkhead     - limits concurrent requests
//
// Note: OpenAPI validation is NOT included here - use ogen-generated servers
// which provide built-in validation with better type safety.
func NewGinModule() fx.Option {
	return fx.Options(
		RecoveryModule(10),
		LoggerModule(20),
		ErrorLoggerModule(30),
		ProblemModule(40),
		TimeoutModule(50),
		CircuitBreakerModule(60),
		RateLimitModule(70),
		HTTPBulkheadModule(80),
		fx.Invoke(registerMiddlewares),
	)
}

// mwIn is used for dependency injection of all middlewares.
type mwIn struct {
	fx.In
	Middlewares []Middleware `group:"gin_mw"`
}

// registerMiddlewares registers all middlewares on the Gin engine sorted by priority.
func registerMiddlewares(engine *gin.Engine, in mwIn) {
	mws := in.Middlewares

	// Sort middlewares by priority (lower priority = executes first)
	sort.Slice(mws, func(i, j int) bool { return mws[i].Priority < mws[j].Priority })

	// Register middlewares
	for _, m := range mws {
		if m.Handler == nil {
			continue // Skip disabled middlewares
		}
		engine.Use(m.Handler)
	}
}
