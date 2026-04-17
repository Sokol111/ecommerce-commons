package tenant

import (
	"errors"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	httpmw "github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TenantSlugHeader is the HTTP header used to propagate tenant slug between services.
const TenantSlugHeader = "X-Tenant-Slug"

// ErrTenantNotFound is returned when tenant slug is not present in the request.
var ErrTenantNotFound = errors.New("tenant not found")

// Middleware resolves tenant slug from the X-Tenant-Slug header and stores it in context.
// It also validates that user tokens have a tenant claim matching the request tenant.
func Middleware(log *zap.Logger) middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		slug := req.Raw.Header.Get(TenantSlugHeader)

		if slug == "" {
			return middleware.Response{}, ErrTenantNotFound
		}

		req.Context = ContextWithSlug(req.Context, slug)

		// Validate tenant claim for tenant-scoped tokens.
		// Security handler runs before middleware, so claims are already in context.
		// Tenant-scoped users must have a tenant claim matching the request tenant.
		// Service accounts and platform admins are not tenant-scoped and can access any tenant.
		if claims := token.ClaimsFromContext(req.Context); claims != nil && claims.IsTenantScoped() {
			if claims.Tenant != slug {
				return middleware.Response{}, token.ErrTenantMismatch
			}
		}

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
