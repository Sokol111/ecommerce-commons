package tracing

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GetTraceID extracts trace ID from context.
func GetTraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// GetSpanID extracts span ID from context.
func GetSpanID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}

// GetTraceIDAndSpanID extracts both trace ID and span ID from context.
func GetTraceIDAndSpanID(ctx context.Context) (string, string) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}

// AddAttribute adds an attribute to the current span.
func AddAttribute(ctx context.Context, key string, value string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String(key, value))
}
