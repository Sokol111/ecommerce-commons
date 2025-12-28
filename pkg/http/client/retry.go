package client

import (
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"
)

// retryTransport wraps http.RoundTripper with automatic retry on transient errors.
// Retries immediately without backoff - designed for pod failover scenarios.
// After exhausting retries, closes idle connections and makes one final attempt.
type retryTransport struct {
	base       http.RoundTripper
	transport  *http.Transport // stored for CloseIdleConnections; nil if base is not *http.Transport
	maxRetries int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; attempt <= t.maxRetries; {
		resp, err := t.doRequest(req, attempt)
		if err == nil {
			return resp, nil
		}

		// Expired connections don't count as retries - just get a new one
		if errors.Is(err, ErrConnExpired) {
			continue
		}

		if !isRetryableError(err) {
			return nil, err
		}
		attempt++
	}

	// All retries exhausted - pool may be full of dead connections
	// Close idle connections and make one final attempt with fresh connection
	if t.transport != nil {
		t.transport.CloseIdleConnections()
	}

	return t.doRequest(req, t.maxRetries+1)
}

func (t *retryTransport) doRequest(req *http.Request, attempt int) (*http.Response, error) {
	reqToSend := req

	// Clone request for retry (body may have been consumed)
	if attempt > 0 {
		reqToSend = req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			reqToSend.Body = body
		}
	}

	return t.base.RoundTrip(reqToSend)
}

// isRetryableError checks if the error is transient and should be retried.
func isRetryableError(err error) bool {
	// Connection refused (pod died, not ready)
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	// Connection reset (pod killed mid-request)
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	// Network unreachable
	if errors.Is(err, syscall.ENETUNREACH) {
		return true
	}

	// EOF (connection closed by server unexpectedly)
	if errors.Is(err, io.EOF) {
		return true
	}

	// Unexpected EOF (partial read before connection closed)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Broken pipe (writing to closed connection)
	if errors.Is(err, syscall.EPIPE) {
		return true
	}

	// Connection closed by peer
	if errors.Is(err, net.ErrClosed) {
		return true
	}

	return false
}
