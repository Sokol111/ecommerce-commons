package tenant

import (
	"errors"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	httpmw "github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrTenantNotFound is returned when tenant slug is not present in the request.
var ErrTenantNotFound = errors.New("tenant not found")

// ErrInvalidTenant is returned when tenant slug is invalid.
var ErrInvalidTenant = errors.New("invalid tenant")

// Middleware resolves tenant slug from the request and stores it in context.
func Middleware(resolver Resolver, log *zap.Logger) middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		slug, err := resolver.Resolve(req.Raw)
		if err != nil {
			log.Error("failed to resolve tenant", zap.Error(err))
			return middleware.Response{}, ErrInvalidTenant
		}

		if slug == "" {
			return middleware.Response{}, ErrTenantNotFound
		}

		req.Context = ContextWithSlug(req.Context, slug)

		reqLog := logger.Get(req.Context).With(zap.String("tenant", slug))
		req.Context = logger.With(req.Context, reqLog)

		return next(req)
	}
}

// MiddlewareModule provides tenant resolution as an ogen middleware with the given priority.
func MiddlewareModule(priority int, resolver Resolver) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(log *zap.Logger) httpmw.Middleware {
				return httpmw.Middleware{
					Priority: priority,
					Handler:  Middleware(resolver, log),
				}
			},
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
