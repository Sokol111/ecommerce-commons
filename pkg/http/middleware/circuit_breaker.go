package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

// newCircuitBreaker creates a configured circuit breaker instance.
func newCircuitBreaker(
	maxRequests uint32,
	interval time.Duration,
	timeout time.Duration,
	failureThreshold uint32,
	log *zap.Logger,
) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        "HTTP-CircuitBreaker",
		MaxRequests: maxRequests,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if consecutive failures exceed threshold
			return counts.ConsecutiveFailures >= failureThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	return gobreaker.NewCircuitBreaker(settings)
}

// newCircuitBreakerMiddleware creates a circuit breaker middleware using sony/gobreaker.
func newCircuitBreakerMiddleware(cb *gobreaker.CircuitBreaker) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		})

		// Handle circuit breaker errors
		if err != nil {
			if errors.Is(err, gobreaker.ErrOpenState) {
				problem := problems.Problem{Detail: "service is temporarily unavailable due to circuit breaker"}
				_ = c.AbortWithError(http.StatusServiceUnavailable, ErrCircuitBreakerOpen).SetMeta(problem) //nolint:errcheck // error already logged via AbortWithError
			}
			// For other errors (like "server error"), request was already processed
		}
	}
}

// CircuitBreakerModule adds circuit breaker middleware to the application.
func CircuitBreakerModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				// Skip if disabled
				if !serverConfig.CircuitBreaker.Enabled {
					return Middleware{
						Priority: priority,
						Handler:  nil,
					}
				}

				log.Info("Circuit breaker middleware initialized",
					zap.Uint32("failure-threshold", serverConfig.CircuitBreaker.FailureThreshold),
					zap.Duration("timeout", serverConfig.CircuitBreaker.Timeout),
					zap.Duration("interval", serverConfig.CircuitBreaker.Interval),
					zap.Uint32("max-requests", serverConfig.CircuitBreaker.MaxRequests),
				)

				return Middleware{
					Priority: priority,
					Handler: newCircuitBreakerMiddleware(
						newCircuitBreaker(
							serverConfig.CircuitBreaker.MaxRequests,
							serverConfig.CircuitBreaker.Interval,
							serverConfig.CircuitBreaker.Timeout,
							serverConfig.CircuitBreaker.FailureThreshold,
							log,
						),
					),
				}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
