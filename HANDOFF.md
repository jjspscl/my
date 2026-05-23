# HANDOFF — Optimistic Mutations + React Hook Form + Structured Logging + Domain Validation

## Part 1: Optimistic Mutations

### 1A. Update `apps/web/src/features/finance/hooks/use-transactions.ts`

Replace entirely:

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import {
  createTransaction as createTransactionApi,
  deleteTransaction as deleteTransactionApi,
  getTodayTotal as getTodayTotalApi,
  listTransactions as listTransactionsApi,
} from '../api/finance.api'
import type { CreateTransaction, Transaction } from '../schemas/transaction.schemas'

export function useTodayTotal() {
  return useQuery({
    queryKey: financeKeys.todayTotal(),
    queryFn: getTodayTotalApi,
    staleTime: 1000 * 60,
  })
}

export function useTransactions(from = '', to = '') {
  return useQuery({
    queryKey: financeKeys.transactionList({ from, to }),
    queryFn: () => listTransactionsApi(from, to),
    staleTime: 1000 * 60,
  })
}

export function useCreateTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateTransaction) => createTransactionApi(data),
    onMutate: async (data) => {
      await queryClient.cancelQueries({ queryKey: financeKeys.transactions() })
      const prevTotal = queryClient.getQueryData(financeKeys.todayTotal())
      return { prevTotal }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prevTotal) {
        queryClient.setQueryData(financeKeys.todayTotal(), ctx.prevTotal)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteTransactionApi(id),
    onMutate: async (id) => {
      // Cancel in-flight fetches
      await queryClient.cancelQueries({ queryKey: financeKeys.transactions() })

      // Snapshot all transaction list queries for rollback
      const queries = queryClient.getQueriesData<Transaction[]>({ queryKey: financeKeys.transactions() })
      const snapshots = queries.map(([key, data]) => ({ key, data }))

      // Optimistically remove from all cached lists
      for (const { key } of snapshots) {
        queryClient.setQueryData<Transaction[]>(key, (old) =>
          old ? old.filter((tx) => tx.id !== id) : old,
        )
      }

      return { snapshots }
    },
    onError: (_err, _vars, ctx) => {
      // Rollback all lists
      if (ctx?.snapshots) {
        for (const { key, data } of ctx.snapshots) {
          queryClient.setQueryData(key, data)
        }
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}
```

### 1B. Update `apps/web/src/features/habits/hooks/use-habits.ts`

Replace entirely:

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { habitKeys } from '../api/habits.keys'
import {
  archiveHabit as archiveHabitApi,
  createHabit as createHabitApi,
  getCompletions as getCompletionsApi,
  getCompletionsMap as getCompletionsMapApi,
  listHabits as listHabitsApi,
  toggleHabit as toggleHabitApi,
} from '../api/habits.api'
import type { CreateHabit, Habit } from '../schemas/habit.schemas'

export function useHabits() {
  return useQuery({
    queryKey: habitKeys.list(),
    queryFn: listHabitsApi,
    staleTime: 1000 * 30,
  })
}

export function useCreateHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateHabit) => createHabitApi(data),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useToggleHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ habitId, date }: { habitId: string; date?: string }) =>
      toggleHabitApi(habitId, date),
    onMutate: async ({ habitId }) => {
      await queryClient.cancelQueries({ queryKey: habitKeys.list() })
      const prev = queryClient.getQueryData<Habit[]>(habitKeys.list())

      // Optimistically flip completedToday
      queryClient.setQueryData<Habit[]>(habitKeys.list(), (old) =>
        old
          ? old.map((h) =>
              h.id === habitId
                ? {
                    ...h,
                    completedToday: !h.completedToday,
                    currentStreak: h.completedToday
                      ? Math.max(0, (h.currentStreak ?? 0) - 1)
                      : (h.currentStreak ?? 0) + 1,
                  }
                : h,
            )
          : old,
      )

      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) {
        queryClient.setQueryData(habitKeys.list(), ctx.prev)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useArchiveHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (habitId: string) => archiveHabitApi(habitId),
    onMutate: async (habitId) => {
      await queryClient.cancelQueries({ queryKey: habitKeys.list() })
      const prev = queryClient.getQueryData<Habit[]>(habitKeys.list())

      // Optimistically remove from list
      queryClient.setQueryData<Habit[]>(habitKeys.list(), (old) =>
        old ? old.filter((h) => h.id !== habitId) : old,
      )

      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) {
        queryClient.setQueryData(habitKeys.list(), ctx.prev)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useCompletions(habitId: string, from?: string, to?: string) {
  return useQuery({
    queryKey: [...habitKeys.completions(habitId), from, to],
    queryFn: () => getCompletionsApi(habitId, from, to),
    staleTime: 1000 * 60,
  })
}

export function useCompletionsMap(from?: string, to?: string) {
  return useQuery({
    queryKey: habitKeys.completionsMap(from, to),
    queryFn: () => getCompletionsMapApi(from, to),
    staleTime: 1000 * 60,
  })
}
```

---

## Part 2: React Hook Form + zodResolver

### 2A. Replace `apps/web/src/features/finance/components/add-expense-dialog.tsx`

```tsx
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { CategoryCombobox } from './category-combobox'
import { useCreateTransaction } from '../hooks/use-transactions'
import { CreateTransactionSchema, type CreateTransaction } from '../schemas/transaction.schemas'
import { z } from 'zod'

// Form schema extends CreateTransactionSchema but uses string for amount (input field)
const FormSchema = z.object({
  amount: z.string().min(1, 'Amount is required').refine(
    (v) => parseFloat(v) > 0,
    'Amount must be positive',
  ),
  category: z.string().min(1, 'Category is required'),
  description: z.string().default(''),
  type: z.enum(['expense', 'income']),
  transactionDate: z.string().min(1),
})
type FormValues = z.infer<typeof FormSchema>

interface AddExpenseDialogProps {
  trigger?: React.ReactNode
  defaultType?: 'expense' | 'income'
}

export function AddExpenseDialog({ trigger, defaultType = 'expense' }: AddExpenseDialogProps) {
  const [open, setOpen] = useState(false)
  const createTx = useCreateTransaction()

  const form = useForm<FormValues>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      amount: '',
      category: '',
      description: '',
      type: defaultType,
      transactionDate: new Date().toISOString().split('T')[0],
    },
  })

  const onSubmit = (values: FormValues) => {
    const data: CreateTransaction = {
      amountCents: Math.round(parseFloat(values.amount) * 100),
      category: values.category,
      description: values.description,
      type: values.type,
      transactionDate: values.transactionDate,
    }

    createTx.mutate(data, {
      onSuccess: () => {
        setOpen(false)
        form.reset()
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm" className="w-full gap-2">
            <Plus className="h-4 w-4" />
            Add expense
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add {form.watch('type') === 'expense' ? 'Expense' : 'Income'}</DialogTitle>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-2">
            <div className="flex gap-2">
              <FormField
                control={form.control}
                name="amount"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel className="text-xs text-muted-foreground">Amount (PHP)</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        step="0.01"
                        min="0"
                        placeholder="0.00"
                        autoFocus
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="text-xs text-muted-foreground">Type</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className="w-[110px]">
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="expense">Expense</SelectItem>
                        <SelectItem value="income">Income</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="category"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Category</FormLabel>
                  <FormControl>
                    <CategoryCombobox value={field.value} onChange={field.onChange} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Description (optional)</FormLabel>
                  <FormControl>
                    <Input placeholder="Coffee, lunch..." {...field} />
                  </FormControl>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="transactionDate"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Date</FormLabel>
                  <FormControl>
                    <Input type="date" {...field} />
                  </FormControl>
                </FormItem>
              )}
            />

            <Button
              type="submit"
              className="w-full"
              size="sm"
              disabled={createTx.isPending}
            >
              {createTx.isPending ? 'Saving...' : `Add ${form.watch('type') === 'expense' ? 'Expense' : 'Income'}`}
            </Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
```

### 2B. Replace `apps/web/src/features/habits/components/add-habit-dialog.tsx`

```tsx
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Check, Plus } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { PALETTE_TOKENS, type PaletteToken } from '@/shared/theme/palette'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { useCreateHabit } from '../hooks/use-habits'
import { CreateHabitSchema, type CreateHabit } from '../schemas/habit.schemas'

interface AddHabitDialogProps {
  trigger?: React.ReactNode
}

export function AddHabitDialog({ trigger }: AddHabitDialogProps) {
  const [open, setOpen] = useState(false)
  const createHabit = useCreateHabit()

  const form = useForm<CreateHabit>({
    resolver: zodResolver(CreateHabitSchema),
    defaultValues: {
      name: '',
      color: 'blue',
      frequency: 'daily',
      targetPerWeek: 1,
    },
  })

  const onSubmit = (values: CreateHabit) => {
    createHabit.mutate(values, {
      onSuccess: () => {
        setOpen(false)
        form.reset()
      },
    })
  }

  const frequency = form.watch('frequency')

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            Add Habit
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Habit</DialogTitle>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-2">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Name</FormLabel>
                  <FormControl>
                    <Input placeholder="Read, Exercise, Meditate..." autoFocus {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="color"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Color</FormLabel>
                  <FormControl>
                    <div className="flex gap-1.5 flex-wrap">
                      {PALETTE_TOKENS.map((t) => (
                        <button
                          key={t}
                          type="button"
                          className={cn(
                            'h-7 w-7 rounded-full flex items-center justify-center transition-transform',
                            field.value === t && 'ring-2 ring-offset-1 ring-foreground scale-110',
                          )}
                          style={{ backgroundColor: `var(--palette-${t})` }}
                          onClick={() => field.onChange(t as PaletteToken)}
                        >
                          {field.value === t && <Check className="h-3.5 w-3.5 text-white" />}
                        </button>
                      ))}
                    </div>
                  </FormControl>
                </FormItem>
              )}
            />

            <div className="flex gap-3">
              <FormField
                control={form.control}
                name="frequency"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel className="text-xs text-muted-foreground">Frequency</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="daily">Daily</SelectItem>
                        <SelectItem value="weekly">Weekly</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormItem>
                )}
              />

              {frequency === 'weekly' && (
                <FormField
                  control={form.control}
                  name="targetPerWeek"
                  render={({ field }) => (
                    <FormItem className="w-24">
                      <FormLabel className="text-xs text-muted-foreground">Times/week</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          min="1"
                          max="7"
                          {...field}
                          onChange={(e) => field.onChange(parseInt(e.target.value) || 1)}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              )}
            </div>

            <Button
              type="submit"
              className="w-full"
              size="sm"
              disabled={createHabit.isPending}
            >
              {createHabit.isPending ? 'Creating...' : 'Create Habit'}
            </Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
```

---

## Part 3: Structured Logging (slog)

### 3A. Create `apps/api/internal/platform/logger/logger.go`

```go
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a structured JSON logger. Level from MY_LOG_LEVEL env (default: info).
func New() *slog.Logger {
	level := parseLevel(os.Getenv("MY_LOG_LEVEL"))
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

### 3B. Create `apps/api/internal/shared/middleware/request_logger.go`

```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// RequestLogger logs method, path, status, duration for every request.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}

			next.ServeHTTP(sw, r)

			log.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", r.Header.Get("X-Request-Id")),
			)
		})
	}
}
```

### 3C. Create `apps/api/internal/shared/middleware/recover.go`

```go
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/jjspscl/my/internal/shared/response"
)

// Recover catches panics, logs stack trace, returns 500.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered",
						slog.Any("error", err),
						slog.String("stack", string(debug.Stack())),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
					)
					response.WriteError(w, r, http.StatusInternalServerError, "internal server error", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
```

### 3D. Update `apps/api/cmd/api/main.go`

Replace `log` stdlib with slog. Replace chi's Logger/Recoverer with our custom middleware:

```go
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
	txSvc := financeapp.NewTransactionService(txRepo, cfg.DefaultCurrency)
	financeHandler := financehttp.NewFinanceHandler(txSvc)

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
			r.Route("/finance", financeHandler.Routes)

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
```

---

## Part 4: Domain-Layer Validation

### 4A. Replace `apps/api/internal/contexts/finance/domain/transaction.go`

Add validation constants and `NewTransaction` constructor:

```go
package domain

import (
	"fmt"
	"time"
)

type TransactionType string

const (
	TransactionExpense TransactionType = "expense"
	TransactionIncome  TransactionType = "income"

	MaxCategoryLen    = 100
	MaxDescriptionLen = 500
)

type Transaction struct {
	ID              string
	UserEmail       string
	AmountCents     int64
	Currency        string
	Category        string
	Description     string
	Type            TransactionType
	TransactionDate time.Time
	CreatedAt       time.Time
}

type DailyTotal struct {
	Date         string
	TotalCents   int64
	ExpenseCents int64
	IncomeCents  int64
	Currency     string
}

// NewTransaction creates a validated transaction. Returns error if invariants fail.
func NewTransaction(id, userEmail, currency, category, description string, amountCents int64, txType TransactionType, txDate time.Time) (*Transaction, error) {
	if amountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if len(category) > MaxCategoryLen {
		return nil, fmt.Errorf("category too long (max %d)", MaxCategoryLen)
	}
	if len(description) > MaxDescriptionLen {
		return nil, fmt.Errorf("description too long (max %d)", MaxDescriptionLen)
	}
	if txType != TransactionExpense && txType != TransactionIncome {
		return nil, fmt.Errorf("invalid transaction type: %s", txType)
	}

	return &Transaction{
		ID:              id,
		UserEmail:       userEmail,
		AmountCents:     amountCents,
		Currency:        currency,
		Category:        category,
		Description:     description,
		Type:            txType,
		TransactionDate: txDate,
		CreatedAt:       time.Now().UTC(),
	}, nil
}
```

### 4B. Replace `apps/api/internal/contexts/habits/domain/habit.go`

Add validation and `NewHabit` constructor:

```go
package domain

import (
	"fmt"
	"time"
)

type Frequency string

const (
	FrequencyDaily  Frequency = "daily"
	FrequencyWeekly Frequency = "weekly"
)

// ValidPaletteTokens matches frontend palette tokens.
var ValidPaletteTokens = map[string]bool{
	"red": true, "orange": true, "amber": true, "yellow": true,
	"green": true, "teal": true, "cyan": true, "blue": true,
	"indigo": true, "purple": true, "pink": true, "slate": true,
}

type Habit struct {
	ID            string
	UserEmail     string
	Name          string
	Color         string
	Frequency     Frequency
	TargetPerWeek int
	Archived      bool
	CreatedAt     time.Time
}

type HabitCompletion struct {
	ID            string
	HabitID       string
	CompletedDate time.Time
	CreatedAt     time.Time
}

type HabitWithStatus struct {
	Habit
	CompletedToday bool
	CurrentStreak  int
}

type HabitStreak struct {
	HabitID     string
	Name        string
	Color       string
	Frequency   Frequency
	StreakCount int
}

// NewHabit creates a validated habit. Returns error on invalid invariants.
func NewHabit(id, userEmail, name, color string, freq Frequency, targetPerWeek int) (*Habit, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return nil, fmt.Errorf("name too long (max 200)")
	}
	if color == "" {
		color = "blue"
	}
	if !ValidPaletteTokens[color] {
		return nil, fmt.Errorf("invalid color: %s", color)
	}
	if freq != FrequencyDaily && freq != FrequencyWeekly {
		freq = FrequencyDaily
	}
	if targetPerWeek < 1 {
		targetPerWeek = 1
	}
	if targetPerWeek > 7 {
		targetPerWeek = 7
	}

	return &Habit{
		ID:            id,
		UserEmail:     userEmail,
		Name:          name,
		Color:         color,
		Frequency:     freq,
		TargetPerWeek: targetPerWeek,
		Archived:      false,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
```

### 4C. Update `apps/api/internal/contexts/finance/application/transaction_service.go`

In the `Create` method, replace manual validation + struct construction with `domain.NewTransaction`:

```go
func (s *TransactionService) Create(ctx context.Context, userEmail string, input CreateTransactionInput) (*domain.Transaction, error) {
	tx, err := domain.NewTransaction(
		uuid.New().String(),
		userEmail,
		s.currency,
		input.Category,
		input.Description,
		input.AmountCents,
		input.Type,
		input.TransactionDate,
	)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, tx); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	return tx, nil
}
```

Keep the rest of the file unchanged.

### 4D. Update `apps/api/internal/contexts/habits/application/habit_service.go`

In the `Create` method, replace manual validation + struct construction with `domain.NewHabit`:

```go
func (s *HabitService) Create(ctx context.Context, userEmail string, input CreateHabitInput) (*domain.Habit, error) {
	h, err := domain.NewHabit(
		uuid.New().String(),
		userEmail,
		input.Name,
		input.Color,
		domain.Frequency(input.Frequency),
		input.TargetPerWeek,
	)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveHabit(ctx, h); err != nil {
		return nil, fmt.Errorf("save habit: %w", err)
	}
	return h, nil
}
```

Keep the rest of the file unchanged.

### 4E. Update handlers to remove duplicate validation

**`apps/api/internal/contexts/finance/interfaces/http/finance_handler.go`** — in `Create`:
Remove the inline validation (lines 78-88) since `domain.NewTransaction` now handles it. Keep JSON decode + date parse. Change error from `InternalServerError` to `BadRequest` when `svc.Create` returns validation error:

```go
func (h *FinanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	txDate := time.Now()
	if req.TransactionDate != "" {
		parsed, err := time.Parse("2006-01-02", req.TransactionDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid transactionDate format, use YYYY-MM-DD", err)
			return
		}
		txDate = parsed
	}

	tx, err := h.svc.Create(r.Context(), email, application.CreateTransactionInput{
		AmountCents:     req.AmountCents,
		Category:        req.Category,
		Description:     req.Description,
		Type:            domain.TransactionType(req.Type),
		TransactionDate: txDate,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toTransactionResponse(tx)})
}
```

**`apps/api/internal/contexts/habits/interfaces/http/habit_handler.go`** — in `Create`:
Remove the `name == ""` check. Change error status to `BadRequest`:

```go
func (h *HabitHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	habit, err := h.svc.Create(r.Context(), email, application.CreateHabitInput{
		Name:          req.Name,
		Color:         req.Color,
		Frequency:     req.Frequency,
		TargetPerWeek: req.TargetPerWeek,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toHabitResponse(habit)})
}
```

### 4F. Update domain tests

**`apps/api/internal/contexts/finance/domain/transaction_test.go`** — replace entirely:

```go
package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction_Valid(t *testing.T) {
	tx, err := NewTransaction("tx-1", "user@test.com", "PHP", "food", "Lunch", 150000, TransactionExpense, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(150000), tx.AmountCents)
	assert.Equal(t, TransactionExpense, tx.Type)
	assert.Equal(t, "food", tx.Category)
	assert.Equal(t, "PHP", tx.Currency)
}

func TestNewTransaction_Income(t *testing.T) {
	tx, err := NewTransaction("tx-2", "user@test.com", "PHP", "salary", "", 5000000, TransactionIncome, time.Now())
	require.NoError(t, err)
	assert.Equal(t, TransactionIncome, tx.Type)
}

func TestNewTransaction_ZeroAmount_Error(t *testing.T) {
	_, err := NewTransaction("tx-3", "user@test.com", "PHP", "food", "", 0, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestNewTransaction_NegativeAmount_Error(t *testing.T) {
	_, err := NewTransaction("tx-4", "user@test.com", "PHP", "food", "", -100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestNewTransaction_EmptyCategory_Error(t *testing.T) {
	_, err := NewTransaction("tx-5", "user@test.com", "PHP", "", "", 100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category")
}

func TestNewTransaction_CategoryTooLong_Error(t *testing.T) {
	longCat := make([]byte, MaxCategoryLen+1)
	for i := range longCat {
		longCat[i] = 'a'
	}
	_, err := NewTransaction("tx-6", "user@test.com", "PHP", string(longCat), "", 100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category too long")
}

func TestNewTransaction_DescriptionTooLong_Error(t *testing.T) {
	longDesc := make([]byte, MaxDescriptionLen+1)
	for i := range longDesc {
		longDesc[i] = 'x'
	}
	_, err := NewTransaction("tx-7", "user@test.com", "PHP", "food", string(longDesc), 100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description too long")
}

func TestNewTransaction_InvalidType_Error(t *testing.T) {
	_, err := NewTransaction("tx-8", "user@test.com", "PHP", "food", "", 100, "invalid", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type")
}

func TestTransactionType_Constants(t *testing.T) {
	assert.Equal(t, TransactionType("expense"), TransactionExpense)
	assert.Equal(t, TransactionType("income"), TransactionIncome)
	assert.NotEqual(t, TransactionExpense, TransactionIncome)
}

func TestDailyTotal_DefaultValues(t *testing.T) {
	total := DailyTotal{Date: "2026-01-15", Currency: "PHP"}
	assert.Equal(t, int64(0), total.TotalCents)
	assert.Equal(t, int64(0), total.ExpenseCents)
}
```

**`apps/api/internal/contexts/habits/domain/habit_test.go`** — replace entirely:

```go
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHabit_Valid(t *testing.T) {
	h, err := NewHabit("h-001", "user@test.com", "Exercise", "green", FrequencyDaily, 7)
	require.NoError(t, err)
	assert.Equal(t, "Exercise", h.Name)
	assert.Equal(t, "green", h.Color)
	assert.Equal(t, FrequencyDaily, h.Frequency)
	assert.Equal(t, 7, h.TargetPerWeek)
	assert.False(t, h.Archived)
}

func TestNewHabit_Weekly(t *testing.T) {
	h, err := NewHabit("h-002", "user@test.com", "Learn Go", "indigo", FrequencyWeekly, 3)
	require.NoError(t, err)
	assert.Equal(t, FrequencyWeekly, h.Frequency)
	assert.Equal(t, 3, h.TargetPerWeek)
}

func TestNewHabit_EmptyName_Error(t *testing.T) {
	_, err := NewHabit("h-003", "user@test.com", "", "blue", FrequencyDaily, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestNewHabit_EmptyColor_DefaultsToBlue(t *testing.T) {
	h, err := NewHabit("h-004", "user@test.com", "Test", "", FrequencyDaily, 1)
	require.NoError(t, err)
	assert.Equal(t, "blue", h.Color)
}

func TestNewHabit_InvalidColor_Error(t *testing.T) {
	_, err := NewHabit("h-005", "user@test.com", "Test", "neon-pink", FrequencyDaily, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid color")
}

func TestNewHabit_InvalidFrequency_DefaultsToDaily(t *testing.T) {
	h, err := NewHabit("h-006", "user@test.com", "Test", "blue", "monthly", 1)
	require.NoError(t, err)
	assert.Equal(t, FrequencyDaily, h.Frequency)
}

func TestNewHabit_ZeroTarget_DefaultsToOne(t *testing.T) {
	h, err := NewHabit("h-007", "user@test.com", "Test", "blue", FrequencyDaily, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, h.TargetPerWeek)
}

func TestNewHabit_TargetOverSeven_CapsToSeven(t *testing.T) {
	h, err := NewHabit("h-008", "user@test.com", "Test", "blue", FrequencyWeekly, 10)
	require.NoError(t, err)
	assert.Equal(t, 7, h.TargetPerWeek)
}

func TestFrequency_Constants(t *testing.T) {
	assert.Equal(t, Frequency("daily"), FrequencyDaily)
	assert.Equal(t, Frequency("weekly"), FrequencyWeekly)
}

func TestHabitWithStatus_Embedding(t *testing.T) {
	h, _ := NewHabit("h-001", "user@test.com", "Exercise", "green", FrequencyDaily, 7)
	ws := HabitWithStatus{Habit: *h, CompletedToday: true, CurrentStreak: 5}
	assert.Equal(t, "h-001", ws.Habit.ID)
	assert.True(t, ws.CompletedToday)
	assert.Equal(t, 5, ws.CurrentStreak)
}

func TestHabitStreak_Fields(t *testing.T) {
	s := HabitStreak{
		HabitID: "h-001", Name: "Exercise", Color: "green",
		Frequency: FrequencyDaily, StreakCount: 10,
	}
	assert.Equal(t, 10, s.StreakCount)
}

func TestValidPaletteTokens_Has12(t *testing.T) {
	assert.Len(t, ValidPaletteTokens, 12)
}
```

### 4G. Update service tests to match new domain constructors

The existing service tests **should still pass unchanged** because:
- `TransactionService.Create` now calls `domain.NewTransaction` internally (same validation, same errors)
- `HabitService.Create` now calls `domain.NewHabit` internally (same behavior)

The tests that check error messages (e.g., `"positive"`, `"category"`, `"name is required"`) still match since `NewTransaction`/`NewHabit` use the same error strings.

However, `TestCreate_InvalidFrequency_DefaultsToDaily` — check it still works since `NewHabit` defaults invalid frequency to daily (same as before). ✅

The test `TestCreate_EmptyColor_DefaultsToBlue` — `NewHabit` with `""` → defaults to `"blue"`. ✅

---

## Execution Order

1. Part 1A + 1B (optimistic mutations)
2. Part 2A + 2B (React Hook Form)
3. Part 3A + 3B + 3C + 3D (slog + middleware + main.go)
4. Part 4A-4G (domain validation)

## Validation

```bash
# Frontend
cd apps/web && pnpm typecheck && pnpm test -- --run && pnpm build

# Backend
cd apps/api && go vet ./... && go test ./... -v -count=1
```

## After Validation

```bash
git add .
git commit -m "✨ feat: optimistic mutations, RHF forms, slog logging, domain validation

Part 1: Optimistic mutations with TanStack Query
- useDeleteTransaction: optimistic remove from cache + rollback on error
- useToggleHabit: optimistic completedToday/streak flip
- useArchiveHabit: optimistic remove from list
- useCreateTransaction: invalidate todayTotal with snapshot

Part 2: React Hook Form + zodResolver
- AddExpenseDialog: full form validation with field-level errors
- AddHabitDialog: form validation against CreateHabitSchema

Part 3: Structured logging (slog)
- JSON structured logger (MY_LOG_LEVEL env, default info)
- RequestLogger middleware (method, path, status, duration, request_id)
- Recover middleware (panic → log stack + 500)
- Replaced stdlib log in main.go

Part 4: Domain-layer validation
- NewTransaction constructor validates amount/type/category/description length
- NewHabit constructor validates name/color/frequency/targetPerWeek
- Handlers thinned: removed duplicate validation, return 400 from service errors
- Domain tests rewritten to test constructors directly
- Service tests pass unchanged (same error contract)"

git push
```

## Success Criteria

- `pnpm typecheck` passes
- `pnpm test -- --run` → 102 tests pass
- `pnpm build` succeeds
- `go vet ./...` passes
- `go test ./... -count=1` → all pass
- No runtime behavior changes (same API contract, same error messages)
- Git push succeeds
