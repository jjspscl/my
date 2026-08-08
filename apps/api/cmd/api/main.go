package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/jjspscl/my/internal/contexts/access/application"
	"github.com/jjspscl/my/internal/contexts/access/infrastructure"
	accesshttp "github.com/jjspscl/my/internal/contexts/access/interfaces/http"
	financeapp "github.com/jjspscl/my/internal/contexts/finance/application"
	financeinfra "github.com/jjspscl/my/internal/contexts/finance/infrastructure"
	financehttp "github.com/jjspscl/my/internal/contexts/finance/interfaces/http"
	habitapp "github.com/jjspscl/my/internal/contexts/habits/application"
	habitinfra "github.com/jjspscl/my/internal/contexts/habits/infrastructure"
	habithttp "github.com/jjspscl/my/internal/contexts/habits/interfaces/http"
	"github.com/jjspscl/my/internal/platform/config"
	"github.com/jjspscl/my/internal/platform/database"
	plogger "github.com/jjspscl/my/internal/platform/logger"
	"github.com/jjspscl/my/internal/platform/mail"
	predis "github.com/jjspscl/my/internal/platform/redis"
	"github.com/jjspscl/my/internal/platform/session"
	"github.com/jjspscl/my/internal/platform/web"
	"github.com/jjspscl/my/internal/shared/middleware"
)

func main() {
	log := plogger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	// Database
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Error("database open failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db, "migrations"); err != nil {
		log.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}

	// Redis
	rdb, err := predis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Error("redis connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer rdb.Close()

	// Dependencies
	sessions := session.NewRedisStore(rdb, cfg.SessionTTL)
	mailer := mail.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom, cfg.SMTPUser, cfg.SMTPPass)
	tokenRepo := infrastructure.NewTokenRepoLibSQL(db)
	authSvc := application.NewAuthService(tokenRepo, sessions, mailer, cfg)
	authHandler := accesshttp.NewAuthHandler(authSvc, cfg.CookieSecret, cfg.CSRFSecret)

	// Finance
	txRepo := financeinfra.NewTransactionRepoLibSQL(db)
	walletRepo := financeinfra.NewWalletRepoLibSQL(db)
	txSvc := financeapp.NewTransactionService(txRepo, walletRepo, cfg.DefaultCurrency)
	financeHandler := financehttp.NewFinanceHandler(txSvc)

	budgetRepo := financeinfra.NewBudgetRepoLibSQL(db)
	budgetSvc := financeapp.NewBudgetService(budgetRepo)
	budgetHandler := financehttp.NewBudgetHandler(budgetSvc)

	billRepo := financeinfra.NewBillRepoLibSQL(db)
	billSvc := financeapp.NewBillService(billRepo)
	billHandler := financehttp.NewBillHandler(billSvc)
	txSvc.WithBillAutoMatcher(billSvc)

	goalRepo := financeinfra.NewGoalRepoLibSQL(db)
	transferRepo := financeinfra.NewTransferRepoLibSQL(db)
	goalSvc := financeapp.NewGoalService(goalRepo, transferRepo, walletRepo)
	goalHandler := financehttp.NewGoalHandler(goalSvc)

	walletSvc := financeapp.NewWalletService(walletRepo)
	walletHandler := financehttp.NewWalletHandler(walletSvc)

	transferSvc := financeapp.NewTransferService(transferRepo, walletRepo)
	transferHandler := financehttp.NewTransferHandler(transferSvc)

	// Habits
	habitRepo := habitinfra.NewHabitRepoLibSQL(db)
	habitSvc := habitapp.NewHabitService(habitRepo)
	habitHandler := habithttp.NewHabitHandler(habitSvc)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Recover(log))
	r.Use(middleware.RequestLogger(log))

	// API
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})

		// Public auth routes
		r.Route("/auth", authHandler.Routes)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(sessions))
			r.Use(middleware.CSRFProtect())

			// Finance
			r.Route("/finance", func(r chi.Router) {
				financeHandler.Routes(r)
				r.Route("/budgets", budgetHandler.Routes)
				r.Route("/bills", billHandler.Routes)
				r.Route("/goals", goalHandler.Routes)
				r.Route("/wallets", walletHandler.Routes)
				r.Route("/transfers", transferHandler.Routes)
			})

			// Habits
			r.Route("/habits", habitHandler.Routes)
		})
	})

	// SPA fallback
	r.Handle("/*", web.Handler())

	addr := ":" + cfg.APIPort
	log.Info("server starting", slog.String("addr", addr), slog.Int("pid", os.Getpid()))
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}
