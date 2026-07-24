package fxconfig

import (
	"sort"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
)

// Interceptor wraps a connect.Interceptor with a priority for ordered execution.
// Lower priority values execute first.
type Interceptor struct {
	Priority int
	Handler  connect.Interceptor
}

// NewInterceptorsModule provides all Connect-RPC interceptor modules.
// Interceptor execution order (by priority, lower = earlier):
//
//	10 - Recovery         - catches panics (must be first)
//	20 - Logger           - logs all RPCs
//	25 - Validation       - rejects invalid proto messages early
//	30 - Timeout          - kills hanging requests
//	40 - RateLimit        - limits requests/second
//	50 - Bulkhead         - limits concurrent requests
func NewInterceptorsModule() fx.Option {
	return fx.Options(
		fx.Supply(
			fx.Annotate(
				Interceptor{Priority: 10, Handler: connect.UnaryInterceptorFunc(interceptor.RecoveryUnaryInterceptor)},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Supply(
			fx.Annotate(
				Interceptor{Priority: 20, Handler: connect.UnaryInterceptorFunc(interceptor.LoggerUnaryInterceptor)},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				provideValidationInterceptor,
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				provideTimeoutInterceptor,
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				provideRateLimitInterceptor,
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				provideBulkheadInterceptor,
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Provide(provideInterceptorChain),
	)
}

func provideBulkheadInterceptor(serverConfig server.Config, log *zap.Logger) Interceptor {
	if !serverConfig.Bulkhead.Enabled {
		return Interceptor{} // nil Handler, will be skipped
	}
	log.Info("Connect bulkhead interceptor initialized",
		zap.Int("max-concurrent", serverConfig.Bulkhead.MaxConcurrent),
		zap.Duration("timeout", serverConfig.Bulkhead.Timeout),
	)
	return Interceptor{
		Priority: 50,
		Handler:  interceptor.NewBulkheadInterceptor(serverConfig.Bulkhead.MaxConcurrent, serverConfig.Bulkhead.Timeout),
	}
}

func provideRateLimitInterceptor(config server.Config) Interceptor {
	if !config.RateLimit.Enabled {
		return Interceptor{} // nil Handler, will be skipped
	}
	return Interceptor{
		Priority: 40,
		Handler:  interceptor.NewRateLimitInterceptor(config.RateLimit.RequestsPerSecond, config.RateLimit.Burst),
	}
}

func provideTimeoutInterceptor(serverConfig server.Config, log *zap.Logger) Interceptor {
	if serverConfig.Timeout.RequestTimeout <= 0 {
		return Interceptor{} // nil Handler, will be skipped
	}
	log.Info("Connect timeout interceptor initialized",
		zap.Duration("request-timeout", serverConfig.Timeout.RequestTimeout),
	)
	return Interceptor{
		Priority: 30,
		Handler:  interceptor.NewTimeoutInterceptor(serverConfig.Timeout.RequestTimeout),
	}
}

func provideValidationInterceptor() (Interceptor, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return Interceptor{}, err
	}
	return Interceptor{
		Priority: 25,
		Handler:  interceptor.NewValidationInterceptor(validator),
	}, nil
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
