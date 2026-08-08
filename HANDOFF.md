# HANDOFF — Track 1: Data Integrity + Security Fixes

Status: **implemented, corrected, and validated** on branch `fix/track-1-security-integrity`.

Commits:

- `5d2aa3b` — wallet ownership and archive validation for goals/contributions
- `3fe08ff` — configurable secure cookies, dead secret removal, CSRF-protected logout
- `3318da7` — auth cache exclusion, 10-minute API cache, logout purge
- pending fix — single auth mount, router regression tests, runtime boot verification

Initial validation missed a Chi startup panic caused by mounting `/auth` twice. Fixed by mounting auth once with nested protected logout middleware. Router tests now cover construction, auth middleware ordering, public auth reachability, and protected routes. Frontend lint retains 18 pre-existing Fast Refresh warnings.

Scope: backend-heavy. No DB migration. No API contract break. One frontend change.

Implementation details below remain completed-task reference. Startup panic correction is recorded in the follow-up section at end.

---

## Context

Five issues confirmed by reading source:

1. `GoalService` never validates wallet ownership/archive state (data integrity, highest risk)
2. Cookie `Secure` flag derived from `r.Host != "localhost:8080"` in 4 places (fragile)
3. `MY_COOKIE_SECRET` / `MY_CSRF_SECRET` loaded, plumbed, never read (dead security theater)
4. `POST /auth/logout` sits outside the CSRF-protected group (low severity CSRF)
5. Workbox caches `/api/v1/*` for 1h with no logout purge (personal data leak after sign-out)

Decisions already made — do not re-litigate:

- Item 3: **delete** both secrets. Sessions are opaque Redis IDs; CSRF is double-submit random token. Neither needs a server secret.
- Item 1: **shared wallet guard** in the `application` package, used by goal/transfer/transaction services. Do not couple `GoalService` to `TransferService`.
- Item 5: API cache TTL → **10 minutes**, and exclude `/api/v1/auth/` from runtime caching.
- `MY_SECURE_COOKIES` unset → derive from `MY_WEB_URL` scheme. Explicit env value always wins.

---

## Commit 1 — `fix(finance): validate wallet ownership on goals and contributions`

### Problem

`apps/api/internal/contexts/finance/application/goal_service.go` holds only `goalRepo` + `transferRepo`. No wallet repo, so:

- `Create` (line 50) and `Update` (line 63) accept any `TargetWalletID` string — nonexistent, another user's, or archived
- `AddContribution` (line 107) builds a `WalletTransfer` directly at line 119 from `SourceWalletID` → `goal.TargetWalletID`, bypassing every check `TransferService.Create` performs

Meanwhile the same validation logic already exists twice:

- `transfer_service.go:30` — `validateWallet`
- `transaction_service.go:53-62` — inline inside `resolveWallet`

### 1A. Extract shared guard

New file: `apps/api/internal/contexts/finance/application/wallet_guard.go`

Package-level helper (not a method) so all three services can use it:

```go
// ensureUsableWallet verifies a wallet exists, belongs to userEmail, and is not archived.
func ensureUsableWallet(ctx context.Context, repo domain.WalletRepository, userEmail, walletID string) (*domain.Wallet, error)
```

Behavior — preserve existing error strings so current tests and API responses do not change:

- empty `walletID` → `wallet is required`
- repo error / not found → `wallet not found: <id>`
- `wallet.UserEmail != userEmail` → `wallet not found: <id>` (do not leak existence)
- `wallet.ArchivedAt != nil` → `wallet is archived: <id>`
- otherwise return the wallet

Note the two existing implementations differ slightly in message format (`transfer_service.go` includes the ID, `transaction_service.go` does not). Standardize on including the ID. Update any existing test assertions that break — check `transaction_service_test.go` for `assert.Contains` on those strings.

### 1B. Refactor existing callers (behavior-preserving)

- `transfer_service.go`: delete `validateWallet` method, call `ensureUsableWallet` in `Create`
- `transaction_service.go`: `resolveWallet` keeps its default-wallet branch (`walletID == ""` → `FindDefault`), but the explicit-ID branch delegates to `ensureUsableWallet`

Do not change `resolveWallet`'s default-wallet fallback semantics.

### 1C. Add wallet validation to `GoalService`

Struct + constructors gain `walletRepo domain.WalletRepository`:

```go
func NewGoalService(goalRepo domain.GoalRepository, transferRepo domain.TransferRepository, walletRepo domain.WalletRepository) *GoalService
func NewGoalServiceNoTransfer(goalRepo domain.GoalRepository, walletRepo domain.WalletRepository) *GoalService
```

Keep `NewGoalServiceNoTransfer` — grep confirms 9 test call sites and zero production uses. Removing it means rewriting those tests for no benefit.

Validation to add:

- `Create`: validate `input.TargetWalletID` before `domain.NewSavingsGoal`
- `Update`: validate `input.TargetWalletID` before constructing the goal
- `AddContribution`: when `input.SourceWalletID != nil`, validate it; also validate `goal.TargetWalletID` before creating the transfer

Same-wallet guard: `domain.NewWalletTransfer` already rejects `fromWalletID == toWalletID` (`transfer.go:32`). Do not duplicate that check — let the domain error surface. Verify the error path returns cleanly rather than saving a partial contribution.

Ordering requirement: validate **before** any repo write. A contribution must not persist if its transfer is invalid.

### 1D. Wire up

`apps/api/cmd/api/main.go:84` → pass `walletRepo` (already in scope at line 69).

### 1E. Tests

`goal_service_test.go` currently has `newMockGoalRepo` but no wallet mock. A `mockWalletRepo` already exists in `transaction_service_test.go:16` — same package, so **reuse it**, do not redefine.

Update all 9 `NewGoalServiceNoTransfer` call sites (lines 87, 102, 115, 124, 144, 169, 191, 209, 226) to pass a wallet repo seeded with the wallet IDs those tests reference (`w-1`, `w-2`), owned by `user@test.com`, unarchived. Existing assertions must still pass.

New table-driven cases:

- create rejects unknown wallet
- create rejects wallet owned by another user
- create rejects archived wallet
- update rejects unknown / foreign / archived wallet
- contribution rejects foreign source wallet
- contribution rejects archived source wallet
- contribution with valid source wallet still creates a transfer and persists (needs `transferRepo`, so use `NewGoalService`)
- contribution where source == goal target surfaces the domain error and persists nothing

### Validate

```
mise run test
mise run lint
```

---

## Commit 2 — `refactor(access): derive cookie Secure from config, drop unused secrets`

Covers items 2, 3, 4. Same files, one coherent change.

### 2A. Config

`apps/api/internal/platform/config/config.go`:

- delete `CookieSecret` and `CSRFSecret` fields (lines 19-20) and their `os.Getenv` loads (lines 43-44)
- add `SecureCookies bool`

Resolution logic:

- if `MY_SECURE_COOKIES` is set, parse it (`strconv.ParseBool`); invalid value → return an error from `Load()`, do not silently default
- if unset, derive: `strings.HasPrefix(WebURL, "https://")`

Rationale for the derived default: production must set `MY_WEB_URL` correctly anyway, since magic-link emails are built from it. A misconfigured `MY_WEB_URL` breaks login before it weakens cookies, so this is not a silent fail-open.

### 2B. Auth handler

`apps/api/internal/contexts/access/interfaces/http/auth_handler.go`:

- replace `cookieSecret` / `csrfSecret` fields with `secureCookies bool`
- `NewAuthHandler(svc *application.AuthService, secureCookies bool, sessionTTL time.Duration) *AuthHandler`
- replace all 4 `Secure: r.Host != "localhost:8080"` occurrences (lines 94, 106, 133, 142) with `Secure: h.secureCookies`
- replace hardcoded `MaxAge: int((7 * 24 * time.Hour).Seconds())` (lines 96, 108) with `int(h.sessionTTL.Seconds())` — currently duplicates the `MY_SESSION_TTL` default and drifts if changed

Leave the two `MaxAge: -1` clearing cookies as-is.

### 2C. Logout CSRF

`apps/api/cmd/api/main.go`: `/auth/*` mounts publicly at line 113, so logout has no CSRF check.

Split the auth routes:

- public: `POST /auth/magic-link`, `POST /auth/verify`, `GET /auth/me`
- protected: `POST /auth/logout` — inside the existing `RequireAuth` + `CSRFProtect` group

Keep middleware wiring in `main.go` rather than nesting a group inside `authHandler.Routes`. Suggested shape: `AuthHandler` exposes `PublicRoutes(r)` and `ProtectedRoutes(r)`.

Note on `/auth/me`: it reads the session cookie directly (line 149) rather than using `RequireAuth`. Leave that as-is — it must stay public so the frontend can probe auth state, and it is a GET.

**No frontend change needed.** Verified: `shared/api/client.ts:31-33` sets `X-CSRF-Token` on all non-GET/HEAD/OPTIONS requests, and `auth.api.ts:25` routes logout through `apiClient` with `POST`. Confirm this still holds before shipping.

### 2D. Env docs

`.env.example` lines 20-21: delete `MY_COOKIE_SECRET` and `MY_CSRF_SECRET`. Add under the auth section:

```
# Cookie security — unset derives from MY_WEB_URL scheme (https -> true)
# MY_SECURE_COOKIES=false
```

Also check the local `.env` (gitignored, do not commit): the removed vars become inert, harmless. Do not edit it unless login breaks.

### 2E. Tests

No `auth_handler_test.go` or `config_test.go` exists yet. Create both:

`config_test.go`:
- `MY_SECURE_COOKIES=true` / `false` respected
- invalid value returns error
- unset + `https://` web URL → true
- unset + `http://` web URL → false

`auth_handler_test.go` (use `httptest`):
- verify sets `my_session` HttpOnly and `my_csrf` non-HttpOnly
- `Secure` follows the config flag in both states
- cookie `MaxAge` matches configured session TTL

Logout CSRF coverage: an `httptest` route-level test asserting logout is rejected without `X-CSRF-Token` and accepted with a matching cookie + header. Existing CSRF middleware tests cover the middleware itself; this covers the wiring.

### Validate

```
mise run test
mise run lint
```

### Manual smoke test — REQUIRED

Automated tests cannot catch a dropped cookie. After this commit:

1. `mise run dev`
2. request a magic link, open Mailpit at `http://localhost:8025`
3. click through, confirm the session sticks and the dashboard loads
4. sign out, confirm redirect to `/login` and that a refresh does not restore the session

If login silently fails, `SecureCookies` resolved to `true` over HTTP. Set `MY_SECURE_COOKIES=false` in local `.env`.

---

## Commit 3 — `fix(web): purge API cache on logout, exclude auth from runtime cache`

### Problem

`apps/web/vite.config.ts:26-41` caches every `/api/v1/*` response for 1 hour, up to 100 entries. `useLogout` (`features/auth/hooks/use-auth.ts:46`) calls `queryClient.clear()`, which only clears TanStack Query's in-memory cache. The Workbox `api-cache` in CacheStorage survives sign-out, holding finance and habit data.

### 3A. Narrow the cache

`vite.config.ts`:

- `urlPattern` must exclude `/api/v1/auth/` so a cached `me` response cannot mislead the router's auth guard. Current pattern is `/^\/api\/v1\//`. Use a negative lookahead (`/^\/api\/v1\/(?!auth\/)/`) or a URL-predicate function — pick whichever reads cleaner.
- `maxAgeSeconds`: `60 * 60` → `60 * 10`

### 3B. Purge on logout

`features/auth/hooks/use-auth.ts`, in the logout `onSuccess` alongside the existing `queryClient.clear()`:

- delete the `api-cache` CacheStorage entry
- guard for absent `caches` (jsdom, non-secure contexts) — must not throw
- do not block navigation on it; a failed purge should still sign the user out

Put the purge in a small named helper so it is testable. `shared/sync/` is the right neighborhood if you prefer it out of the auth feature — your call, but keep the cache name in one place.

### 3C. Test

Unit test asserting logout attempts deletion of `api-cache`, with a stubbed `caches`, plus a case where `caches` is undefined and logout still succeeds.

### Validate

```
mise run test
mise run typecheck
mise run lint
mise run build
```

---

## Docs to update (fold into commit 3, or a 4th docs commit if it grows)

- `README.md` — env var changes: secrets removed, `MY_SECURE_COOKIES` added
- `docs/offline-sync.md` — API cache TTL now 10 minutes, auth endpoints excluded, cache purged on logout. This doc currently describes the cache at `:15-16` without mentioning purge behavior.
- `docs/backend-ddd.md` — check whether it documents cookie secrets or the `Secure` heuristic; correct if so

Do not touch `ROADMAP.md` in this track — it needs a separate cleanup pass.

---

## Out of scope for Track 1

Do not start these, even if they look adjacent:

- offline queue idempotency keys
- `/api/v1/sync/*` backend
- rate limiting, CORS
- production Docker image
- untracking the stray `apps/api/api` binary
- `ROADMAP.md` refresh

---

## Commit workflow

Before the first commit, load the `git-commit-and-push` skill and follow it for message format and validation gates.

Working tree already has unrelated uncommitted changes:

```
 M AGENTS.md
 M docs/opencode.md
 M opencode.jsonc
?? .opencode/skills/pull-requests/
```

These are OpenCode workflow config, unrelated to Track 1. **Do not include them** in the three commits. Stage specific files, never `git add .`.

Branch: create a feature branch, do not commit directly to `main`.

---

## Report back with

- files created/changed per commit
- validation output for `test`, `lint`, `typecheck`, `build`
- manual login smoke test result (commit 2) — state explicitly whether it was run
- any existing test assertions that had to change, and why
- anything that turned out different from this handoff's assumptions

---

## Follow-up — Chi auth mount panic

The first implementation mounted `/api/v1/auth` twice: once for public routes and once inside the protected route group. Chi panicked during startup with:

```
chi: attempting to Mount() a handler on an existing path, '/auth'
```

Correction: `cmd/api/router.go` now mounts `/auth` once and nests protected logout middleware inside that mount. `cmd/api/router_test.go` constructs the complete router and checks auth/public/protected route behavior, preventing regression.

Manual `mise run dev` startup was rerun by the user and confirmed working.
