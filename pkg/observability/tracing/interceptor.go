package tracing

import (
	"context"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ConnectInterceptorPriority is the recommended priority for the tracing
// Connect interceptor (between recovery 10 and logger 20).
const ConnectInterceptorPriority = 15

// NewConnectInterceptorModule provides the trace-context Connect-RCP
// interceptor via FX. It injects trace_id and span_id into the request-scoped
// logger so that all subsequent logging includes trace fields.
func NewConnectInterceptorModule() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() interceptor.Interceptor {
				return interceptor.Interceptor{
					Priority: ConnectInterceptorPriority,
					Handler:  connect.UnaryInterceptorFunc(traceContextUnaryInterceptor),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func traceContextUnaryInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		traceID, spanID := GetTraceIDAndSpanID(ctx)
		if traceID != "" {
			scoped := logger.Get(ctx).With(
				zap.String("trace_id", traceID),
				zap.String("span_id", spanID),
			)
			ctx = logger.With(ctx, scoped)
		}
		return next(ctx, req)
	}
}
