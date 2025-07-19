package main

import (
	"flag"
	"log"
	"path/filepath"

	"scheduler-api/database"
)

func main() {
	var (
		migrationsDir = flag.String("migrations-dir", "./migrations", "Directory containing migration files")
		useNeon       = flag.Bool("neon", false, "Use Neon database configuration")
	)
	flag.Parse()

	// Load configuration
	var config *database.Config
	if *useNeon {
		config = database.LoadNeonConfig()
		log.Println("Using Neon database configuration")
	} else {
		config = database.LoadConfigFromEnv()
		log.Println("Using local database configuration")
	}

	// Connect to database
	db, err := database.Connect(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get absolute path for migrations directory
	absPath, err := filepath.Abs(*migrationsDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for migrations directory: %v", err)
	}

	// Run migrations
	log.Printf("Running migrations from directory: %s", absPath)
	if err := database.Migrate(db, absPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("All migrations completed successfully!")
}

