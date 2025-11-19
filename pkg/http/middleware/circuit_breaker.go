package middleware

import (
	"errors"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewCircuitBreakerMiddleware creates a circuit breaker middleware using sony/gobreaker
func NewCircuitBreakerMiddleware(serverConfig server.Config, log *zap.Logger, priority int) Middleware {
	config := serverConfig.CircuitBreaker

	// Skip if disabled
	if !config.Enabled {
		return Middleware{
			Priority: priority,
			Handler:  nil,
		}
	}

	// Configure gobreaker settings
	settings := gobreaker.Settings{
		Name:        "HTTP-CircuitBreaker",
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if consecutive failures exceed threshold
			return counts.ConsecutiveFailures >= config.FailureThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	cb := gobreaker.NewCircuitBreaker(settings)

	log.Info("Circuit breaker middleware initialized",
		zap.Uint32("failure-threshold", config.FailureThreshold),
		zap.Duration("timeout", config.Timeout),
		zap.Duration("interval", config.Interval),
		zap.Uint32("max-requests", config.MaxRequests),
	)

	return Middleware{
		Priority: priority,
		Handler: func(c *gin.Context) {
			// Allow health checks without circuit breaker
			if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
				c.Next()
				return
			}

			// Execute request through circuit breaker
			_, err := cb.Execute(func() (interface{}, error) {
				c.Next()

				// If request was aborted by previous middleware (rate limiter, etc), ignore
				if c.IsAborted() {
					return nil, nil
				}

				// Return error if response status is 5xx (actual service errors)
				if c.Writer.Status() >= 500 {
					return nil, errors.New("server error")
				}
				return nil, nil
			}) // Handle circuit breaker errors
			if err != nil {
				if err == gobreaker.ErrOpenState {
					log.Warn("Circuit breaker is open - rejecting request",
						zap.String("path", c.Request.URL.Path),
						zap.String("method", c.Request.Method),
					)

					problem := problems.New(http.StatusServiceUnavailable, "service is temporarily unavailable due to circuit breaker")
					problem.Instance = c.Request.URL.Path
					c.Error(errors.New(problem.Detail)).SetMeta(problem)
					c.Abort()
				}
				// For other errors (like "server error"), request was already processed
			}
		},
	}
}

// CircuitBreakerModule adds circuit breaker middleware to the application
func CircuitBreakerModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				return NewCircuitBreakerMiddleware(serverConfig, log, priority)
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
