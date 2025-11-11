package problems

import "net/http"

// Problem represents RFC7807 Problem Details for HTTP APIs
type Problem struct {
	Type     string       `json:"type,omitempty"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	TraceID  string       `json:"traceId,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// New creates a new Problem with the given status and detail
func New(status int, detail string) *Problem {
	return &Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
}

// Common HTTP error constructors

func TooManyRequests(detail string) *Problem {
	return New(http.StatusTooManyRequests, detail)
}

func ServiceUnavailable(detail string) *Problem {
	return New(http.StatusServiceUnavailable, detail)
}

func GatewayTimeout(detail string) *Problem {
	return New(http.StatusGatewayTimeout, detail)
}

func BadRequest(detail string) *Problem {
	return New(http.StatusBadRequest, detail)
}

func NotFound(detail string) *Problem {
	return New(http.StatusNotFound, detail)
}

func Unauthorized(detail string) *Problem {
	return New(http.StatusUnauthorized, detail)
}

func Forbidden(detail string) *Problem {
	return New(http.StatusForbidden, detail)
}

func Conflict(detail string) *Problem {
	return New(http.StatusConflict, detail)
}

func InternalServerError(detail string) *Problem {
	return New(http.StatusInternalServerError, detail)
}
