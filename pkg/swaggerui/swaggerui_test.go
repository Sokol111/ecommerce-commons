package swaggerui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRegisterSwaggerUI_OpenAPIEndpoint(t *testing.T) {
	router := gin.New()
	openAPIContent := []byte("openapi: 3.0.0\ninfo:\n  title: Test API")

	registerSwaggerUI(router, SwaggerConfig{
		OpenAPIContent: openAPIContent,
	})

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/yaml", w.Header().Get("Content-Type"))
	assert.Equal(t, string(openAPIContent), w.Body.String())
}

func TestRegisterSwaggerUI_DefaultRoute(t *testing.T) {
	router := gin.New()

	registerSwaggerUI(router, SwaggerConfig{
		OpenAPIContent: []byte("test"),
	})

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "swagger-ui")
	assert.Contains(t, w.Body.String(), "/openapi.yaml")
}

func TestRegisterSwaggerUI_CustomRoute(t *testing.T) {
	router := gin.New()

	registerSwaggerUI(router, SwaggerConfig{
		OpenAPIContent: []byte("test"),
		Route:          "/api-docs",
	})

	// Custom route should work
	req := httptest.NewRequest(http.MethodGet, "/api-docs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "swagger-ui")

	// Default route should not exist
	req = httptest.NewRequest(http.MethodGet, "/swagger", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterSwaggerUI_HTMLContent(t *testing.T) {
	router := gin.New()

	registerSwaggerUI(router, SwaggerConfig{
		OpenAPIContent: []byte("test"),
	})

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "<!DOCTYPE html>")
	assert.Contains(t, body, "<title>Swagger UI</title>")
	assert.Contains(t, body, "swagger-ui-dist@5/swagger-ui.css")
	assert.Contains(t, body, "swagger-ui-dist@5/swagger-ui-bundle.js")
	assert.Contains(t, body, "SwaggerUIBundle")
	assert.Contains(t, body, "dom_id: '#swagger-ui'")
}

func TestNewSwaggerModule(t *testing.T) {
	cfg := SwaggerConfig{
		OpenAPIContent: []byte("openapi: 3.0.0"),
		Route:          "/docs",
	}

	module := NewSwaggerModule(cfg)
	assert.NotNil(t, module)
}
