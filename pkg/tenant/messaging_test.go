package tenant

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestSaveToHeaders_NoTenant(t *testing.T) {
	t.Parallel()

	headers := map[string]string{"existing": "value"}
	result := SaveToHeaders(context.Background(), headers)

	assert.Equal(t, headers, result)
	assert.Len(t, result, 1)
}

func TestSaveToHeaders_NilHeaders(t *testing.T) {
	t.Parallel()

	ctx := ContextWithSlug(context.Background(), "shop")
	result := SaveToHeaders(ctx, nil)

	assert.Equal(t, map[string]string{HeaderKey: "shop"}, result)
}

func TestSaveToHeaders_AddsToExisting(t *testing.T) {
	t.Parallel()

	ctx := ContextWithSlug(context.Background(), "shop")
	headers := map[string]string{"existing": "value"}
	result := SaveToHeaders(ctx, headers)

	assert.Equal(t, "shop", result[HeaderKey])
	assert.Equal(t, "value", result["existing"])
}

func TestContextFromKafkaHeaders_NoHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := ContextFromKafkaHeaders(ctx, []kgo.RecordHeader{})

	_, ok := SlugFromContext(result)
	assert.False(t, ok)
}

func TestContextFromKafkaHeaders_MissingKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	headers := []kgo.RecordHeader{{Key: "other", Value: []byte("shop")}}
	result := ContextFromKafkaHeaders(ctx, headers)

	_, ok := SlugFromContext(result)
	assert.False(t, ok)
}

func TestContextFromKafkaHeaders_EmptyValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	headers := []kgo.RecordHeader{{Key: HeaderKey, Value: []byte("")}}
	result := ContextFromKafkaHeaders(ctx, headers)

	_, ok := SlugFromContext(result)
	assert.False(t, ok)
}

func TestContextFromKafkaHeaders_SetsSlug(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	headers := []kgo.RecordHeader{{Key: HeaderKey, Value: []byte("shop")}}
	result := ContextFromKafkaHeaders(ctx, headers)

	slug, ok := SlugFromContext(result)
	assert.True(t, ok)
	assert.Equal(t, "shop", slug)
}

func TestContextFromKafkaHeaders_LaterHeadersIgnored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	headers := []kgo.RecordHeader{
		{Key: HeaderKey, Value: []byte("first")},
		{Key: HeaderKey, Value: []byte("second")},
	}
	result := ContextFromKafkaHeaders(ctx, headers)

	slug, ok := SlugFromContext(result)
	assert.True(t, ok)
	assert.Equal(t, "first", slug)
}
