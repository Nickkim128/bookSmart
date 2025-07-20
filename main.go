package main

import (
	"context"
	"os"

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

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
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
