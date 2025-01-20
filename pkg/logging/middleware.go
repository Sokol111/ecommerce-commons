package handler

import (
	"log/slog"
)

func ErrorLoggerMiddleware(c LoggingContext) {
	c.Next()
	
	for _, err := range c.Errors() {
		slog.Error(err.Error(), slog.Group("request", "method", c.RequestMethod(), "path", c.RequestPath()))
	}
}
