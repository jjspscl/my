package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jjspscl/my/internal/platform/bootstrap"
	"github.com/jjspscl/my/internal/platform/config"
	platformmcp "github.com/jjspscl/my/internal/platform/mcp"
	"github.com/jjspscl/my/internal/platform/update"
	"github.com/jjspscl/my/internal/platform/version"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	checkUpdate := flag.Bool("check-update", false, "check GitHub for a newer release")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.String())
		return
	}
	if *checkUpdate {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		result, err := update.Check(ctx, version.Version)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(result.Message())
		return
	}

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}
	app, err := bootstrap.NewWithOptions(cfg, log, bootstrap.Options{SkipMigrations: true})
	if err != nil {
		log.Error("application bootstrap failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Error("application close failed", slog.Any("error", err))
		}
	}()

	server := platformmcp.NewServer(app, platformmcp.Options{ReadOnly: cfg.MCPReadOnly})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil && ctx.Err() == nil {
		log.Error("MCP stdio server failed", slog.Any("error", err))
		os.Exit(1)
	}
}
