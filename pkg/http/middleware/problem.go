package middleware

import (
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// problemMiddleware converts Gin errors to Problem Details (RFC 7807) responses
func problemMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only handle if there are errors and response hasn't been written yet
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		// Build Problem Details from the first error
		firstErr := c.Errors[0]
		var problem problems.Problem

		// If meta is already a Problem, use it directly
		if existingProblem, ok := firstErr.Meta.(*problems.Problem); ok {
			problem = *existingProblem
			// Ensure status is set from Problem
			if problem.Status == 0 {
				problem.Status = http.StatusInternalServerError
			}
		} else {
			// Get status code, default to 500 if not set
			status := c.Writer.Status()
			if status == 0 || status == http.StatusOK {
				status = http.StatusInternalServerError
			}

			// Build default Problem Details
			problem = problems.Problem{
				Type:     "about:blank",
				Title:    http.StatusText(status),
				Status:   status,
				Detail:   firstErr.Error(),
				Instance: c.Request.URL.Path,
			}

			// If meta contains field errors, add them
			if meta, ok := firstErr.Meta.(map[string]string); ok {
				for field, msg := range meta {
					problem.Errors = append(problem.Errors, problems.FieldError{
						Field:   field,
						Message: msg,
					})
				}
			}
		}

		// Try to extract trace ID from context if available (if not already set)
		if problem.TraceID == "" {
			if traceID, exists := c.Get(string(logger.LoggerCtxKey)); exists {
				if tid, ok := traceID.(string); ok {
					problem.TraceID = tid
				}
			}
		}

		c.JSON(problem.Status, problem)
	}
}

// ProblemModule provides problem details middleware
func ProblemModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: priority, Handler: problemMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
