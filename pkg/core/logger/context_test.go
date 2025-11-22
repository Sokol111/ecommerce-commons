package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGet_WithNilContext(t *testing.T) {
	// Given: nil context and default logger
	original := defaultLogger
	defaultLogger = zap.NewNop()
	t.Cleanup(func() { defaultLogger = original })

	// When: getting logger from nil context
	logger := Get(nil)

	// Then: should return default logger
	assert.NotNil(t, logger)
	assert.Equal(t, defaultLogger, logger)
}

func TestGet_WithEmptyContext(t *testing.T) {
	// Given: empty context and default logger
	original := defaultLogger
	defaultLogger = zap.NewNop()
	t.Cleanup(func() { defaultLogger = original })
	ctx := context.Background()

	// When: getting logger from empty context
	logger := Get(ctx)

	// Then: should return default logger
	assert.NotNil(t, logger)
	assert.Equal(t, defaultLogger, logger)
}

func TestGet_WithLoggerInContext(t *testing.T) {
	// Given: context with custom logger
	original := defaultLogger
	defaultLogger = zap.NewNop()
	t.Cleanup(func() { defaultLogger = original })

	// Create a different logger instance using observer
	core, _ := observer.New(zapcore.InfoLevel)
	customLogger := zap.New(core)
	ctx := context.WithValue(context.Background(), loggerCtxKey, customLogger)

	// When: getting logger from context
	logger := Get(ctx)

	// Then: should return custom logger from context
	assert.NotNil(t, logger)
	assert.Equal(t, customLogger, logger)
	assert.NotEqual(t, defaultLogger, logger)
}

func TestGet_WithNilLoggerInContext(t *testing.T) {
	// Given: context with nil logger value
	original := defaultLogger
	defaultLogger = zap.NewNop()
	t.Cleanup(func() { defaultLogger = original })

	ctx := context.WithValue(context.Background(), loggerCtxKey, (*zap.Logger)(nil))

	// When: getting logger from context
	logger := Get(ctx)

	// Then: should return default logger
	assert.NotNil(t, logger)
	assert.Equal(t, defaultLogger, logger)
}

func TestGet_WithWrongTypeInContext(t *testing.T) {
	// Given: context with wrong type value
	original := defaultLogger
	defaultLogger = zap.NewNop()
	t.Cleanup(func() { defaultLogger = original })

	ctx := context.WithValue(context.Background(), loggerCtxKey, "not a logger")

	// When: getting logger from context
	logger := Get(ctx)

	// Then: should return default logger
	assert.NotNil(t, logger)
	assert.Equal(t, defaultLogger, logger)
}

func TestWith_CreatesNewContext(t *testing.T) {
	// Given: background context and custom logger
	ctx := context.Background()
	customLogger := zap.NewNop()

	// When: attaching logger to context
	newCtx := With(ctx, customLogger)

	// Then: new context should contain the logger
	assert.NotNil(t, newCtx)
	assert.NotEqual(t, ctx, newCtx)

	loggerFromCtx := Get(newCtx)
	assert.Equal(t, customLogger, loggerFromCtx)
}

func TestWith_WithNilContext(t *testing.T) {
	// Given: nil context and custom logger
	customLogger := zap.NewNop()

	// When: attaching logger to nil context
	newCtx := With(nil, customLogger)

	// Then: should create new context with logger
	assert.NotNil(t, newCtx)

	loggerFromCtx := Get(newCtx)
	assert.Equal(t, customLogger, loggerFromCtx)
}

func TestWith_ReplacesExistingLogger(t *testing.T) {
	// Given: context with existing logger
	firstLogger := zap.NewNop()

	core, _ := observer.New(zapcore.InfoLevel)
	secondLogger := zap.New(core)

	ctx := With(context.Background(), firstLogger)

	// When: replacing logger in context
	newCtx := With(ctx, secondLogger)

	// Then: new context should have the second logger
	loggerFromCtx := Get(newCtx)
	assert.Equal(t, secondLogger, loggerFromCtx)
}

func TestGet_With_Integration(t *testing.T) {
	// Given: observer core to track logs
	core, recorded := observer.New(zapcore.InfoLevel)
	customLogger := zap.New(core)

	// When: using logger from context
	ctx := With(context.Background(), customLogger)
	logger := Get(ctx)
	logger.Info("test message", zap.String("key", "value"))

	// Then: log should be recorded
	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "test message", logs[0].Message)
	assert.Equal(t, "value", logs[0].ContextMap()["key"])
}

func TestContextKey_Uniqueness(t *testing.T) {
	// Given: two context keys
	key1 := contextKey{}
	key2 := contextKey{}

	// Then: keys should be different instances but same type
	assert.Equal(t, key1, key2) // Empty structs are equal
	assert.IsType(t, contextKey{}, loggerCtxKey)
}

func TestWith_ChainedContexts(t *testing.T) {
	// Given: multiple loggers
	logger1 := zap.NewNop()
	logger2 := zap.NewNop()
	logger3 := zap.NewNop()

	// When: chaining context updates
	ctx1 := With(context.Background(), logger1)
	ctx2 := With(ctx1, logger2)
	ctx3 := With(ctx2, logger3)

	// Then: each context should have its own logger
	assert.Equal(t, logger1, Get(ctx1))
	assert.Equal(t, logger2, Get(ctx2))
	assert.Equal(t, logger3, Get(ctx3))
}

func TestGet_ConcurrentAccess(t *testing.T) {
	// Given: setup default logger
	original := defaultLogger
	defaultLogger = zap.NewNop()
	t.Cleanup(func() { defaultLogger = original })

	customLogger := zap.NewNop()
	ctx := With(context.Background(), customLogger)

	// When: accessing logger concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			logger := Get(ctx)
			assert.NotNil(t, logger)
			assert.Equal(t, customLogger, logger)
			done <- true
		}()
	}

	// Then: all goroutines should complete successfully
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestWith_ConcurrentContextCreation(t *testing.T) {
	// Given: base context
	baseCtx := context.Background()

	// When: creating contexts concurrently
	done := make(chan context.Context, 10)
	for i := 0; i < 10; i++ {
		go func() {
			logger := zap.NewNop()
			ctx := With(baseCtx, logger)
			done <- ctx
		}()
	}

	// Then: all contexts should be created successfully
	contexts := make([]context.Context, 0, 10)
	for i := 0; i < 10; i++ {
		ctx := <-done
		assert.NotNil(t, ctx)
		contexts = append(contexts, ctx)
	}

	// Each context should have its own logger
	for i, ctx := range contexts {
		logger := Get(ctx)
		assert.NotNil(t, logger, "context %d should have a logger", i)
	}
}
