package observability

import (
	"context"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/zap"
)

// TraceContextUnaryInterceptor is a connect unary interceptor that injects trace and span IDs into the request logger.
func TraceContextUnaryInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
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
