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
	"github.com/jjspscl/my/internal/platform/config"
	"github.com/jjspscl/my/internal/platform/database"
	"github.com/jjspscl/my/internal/platform/mail"
	predis "github.com/jjspscl/my/internal/platform/redis"
	"github.com/jjspscl/my/internal/platform/session"
	"github.com/jjspscl/my/internal/platform/timeutil"
	redis "github.com/redis/go-redis/v9"
)

type App struct {
	Cfg      *config.Config
	Log      *slog.Logger
	DB       *sql.DB
	Redis    *redis.Client
	Sessions session.Store

	Auth      *accessapp.AuthService
	Tx        *financeapp.TransactionService
	Budget    *financeapp.BudgetService
	Bill      *financeapp.BillService
	Goal      *financeapp.GoalService
	Wallet    *financeapp.WalletService
	Transfer  *financeapp.TransferService
	Category  *financeapp.CategoryService
	Analytics *financeapp.AnalyticsService
	Habit     *habitapp.HabitService

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
		if err := database.Migrate(db, "migrations"); err != nil {
			_ = db.Close()
			return nil, err
		}
	} else if err := database.VerifySchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

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
	billSvc := financeapp.NewBillService(billRepo).WithCurrency(cfg.DefaultCurrency).WithClock(clock)
	txSvc.WithBillAutoMatcher(billSvc)

	goalRepo := financeinfra.NewGoalRepoLibSQL(db)
	transferRepo := financeinfra.NewTransferRepoLibSQL(db)
	goalSvc := financeapp.NewGoalService(goalRepo, transferRepo, walletRepo).WithClock(clock).WithCoordinator(coordinator)
	walletSvc := financeapp.NewWalletService(walletRepo)
	transferSvc := financeapp.NewTransferService(transferRepo, walletRepo)

	categoryRepo := financeinfra.NewCategoryRepoLibSQL(db)
	categorySvc := financeapp.NewCategoryService(categoryRepo)

	analyticsRepo := financeinfra.NewAnalyticsRepoLibSQL(db)
	analyticsSvc := financeapp.NewAnalyticsService(analyticsRepo, budgetRepo, goalRepo).WithClock(clock)

	habitRepo := habitinfra.NewHabitRepoLibSQL(db)
	habitSvc := habitapp.NewHabitService(habitRepo)

	return &App{
		Cfg:       cfg,
		Log:       log,
		DB:        db,
		Redis:     rdb,
		Sessions:  sessions,
		Auth:      authSvc,
		Tx:        txSvc,
		Budget:    budgetSvc,
		Bill:      billSvc,
		Goal:      goalSvc,
		Wallet:    walletSvc,
		Transfer:  transferSvc,
		Category:  categorySvc,
		Analytics: analyticsSvc,
		Habit:     habitSvc,
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
