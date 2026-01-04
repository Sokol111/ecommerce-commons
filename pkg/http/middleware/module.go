package middleware

import (
	"sort"

	"github.com/ogen-go/ogen/middleware"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// NewOgenMiddlewareModule provides all ogen middleware modules.
// Middleware execution order (by priority, lower = earlier):
//
//	10 - Recovery         - catches panics (must be first)
//	20 - Logger           - logs all requests
//	30 - Timeout          - kills hanging requests
//	40 - RateLimit        - limits requests/second
//	50 - HTTPBulkhead     - limits concurrent requests
//
// Note: ErrorLogger and Problem middlewares are not needed with ogen -
// use OgenErrorHandler which handles both logging and RFC 7807 responses.
func NewOgenMiddlewareModule() fx.Option {
	return fx.Options(
		RecoveryModule(10),
		LoggerModule(20),
		TimeoutModule(30),
		RateLimitModule(40),
		HTTPBulkheadModule(50),
		fx.Provide(provideMiddlewareChain),
	)
}

// mwIn is used for dependency injection of all middlewares.
type mwIn struct {
	fx.In
	Middlewares []Middleware `group:"ogen_mw"`
}

// provideMiddlewareChain collects all middlewares, sorts by priority, and returns a slice.
func provideMiddlewareChain(in mwIn) []middleware.Middleware {
	// Sort by priority (lower = executes first)
	sort.Slice(in.Middlewares, func(i, j int) bool {
		return in.Middlewares[i].Priority < in.Middlewares[j].Priority
	})

	// Filter nil handlers and extract middleware.Middleware
	return lo.FilterMap(in.Middlewares, func(m Middleware, _ int) (middleware.Middleware, bool) {
		return m.Handler, m.Handler != nil
	})
}
