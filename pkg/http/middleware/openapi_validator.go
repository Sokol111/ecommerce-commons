package middleware

import (
	"errors"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	oapiMiddleware "github.com/oapi-codegen/gin-middleware"
	"go.uber.org/fx"
)

// openAPIErrorHandler handles OpenAPI validation errors.
func openAPIErrorHandler(c *gin.Context, message string, statusCode int) {
	_ = c.AbortWithError(statusCode, errors.New(message)) //nolint:errcheck // error is intentionally not handled
}

// createOpenAPIValidator creates OpenAPI request validator with custom options.
func createOpenAPIValidator(swagger *openapi3.T) gin.HandlerFunc {
	swagger.Servers = nil

	options := &oapiMiddleware.Options{
		ErrorHandler:          openAPIErrorHandler,
		SilenceServersWarning: true,
	}

	return oapiMiddleware.OapiRequestValidatorWithOptions(swagger, options)
}

// createOpenAPIValidatorMiddleware creates OpenAPI request validator middleware
// that skips validation for health endpoints.
func createOpenAPIValidatorMiddleware(validator gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip OpenAPI validation for health endpoints
		if strings.HasPrefix(c.Request.URL.Path, "/health/") {
			c.Next()
			return
		}

		validator(c)
	}
}

// OpenAPIValidatorModule provides OpenAPI validator middleware.
func OpenAPIValidatorModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(swagger *openapi3.T) Middleware {
				// Skip if nil
				if swagger == nil {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped in newEngine
					}
				}

				validator := createOpenAPIValidator(swagger)
				return Middleware{
					Priority: priority,
					Handler:  createOpenAPIValidatorMiddleware(validator),
				}
			},
			fx.ParamTags(`optional:"true"`),
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
