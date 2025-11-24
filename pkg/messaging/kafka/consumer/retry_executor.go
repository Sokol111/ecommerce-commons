package consumer

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
)

// PanicError представляє помилку, що виникла через panic
type PanicError struct {
	Panic interface{}
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Panic)
}

// RetryExecutor виконує операцію з retry логікою та panic recovery
type RetryExecutor interface {
	// Execute виконує функцію з повторними спробами при помилках
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

type retryExecutor struct {
	maxAttempts int
	maxBackoff  time.Duration
	log         *zap.Logger
}

func newRetryExecutor(maxAttempts int, maxBackoff time.Duration, log *zap.Logger) RetryExecutor {
	return &retryExecutor{
		maxAttempts: maxAttempts,
		maxBackoff:  maxBackoff,
		log:         log,
	}
}

func (r *retryExecutor) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	for attempt := 1; attempt <= r.maxAttempts && ctx.Err() == nil; attempt++ {
		// Виконуємо функцію з panic recovery
		err := r.executeWithPanicRecovery(ctx, fn)

		if err == nil {
			// Успіх
			return nil
		}

		// Перевіряємо чи повідомлення треба пропустити
		if errors.Is(err, ErrSkipMessage) {
			return err
		}

		// Перевіряємо чи помилка перманентна
		if errors.Is(err, ErrPermanent) {
			return err
		}

		// Логуємо помилку
		r.logError(err, attempt)

		// Якщо це остання спроба, повертаємо помилку
		if attempt >= r.maxAttempts {
			return fmt.Errorf("max retry attempts reached: %w", err)
		}

		// Чекаємо перед наступною спробою з backoff
		sleep(ctx, backoffDuration(attempt, r.maxBackoff))
	}

	// Контекст скасований під час retry
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return fmt.Errorf("unexpected end of retry loop")
}

func (r *retryExecutor) executeWithPanicRecovery(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			// Panic - це перманентна помилка (вказує на баг у коді)
			err = fmt.Errorf("%w: %v", ErrPermanent, &PanicError{
				Panic: rec,
				Stack: debug.Stack(),
			})
		}
	}()

	return fn(ctx)
}

func (r *retryExecutor) logError(err error, attempt int) {
	logFields := []zap.Field{
		zap.Int("attempt", attempt),
		zap.Int("maxAttempts", r.maxAttempts),
	}

	// Додаємо специфічні поля для panic помилок
	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		logFields = append(logFields,
			zap.Any("panic", panicErr.Panic),
			zap.ByteString("stack", panicErr.Stack),
		)
	} else {
		logFields = append(logFields, zap.Error(err))
	}

	r.log.Error("failed to process message", logFields...)
}
