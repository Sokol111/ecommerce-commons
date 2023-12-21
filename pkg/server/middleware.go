package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func errorHandler(c *gin.Context) {
	c.Next()

	for _, err := range c.Errors {
		log.Error().Msg(err.Error())
	}

	if len(c.Errors) > 0 {
		c.JSON(-1, gin.H{"error": c.Errors[len(c.Errors)-1].Error()})
	}
}
