package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"

	"scheduler-api/internal/auth"
	"scheduler-api/internal/scheduler"
	"scheduler-api/database"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func initPostgres(log *zap.Logger) (*pgxpool.Pool, error) {
	databaseURL := os.Getenv("DATABASE_URL")

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

func initSQLDatabase(log *zap.Logger) (*sql.DB, error) {
	// Load database configuration
	var config *database.Config
	if os.Getenv("DATABASE_URL") != "" {
		config = database.LoadNeonConfig()
	} else {
		config = database.LoadConfigFromEnv()
	}

	// Connect to database
	db, err := database.Connect(config)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := database.Migrate(db, "migrations"); err != nil {
		log.Error("Failed to run migrations", zap.Error(err))
		// Don't fail completely - migrations might already be applied
	}

	return db, nil
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	r := gin.Default()

	// Add CORS middleware for frontend integration - MUST be before route registration
	allowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if the origin is in the allowed list
		for _, allowedOrigin := range allowedOrigins {
			if strings.TrimSpace(allowedOrigin) == origin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Initialize PGX pool connection
	pgxPool, err := initPostgres(logger)
	if err != nil {
		logger.Fatal("Failed to initialize Postgres:", zap.Error(err))
	}
	defer pgxPool.Close()

	// Initialize SQL database connection (for auth service)
	sqlDB, err := initSQLDatabase(logger)
	if err != nil {
		logger.Fatal("Failed to initialize SQL database:", zap.Error(err))
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			logger.Error("Failed to close SQL database", zap.Error(closeErr))
		}
	}()

	// Initialize Firebase service
	firebaseService, err := auth.NewFirebaseService()
	if err != nil {
		logger.Fatal("Failed to initialize Firebase service:", zap.Error(err))
	}

	// Initialize auth middleware
	authMiddleware := auth.NewAuthMiddleware(firebaseService, sqlDB, logger)

	// Create middleware functions for different protection levels
	requireAuth := authMiddleware.RequireAuth()
	requireAdmin := authMiddleware.RequireRole("admin")
	// requireTutor := authMiddleware.RequireRole("tutor")
	// requireStudent := authMiddleware.RequireRole("student")

	// Define which routes need authentication
	authMiddlewares := []scheduler.MiddlewareFunc{
		// Convert Gin middleware to MiddlewareFunc
		func(c *gin.Context) {
			requireAuth(c)
		},
	}

	// Register handlers with authentication middleware
	scheduler.RegisterHandlersWithOptions(r, scheduler.NewService(logger, pgxPool, sqlDB, firebaseService), scheduler.GinServerOptions{
		Middlewares: authMiddlewares,
	})


	// Register Swagger documentation endpoints
	scheduler.RegisterSwaggerHandlers(r)

	// Add some public routes (if any) - these would be registered separately
	// For now, all routes require authentication

	// Add admin-only routes group
	adminGroup := r.Group("/v1/admin")
	adminGroup.Use(requireAuth, requireAdmin)
	{
		// Add admin-specific routes here if needed
		adminGroup.GET("/users", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Admin users endpoint"})
		})
	}

	logger.Info("Starting server on :8000")
	logger.Info("Firebase authentication enabled")

	if err := r.Run(":8000"); err != nil {
		logger.Fatal("Failed to start server:", zap.Error(err))
	}
}
