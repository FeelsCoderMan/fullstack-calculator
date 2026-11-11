package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/api"
)

// swaggerUI serves the Swagger UI page, which renders the spec from pathSwaggerSpec.
func swaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, contentTypeHTML, api.SwaggerUI)
}

// openAPISpec serves the embedded OpenAPI document, the contract the frontend
// generates its types from.
func openAPISpec(c *gin.Context) {
	c.Data(http.StatusOK, contentTypeYAML, api.Spec)
}
