package logger

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogThrottler_DefaultInterval(t *testing.T) {
	// Given: zero interval
	log := zap.NewNop()

	// When: creating throttler with zero interval
	throttler := NewLogThrottler(log, 0)

	// Then: interval should default to 5 minutes
	require.NotNil(t, throttler)
	assert.Equal(t, 5*time.Minute, throttler.interval)
}

func TestNewLogThrottler_CustomInterval(t *testing.T) {
	// Given: custom interval
	log := zap.NewNop()
	customInterval := 10 * time.Second

	// When: creating throttler with custom interval
	throttler := NewLogThrottler(log, customInterval)

	// Then: interval should be set to custom value
	require.NotNil(t, throttler)
	assert.Equal(t, customInterval, throttler.interval)
}

func TestLogThrottler_Warn_FirstCallLogsWarn(t *testing.T) {
	// Given: a new throttler with observer
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Minute)

	// When: first call with a key
	throttler.Warn("test-key", "test message", zap.String("field", "value"))

	// Then: should log as WARN
	require.Equal(t, 1, logs.Len())
	entry := logs.All()[0]
	assert.Equal(t, zapcore.WarnLevel, entry.Level)
	assert.Equal(t, "test message", entry.Message)
	assert.Equal(t, "value", entry.ContextMap()["field"])
}

func TestLogThrottler_Warn_SubsequentCallsLogDebug(t *testing.T) {
	// Given: a throttler with a long interval
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour) // Long interval to ensure no token refresh

	// When: multiple calls with the same key
	throttler.Warn("test-key", "first message")
	throttler.Warn("test-key", "second message")
	throttler.Warn("test-key", "third message")

	// Then: first should be WARN, rest should be DEBUG
	require.Equal(t, 3, logs.Len())

	assert.Equal(t, zapcore.WarnLevel, logs.All()[0].Level)
	assert.Equal(t, "first message", logs.All()[0].Message)

	assert.Equal(t, zapcore.DebugLevel, logs.All()[1].Level)
	assert.Equal(t, "second message", logs.All()[1].Message)

	assert.Equal(t, zapcore.DebugLevel, logs.All()[2].Level)
	assert.Equal(t, "third message", logs.All()[2].Message)
}

func TestLogThrottler_Warn_DifferentKeysAreIndependent(t *testing.T) {
	// Given: a throttler
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour)

	// When: first calls with different keys
	throttler.Warn("key-1", "message for key 1")
	throttler.Warn("key-2", "message for key 2")
	throttler.Warn("key-3", "message for key 3")

	// Then: all should be WARN (each key has its own limiter)
	require.Equal(t, 3, logs.Len())

	for i, entry := range logs.All() {
		assert.Equal(t, zapcore.WarnLevel, entry.Level, "Entry %d should be WARN", i)
	}
}

func TestLogThrottler_Warn_MixedKeys(t *testing.T) {
	// Given: a throttler
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour)

	// When: interleaved calls with different keys
	throttler.Warn("key-a", "first A")  // WARN
	throttler.Warn("key-b", "first B")  // WARN
	throttler.Warn("key-a", "second A") // DEBUG (key-a already used)
	throttler.Warn("key-b", "second B") // DEBUG (key-b already used)
	throttler.Warn("key-c", "first C")  // WARN (new key)

	// Then: verify correct levels
	require.Equal(t, 5, logs.Len())

	expected := []struct {
		level   zapcore.Level
		message string
	}{
		{zapcore.WarnLevel, "first A"},
		{zapcore.WarnLevel, "first B"},
		{zapcore.DebugLevel, "second A"},
		{zapcore.DebugLevel, "second B"},
		{zapcore.WarnLevel, "first C"},
	}

	for i, exp := range expected {
		assert.Equal(t, exp.level, logs.All()[i].Level, "Entry %d level mismatch", i)
		assert.Equal(t, exp.message, logs.All()[i].Message, "Entry %d message mismatch", i)
	}
}

func TestLogThrottler_Warn_WithFields(t *testing.T) {
	// Given: a throttler
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour)

	// When: logging with multiple fields
	throttler.Warn("test-key", "message with fields",
		zap.String("string_field", "value"),
		zap.Int("int_field", 42),
		zap.Bool("bool_field", true),
		zap.Error(nil),
	)

	// Then: all fields should be present
	require.Equal(t, 1, logs.Len())
	entry := logs.All()[0]

	assert.Equal(t, "value", entry.ContextMap()["string_field"])
	assert.Equal(t, int64(42), entry.ContextMap()["int_field"])
	assert.Equal(t, true, entry.ContextMap()["bool_field"])
}

func TestLogThrottler_Warn_EmptyKey(t *testing.T) {
	// Given: a throttler
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour)

	// When: using empty string as key
	throttler.Warn("", "first with empty key")
	throttler.Warn("", "second with empty key")

	// Then: empty key should work like any other key
	require.Equal(t, 2, logs.Len())
	assert.Equal(t, zapcore.WarnLevel, logs.All()[0].Level)
	assert.Equal(t, zapcore.DebugLevel, logs.All()[1].Level)
}

func TestLogThrottler_ConcurrentAccess(t *testing.T) {
	// Given: a throttler
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour)

	// When: concurrent access from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 100
	numCallsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
				throttler.Warn("shared-key", "concurrent message", zap.Int("goroutine", id))
			}
		}(i)
	}

	wg.Wait()

	// Then: should not panic and should have logged all messages
	totalCalls := numGoroutines * numCallsPerGoroutine
	assert.Equal(t, totalCalls, logs.Len())

	// Only one should be WARN (the first one to acquire the token)
	warnCount := 0
	debugCount := 0
	for _, entry := range logs.All() {
		switch entry.Level {
		case zapcore.WarnLevel:
			warnCount++
		case zapcore.DebugLevel:
			debugCount++
		}
	}

	assert.Equal(t, 1, warnCount, "Should have exactly one WARN log")
	assert.Equal(t, totalCalls-1, debugCount, "Rest should be DEBUG logs")
}

func TestLogThrottler_ConcurrentAccessDifferentKeys(t *testing.T) {
	// Given: a throttler
	core, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	throttler := NewLogThrottler(log, time.Hour)

	// When: concurrent access with different keys
	var wg sync.WaitGroup
	numKeys := 50
	numCallsPerKey := 5

	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(keyID int) {
			defer wg.Done()
			key := string(rune('A' + keyID%26))
			for j := 0; j < numCallsPerKey; j++ {
				throttler.Warn(key, "message", zap.Int("key", keyID))
			}
		}(i)
	}

	wg.Wait()

	// Then: should not panic
	totalCalls := numKeys * numCallsPerKey
	assert.Equal(t, totalCalls, logs.Len())
}

func TestLogThrottler_GetLimiter_ReturnsSameLimiter(t *testing.T) {
	// Given: a throttler
	log := zap.NewNop()
	throttler := NewLogThrottler(log, time.Minute)

	// When: getting limiter for the same key multiple times
	limiter1 := throttler.getLimiter("test-key")
	limiter2 := throttler.getLimiter("test-key")
	limiter3 := throttler.getLimiter("test-key")

	// Then: should return the same limiter instance
	assert.Same(t, limiter1, limiter2)
	assert.Same(t, limiter2, limiter3)
}

func TestLogThrottler_GetLimiter_DifferentKeysReturnDifferentLimiters(t *testing.T) {
	// Given: a throttler
	log := zap.NewNop()
	throttler := NewLogThrottler(log, time.Minute)

	// When: getting limiters for different keys
	limiter1 := throttler.getLimiter("key-1")
	limiter2 := throttler.getLimiter("key-2")

	// Then: should return different limiter instances
	assert.NotSame(t, limiter1, limiter2)
}

func TestLogThrottler_MultipleInstances_AreIndependent(t *testing.T) {
	// Given: two separate throttler instances
	core1, logs1 := observer.New(zapcore.DebugLevel)
	core2, logs2 := observer.New(zapcore.DebugLevel)

	throttler1 := NewLogThrottler(zap.New(core1), time.Hour)
	throttler2 := NewLogThrottler(zap.New(core2), time.Hour)

	// When: using the same key on both throttlers
	throttler1.Warn("shared-key", "message from throttler 1")
	throttler2.Warn("shared-key", "message from throttler 2")

	// Then: both should log as WARN (independent limiters)
	require.Equal(t, 1, logs1.Len())
	require.Equal(t, 1, logs2.Len())

	assert.Equal(t, zapcore.WarnLevel, logs1.All()[0].Level)
	assert.Equal(t, zapcore.WarnLevel, logs2.All()[0].Level)
}
