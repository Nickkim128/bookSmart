package main

import (
	"context"
	"os"
	"strings"
	"time"

	"scheduler-api/internal/scheduler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func initPostgres(log *zap.Logger) (*pgxpool.Pool, error) {
	databaseURL := os.Getenv("NEON_DB_CONN")

	pgxConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatal("Failed to parse database URL:", zap.Error(err))
	}

	pgxPool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
	if err != nil {
		log.Fatal("Failed to create pool:", zap.Error(err))
	}

	return pgxPool, nil
}

func main() {
	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()

	// Get allowed origins from environment variable or use defaults
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		// Default origins for development
		config.AllowOrigins = []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		}
	} else {
		// Parse comma-separated origins from environment variable
		origins := strings.Split(allowedOrigins, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		config.AllowOrigins = origins
	}

	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"X-Requested-With",
		"Accept",
		"Cache-Control",
		"X-CSRF-Token",
	}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	// Apply CORS middleware
	r.Use(cors.New(config))

	log, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to create logger:", zap.Error(err))
	}

	pgxPool, err := initPostgres(log)
	if err != nil {
		log.Fatal("Failed to initialize Postgres:", zap.Error(err))
	}

	scheduler.RegisterHandlers(r, scheduler.NewService(
		log,
		pgxPool,
	))

	// Register Swagger documentation endpoints
	scheduler.RegisterSwaggerHandlers(r)

	log.Info("Starting server on :8000")

	if err := r.Run(":8000"); err != nil {
		log.Fatal("Failed to start server:", zap.Error(err))
	}
}
