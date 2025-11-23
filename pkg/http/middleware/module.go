package middleware

import (
	"go.uber.org/fx"
)

// NewGinModule provides all Gin middleware modules
// Middleware execution order (by priority, lower = earlier):
//
//	10 - Recovery         - catches panics (must be first)
//	20 - Logger           - logs all requests
//	30 - ErrorLogger      - logs errors from ALL middleware and handlers
//	40 - Problem          - converts errors to RFC 7807 (must wrap all error sources)
//	50 - Timeout          - kills hanging requests
//	60 - CircuitBreaker   - protects against cascading failures
//	70 - RateLimit        - limits requests/second
//	80 - HTTPBulkhead     - limits concurrent requests
//	90 - OpenAPIValidator - validates against schema
func NewGinModule() fx.Option {
	return fx.Options(
		RecoveryModule(10),
		LoggerModule(20),
		ErrorLoggerModule(30),
		ProblemModule(40),
		TimeoutModule(50),
		CircuitBreakerModule(60),
		RateLimitModule(70),
		HTTPBulkheadModule(80),
		OpenAPIValidatorModule(90),
		fx.Provide(provideGinAndHandler),
	)
}
