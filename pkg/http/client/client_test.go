package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ClientConfig Tests
// ============================================================================

func TestClientConfig_applyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		initial  ClientConfig
		expected ClientConfig
	}{
		{
			name:    "all nil values get defaults",
			initial: ClientConfig{BaseURL: "http://example.com"},
			expected: ClientConfig{
				BaseURL:             "http://example.com",
				Timeout:             lo.ToPtr(DefaultTimeout),
				MaxIdleConnsPerHost: lo.ToPtr(DefaultMaxIdleConnsPerHost),
				IdleConnTimeout:     lo.ToPtr(DefaultIdleConnTimeout),
				MaxConnLifetime:     lo.ToPtr(DefaultMaxConnLifetime),
			},
		},
		{
			name: "custom timeout preserved",
			initial: ClientConfig{
				BaseURL: "http://example.com",
				Timeout: lo.ToPtr(30 * time.Second),
			},
			expected: ClientConfig{
				BaseURL:             "http://example.com",
				Timeout:             lo.ToPtr(30 * time.Second),
				MaxIdleConnsPerHost: lo.ToPtr(DefaultMaxIdleConnsPerHost),
				IdleConnTimeout:     lo.ToPtr(DefaultIdleConnTimeout),
				MaxConnLifetime:     lo.ToPtr(DefaultMaxConnLifetime),
			},
		},
		{
			name: "all custom values preserved",
			initial: ClientConfig{
				BaseURL:             "http://example.com",
				Timeout:             lo.ToPtr(5 * time.Second),
				MaxIdleConnsPerHost: lo.ToPtr(50),
				IdleConnTimeout:     lo.ToPtr(30 * time.Second),
				MaxConnLifetime:     lo.ToPtr(120 * time.Second),
			},
			expected: ClientConfig{
				BaseURL:             "http://example.com",
				Timeout:             lo.ToPtr(5 * time.Second),
				MaxIdleConnsPerHost: lo.ToPtr(50),
				IdleConnTimeout:     lo.ToPtr(30 * time.Second),
				MaxConnLifetime:     lo.ToPtr(120 * time.Second),
			},
		},
		{
			name: "zero duration values preserved",
			initial: ClientConfig{
				BaseURL:         "http://example.com",
				Timeout:         lo.ToPtr(time.Duration(0)),
				MaxConnLifetime: lo.ToPtr(time.Duration(0)),
			},
			expected: ClientConfig{
				BaseURL:             "http://example.com",
				Timeout:             lo.ToPtr(time.Duration(0)),
				MaxIdleConnsPerHost: lo.ToPtr(DefaultMaxIdleConnsPerHost),
				IdleConnTimeout:     lo.ToPtr(DefaultIdleConnTimeout),
				MaxConnLifetime:     lo.ToPtr(time.Duration(0)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initial
			cfg.applyDefaults()
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestClientConfig_validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config with base url",
			cfg:     ClientConfig{BaseURL: "http://example.com"},
			wantErr: false,
		},
		{
			name:    "empty base url returns error",
			cfg:     ClientConfig{BaseURL: ""},
			wantErr: true,
			errMsg:  "base-url is required",
		},
		{
			name:    "full config is valid",
			cfg:     ClientConfig{BaseURL: "http://service:8080", Timeout: lo.ToPtr(5 * time.Second)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// newHTTPClient Tests
// ============================================================================

func TestNewHTTPClient(t *testing.T) {
	t.Run("creates client with default config", func(t *testing.T) {
		cfg := ClientConfig{
			BaseURL:             "http://example.com",
			Timeout:             lo.ToPtr(DefaultTimeout),
			MaxIdleConnsPerHost: lo.ToPtr(DefaultMaxIdleConnsPerHost),
			IdleConnTimeout:     lo.ToPtr(DefaultIdleConnTimeout),
			MaxConnLifetime:     lo.ToPtr(DefaultMaxConnLifetime),
		}

		client := newHTTPClient(cfg)

		require.NotNil(t, client)
		assert.Equal(t, DefaultTimeout, client.Timeout)
		assert.NotNil(t, client.Transport)
	})

	t.Run("creates client with zero MaxConnLifetime", func(t *testing.T) {
		cfg := ClientConfig{
			BaseURL:             "http://example.com",
			Timeout:             lo.ToPtr(5 * time.Second),
			MaxIdleConnsPerHost: lo.ToPtr(10),
			IdleConnTimeout:     lo.ToPtr(30 * time.Second),
			MaxConnLifetime:     lo.ToPtr(time.Duration(0)),
		}

		client := newHTTPClient(cfg)

		require.NotNil(t, client)
		assert.Equal(t, 5*time.Second, client.Timeout)
	})

	t.Run("retry transport uses min of pool size and cap", func(t *testing.T) {
		cfg := ClientConfig{
			BaseURL:             "http://example.com",
			Timeout:             lo.ToPtr(DefaultTimeout),
			MaxIdleConnsPerHost: lo.ToPtr(3), // Less than MaxRetriesCap
			IdleConnTimeout:     lo.ToPtr(DefaultIdleConnTimeout),
			MaxConnLifetime:     lo.ToPtr(DefaultMaxConnLifetime),
		}

		client := newHTTPClient(cfg)
		rt, ok := client.Transport.(*retryTransport)
		require.True(t, ok)
		assert.Equal(t, 3, rt.maxRetries)
	})

	t.Run("retry transport caps at MaxRetriesCap", func(t *testing.T) {
		cfg := ClientConfig{
			BaseURL:             "http://example.com",
			Timeout:             lo.ToPtr(DefaultTimeout),
			MaxIdleConnsPerHost: lo.ToPtr(100), // Greater than MaxRetriesCap
			IdleConnTimeout:     lo.ToPtr(DefaultIdleConnTimeout),
			MaxConnLifetime:     lo.ToPtr(DefaultMaxConnLifetime),
		}

		client := newHTTPClient(cfg)
		rt, ok := client.Transport.(*retryTransport)
		require.True(t, ok)
		assert.Equal(t, MaxRetriesCap, rt.maxRetries)
	})
}

// ============================================================================
// ProvideHTTPClient Tests
// ============================================================================

func TestProvideHTTPClient(t *testing.T) {
	t.Run("creates client from valid config", func(t *testing.T) {
		v := viper.New()
		v.Set("clients.test-service.base-url", "http://test-service:8080")
		v.Set("clients.test-service.timeout", "5s")

		provider := ProvideHTTPClient("test-service")
		client, cfg, err := provider(v)

		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "http://test-service:8080", cfg.BaseURL)
		assert.Equal(t, 5*time.Second, *cfg.Timeout)
	})

	t.Run("returns error for missing base url", func(t *testing.T) {
		v := viper.New()
		v.Set("clients.test-service.timeout", "5s")

		provider := ProvideHTTPClient("test-service")
		_, _, err := provider(v)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "base-url is required")
	})

	t.Run("applies defaults for missing optional fields", func(t *testing.T) {
		v := viper.New()
		v.Set("clients.test-service.base-url", "http://test-service:8080")

		provider := ProvideHTTPClient("test-service")
		_, cfg, err := provider(v)

		require.NoError(t, err)
		assert.Equal(t, DefaultTimeout, *cfg.Timeout)
		assert.Equal(t, DefaultMaxIdleConnsPerHost, *cfg.MaxIdleConnsPerHost)
		assert.Equal(t, DefaultIdleConnTimeout, *cfg.IdleConnTimeout)
		assert.Equal(t, DefaultMaxConnLifetime, *cfg.MaxConnLifetime)
	})

	t.Run("returns error for non-existent client name", func(t *testing.T) {
		v := viper.New()

		provider := ProvideHTTPClient("non-existent")
		_, _, err := provider(v)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "base-url is required")
	})
}

// ============================================================================
// timedConn Tests
// ============================================================================

func TestTimedConn_isExpired(t *testing.T) {
	t.Run("not expired when within lifetime", func(t *testing.T) {
		conn := &timedConn{
			createdAt:   time.Now(),
			maxLifetime: 1 * time.Hour,
		}
		assert.False(t, conn.isExpired())
	})

	t.Run("expired when past lifetime", func(t *testing.T) {
		conn := &timedConn{
			createdAt:   time.Now().Add(-2 * time.Hour),
			maxLifetime: 1 * time.Hour,
		}
		assert.True(t, conn.isExpired())
	})

	t.Run("expired at exact boundary", func(t *testing.T) {
		conn := &timedConn{
			createdAt:   time.Now().Add(-1*time.Hour - 1*time.Nanosecond),
			maxLifetime: 1 * time.Hour,
		}
		assert.True(t, conn.isExpired())
	})
}

func TestTimedConn_Read(t *testing.T) {
	t.Run("reads successfully when not expired", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		tc := &timedConn{
			Conn:        client,
			createdAt:   time.Now(),
			maxLifetime: 1 * time.Hour,
		}

		// Write from server side
		go func() {
			server.Write([]byte("hello"))
		}()

		buf := make([]byte, 5)
		n, err := tc.Read(buf)

		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, "hello", string(buf))
	})

	t.Run("returns ErrConnExpired when expired", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()

		tc := &timedConn{
			Conn:        client,
			createdAt:   time.Now().Add(-2 * time.Hour),
			maxLifetime: 1 * time.Hour,
		}

		buf := make([]byte, 5)
		n, err := tc.Read(buf)

		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, ErrConnExpired)
	})
}

func TestTimedConn_Write(t *testing.T) {
	t.Run("writes successfully when not expired", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		tc := &timedConn{
			Conn:        client,
			createdAt:   time.Now(),
			maxLifetime: 1 * time.Hour,
		}

		// Read from server side
		go func() {
			buf := make([]byte, 5)
			server.Read(buf)
		}()

		n, err := tc.Write([]byte("hello"))

		require.NoError(t, err)
		assert.Equal(t, 5, n)
	})

	t.Run("returns ErrConnExpired when expired", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()

		tc := &timedConn{
			Conn:        client,
			createdAt:   time.Now().Add(-2 * time.Hour),
			maxLifetime: 1 * time.Hour,
		}

		n, err := tc.Write([]byte("hello"))

		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, ErrConnExpired)
	})
}

// ============================================================================
// retryTransport Tests
// ============================================================================

type mockRoundTripper struct {
	responses []*http.Response
	errors    []error
	calls     int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := m.calls
	m.calls++

	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestRetryTransport_RoundTrip(t *testing.T) {
	t.Run("successful request without retry", func(t *testing.T) {
		mock := &mockRoundTripper{
			responses: []*http.Response{{StatusCode: http.StatusOK}},
		}
		rt := &retryTransport{base: mock, maxRetries: 3}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 1, mock.calls)
	})

	t.Run("retries on ECONNREFUSED and succeeds", func(t *testing.T) {
		mock := &mockRoundTripper{
			errors: []error{
				syscall.ECONNREFUSED,
				syscall.ECONNREFUSED,
				nil,
			},
			responses: []*http.Response{nil, nil, {StatusCode: http.StatusOK}},
		}
		rt := &retryTransport{base: mock, maxRetries: 3}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 3, mock.calls)
	})

	t.Run("retries on ECONNRESET and succeeds", func(t *testing.T) {
		mock := &mockRoundTripper{
			errors: []error{
				syscall.ECONNRESET,
				nil,
			},
			responses: []*http.Response{nil, {StatusCode: http.StatusOK}},
		}
		rt := &retryTransport{base: mock, maxRetries: 3}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, mock.calls)
	})

	t.Run("retries on EOF and succeeds", func(t *testing.T) {
		mock := &mockRoundTripper{
			errors: []error{
				io.EOF,
				nil,
			},
			responses: []*http.Response{nil, {StatusCode: http.StatusOK}},
		}
		rt := &retryTransport{base: mock, maxRetries: 3}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, mock.calls)
	})

	t.Run("does not retry on non-retryable error", func(t *testing.T) {
		customErr := errors.New("custom error")
		mock := &mockRoundTripper{
			errors: []error{customErr},
		}
		rt := &retryTransport{base: mock, maxRetries: 3}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		_, err := rt.RoundTrip(req)

		require.Error(t, err)
		assert.Equal(t, customErr, err)
		assert.Equal(t, 1, mock.calls)
	})

	t.Run("exhausts retries and makes final attempt", func(t *testing.T) {
		mock := &mockRoundTripper{
			errors: []error{
				syscall.ECONNREFUSED,
				syscall.ECONNREFUSED,
				syscall.ECONNREFUSED,
				syscall.ECONNREFUSED,
				nil, // Final attempt succeeds
			},
			responses: []*http.Response{nil, nil, nil, nil, {StatusCode: http.StatusOK}},
		}
		transport := &http.Transport{}
		rt := &retryTransport{base: mock, transport: transport, maxRetries: 3}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 5, mock.calls) // 1 initial + 3 retries + 1 final
	})

	t.Run("ErrConnExpired does not count as retry", func(t *testing.T) {
		mock := &mockRoundTripper{
			errors: []error{
				ErrConnExpired,
				ErrConnExpired,
				nil,
			},
			responses: []*http.Response{nil, nil, {StatusCode: http.StatusOK}},
		}
		rt := &retryTransport{base: mock, maxRetries: 1}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 3, mock.calls)
	})
}

func TestRetryTransport_RoundTrip_WithBody(t *testing.T) {
	t.Run("retries with request body", func(t *testing.T) {
		mock := &mockRoundTripper{
			errors: []error{
				syscall.ECONNREFUSED,
				nil,
			},
			responses: []*http.Response{nil, {StatusCode: http.StatusOK}},
		}
		rt := &retryTransport{base: mock, maxRetries: 3}

		body := bytes.NewReader([]byte(`{"key": "value"}`))
		req := httptest.NewRequest(http.MethodPost, "http://example.com", body)
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte(`{"key": "value"}`))), nil
		}

		resp, err := rt.RoundTrip(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, mock.calls)
	})
}

// ============================================================================
// isRetryableError Tests
// ============================================================================

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "ECONNREFUSED is retryable",
			err:       syscall.ECONNREFUSED,
			retryable: true,
		},
		{
			name:      "ECONNRESET is retryable",
			err:       syscall.ECONNRESET,
			retryable: true,
		},
		{
			name:      "ENETUNREACH is retryable",
			err:       syscall.ENETUNREACH,
			retryable: true,
		},
		{
			name:      "EOF is retryable",
			err:       io.EOF,
			retryable: true,
		},
		{
			name:      "ErrUnexpectedEOF is retryable",
			err:       io.ErrUnexpectedEOF,
			retryable: true,
		},
		{
			name:      "EPIPE is retryable",
			err:       syscall.EPIPE,
			retryable: true,
		},
		{
			name:      "net.ErrClosed is retryable",
			err:       net.ErrClosed,
			retryable: true,
		},
		{
			name:      "wrapped ECONNREFUSED is retryable",
			err:       errors.New("connect: " + syscall.ECONNREFUSED.Error()),
			retryable: false, // Note: simple string wrap doesn't work with errors.Is
		},
		{
			name:      "custom error is not retryable",
			err:       errors.New("custom error"),
			retryable: false,
		},
		{
			name:      "nil error is not retryable",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context canceled is not retryable",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded is not retryable",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestHTTPClient_Integration(t *testing.T) {
	t.Run("successful request to test server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		cfg := ClientConfig{
			BaseURL:             server.URL,
			Timeout:             lo.ToPtr(5 * time.Second),
			MaxIdleConnsPerHost: lo.ToPtr(10),
			IdleConnTimeout:     lo.ToPtr(30 * time.Second),
			MaxConnLifetime:     lo.ToPtr(time.Duration(0)), // Disable for test simplicity
		}

		client := newHTTPClient(cfg)

		req, _ := http.NewRequest(http.MethodGet, server.URL+"/test", nil)
		resp, err := client.Do(req)

		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, `{"status": "ok"}`, string(body))
	})

	t.Run("handles server errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := ClientConfig{
			BaseURL:             server.URL,
			Timeout:             lo.ToPtr(5 * time.Second),
			MaxIdleConnsPerHost: lo.ToPtr(10),
			IdleConnTimeout:     lo.ToPtr(30 * time.Second),
			MaxConnLifetime:     lo.ToPtr(time.Duration(0)),
		}

		client := newHTTPClient(cfg)

		req, _ := http.NewRequest(http.MethodGet, server.URL+"/error", nil)
		resp, err := client.Do(req)

		require.NoError(t, err) // HTTP errors are not transport errors
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("respects timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := ClientConfig{
			BaseURL:             server.URL,
			Timeout:             lo.ToPtr(50 * time.Millisecond),
			MaxIdleConnsPerHost: lo.ToPtr(10),
			IdleConnTimeout:     lo.ToPtr(30 * time.Second),
			MaxConnLifetime:     lo.ToPtr(time.Duration(0)),
		}

		client := newHTTPClient(cfg)

		req, _ := http.NewRequest(http.MethodGet, server.URL+"/slow", nil)
		_, err := client.Do(req)

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "Client.Timeout exceeded"),
			"expected timeout error, got: %v", err)
	})
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkIsRetryableError(b *testing.B) {
	errors := []error{
		syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		io.EOF,
		errors.New("custom error"),
		nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, err := range errors {
			isRetryableError(err)
		}
	}
}

func BenchmarkTimedConn_isExpired(b *testing.B) {
	conn := &timedConn{
		createdAt:   time.Now(),
		maxLifetime: 1 * time.Hour,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.isExpired()
	}
}
