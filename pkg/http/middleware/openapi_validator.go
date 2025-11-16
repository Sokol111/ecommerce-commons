package middleware

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	oapiMiddleware "github.com/oapi-codegen/gin-middleware"
	"go.uber.org/fx"
)

// createOpenAPIValidatorHandler creates OpenAPI request validator middleware
// that skips validation for health endpoints
func createOpenAPIValidatorHandler(swagger *openapi3.T) gin.HandlerFunc {
	// If swagger is not provided, return nil to skip this middleware
	if swagger == nil {
		return nil
	}

	swagger.Servers = nil
	validator := oapiMiddleware.OapiRequestValidator(swagger)

	return func(c *gin.Context) {
		// Skip OpenAPI validation for health endpoints
		if strings.HasPrefix(c.Request.URL.Path, "/health/") {
			c.Next()
			return
		}

		validator(c)
	}
}

// OpenAPIValidatorModule provides OpenAPI validator middleware
func OpenAPIValidatorModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(swagger *openapi3.T) Middleware {
				return Middleware{
					Priority: priority,
					Handler:  createOpenAPIValidatorHandler(swagger),
				}
			},
			fx.ParamTags(`optional:"true"`),
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
