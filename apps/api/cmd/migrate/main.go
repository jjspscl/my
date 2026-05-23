package main

import (
	"log"

	"github.com/jjspscl/my/internal/platform/config"
	"github.com/jjspscl/my/internal/platform/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	log.Println("migrations applied successfully")
}
