package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// health reports that the process is up and able to serve requests. The status
// code is the whole answer: 204 No Content, no body (see api/openapi.yaml).
func health(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
