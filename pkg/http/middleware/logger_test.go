package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/ogen-go/ogen/middleware"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggerMiddleware(t *testing.T) {
	// Create a test logger and context
	testLogger := zap.NewNop()
	ctx := logger.With(context.Background(), testLogger)

	t.Run("passes request through and returns response", func(t *testing.T) {
		mw := loggerMiddleware()

		nextCalled := false
		expectedResp := middleware.Response{Type: "test-response"}
		next := func(_ middleware.Request) (middleware.Response, error) {
			nextCalled = true
			return expectedResp, nil
		}

		req := middleware.Request{
			Context:       ctx,
			Raw:           httptest.NewRequest("GET", "/test?foo=bar", nil),
			OperationName: "TestOperation",
		}

		resp, err := mw(req, next)

		assert.NoError(t, err)
		assert.True(t, nextCalled)
		assert.Equal(t, expectedResp.Type, resp.Type)
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		mw := loggerMiddleware()

		expectedErr := assert.AnError
		next := func(_ middleware.Request) (middleware.Response, error) {
			return middleware.Response{}, expectedErr
		}

		req := middleware.Request{
			Context:       ctx,
			Raw:           httptest.NewRequest("POST", "/api/v1/resource", nil),
			OperationName: "CreateResource",
		}

		_, err := mw(req, next)

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("handles request with user agent", func(t *testing.T) {
		mw := loggerMiddleware()

		next := func(_ middleware.Request) (middleware.Response, error) {
			return middleware.Response{}, nil
		}

		httpReq := httptest.NewRequest("GET", "/test", nil)
		httpReq.Header.Set("User-Agent", "TestAgent/1.0")

		req := middleware.Request{
			Context:       ctx,
			Raw:           httpReq,
			OperationName: "TestOp",
		}

		_, err := mw(req, next)

		assert.NoError(t, err)
	})
}

func TestRequestFields(t *testing.T) {
	t.Run("extracts fields from request", func(t *testing.T) {
		httpReq := httptest.NewRequest("POST", "/api/v1/users?page=1&size=10", nil)

		fields := requestFields(httpReq)

		assert.Len(t, fields, 3)

		// Check field values
		fieldMap := make(map[string]interface{})
		for _, f := range fields {
			fieldMap[f.Key] = f.String
		}

		assert.Equal(t, "POST", fieldMap["method"])
		assert.Equal(t, "/api/v1/users", fieldMap["path"])
		assert.Equal(t, "page=1&size=10", fieldMap["query"])
	})

	t.Run("handles empty query string", func(t *testing.T) {
		httpReq := httptest.NewRequest("GET", "/health", nil)

		fields := requestFields(httpReq)

		fieldMap := make(map[string]interface{})
		for _, f := range fields {
			fieldMap[f.Key] = f.String
		}

		assert.Equal(t, "GET", fieldMap["method"])
		assert.Equal(t, "/health", fieldMap["path"])
		assert.Equal(t, "", fieldMap["query"])
	})
}
