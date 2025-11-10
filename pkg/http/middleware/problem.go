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

		// Get status code, default to 500 if not set
		status := c.Writer.Status()
		if status == 0 || status == http.StatusOK {
			status = http.StatusInternalServerError
		}

		// Build Problem Details from the first error
		firstErr := c.Errors[0]
		problem := problems.Problem{
			Type:     "about:blank",
			Title:    http.StatusText(status),
			Status:   status,
			Detail:   firstErr.Error(),
			Instance: c.Request.URL.Path,
		}

		// Try to extract trace ID from context if available
		if traceID, exists := c.Get(string(logger.LoggerCtxKey)); exists {
			if tid, ok := traceID.(string); ok {
				problem.TraceID = tid
			}
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

		// If meta is already a Problem, use it
		if existingProblem, ok := firstErr.Meta.(*problems.Problem); ok {
			problem = *existingProblem
		}

		c.JSON(status, problem)
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
