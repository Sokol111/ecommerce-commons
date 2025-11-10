package middleware

import (
	"go.uber.org/fx"
)

// NewGinModule provides all Gin middleware modules
// Middleware execution order (by priority, lower = earlier):
//
//	10 - Timeout          - kills hanging requests
//	20 - RateLimit        - limits requests/second
//	30 - HTTPBulkhead     - limits concurrent requests
//	40 - Recovery         - catches panics
//	50 - Logger           - logs requests
//	60 - OpenAPIValidator - validates against schema
//	70 - ErrorLogger      - logs errors from handlers
//	80 - Problem          - converts errors to RFC 7807
func NewGinModule() fx.Option {
	return fx.Options(
		TimeoutModule(10),
		RateLimitModule(20),
		HTTPBulkheadModule(30),
		RecoveryModule(40),
		LoggerModule(50),
		OpenAPIValidatorModule(60),
		ErrorLoggerModule(70),
		ProblemModule(80),
		fx.Provide(provideGinAndHandler),
	)
}
