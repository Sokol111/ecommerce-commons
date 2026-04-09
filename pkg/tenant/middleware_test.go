package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	corelogger "github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/ogen-go/ogen/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMiddleware_WithHeader(t *testing.T) {
	mw := Middleware(zap.NewNop())
	ctx := corelogger.With(context.Background(), zap.NewNop())
	httpReq := httptest.NewRequest(http.MethodGet, "http://localhost/api/products", nil)
	httpReq.Header.Set(TenantSlugHeader, "acme")

	nextCalled := false
	next := func(req middleware.Request) (middleware.Response, error) {
		nextCalled = true

		slug, ok := SlugFromContext(req.Context)
		require.True(t, ok)
		assert.Equal(t, "acme", slug)

		return middleware.Response{Type: "ok"}, nil
	}

	resp, err := mw(middleware.Request{Context: ctx, Raw: httpReq}, next)

	require.NoError(t, err)
	assert.True(t, nextCalled)
	assert.Equal(t, "ok", resp.Type)
}

func TestMiddleware_NoHeader(t *testing.T) {
	mw := Middleware(zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "http://localhost/api/products", nil)

	nextCalled := false
	next := func(req middleware.Request) (middleware.Response, error) {
		nextCalled = true
		return middleware.Response{Type: "unexpected"}, nil
	}

	resp, err := mw(middleware.Request{Context: context.Background(), Raw: req}, next)

	assert.ErrorIs(t, err, ErrTenantNotFound)
	assert.False(t, nextCalled)
	assert.Empty(t, resp.Type)
}
