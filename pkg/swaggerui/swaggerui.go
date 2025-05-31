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

func registerSwaggerUI(r *gin.Engine) error {
	subFS, err := fs.Sub(swaggerFS, "swagger-ui")
	if err != nil {
		return err
	}
	r.StaticFS("/swagger", http.FS(subFS))
	r.StaticFile("/openapi.yaml", "./api/openapi.yaml")
	return nil
}

func NewSwaggerModule() fx.Option {
	return fx.Invoke(registerSwaggerUI)
}
