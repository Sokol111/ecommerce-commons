package swaggerui

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type SwaggerConfig struct {
	OpenAPIContent []byte
	Route          string
}

func registerSwaggerUI(router *gin.Engine, cfg SwaggerConfig) {
	router.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", cfg.OpenAPIContent)
	})

	route := cfg.Route
	if route == "" {
		route = "/swagger"
	}
	router.GET(route, func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<!DOCTYPE html>
<html>
<head>
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/openapi.yaml',
      dom_id: '#swagger-ui'
    });
  </script>
</body>
</html>`)
	})
}

func NewSwaggerModule(cfg SwaggerConfig) fx.Option {
	return fx.Invoke(func(r *gin.Engine) {
		registerSwaggerUI(r, cfg)
	})
}
