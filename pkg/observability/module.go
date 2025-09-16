package observability

import (
	"context"
	"strings"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/config"
	"github.com/Sokol111/ecommerce-commons/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"go.opentelemetry.io/otel/trace"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewTracingModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newConfig,
		),
		fx.Options(
			fx.Provide(func(lc fx.Lifecycle, log *zap.Logger, conf Config, appConf config.Config) (*sdktrace.TracerProvider, error) {
				if !conf.Enabled {
					log.Info("tracing disabled")
					return nil, nil
				}
				return provideTracerProvider(lc, log, conf, appConf)
			}),
			fx.Invoke(func(conf Config, engine *gin.Engine, appConf config.Config) {
				if conf.Enabled {
					addGinMiddleware(engine, appConf)
				}
			}),
			fx.Invoke(func(conf Config, engine *gin.Engine, log *zap.Logger) {
				if conf.Enabled {
					addTraceLoggerToContext(engine, log)
				}
			}),
		),
	)
}

func provideTracerProvider(lc fx.Lifecycle, log *zap.Logger, conf Config, appConf config.Config) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(conf.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(appConf.ServiceName),
		semconv.ServiceVersionKey.String(appConf.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(string(appConf.Environment)),
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

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetTracerProvider(tp)
			log.Info("otel tracing initialized", zap.String("endpoint", conf.Endpoint))
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

func addGinMiddleware(engine *gin.Engine, appConf config.Config) {
	engine.Use(otelgin.Middleware(appConf.ServiceName,
		otelgin.WithGinFilter(func(c *gin.Context) bool {
			route := c.FullPath()
			if route == "" {
				route = c.Request.URL.Path
			}
			if strings.HasPrefix(route, "/health") || strings.HasPrefix(route, "/metrics") {
				return false
			}
			return true
		}),
	))
}

func addTraceLoggerToContext(engine *gin.Engine, log *zap.Logger) {
	engine.Use(func(c *gin.Context) {
		l := withTrace(c.Request.Context(), log)
		ctx := context.WithValue(c.Request.Context(), logger.CtxKey, l)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
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
