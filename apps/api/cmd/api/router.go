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
	"github.com/jjspscl/my/internal/platform/session"
	platformversion "github.com/jjspscl/my/internal/platform/version"
	"github.com/jjspscl/my/internal/platform/web"
	"github.com/jjspscl/my/internal/shared/middleware"
)

type routerDeps struct {
	log             *slog.Logger
	sessions        session.Store
	authHandler     *accesshttp.AuthHandler
	financeHandler  *financehttp.FinanceHandler
	budgetHandler   *financehttp.BudgetHandler
	billHandler     *financehttp.BillHandler
	goalHandler     *financehttp.GoalHandler
	walletHandler   *financehttp.WalletHandler
	transferHandler *financehttp.TransferHandler
	habitHandler    *habithttp.HabitHandler
	mcpHandler      http.Handler
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
			deps.authHandler.PublicRoutes(r)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(deps.sessions))
				r.Use(middleware.CSRFProtect())
				deps.authHandler.ProtectedRoutes(r)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(deps.sessions))
			r.Use(middleware.CSRFProtect())

			r.Route("/finance", func(r chi.Router) {
				deps.financeHandler.Routes(r)
				r.Route("/budgets", deps.budgetHandler.Routes)
				r.Route("/bills", deps.billHandler.Routes)
				r.Route("/goals", deps.goalHandler.Routes)
				r.Route("/wallets", deps.walletHandler.Routes)
				r.Route("/transfers", deps.transferHandler.Routes)
			})

			r.Route("/habits", deps.habitHandler.Routes)
		})
	})
	if deps.mcpHandler != nil {
		r.Handle("/mcp", deps.mcpHandler)
	} else {
		r.Handle("/mcp", http.NotFoundHandler())
	}

	r.Handle("/*", web.Handler())
	return r
}
