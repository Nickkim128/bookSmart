package scheduler

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed swagger-ui/*
var swaggerUIFiles embed.FS

// SwaggerUIHandler serves the Swagger UI
func SwaggerUIHandler(c *gin.Context) {
	// Get the OpenAPI spec
	swagger, err := GetSwagger()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load OpenAPI specification"})
		return
	}

	// Convert to JSON
	specJSON, err := swagger.MarshalJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal OpenAPI specification"})
		return
	}

	// Parse the Swagger UI template
	tmpl, err := template.ParseFS(swaggerUIFiles, "swagger-ui/index.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Swagger UI template"})
		return
	}

	// Set content type to HTML
	c.Header("Content-Type", "text/html")

	// Execute template with the spec
	err = tmpl.Execute(c.Writer, gin.H{
		"spec": string(specJSON),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render Swagger UI"})
		return
	}
}

// SwaggerSpecHandler serves the raw OpenAPI specification as JSON
func SwaggerSpecHandler(c *gin.Context) {
	swagger, err := GetSwagger()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load OpenAPI specification"})
		return
	}

	c.JSON(http.StatusOK, swagger)
}

// RegisterSwaggerHandlers registers the Swagger UI and spec endpoints
func RegisterSwaggerHandlers(router gin.IRouter) {
	// Serve Swagger UI static files
	router.StaticFS("/swagger-ui", http.FS(swaggerUIFiles))

	// Serve Swagger UI at /docs
	router.GET("/docs", SwaggerUIHandler)

	// Serve raw OpenAPI spec at /docs/swagger.json
	router.GET("/docs/swagger.json", SwaggerSpecHandler)
}
