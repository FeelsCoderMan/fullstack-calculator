package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/api"
)

// SwaggerUI serves the Swagger UI page, which renders the spec from PathSwaggerSpec.
func SwaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, contentTypeHTML, api.SwaggerUI)
}

// OpenAPISpec serves the embedded OpenAPI document, the contract the frontend
// generates its types from.
func OpenAPISpec(c *gin.Context) {
	c.Data(http.StatusOK, contentTypeYAML, api.Spec)
}
