package logger

import (
	"context"

	"go.uber.org/zap"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey struct{}

// loggerCtxKey is the context key used to store and retrieve logger instances from context.
var loggerCtxKey = contextKey{}

// Get extracts a logger from the context.
// If no logger is found in the context, it returns the default logger.
// This function is safe to call with a nil context.
func Get(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return defaultLogger
	}
	if ctxLogger, ok := ctx.Value(loggerCtxKey).(*zap.Logger); ok && ctxLogger != nil {
		return ctxLogger
	}
	return defaultLogger
}

// With returns a new context with the provided logger attached.
// This allows propagating logger instances through the application context.
func With(ctx context.Context, logger *zap.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerCtxKey, logger)
}
