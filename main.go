package main

import (
	"log"

	"scheduler-api/internal/scheduler"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Create Gin router
	r := gin.Default()

	// Create server implementation
	server := scheduler.NewService(zap.NewExample())

	// Register routes
	scheduler.RegisterHandlers(r, server)

	log.Println("Starting server on :8000")
	log.Println("API documentation available at: http://localhost:8000/swagger/")

	if err := r.Run(":8000"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
