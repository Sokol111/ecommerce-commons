package middleware

import (
	"errors"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

// NewRateLimitMiddleware creates a rate limiting middleware
func NewRateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow health checks without rate limiting
		if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
			c.Next()
			return
		}

		// Check rate limit
		if !limiter.Allow() {
			problem := problems.Problem{Detail: "rate limit exceeded, please try again later"}
			c.AbortWithError(http.StatusTooManyRequests, errors.New("rate limit exceeded")).SetMeta(problem)
			return
		}

		c.Next()
	}
}

// RateLimitModule adds rate limiting middleware to the application
func RateLimitModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(config server.Config) Middleware {
				if !config.RateLimit.Enabled {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped in newEngine
					}
				}
				limiter := rate.NewLimiter(rate.Limit(config.RateLimit.RequestsPerSecond), config.RateLimit.Burst)
				return Middleware{
					Priority: priority,
					Handler:  NewRateLimitMiddleware(limiter),
				}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
