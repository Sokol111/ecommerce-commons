package observability

import (
	"context"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	commongin "github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"go.opentelemetry.io/otel/trace"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewTracingModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newConfig,
		),
		fx.Provide(
			func(lc fx.Lifecycle, log *zap.Logger, conf Config, appConf config.Config, readiness health.Readiness) (trace.TracerProvider, error) {
				if !conf.TracingEnabled {
					log.Info("tracing disabled")
					return nil, nil
				}
				return provideTracerProvider(lc, log, conf, appConf, readiness)
			},
		),
		fx.Provide(
			fx.Annotate(
				func(conf Config, log *zap.Logger) commongin.Middleware {
					if !conf.TracingEnabled {
						return commongin.Middleware{}
					}
					return commongin.Middleware{Priority: 50, Handler: tracingLoggerMiddleware(log)}
				},
				fx.ResultTags(`group:"gin_mw"`),
			),
		),
		fx.Invoke(func(trace.TracerProvider) {}),
	)
}

func provideTracerProvider(lc fx.Lifecycle, log *zap.Logger, conf Config, appConf config.Config, readiness health.Readiness) (trace.TracerProvider, error) {
	readiness.AddOne()

	ctx := context.Background()

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(appConf.ServiceName),
		semconv.ServiceVersionKey.String(appConf.ServiceVersion),
		semconv.DeploymentEnvironmentNameKey.String(appConf.Environment),
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),            // allows adding OTEL_RESOURCE_ATTRIBUTES if needed
		resource.WithProcess(),            // pid, runtime
		resource.WithOS(),                 // OS info
		resource.WithHost(),               // hostname
		resource.WithAttributes(attrs...), // the rest of attributes
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider

	if conf.OtelCollectorEndpoint != "" {
		exp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(conf.OtelCollectorEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		)
	} else {
		log.Info("otel tracing: no collector endpoint provided; running in local in-process mode (no export)")
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
			sdktrace.WithResource(res),
		)
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			log.Info("otel tracing initialized", zap.String("endpoint", conf.OtelCollectorEndpoint))
			readiness.Done()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			c, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return tp.Shutdown(c)
		},
	})

	return tp, nil
}

func GetTraceId(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

func tracingLoggerMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := withTrace(c.Request.Context(), log)
		ctx := context.WithValue(c.Request.Context(), logger.LoggerCtxKey, l)
		c.Request = c.Request.WithContext(ctx)
		c.Set(string(logger.LoggerCtxKey), l)
		c.Next()
	}
}

func withTrace(ctx context.Context, log *zap.Logger) *zap.Logger {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return log
	}
	return log.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}

// TraceFunc wraps a function with automatic span creation and error handling
// Usage: TraceFunc(ctx, "operation.name", opts, func(ctx context.Context) error { ... })
func TraceFunc(ctx context.Context, spanName string, opts []trace.SpanStartOption, fn func(context.Context) error) error {
	tracer := otel.Tracer("app")
	ctx, span := tracer.Start(ctx, spanName, opts...)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "success")
	return nil
}

// TraceFuncWithResult wraps a function that returns a value and error
// Usage: result, err := TraceFuncWithResult(ctx, "operation.name", opts, func(ctx context.Context) (T, error) { ... })
func TraceFuncWithResult[T any](ctx context.Context, spanName string, opts []trace.SpanStartOption, fn func(context.Context) (T, error)) (T, error) {
	tracer := otel.Tracer("app")
	ctx, span := tracer.Start(ctx, spanName, opts...)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		var zero T
		return zero, err
	}

	span.SetStatus(codes.Ok, "success")
	return result, nil
}

// WithSpan creates a new span and returns updated context
// Usage: ctx, endSpan := WithSpan(ctx, "operation.name", opts); defer endSpan(nil)
func WithSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, func(error)) {
	tracer := otel.Tracer("app")
	ctx, span := tracer.Start(ctx, spanName, opts...)

	endFunc := func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}

	return ctx, endFunc
}
