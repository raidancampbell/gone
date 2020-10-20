package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func HealthHandler(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusOK)
}
