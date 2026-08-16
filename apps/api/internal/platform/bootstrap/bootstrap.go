package bootstrap

import (
	"database/sql"
	"errors"
	"log/slog"
	"sync"

	accessapp "github.com/jjspscl/my/internal/contexts/access/application"
	accessinfra "github.com/jjspscl/my/internal/contexts/access/infrastructure"
	financeapp "github.com/jjspscl/my/internal/contexts/finance/application"
	financeinfra "github.com/jjspscl/my/internal/contexts/finance/infrastructure"
	habitapp "github.com/jjspscl/my/internal/contexts/habits/application"
	habitinfra "github.com/jjspscl/my/internal/contexts/habits/infrastructure"
	intelapp "github.com/jjspscl/my/internal/contexts/intelligence/application"
	intelinfra "github.com/jjspscl/my/internal/contexts/intelligence/infrastructure"
	"github.com/jjspscl/my/internal/platform/config"
	"github.com/jjspscl/my/internal/platform/database"
	"github.com/jjspscl/my/internal/platform/mail"
	predis "github.com/jjspscl/my/internal/platform/redis"
	"github.com/jjspscl/my/internal/platform/session"
	"github.com/jjspscl/my/internal/platform/timeutil"
	"github.com/jjspscl/my/migrations"
	redis "github.com/redis/go-redis/v9"
)

type App struct {
	Cfg      *config.Config
	Log      *slog.Logger
	DB       *sql.DB
	Redis    *redis.Client
	Sessions session.Store

	Auth             *accessapp.AuthService
	Tx               *financeapp.TransactionService
	Budget           *financeapp.BudgetService
	Bill             *financeapp.BillService
	Goal             *financeapp.GoalService
	Wallet           *financeapp.WalletService
	Transfer         *financeapp.TransferService
	Category         *financeapp.CategoryService
	Import           *financeapp.ImportService
	Analytics        *financeapp.AnalyticsService
	DerivedAnalytics *financeapp.DerivedAnalyticsService
	Habit            *habitapp.HabitService
	Intelligence     *intelapp.SettingsService
	Analysis         *intelapp.AnalysisService

	closeOnce sync.Once
	closeErr  error
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	return NewWithOptions(cfg, log, Options{})
}

type Options struct {
	SkipMigrations bool
}

func NewWithOptions(cfg *config.Config, log *slog.Logger, opts Options) (*App, error) {
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if !opts.SkipMigrations {
		if err := database.Migrate(db, migrations.FS, log); err != nil {
			_ = db.Close()
			return nil, err
		}
	} else if err := database.VerifySchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	reportOrphans(db, log)

	rdb, err := predis.NewClient(cfg.RedisURL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	sessions := session.NewRedisStore(rdb, cfg.SessionTTL)
	mailer := mail.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom, cfg.SMTPUser, cfg.SMTPPass)
	tokenRepo := accessinfra.NewTokenRepoLibSQL(db)
	authSvc := accessapp.NewAuthService(tokenRepo, sessions, mailer, cfg)

	txRepo := financeinfra.NewTransactionRepoLibSQL(db)
	walletRepo := financeinfra.NewWalletRepoLibSQL(db)
	coordinator := financeinfra.NewCoordinator(db)
	clock := timeutil.New(cfg.Location)
	txSvc := financeapp.NewTransactionService(txRepo, walletRepo, clock)

	budgetRepo := financeinfra.NewBudgetRepoLibSQL(db)
	budgetSvc := financeapp.NewBudgetService(budgetRepo).WithCurrency(cfg.DefaultCurrency).WithClock(clock)

	billRepo := financeinfra.NewBillRepoLibSQL(db)
	billSvc := financeapp.NewBillService(billRepo).WithCurrency(cfg.DefaultCurrency).WithClock(clock).WithTransactionSupport(txRepo, walletRepo).WithCoordinator(coordinator)
	txSvc.WithBillAutoMatcher(billSvc)

	importRepo := financeinfra.NewImportRepoLibSQL(db)
	// Transaction edits/deletes must reconcile bill links and import
	// provenance atomically, so the service is wired with both.
	txSvc.WithCoordinator(coordinator).WithBillLinkRepo(billRepo).WithImportProvenanceMarker(importRepo)

	goalRepo := financeinfra.NewGoalRepoLibSQL(db)
	transferRepo := financeinfra.NewTransferRepoLibSQL(db)
	goalSvc := financeapp.NewGoalService(goalRepo, transferRepo, walletRepo).WithClock(clock).WithCoordinator(coordinator)
	walletSvc := financeapp.NewWalletService(walletRepo)
	transferSvc := financeapp.NewTransferService(transferRepo, walletRepo)

	importSvc := financeapp.NewImportService(importRepo, txRepo, transferRepo, walletRepo, coordinator)
	categoryRepo := financeinfra.NewCategoryRepoLibSQL(db)
	categorySvc := financeapp.NewCategoryService(categoryRepo)

	analyticsRepo := financeinfra.NewAnalyticsRepoLibSQL(db)
	analyticsSvc := financeapp.NewAnalyticsService(analyticsRepo, budgetRepo, goalRepo).WithClock(clock)
	derivedAnalyticsSvc := financeapp.NewDerivedAnalyticsService(analyticsRepo, walletRepo, billRepo, analyticsSvc, billSvc).WithClock(clock)

	habitRepo := habitinfra.NewHabitRepoLibSQL(db)
	habitSvc := habitapp.NewHabitService(habitRepo)

	// Intelligence: confidence-gated LLM analysis. Services are always built
	// so settings/status stay reachable; the SecretBox may be nil (no master
	// key), in which case credential writes fail closed and analysis is
	// unavailable. Analysis additionally requires MY_LLM_ENABLED.
	intelRepo := intelinfra.NewIntelligenceRepoLibSQL(db)
	box, _ := intelinfra.NewSecretBox(cfg.LLMMasterKey) // nil when unconfigured
	runtime := intelapp.RuntimeConfig{LLMEnabled: cfg.LLMEnabled, CodexPath: cfg.LLMCodexPath}
	intelligenceSvc := intelapp.NewSettingsService(intelRepo, box, runtime)
	confidenceSvc := intelapp.NewConfidenceService()
	gateway := intelinfra.NewSearchService()
	analysisSvc := intelapp.NewAnalysisService(intelRepo, intelligenceSvc, confidenceSvc, gateway, box != nil && cfg.LLMEnabled)

	return &App{
		Cfg:              cfg,
		Log:              log,
		DB:               db,
		Redis:            rdb,
		Sessions:         sessions,
		Auth:             authSvc,
		Tx:               txSvc,
		Budget:           budgetSvc,
		Bill:             billSvc,
		Goal:             goalSvc,
		Wallet:           walletSvc,
		Transfer:         transferSvc,
		Category:         categorySvc,
		Import:           importSvc,
		Analytics:        analyticsSvc,
		DerivedAnalytics: derivedAnalyticsSvc,
		Habit:            habitSvc,
		Intelligence:     intelligenceSvc,
		Analysis:         analysisSvc,
	}, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}

	a.closeOnce.Do(func() {
		var errs []error
		if a.Redis != nil {
			errs = append(errs, a.Redis.Close())
		}
		if a.DB != nil {
			errs = append(errs, a.DB.Close())
		}
		a.closeErr = errors.Join(errs...)
	})
	return a.closeErr
}

// reportOrphans surfaces pre-existing orphaned child rows (created while
// foreign_keys was OFF) without failing boot. New orphans cannot form: the
// goal and bill repos delete children explicitly, and FKs are ON.
func reportOrphans(db *sql.DB, log *slog.Logger) {
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		log.Warn("foreign_key_check failed", slog.Any("error", err))
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count > 0 {
		log.Warn("foreign_key_check found orphaned rows (pre-existing from the foreign_keys=OFF era); review before relying on cascades",
			slog.Int("count", count))
	} else {
		log.Debug("foreign_key_check clean")
	}
}
