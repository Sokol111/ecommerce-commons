package tenant

import (
	"errors"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	httpmw "github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TenantSlugHeader is the HTTP header used to propagate tenant slug between services.
const TenantSlugHeader = "X-Tenant-Slug"

// ErrTenantNotFound is returned when tenant slug is not present in the request.
var ErrTenantNotFound = errors.New("tenant not found")

// Middleware resolves tenant slug from the X-Tenant-Slug header and stores it in context.
func Middleware(log *zap.Logger) middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		slug := req.Raw.Header.Get(TenantSlugHeader)

		if slug == "" {
			return middleware.Response{}, ErrTenantNotFound
		}

		req.Context = ContextWithSlug(req.Context, slug)

		reqLog := logger.Get(req.Context).With(zap.String("tenant", slug))
		req.Context = logger.With(req.Context, reqLog)

		return next(req)
	}
}

// MiddlewareModule provides tenant resolution as an ogen middleware.
// Priority 25: after Recovery(10) and Logger(20), before Timeout(30).
func MiddlewareModule() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(log *zap.Logger) httpmw.Middleware {
				return httpmw.Middleware{
					Priority: 25,
					Handler:  Middleware(log),
				}
			},
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
