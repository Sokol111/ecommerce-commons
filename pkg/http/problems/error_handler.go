package problems

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/ogen-go/ogen/ogenerrors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewErrorHandlerModule provides the error handler for ogen servers.
func NewErrorHandlerModule() fx.Option {
	return fx.Provide(NewErrorHandler)
}

// NewErrorHandler returns an ogen ErrorHandler that logs errors and returns
// RFC 7807 Problem Details responses.
func NewErrorHandler() ogenerrors.ErrorHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		code := errorToStatusCode(err)

		problem := Problem{
			Type:     "about:blank",
			Title:    http.StatusText(code),
			Status:   code,
			Detail:   err.Error(),
			Instance: r.URL.Path,
		}

		// Add trace ID if available
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			problem.TraceID = sc.TraceID().String()
		}

		// Log error
		log := logger.Get(ctx)
		log.Error("Request error",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.Int("status", code),
			zap.String("error", err.Error()),
			zap.Any("meta", &problem),
		)

		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(problem)
	}
}

// errorToStatusCode maps errors to HTTP status codes.
func errorToStatusCode(err error) int {
	switch {
	case errors.Is(err, middleware.ErrRequestTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, middleware.ErrRateLimitExceeded):
		return http.StatusTooManyRequests
	case errors.Is(err, middleware.ErrBulkheadFull):
		return http.StatusServiceUnavailable
	case errors.Is(err, middleware.ErrPanic):
		return http.StatusInternalServerError
	default:
		return ogenerrors.ErrorCode(err)
	}
}
