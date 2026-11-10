package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthResponse struct {
	Status HealthStatus `json:"status"`
}

// Health reports that the process is up and able to serve requests.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: HealthStatusOK})
}
