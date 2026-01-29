package problems

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestErrorToStatusCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "ErrRequestTimeout returns 504 Gateway Timeout",
			err:          middleware.ErrRequestTimeout,
			expectedCode: http.StatusGatewayTimeout,
		},
		{
			name:         "ErrRateLimitExceeded returns 429 Too Many Requests",
			err:          middleware.ErrRateLimitExceeded,
			expectedCode: http.StatusTooManyRequests,
		},
		{
			name:         "ErrBulkheadFull returns 503 Service Unavailable",
			err:          middleware.ErrBulkheadFull,
			expectedCode: http.StatusServiceUnavailable,
		},
		{
			name:         "ErrPanic returns 500 Internal Server Error",
			err:          middleware.ErrPanic,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "ErrInvalidPublicKey returns 500 Internal Server Error",
			err:          token.ErrInvalidPublicKey,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "ErrInsufficientPermissions returns 403 Forbidden",
			err:          token.ErrInsufficientPermissions,
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "wrapped ErrRequestTimeout returns 504",
			err:          errors.Join(errors.New("context"), middleware.ErrRequestTimeout),
			expectedCode: http.StatusGatewayTimeout,
		},
		{
			name:         "unknown error returns default code",
			err:          errors.New("unknown error"),
			expectedCode: http.StatusInternalServerError, // ogenerrors.ErrorCode default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := errorToStatusCode(tt.err)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestNewErrorHandler(t *testing.T) {
	// Setup logger context
	testLogger := zap.NewNop()
	ctx := logger.With(context.Background(), testLogger)

	handler := NewErrorHandler()

	t.Run("returns RFC7807 Problem response", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/products", nil)
		r = r.WithContext(ctx)

		testErr := errors.New("something went wrong")
		handler(ctx, w, r, testErr)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "about:blank", problem.Type)
		assert.Equal(t, "Internal Server Error", problem.Title)
		assert.Equal(t, http.StatusInternalServerError, problem.Status)
		assert.Equal(t, "something went wrong", problem.Detail)
		assert.Equal(t, "/api/v1/products", problem.Instance)
	})

	t.Run("handles ErrRequestTimeout", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/orders", nil)
		r = r.WithContext(ctx)

		handler(ctx, w, r, middleware.ErrRequestTimeout)

		assert.Equal(t, http.StatusGatewayTimeout, w.Code)

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "Gateway Timeout", problem.Title)
		assert.Equal(t, http.StatusGatewayTimeout, problem.Status)
	})

	t.Run("handles ErrRateLimitExceeded", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/search", nil)
		r = r.WithContext(ctx)

		handler(ctx, w, r, middleware.ErrRateLimitExceeded)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "Too Many Requests", problem.Title)
		assert.Equal(t, http.StatusTooManyRequests, problem.Status)
	})

	t.Run("handles ErrBulkheadFull", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/heavy", nil)
		r = r.WithContext(ctx)

		handler(ctx, w, r, middleware.ErrBulkheadFull)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "Service Unavailable", problem.Title)
	})

	t.Run("handles ErrInsufficientPermissions", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/api/v1/admin/users/123", nil)
		r = r.WithContext(ctx)

		handler(ctx, w, r, token.ErrInsufficientPermissions)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "Forbidden", problem.Title)
		assert.Equal(t, http.StatusForbidden, problem.Status)
	})

	t.Run("includes instance path from request", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/products/123/reviews", nil)
		r = r.WithContext(ctx)

		handler(ctx, w, r, errors.New("not found"))

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "/api/v1/products/123/reviews", problem.Instance)
	})

	t.Run("handles panic error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/crash", nil)
		r = r.WithContext(ctx)

		handler(ctx, w, r, middleware.ErrPanic)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var problem Problem
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "Internal Server Error", problem.Title)
	})
}

func TestProblem_JSONSerialization(t *testing.T) {
	t.Run("serializes all fields", func(t *testing.T) {
		problem := Problem{
			Type:     "https://example.com/errors/validation",
			Title:    "Validation Error",
			Status:   400,
			Detail:   "Invalid input data",
			Instance: "/api/v1/users",
			TraceID:  "abc123",
			Errors: []FieldError{
				{Field: "email", Message: "invalid format"},
				{Field: "name", Message: "required"},
			},
		}

		data, err := json.Marshal(problem)
		require.NoError(t, err)

		var decoded Problem
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, problem, decoded)
	})

	t.Run("omits empty fields", func(t *testing.T) {
		problem := Problem{
			Title:  "Not Found",
			Status: 404,
		}

		data, err := json.Marshal(problem)
		require.NoError(t, err)

		// Check that omitempty works
		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.NotContains(t, raw, "type")
		assert.NotContains(t, raw, "detail")
		assert.NotContains(t, raw, "instance")
		assert.NotContains(t, raw, "traceId")
		assert.NotContains(t, raw, "errors")
		assert.Contains(t, raw, "title")
		assert.Contains(t, raw, "status")
	})
}

func TestFieldError_JSONSerialization(t *testing.T) {
	t.Run("serializes field error", func(t *testing.T) {
		fieldErr := FieldError{
			Field:   "password",
			Message: "must be at least 8 characters",
		}

		data, err := json.Marshal(fieldErr)
		require.NoError(t, err)

		var decoded FieldError
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, fieldErr, decoded)
	})

	t.Run("omits empty field name", func(t *testing.T) {
		fieldErr := FieldError{
			Message: "general validation error",
		}

		data, err := json.Marshal(fieldErr)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.NotContains(t, raw, "field")
		assert.Contains(t, raw, "message")
	})
}
