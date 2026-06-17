package tenant

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/security/validation"
	"go.uber.org/zap"
)

// TenantSlugHeader is the HTTP header used to propagate tenant slug between services.
const TenantSlugHeader = "X-Tenant-Slug"

// NewResolverInterceptor creates a Connect-RPC interceptor that resolves tenant
// slug from the X-Tenant-Slug header and stores it in context.
// It also enriches the request-scoped logger with the tenant field so that
// subsequent interceptors (e.g. logger) include it in every log line.
func NewResolverInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			slug := req.Header().Get(TenantSlugHeader)

			if slug == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tenant not found in request header"))
			}

			ctx = ContextWithSlug(ctx, slug)

			reqLog := logger.Get(ctx).With(zap.String("tenant", slug))
			ctx = logger.With(ctx, reqLog)

			return next(ctx, req)
		}
	}
}

// NewValidatorInterceptor creates a Connect-RPC interceptor that validates
// tenant-scoped tokens have a tenant claim matching the request tenant.
// Service accounts and platform admins are not tenant-scoped and can access any tenant.
func NewValidatorInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if claims := validation.ClaimsFromContext(ctx); claims != nil && claims.IsTenantScoped() {
				slug := MustSlugFromContext(ctx)
				if claims.Tenant != slug {
					return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("token tenant %q does not match request tenant %q", claims.Tenant, slug))
				}
			}

			return next(ctx, req)
		}
	}
}
