package swaggerui

import (
	"net/http"

	"go.uber.org/fx"
)

type SwaggerConfig struct {
	OpenAPIContent []byte
	Route          string
}

func registerSwaggerUI(mux *http.ServeMux, cfg SwaggerConfig) {
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cfg.OpenAPIContent) //nolint:errcheck // HTTP handler, error logged by net/http
	})

	route := cfg.Route
	if route == "" {
		route = "/swagger"
	}
	mux.HandleFunc(route, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		//nolint:errcheck // HTTP handler, error logged by net/http
		_, _ = w.Write([]byte(`<!DOCTYPE html>
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
</html>`))
	})
}

func NewSwaggerModule(cfg SwaggerConfig) fx.Option {
	return fx.Invoke(func(mux *http.ServeMux) {
		registerSwaggerUI(mux, cfg)
	})
}
