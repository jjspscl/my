package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	financeHandler := financehttp.NewFinanceHandler(app.Tx, cfg.DefaultCurrency)
	budgetHandler := financehttp.NewBudgetHandler(app.Budget)
	billHandler := financehttp.NewBillHandler(app.Bill)
	goalHandler := financehttp.NewGoalHandler(app.Goal)
	walletHandler := financehttp.NewWalletHandler(app.Wallet)
	transferHandler := financehttp.NewTransferHandler(app.Transfer)

	// Habits
	habitHandler := habithttp.NewHabitHandler(app.Habit)

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
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiServer := newHTTPServer(":"+cfg.APIPort, r)
	servers := []*http.Server{apiServer}
	log.Info("server starting", slog.String("addr", apiServer.Addr), slog.Int("pid", os.Getpid()))

	if cfg.MCPEnabled {
		mcpServer := newHTTPServer(cfg.MCPAddr(), mcpHandler(app, cfg, log))
		servers = append(servers, mcpServer)
		if !isLoopbackHost(cfg.MCPBind) {
			log.Warn("MCP listener is not loopback-bound; full finance read and write surface is reachable from the network with only a static bearer token",
				slog.String("addr", mcpServer.Addr))
		}
		log.Info("MCP server starting",
			slog.String("addr", mcpServer.Addr),
			slog.Bool("read_only", cfg.MCPReadOnly))
	}

	errs := make(chan error, len(servers))
	for _, srv := range servers {
		go func(srv *http.Server) {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errs <- fmt.Errorf("listen %s: %w", srv.Addr, err)
			}
		}(srv)
	}

	var listenErr error
	select {
	case <-ctx.Done():
	case listenErr = <-errs:
		log.Error("server failed", slog.Any("error", listenErr))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, srv := range servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown failed", slog.String("addr", srv.Addr), slog.Any("error", err))
		}
	}
	if listenErr != nil {
		os.Exit(1)
	}
	log.Info("server stopped")
}

// newHTTPServer applies timeouts so a slow or idle client cannot hold a
// connection open indefinitely (Slowloris).
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// mcpHandler builds the MCP endpoint. Requests are authenticated with a bearer
// token and capped in size. Streamable HTTP responses may stream indefinitely,
// so no write timeout is applied to this listener.
func mcpHandler(app *bootstrap.App, cfg *config.Config, log *slog.Logger) http.Handler {
	server := platformmcp.NewServer(app, platformmcp.Options{ReadOnly: cfg.MCPReadOnly})
	var handler http.Handler = mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return server },
		&mcpsdk.StreamableHTTPOptions{JSONResponse: true, Logger: log},
	)
	handler = http.MaxBytesHandler(handler, 1<<20)
	handler = middleware.RequireBearerToken(cfg.MCPToken)(handler)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	return mux
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
