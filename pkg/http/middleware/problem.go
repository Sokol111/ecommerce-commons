package middleware

import (
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// problemMiddleware converts Gin errors to Problem Details (RFC 7807) responses.
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
		} else if existingProblem, ok := firstErr.Meta.(problems.Problem); ok {
			problem = existingProblem
		} else {
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

		if problem.Status == 0 {
			problem.Status = c.Writer.Status()
			if problem.Status == 0 {
				problem.Status = http.StatusInternalServerError
			}
		}
		if problem.Instance == "" {
			problem.Instance = c.Request.URL.Path
		}
		if problem.Title == "" {
			problem.Title = http.StatusText(problem.Status)
		}
		if problem.Type == "" {
			problem.Type = "about:blank"
		}
		if problem.Detail == "" {
			problem.Detail = http.StatusText(problem.Status)
		}
		// Try to extract trace ID from OpenTelemetry context if available (if not already set)
		if problem.TraceID == "" {
			sc := trace.SpanContextFromContext(c.Request.Context())
			if sc.IsValid() {
				problem.TraceID = sc.TraceID().String()
			}
		}

		c.JSON(problem.Status, problem)
	}
}

// ProblemModule provides problem details middleware.
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
