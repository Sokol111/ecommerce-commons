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
func NewRateLimitMiddleware(serverConfig server.Config, priority int) Middleware {
	config := serverConfig.RateLimit

	// Skip if disabled
	if !config.Enabled {
		return Middleware{
			Priority: priority,
			Handler:  nil, // Will be skipped in newEngine
		}
	}

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)

	return Middleware{
		Priority: priority,
		Handler: func(c *gin.Context) {
			// Allow health checks without rate limiting
			if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
				c.Next()
				return
			}

			// Check rate limit
			if !limiter.Allow() {
				problem := problems.New(http.StatusTooManyRequests, "rate limit exceeded, please try again later")
				problem.Instance = c.Request.URL.Path
				c.Error(errors.New(problem.Detail)).SetMeta(problem)
				c.Abort()
				return
			}

			c.Next()
		},
	}
}

// RateLimitModule adds rate limiting middleware to the application
func RateLimitModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config) Middleware {
				return NewRateLimitMiddleware(serverConfig, priority)
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
