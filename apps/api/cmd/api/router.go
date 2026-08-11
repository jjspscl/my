package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	accesshttp "github.com/jjspscl/my/internal/contexts/access/interfaces/http"
	financehttp "github.com/jjspscl/my/internal/contexts/finance/interfaces/http"
	habithttp "github.com/jjspscl/my/internal/contexts/habits/interfaces/http"
	"github.com/jjspscl/my/internal/platform/backup"
	"github.com/jjspscl/my/internal/platform/session"
	platformversion "github.com/jjspscl/my/internal/platform/version"
	"github.com/jjspscl/my/internal/platform/web"
	"github.com/jjspscl/my/internal/shared/middleware"
)

type routerDeps struct {
	log                     *slog.Logger
	sessions                session.Store
	authHandler             *accesshttp.AuthHandler
	backupHandler           *backup.Handler
	magicLinkRate           int
	financeHandler          *financehttp.FinanceHandler
	budgetHandler           *financehttp.BudgetHandler
	billHandler             *financehttp.BillHandler
	goalHandler             *financehttp.GoalHandler
	walletHandler           *financehttp.WalletHandler
	transferHandler         *financehttp.TransferHandler
	categoryHandler         *financehttp.CategoryHandler
	analyticsHandler        *financehttp.AnalyticsHandler
	derivedAnalyticsHandler *financehttp.DerivedAnalyticsHandler
	habitHandler            *habithttp.HabitHandler
}

func newRouter(deps routerDeps) chi.Router {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Recover(deps.log))
	r.Use(middleware.RequestLogger(deps.log))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"version": platformversion.String(),
			})
		})

		// Mount auth once. Public routes and protected logout share this subrouter.
		r.Route("/auth", func(r chi.Router) {
			deps.authHandler.PublicRoutes(r, deps.magicLinkRate)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(deps.sessions))
				r.Use(middleware.CSRFProtect())
				deps.authHandler.ProtectedRoutes(r)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(deps.sessions))
			r.Use(middleware.CSRFProtect())

			// Database snapshot and JSON export — full data dump, so they sit
			// behind session auth and are deliberately NOT exposed via MCP.
			r.Get("/backup", deps.backupHandler.Snapshot)
			r.Get("/export", deps.backupHandler.Export)

			r.Route("/finance", func(r chi.Router) {
				deps.financeHandler.Routes(r)
				r.Route("/budgets", deps.budgetHandler.Routes)
				r.Route("/bills", deps.billHandler.Routes)
				r.Route("/goals", deps.goalHandler.Routes)
				r.Route("/wallets", deps.walletHandler.Routes)
				r.Route("/transfers", deps.transferHandler.Routes)
				r.Route("/categories", deps.categoryHandler.Routes)
				r.Route("/analytics", func(r chi.Router) {
					deps.analyticsHandler.Routes(r)
					deps.derivedAnalyticsHandler.Routes(r)
				})
			})

			r.Route("/habits", deps.habitHandler.Routes)
		})
	})
	// MCP is served on its own listener (see cmd/api/main.go) so MY_MCP_BIND can
	// genuinely restrict it. Answer /mcp here explicitly so a client pointed at
	// the dashboard port gets a 404 instead of the SPA's index.html.
	r.Handle("/mcp", http.NotFoundHandler())

	r.Handle("/*", web.Handler())
	return r
}
