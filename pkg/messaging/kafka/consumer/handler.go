package consumer

import (
	"context"
	"errors"
)

// Sentinel errors for message processing control flow.

// ErrSkipMessage indicates that the message should be skipped without retry and committed.
// Use this when:
//   - The handler doesn't need to process this particular event type.
//   - The message is intentionally ignored (e.g., filtered out).
//   - The event is outdated or irrelevant.
var ErrSkipMessage = errors.New("skip message processing")

// ErrPermanent indicates a permanent error that should send message to DLQ.
// Use this when:
// - Business logic validation fails (invalid data that won't change on retry)
// - Required entity not found and won't be created
// - Data integrity violation
// - Panic occurred (bug in code)
// - Any error that retrying won't fix
// If you don't wrap your error with ErrPermanent or ErrSkipMessage,
// it will be treated as a retryable error with exponential backoff.
var ErrPermanent = errors.New("permanent error")

type Handler interface {
	Process(ctx context.Context, event any) error
}
