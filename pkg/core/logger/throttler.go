package logger

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// LogThrottler provides rate-limited logging functionality.
// Each instance maintains its own isolated map of rate limiters,
// allowing different components to have independent throttling.
type LogThrottler struct {
	log      *zap.Logger
	limiters sync.Map // map[string]*rate.Limiter
	interval time.Duration
}

// NewLogThrottler creates a new LogThrottler with the given logger.
// The interval parameter specifies how often a WARN log is allowed per key.
// If interval is 0, it defaults to 5 minutes.
func NewLogThrottler(log *zap.Logger, interval time.Duration) *LogThrottler {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &LogThrottler{
		log:      log,
		interval: interval,
	}
}

// Warn logs as WARN once per interval per key, DEBUG otherwise.
// This is useful for suppressing repetitive error logs while
// still maintaining visibility of ongoing issues.
func (t *LogThrottler) Warn(key string, msg string, fields ...zap.Field) {
	limiter := t.getLimiter(key)

	if limiter.Allow() {
		t.log.Warn(msg, fields...)
	} else {
		t.log.Debug(msg, fields...)
	}
}

func (t *LogThrottler) getLimiter(key string) *rate.Limiter {
	if limiter, ok := t.limiters.Load(key); ok {
		return limiter.(*rate.Limiter)
	}

	// 1 event per interval, no burst
	limiter := rate.NewLimiter(rate.Every(t.interval), 1)
	actual, _ := t.limiters.LoadOrStore(key, limiter)
	return actual.(*rate.Limiter)
}
