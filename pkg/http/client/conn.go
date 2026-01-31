package client

import (
	"errors"
	"net"
	"time"
)

// ErrConnExpired is returned when a connection exceeds its max lifetime.
// This error is handled specially by retryTransport - it doesn't count as a retry attempt.
var ErrConnExpired = errors.New("connection expired")

// timedConn wraps net.Conn to track creation time and enforce max lifetime.
// When maxLifetime expires, the connection reports itself as closed on next read/write,
// forcing http.Transport to create a new connection (with fresh DNS lookup).
type timedConn struct {
	net.Conn
	createdAt   time.Time
	maxLifetime time.Duration
}

func (c *timedConn) isExpired() bool {
	return time.Since(c.createdAt) > c.maxLifetime
}

func (c *timedConn) Read(b []byte) (n int, err error) {
	if c.isExpired() {
		_ = c.Close() //nolint:errcheck // Best effort cleanup on expiry
		return 0, ErrConnExpired
	}
	return c.Conn.Read(b)
}

func (c *timedConn) Write(b []byte) (n int, err error) {
	if c.isExpired() {
		_ = c.Close() //nolint:errcheck // Best effort cleanup on expiry
		return 0, ErrConnExpired
	}
	return c.Conn.Write(b)
}
