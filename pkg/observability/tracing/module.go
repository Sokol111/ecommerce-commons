package tracing

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	appmiddleware "github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	otelconfig "github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	"github.com/ogen-go/ogen/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// providerParams holds dependencies for tracing provider.
type providerParams struct {
	fx.In
	Lc        fx.Lifecycle
	Log       *zap.Logger
	Cfg       otelconfig.Config
	AppCfg    appconfig.AppConfig
	Readiness health.ComponentManager
}

// NewTracingModule returns fx.Option for tracing.
// If tracing is disabled, it provides a nil TracerProvider and empty middlewares.
//
// Note: HTTP request tracing is handled by ogen-generated servers automatically.
// This module only provides the TracerProvider and logger middleware for trace context propagation.
func NewTracingModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(p providerParams) (trace.TracerProvider, error) {
				if !p.Cfg.Tracing.Enabled {
					p.Log.Info("tracing: disabled")
					return nil, nil
				}
				return provideTracerProvider(p)
			},
			fx.Annotate(
				func(log *zap.Logger, tp trace.TracerProvider) appmiddleware.Middleware {
					if tp == nil {
						return appmiddleware.Middleware{}
					}
					return provideLoggerMiddleware(log)
				},
				fx.ResultTags(`group:"ogen_mw"`),
			),
		),
		fx.Invoke(func(trace.TracerProvider) {}),
	)
}

func provideTracerProvider(p providerParams) (trace.TracerProvider, error) {
	tp, err := newTracerProvider(context.Background(), p.Log, p.Cfg, p.AppCfg)
	if err != nil {
		return nil, err
	}

	p.Readiness.AddComponent(otelconfig.TracingComponentName)

	p.Lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			p.Log.Info("tracing initialized", zap.String("endpoint", p.Cfg.OtelCollectorEndpoint))
			p.Readiness.MarkReady(otelconfig.TracingComponentName)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, otelconfig.DefaultShutdownTimeout)
			defer cancel()
			return tp.Shutdown(shutdownCtx)
		},
	})

	return tp, nil
}

func provideLoggerMiddleware(log *zap.Logger) appmiddleware.Middleware {
	return appmiddleware.Middleware{
		Priority: 15,
		Handler:  loggerHandler(log),
	}
}

func loggerHandler(log *zap.Logger) middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		traceID, spanID := GetTraceIDAndSpanID(req.Context)
		if traceID != "" {
			// Create a new logger instance with trace fields for this request only
			reqLog := log.With(zap.String("trace_id", traceID), zap.String("span_id", spanID))
			req.SetContext(logger.With(req.Context, reqLog))
		}
		return next(req)
	}
}
