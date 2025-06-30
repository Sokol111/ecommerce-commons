package swaggerui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

//go:embed swagger-ui/*
var swaggerFS embed.FS

type SwaggerConfig struct {
	OpenAPIContent []byte
}

func registerSwaggerUI(router *gin.Engine, cfg SwaggerConfig) error {
	subFS, err := fs.Sub(swaggerFS, "swagger-ui")
	if err != nil {
		return err
	}

	router.StaticFS("/swagger", http.FS(subFS))

	router.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", cfg.OpenAPIContent)
	})

	return nil
}

func NewSwaggerModule(cfg SwaggerConfig) fx.Option {
	return fx.Invoke(func(r *gin.Engine) error {
		return registerSwaggerUI(r, cfg)
	})
}
