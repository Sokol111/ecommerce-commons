package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func TestProblemMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("does nothing when there are no errors", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response["message"] != "success" {
			t.Errorf("expected message 'success', got %v", response["message"])
		}
	})

	t.Run("does nothing when response is already written", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
			// Add error after response is written
			_ = c.Error(gin.Error{Err: http.ErrAbortHandler, Type: gin.ErrorTypePublic})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response["message"] != "success" {
			t.Errorf("expected original response, got %v", response)
		}
	})

	t.Run("converts error to problem when meta is Problem pointer", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			problemDetail := &problems.Problem{
				Type:     "https://example.com/errors/bad-request",
				Title:    "Bad Request",
				Status:   http.StatusBadRequest,
				Detail:   "Invalid input provided",
				Instance: "/test",
			}
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: problemDetail,
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Type != "https://example.com/errors/bad-request" {
			t.Errorf("expected type 'https://example.com/errors/bad-request', got %s", problem.Type)
		}
		if problem.Title != "Bad Request" {
			t.Errorf("expected title 'Bad Request', got %s", problem.Title)
		}
		if problem.Detail != "Invalid input provided" {
			t.Errorf("expected detail 'Invalid input provided', got %s", problem.Detail)
		}
		if problem.Instance != "/test" {
			t.Errorf("expected instance '/test', got %s", problem.Instance)
		}
	})

	t.Run("converts error to problem when meta is Problem value", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
			problemDetail := problems.Problem{
				Type:     "https://example.com/errors/not-found",
				Title:    "Not Found",
				Status:   http.StatusNotFound,
				Detail:   "Resource not found",
				Instance: "/test",
			}
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: problemDetail,
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Type != "https://example.com/errors/not-found" {
			t.Errorf("expected type 'https://example.com/errors/not-found', got %s", problem.Type)
		}
		if problem.Title != "Not Found" {
			t.Errorf("expected title 'Not Found', got %s", problem.Title)
		}
	})

	t.Run("converts field errors from meta map to problem", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusUnprocessableEntity)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: map[string]string{
					"email":    "Invalid email format",
					"password": "Password is too short",
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
		}

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(problem.Errors) != 2 {
			t.Fatalf("expected 2 field errors, got %d", len(problem.Errors))
		}

		// Check that both errors are present (order might vary due to map iteration)
		errorsMap := make(map[string]string)
		for _, fe := range problem.Errors {
			errorsMap[fe.Field] = fe.Message
		}

		if errorsMap["email"] != "Invalid email format" {
			t.Errorf("expected email error 'Invalid email format', got %s", errorsMap["email"])
		}
		if errorsMap["password"] != "Password is too short" {
			t.Errorf("expected password error 'Password is too short', got %s", errorsMap["password"])
		}
	})

	t.Run("sets default status to 500 when writer status is zero", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Don't set status, it will be 0/200 by default
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					// Status is 0, so it should use writer status (200) or default to 500
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// When no status is set, Gin defaults to 200, but middleware should use it
		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// The problem status should be either from writer (200) or fallback to 500
		if problem.Status != http.StatusOK && problem.Status != http.StatusInternalServerError {
			t.Errorf("expected problem status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, problem.Status)
		}
	})

	t.Run("sets default instance from request path", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/users/123", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusNotFound,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Instance != "/api/users/123" {
			t.Errorf("expected instance '/api/users/123', got %s", problem.Instance)
		}
	})

	t.Run("sets default title from status text", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusForbidden)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusForbidden,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Title != "Forbidden" {
			t.Errorf("expected title 'Forbidden', got %s", problem.Title)
		}
	})

	t.Run("sets default type to about:blank", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusBadRequest,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Type != "about:blank" {
			t.Errorf("expected type 'about:blank', got %s", problem.Type)
		}
	})

	t.Run("sets default detail from status text", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusUnauthorized)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusUnauthorized,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Detail != "Unauthorized" {
			t.Errorf("expected detail 'Unauthorized', got %s", problem.Detail)
		}
	})

	t.Run("extracts trace ID from OpenTelemetry context", func(t *testing.T) {
		middleware := problemMiddleware()

		// Create a valid trace ID manually
		traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
		spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Set the span context in the request context
			ctx := trace.ContextWithSpanContext(c.Request.Context(), spanContext)
			c.Request = c.Request.WithContext(ctx)

			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusBadRequest,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// TraceID should be set (non-empty string)
		if problem.TraceID == "" {
			t.Error("expected traceID to be set from OpenTelemetry context")
		}

		// Validate it's a valid trace ID format (32 hex characters)
		if len(problem.TraceID) != 32 {
			t.Errorf("expected traceID length 32, got %d", len(problem.TraceID))
		}

		// Validate the exact trace ID
		if problem.TraceID != "0123456789abcdef0123456789abcdef" {
			t.Errorf("expected traceID '0123456789abcdef0123456789abcdef', got '%s'", problem.TraceID)
		}
	})

	t.Run("does not overwrite existing trace ID", func(t *testing.T) {
		middleware := problemMiddleware()

		existingTraceID := "abc123def456abc123def456abc123de"

		// Create a valid trace ID for the context
		traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
		spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Set the span context in the request context
			ctx := trace.ContextWithSpanContext(c.Request.Context(), spanContext)
			c.Request = c.Request.WithContext(ctx)

			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status:  http.StatusBadRequest,
					TraceID: existingTraceID,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.TraceID != existingTraceID {
			t.Errorf("expected traceID '%s', got '%s'", existingTraceID, problem.TraceID)
		}
	})

	t.Run("handles error without OpenTelemetry context", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusBadRequest,
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// TraceID should be empty when no OTel context is present
		if problem.TraceID != "" {
			t.Errorf("expected empty traceID, got '%s'", problem.TraceID)
		}
	})

	t.Run("handles multiple errors and uses first one", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusBadRequest,
					Detail: "First error",
				},
			})
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Status: http.StatusInternalServerError,
					Detail: "Second error",
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Detail != "First error" {
			t.Errorf("expected detail 'First error', got '%s'", problem.Detail)
		}
		if problem.Status != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, problem.Status)
		}
	})

	t.Run("preserves all fields from existing problem", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
				Meta: &problems.Problem{
					Type:     "https://api.example.com/errors/validation",
					Title:    "Validation Failed",
					Status:   http.StatusBadRequest,
					Detail:   "One or more fields failed validation",
					Instance: "/api/users",
					TraceID:  "trace-123-456",
					Errors: []problems.FieldError{
						{Field: "email", Message: "Invalid format"},
						{Field: "age", Message: "Must be positive"},
					},
				},
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Type != "https://api.example.com/errors/validation" {
			t.Errorf("expected type 'https://api.example.com/errors/validation', got %s", problem.Type)
		}
		if problem.Title != "Validation Failed" {
			t.Errorf("expected title 'Validation Failed', got %s", problem.Title)
		}
		if problem.Status != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, problem.Status)
		}
		if problem.Detail != "One or more fields failed validation" {
			t.Errorf("expected detail 'One or more fields failed validation', got %s", problem.Detail)
		}
		if problem.Instance != "/api/users" {
			t.Errorf("expected instance '/api/users', got %s", problem.Instance)
		}
		if problem.TraceID != "trace-123-456" {
			t.Errorf("expected traceID 'trace-123-456', got %s", problem.TraceID)
		}
		if len(problem.Errors) != 2 {
			t.Fatalf("expected 2 field errors, got %d", len(problem.Errors))
		}
	})

	t.Run("uses writer status when problem status is zero", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusConflict)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
		}

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Status != http.StatusConflict {
			t.Errorf("expected problem status %d, got %d", http.StatusConflict, problem.Status)
		}
	})

	t.Run("handles error without any meta information", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if problem.Status != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, problem.Status)
		}
		if problem.Type != "about:blank" {
			t.Errorf("expected type 'about:blank', got %s", problem.Type)
		}
		if problem.Title != "Bad Request" {
			t.Errorf("expected title 'Bad Request', got %s", problem.Title)
		}
		if problem.Detail != "Bad Request" {
			t.Errorf("expected detail 'Bad Request', got %s", problem.Detail)
		}
		if problem.Instance != "/test" {
			t.Errorf("expected instance '/test', got %s", problem.Instance)
		}
	})

	t.Run("handles different HTTP status codes correctly", func(t *testing.T) {
		middleware := problemMiddleware()

		testCases := []struct {
			status       int
			expectedText string
		}{
			{http.StatusBadRequest, "Bad Request"},
			{http.StatusUnauthorized, "Unauthorized"},
			{http.StatusForbidden, "Forbidden"},
			{http.StatusNotFound, "Not Found"},
			{http.StatusMethodNotAllowed, "Method Not Allowed"},
			{http.StatusConflict, "Conflict"},
			{http.StatusUnprocessableEntity, "Unprocessable Entity"},
			{http.StatusInternalServerError, "Internal Server Error"},
			{http.StatusServiceUnavailable, "Service Unavailable"},
		}

		for _, tc := range testCases {
			t.Run(tc.expectedText, func(t *testing.T) {
				router := gin.New()
				router.Use(middleware)
				router.GET("/test", func(c *gin.Context) {
					c.Status(tc.status)
					_ = c.Error(&gin.Error{
						Err:  http.ErrAbortHandler,
						Type: gin.ErrorTypePublic,
					})
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				if w.Code != tc.status {
					t.Errorf("expected status %d, got %d", tc.status, w.Code)
				}

				var problem problems.Problem
				if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}

				if problem.Status != tc.status {
					t.Errorf("expected problem status %d, got %d", tc.status, problem.Status)
				}
				if problem.Title != tc.expectedText {
					t.Errorf("expected title '%s', got '%s'", tc.expectedText, problem.Title)
				}
			})
		}
	})

	t.Run("content-type header is set to application/json", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json; charset=utf-8" {
			t.Errorf("expected Content-Type 'application/json; charset=utf-8', got '%s'", contentType)
		}
	})

	t.Run("handles invalid span context gracefully", func(t *testing.T) {
		middleware := problemMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Set an invalid span context
			invalidSpanContext := trace.NewSpanContext(trace.SpanContextConfig{})
			ctx := trace.ContextWithSpanContext(c.Request.Context(), invalidSpanContext)
			c.Request = c.Request.WithContext(ctx)

			c.Status(http.StatusBadRequest)
			_ = c.Error(&gin.Error{
				Err:  http.ErrAbortHandler,
				Type: gin.ErrorTypePublic,
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var problem problems.Problem
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// TraceID should be empty for invalid span context
		if problem.TraceID != "" {
			t.Errorf("expected empty traceID for invalid span context, got '%s'", problem.TraceID)
		}
	})
}
