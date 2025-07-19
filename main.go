package main

import (
	"log"
	"net/http"

	"scheduler-api/internal/scheduler"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create Gin router
	r := gin.Default()

	// Add CORS middleware for easier testing
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Create server implementation
	server := scheduler.NewServer()

	// Register routes
	scheduler.RegisterHandlers(r, server)

	// Add a health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	log.Println("Starting server on :8000")
	log.Println("API documentation available at: http://localhost:8000/swagger/")
	log.Println("Health check available at: http://localhost:8000/health")

	if err := r.Run(":8000"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
