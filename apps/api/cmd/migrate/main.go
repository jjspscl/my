package main

import (
	"log/slog"
	"os"

	"github.com/jjspscl/my/internal/platform/config"
	"github.com/jjspscl/my/internal/platform/database"
	"github.com/jjspscl/my/migrations"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Error("database open failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	// Migrations are embedded in the binary; no filesystem dependency and no
	// dependence on the working directory.
	if err := database.Migrate(db, migrations.FS, log); err != nil {
		log.Error("migrate failed", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("migrations applied successfully")
}
