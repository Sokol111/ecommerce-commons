package consumer

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// errorTracker tracks retriable errors and implements smart logging
// to avoid log spam while providing visibility into persistent issues
type errorTracker struct {
	trackers map[string]*errorCounter
	mu       sync.RWMutex
	log      *zap.Logger
}

type errorCounter struct {
	count     int
	firstSeen time.Time
	lastWarn  time.Time
}

func newErrorTracker(log *zap.Logger) *errorTracker {
	return &errorTracker{
		trackers: make(map[string]*errorCounter),
		log:      log,
	}
}

// logReaderError logs a retriable error with smart throttling
// It logs as WARN every 5 minutes, and as DEBUG otherwise
func (t *errorTracker) logReaderError(err *readerError) {
	t.mu.Lock()
	defer t.mu.Unlock()

	counter, exists := t.trackers[err.errorKey]
	if !exists {
		counter = &errorCounter{firstSeen: time.Now()}
		t.trackers[err.errorKey] = counter
	}
	counter.count++

	shouldWarn := counter.lastWarn.IsZero() || time.Since(counter.lastWarn) > 5*time.Minute

	if shouldWarn {
		t.log.Warn(err.description,
			zap.Error(err),
			zap.Int("attempts", counter.count),
			zap.Duration("duration", time.Since(counter.firstSeen)))
		counter.lastWarn = time.Now()
	} else {
		t.log.Debug(err.description, zap.Error(err))
	}
}
