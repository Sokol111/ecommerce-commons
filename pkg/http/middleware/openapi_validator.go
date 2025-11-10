package middleware

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	oapiMiddleware "github.com/oapi-codegen/gin-middleware"
	"go.uber.org/fx"
)

// createOpenAPIValidatorHandler creates OpenAPI request validator middleware
func createOpenAPIValidatorHandler(swagger *openapi3.T) gin.HandlerFunc {
	swagger.Servers = nil
	return oapiMiddleware.OapiRequestValidator(swagger)
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
