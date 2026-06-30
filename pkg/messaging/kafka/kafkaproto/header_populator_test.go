package kafkaproto

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// testProtoEvent returns a proto.Message for testing (uses well-known emptypb.Empty).
func testProtoEvent() proto.Message {
	return &emptypb.Empty{}
}

func TestHeaderPopulator_PopulateHeaders(t *testing.T) {
	t.Run("populates all header fields", func(t *testing.T) {
		populator := NewHeaderPopulator("test-service")
		headers := make(map[string]string)

		beforeMs := time.Now().UTC().UnixMilli()
		eventID := populator.PopulateHeaders(context.Background(), testProtoEvent(), headers)
		afterMs := time.Now().UTC().UnixMilli()

		assert.NotEmpty(t, eventID)
		assert.Equal(t, eventID, headers["event_id"])
		assert.Equal(t, "google.protobuf.Empty", headers["event_type"])
		assert.Equal(t, "test-service", headers["source"])

		ts, err := strconv.ParseInt(headers["timestamp"], 10, 64)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ts, beforeMs)
		assert.LessOrEqual(t, ts, afterMs)
	})

	t.Run("returns generated event ID", func(t *testing.T) {
		populator := NewHeaderPopulator("test-service")
		headers := make(map[string]string)

		eventID := populator.PopulateHeaders(context.Background(), testProtoEvent(), headers)

		require.NotEmpty(t, eventID)
		assert.Equal(t, eventID, headers["event_id"])
	})

	t.Run("sets event_type from proto full name", func(t *testing.T) {
		populator := NewHeaderPopulator("svc")
		headers := make(map[string]string)

		populator.PopulateHeaders(context.Background(), testProtoEvent(), headers)

		assert.Equal(t, "google.protobuf.Empty", headers["event_type"])
	})
}
