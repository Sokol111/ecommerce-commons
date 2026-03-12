package consumer

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// offsetMarker is an interface for marking records for commit.
type offsetMarker interface {
	MarkCommitRecords(...*kgo.Record)
}

// resultHandler handles message processing results.
type resultHandler struct {
	log          *zap.Logger
	dlqHandler   DLQHandler
	offsetMarker offsetMarker
}

func newResultHandler(
	log *zap.Logger,
	dlqHandler DLQHandler,
	client *kgo.Client,
) *resultHandler {
	return &resultHandler{
		log:          log,
		dlqHandler:   dlqHandler,
		offsetMarker: client,
	}
}

// handle processes the result of message handling and takes appropriate action.
func (h *resultHandler) handle(ctx context.Context, err error, record *kgo.Record, span trace.Span) {
	defer h.offsetMarker.MarkCommitRecords(record)

	switch {
	case err == nil:
		span.SetStatus(codes.Ok, "message processed successfully")

	case errors.Is(err, ErrSkipMessage):
		span.SetStatus(codes.Ok, "message skipped")
		h.log.Info("skipping message", h.recordFields(record)...)

	case errors.Is(err, ErrPermanent):
		span.SetStatus(codes.Error, "permanent error - sending to DLQ")
		h.log.Error("permanent error - sending message to DLQ", h.recordFieldsWithError(record, err)...)
		h.dlqHandler.SendToDLQ(ctx, record, err)

	default:
		// Retry exhausted or context cancelled
		span.RecordError(err)
		span.SetStatus(codes.Error, "message processing failed - sending to DLQ")
		h.log.Error("message processing failed after retries - sending to DLQ", h.recordFieldsWithError(record, err)...)
		h.dlqHandler.SendToDLQ(ctx, record, err)
	}
}

func (h *resultHandler) recordFields(record *kgo.Record) []zap.Field {
	return []zap.Field{
		zap.String("key", string(record.Key)),
		zap.Int32("partition", record.Partition),
		zap.Int64("offset", record.Offset),
	}
}

func (h *resultHandler) recordFieldsWithError(record *kgo.Record, err error) []zap.Field {
	return append(h.recordFields(record), zap.Error(err))
}
