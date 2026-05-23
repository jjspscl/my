# ✅ COMPLETED

# HANDOFF — Tests + DX (Air Hot-Reload + Thorough Test Coverage)

## Part A: Air Hot-Reload with Browser Signal

### 1. Install Air
Add to `mise.toml` tools section:
```toml
[tools]
go = "latest"
node = "lts"
pnpm = "latest"
"go:github.com/air-verse/air" = "latest"
```

### 2. Create `apps/api/.air.toml`
```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/main ./cmd/api"
entrypoint = ["./tmp/main"]
include_ext = ["go", "sql"]
exclude_dir = ["tmp", "vendor", "scripts"]
exclude_regex = ["_test\\.go$"]
delay = 500

[log]
time = true

[color]
main = "cyan"
watcher = "green"
build = "yellow"

[proxy]
enabled = true
proxy_port = 8090
app_port = 8080
app_start_timeout = 5000
```

### 3. Update `mise.toml` dev:api task
Change from:
```toml
[tasks."dev:api"]
description = "Run Go API dev server (ensures Redis is running)"
run = [
  "docker start my-redis 2>/dev/null || docker run -d --name my-redis -p 6379:6379 redis:7-alpine",
  "go run ./cmd/api"
]
dir = "apps/api"
```
To:
```toml
[tasks."dev:api"]
description = "Run Go API dev server with hot-reload (ensures Redis is running)"
run = [
  "docker start my-redis 2>/dev/null || docker run -d --name my-redis -p 6379:6379 redis:7-alpine",
  "air"
]
dir = "apps/api"
```

### 4. Update `.gitignore`
Add at bottom:
```
# Air tmp
apps/api/tmp/
```

### 5. Notes
- Frontend dev: Vite `:5173` → proxy `/api` to Go `:8080` (unchanged)
- Embedded SPA testing: browser hits Air proxy `:8090` (auto-reloads on Go rebuild)
- These are separate — Vite HMR for frontend, Air reload for backend-only changes

---

## Part B: Go Tests (Thorough Coverage)

### Dependencies
Add `testify` for assertions:
```bash
cd apps/api && go get github.com/stretchr/testify
```

### Test Files to Create

#### B1. `internal/shared/response/response_test.go`
Test:
- `WriteJSON` writes correct status, Content-Type header, JSON body with `ok: true` envelope
- `WriteError` writes error status, `ok: false`, sanitized error message (not internal details)
- `WriteError` with different status codes (400, 401, 404, 500)

#### B2. `internal/contexts/access/domain/token_test.go`
Test:
- `NewMagicToken` generates valid UUID-format token
- `NewMagicToken` sets correct email and expiry (15 min from now)
- `IsExpired()` returns false for fresh token, true for expired

#### B3. `internal/contexts/access/application/auth_service_test.go`
Create mock: `mockTokenRepo` implementing `TokenRepository` interface.
Test:
- `RequestMagicLink` stores token and sends email (mock mail sender)
- `VerifyToken` with valid token returns email, deletes token
- `VerifyToken` with expired token returns error
- `VerifyToken` with non-existent token returns error

#### B4. `internal/contexts/finance/domain/transaction_test.go`
Test:
- Valid transaction creation (all fields populated)
- Validation: amount_cents must be > 0
- Validation: type must be "expense" or "income"
- Validation: category cannot be empty
- Validation: transaction_date must be valid date

#### B5. `internal/contexts/finance/application/transaction_service_test.go`
Create mock: `mockTransactionRepo` implementing `TransactionRepository`.
Test:
- `CreateTransaction` with valid input stores and returns transaction
- `CreateTransaction` with invalid input returns validation error
- `GetTodayTotal` aggregates correctly (multiple expenses + income)
- `GetTodayTotal` returns zeros when no transactions
- `ListByDateRange` passes correct params to repo
- `DeleteTransaction` calls repo delete

#### B6. `internal/contexts/finance/infrastructure/datetime_test.go`
Extract `parseDatetime` into its own file or test it where it lives.
Test:
- Parses `"2026-01-15"` format
- Parses `"2026-01-15 14:30:00"` format
- Parses `"2026-01-15T14:30:00Z"` format
- Parses RFC3339 `"2026-01-15T14:30:00+08:00"` format
- Returns error for invalid string

#### B7. `internal/contexts/habits/domain/habit_test.go`
Test:
- Valid habit creation
- Validation: name cannot be empty
- Validation: color must be valid palette token
- Validation: frequency must be "daily" or "weekly"
- Validation: target_per_week must be 1-7

#### B8. `internal/contexts/habits/application/habit_service_test.go`
Create mock: `mockHabitRepo` implementing `HabitRepository`.
Test:
- `CreateHabit` stores and returns
- `ListWithStatus` returns habits with completedToday + currentStreak
- `ToggleCompletion` when not completed → creates completion, returns `completed: true`
- `ToggleCompletion` when already completed → deletes completion, returns `completed: false`
- `ArchiveHabit` sets archived flag
- `GetAllCompletionsGrouped` returns correct map structure

#### B9. `internal/contexts/habits/infrastructure/datetime_test.go`
Same as B6 but for `parseDatetimeHabit`.

#### B10. `internal/shared/middleware/auth_test.go`
Test (need `httptest` + mock session store):
- Request with valid session cookie → sets email in context, calls next
- Request with invalid/expired session → returns 401 JSON
- Request with no session cookie → returns 401 JSON

#### B11. `internal/shared/middleware/csrf_test.go`
Test:
- GET request passes through without CSRF check
- POST with matching cookie + header → passes
- POST with mismatched cookie + header → returns 403
- POST with missing CSRF header → returns 403

### Mock Pattern
Each mock goes in the same `_test.go` file. Example:
```go
type mockHabitRepo struct {
    habits      []domain.Habit
    completions []domain.HabitCompletion
    createFn    func(h *domain.Habit) error
    // etc.
}
```

### Validation
```bash
cd apps/api && go test ./... -v -count=1
```
All tests must pass.

---

## Part C: Frontend Tests (Vitest — Thorough)

### Dependencies
```bash
cd apps/web && pnpm add -D vitest-fetch-mock
```

### Setup: `apps/web/src/test/setup.ts`
```ts
import createFetchMock from 'vitest-fetch-mock'
import { vi } from 'vitest'

const fetchMocker = createFetchMock(vi)
fetchMocker.enableMocks()
```

Update `apps/web/vitest.config.ts` (or `vite.config.ts` vitest section) to include:
```ts
test: {
  setupFiles: ['./src/test/setup.ts'],
  environment: 'jsdom',
}
```

### Test Files to Create

#### C1. `src/shared/theme/palette.test.ts`
- `PALETTE_TOKENS` has 12 entries
- `paletteVar('green')` → `'var(--palette-green)'`
- `paletteBgStyle('blue')` → `{ backgroundColor: 'var(--palette-blue)' }`
- `paletteTextClass('red')` → `'text-[var(--palette-red)]'`
- Every token in `DEFAULT_PALETTE` has a hex value starting with `#`

#### C2. `src/shared/api/client.test.ts`
- Successful GET: unwraps envelope `{ ok: true, data: {...} }` → returns data
- Failed request (4xx): throws with error message from envelope
- Network error: throws appropriate error
- CSRF: POST includes `X-CSRF-Token` header from cookie
- Credentials: all requests include `credentials: 'include'`

#### C3. `src/features/auth/schemas/auth.schemas.test.ts`
- Valid login schema parse: `{ email: "test@example.com" }`
- Invalid login schema: missing email, invalid email format
- Verify schema: `{ token: "abc-123" }`
- Me response schema: `{ email: "user@test.com" }`

#### C4. `src/features/finance/schemas/transaction.schemas.test.ts`
- Valid transaction parses (all fields present)
- `amountCents` must be number > 0
- `type` must be "expense" | "income"
- `transactionDate` must be string (YYYY-MM-DD)
- Create input schema validates required fields
- Today total schema: `{ totalExpense, totalIncome, net }`

#### C5. `src/features/habits/schemas/habit.schemas.test.ts`
- Valid habit parses
- `color` accepts all palette tokens
- `frequency` only "daily" | "weekly"
- `completedToday` is boolean
- `currentStreak` is number
- `CompletionsMapSchema` parses `{ completions: { "2026-01-01": ["id1"] }, totalHabits: 5 }`

#### C6. `src/features/dashboard/lib/widget-registry.test.ts`
- Register widget → getWidget returns it
- Register same ID twice → no error (idempotent)
- getWidgets returns all registered
- getWidget with unknown ID → returns undefined
- Widget has required fields: id, title, component, size

#### C7. `src/features/habits/api/habits.keys.test.ts`
- Query keys are arrays
- Key factory produces unique keys per param (e.g., `habitKeys.list()` ≠ `habitKeys.completions(id)`)

#### C8. `src/features/finance/api/finance.keys.test.ts`
- Same pattern as C7 for finance query keys

### Validation
```bash
cd apps/web && pnpm test -- --run
```
All tests must pass.

---

## Execution Order
1. Part A (Air) — quick, unblocks DX
2. Part B (Go tests) — heavier, 11 test files
3. Part C (Frontend tests) — 8 test files

## Success Criteria
- `mise run dev:api` starts Air, hot-reloads on .go change, proxy on :8090
- `go test ./... -count=1` → ALL PASS (11+ test files)
- `pnpm --filter web test -- --run` → ALL PASS (8 test files)
- `pnpm --filter web build` → still succeeds
- No changes to existing non-test source files (except mise.toml, .gitignore, vitest config)
