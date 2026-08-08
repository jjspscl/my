package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	accesshttp "github.com/jjspscl/my/internal/contexts/access/interfaces/http"
	financehttp "github.com/jjspscl/my/internal/contexts/finance/interfaces/http"
	habithttp "github.com/jjspscl/my/internal/contexts/habits/interfaces/http"
	"github.com/jjspscl/my/internal/platform/bootstrap"
	"github.com/jjspscl/my/internal/platform/config"
	plogger "github.com/jjspscl/my/internal/platform/logger"
	platformmcp "github.com/jjspscl/my/internal/platform/mcp"
	platformversion "github.com/jjspscl/my/internal/platform/version"
	"github.com/jjspscl/my/internal/shared/middleware"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(platformversion.String())
		return
	}

	log := plogger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	app, err := bootstrap.New(cfg, log)
	if err != nil {
		log.Error("application bootstrap failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Error("application close failed", slog.Any("error", err))
		}
	}()

	authHandler := accesshttp.NewAuthHandler(app.Auth, cfg.SecureCookies, cfg.SessionTTL)

	// Finance
	financeHandler := financehttp.NewFinanceHandler(app.Tx)
	budgetHandler := financehttp.NewBudgetHandler(app.Budget)
	billHandler := financehttp.NewBillHandler(app.Bill)
	goalHandler := financehttp.NewGoalHandler(app.Goal)
	walletHandler := financehttp.NewWalletHandler(app.Wallet)
	transferHandler := financehttp.NewTransferHandler(app.Transfer)

	// Habits
	habitHandler := habithttp.NewHabitHandler(app.Habit)

	var mcpHandler http.Handler
	if cfg.MCPEnabled {
		if cfg.MCPBind != "127.0.0.1" && cfg.MCPBind != "localhost" {
			log.Warn("MCP server configured beyond localhost; bearer token protects full data mutation surface", slog.String("bind", cfg.MCPBind))
		}
		mcpServer := platformmcp.NewServer(app, platformmcp.Options{ReadOnly: cfg.MCPReadOnly})
		mcpHandler = mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
			return mcpServer
		}, &mcpsdk.StreamableHTTPOptions{
			JSONResponse: true,
			Logger:       log,
		})
		mcpHandler = http.MaxBytesHandler(mcpHandler, 1<<20)
		mcpHandler = middleware.RequireBearerToken(cfg.MCPToken, cfg.MCPBind == "127.0.0.1" || cfg.MCPBind == "localhost")(mcpHandler)
	}

	r := newRouter(routerDeps{
		log:             log,
		sessions:        app.Sessions,
		authHandler:     authHandler,
		financeHandler:  financeHandler,
		budgetHandler:   budgetHandler,
		billHandler:     billHandler,
		goalHandler:     goalHandler,
		walletHandler:   walletHandler,
		transferHandler: transferHandler,
		habitHandler:    habitHandler,
		mcpHandler:      mcpHandler,
	})

	addr := ":" + cfg.APIPort
	log.Info("server starting", slog.String("addr", addr), slog.Int("pid", os.Getpid()))
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}
