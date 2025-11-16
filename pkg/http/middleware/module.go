package middleware

import (
	"go.uber.org/fx"
)

// NewGinModule provides all Gin middleware modules
// Middleware execution order (by priority, lower = earlier):
//
//	10 - Recovery         - catches panics (must be first)
//	20 - Logger           - logs all requests
//	30 - Timeout          - kills hanging requests
//	35 - CircuitBreaker   - protects against cascading failures
//	40 - RateLimit        - limits requests/second
//	50 - HTTPBulkhead     - limits concurrent requests
//	60 - OpenAPIValidator - validates against schema
//	70 - ErrorLogger      - logs errors from handlers
//	80 - Problem          - converts errors to RFC 7807 (must be last)
func NewGinModule() fx.Option {
	return fx.Options(
		RecoveryModule(10),
		LoggerModule(20),
		TimeoutModule(30),
		CircuitBreakerModule(35),
		RateLimitModule(40),
		HTTPBulkheadModule(50),
		OpenAPIValidatorModule(60),
		ErrorLoggerModule(70),
		ProblemModule(80),
		fx.Provide(provideGinAndHandler),
	)
}
