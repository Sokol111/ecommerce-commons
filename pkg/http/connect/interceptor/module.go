package interceptor

import (
	"sort"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// Interceptor wraps a connect.Interceptor with a priority for ordered execution.
// Lower priority values execute first.
type Interceptor struct {
	Priority int
	Handler  connect.Interceptor
}

// NewModule provides all Connect-RPC interceptor modules.
// Interceptor execution order (by priority, lower = earlier):
//
//	10 - Recovery         - catches panics (must be first)
//	20 - Logger           - logs all RPCs
//	25 - Validation       - rejects invalid proto messages early
//	30 - Timeout          - kills hanging requests
//	40 - RateLimit        - limits requests/second
//	50 - Bulkhead         - limits concurrent requests
func NewModule() fx.Option {
	return fx.Options(
		RecoveryModule(10),
		LoggerModule(20),
		ValidationModule(25),
		TimeoutModule(30),
		RateLimitModule(40),
		BulkheadModule(50),
		fx.Provide(provideInterceptorChain),
	)
}

// interceptorIn is used for dependency injection of all interceptors.
type interceptorIn struct {
	fx.In
	Interceptors []Interceptor `group:"connect_interceptor"`
}

// provideInterceptorChain collects all interceptors, sorts by priority,
// filters nil handlers, and returns a []connect.Interceptor ready for
// connect.WithInterceptors(...).
func provideInterceptorChain(in interceptorIn) []connect.Interceptor {
	// Sort by priority (lower = executes first)
	sort.Slice(in.Interceptors, func(i, j int) bool {
		return in.Interceptors[i].Priority < in.Interceptors[j].Priority
	})

	// Filter nil handlers and extract connect.Interceptor
	return lo.FilterMap(in.Interceptors, func(i Interceptor, _ int) (connect.Interceptor, bool) {
		return i.Handler, i.Handler != nil
	})
}
