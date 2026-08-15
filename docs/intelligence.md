# Intelligence

Confidence-gated LLM analysis for the `my` dashboard: a bounded context that
asks configured model providers for structured suggestions, calibrates every
score from evidence, and never writes finance data itself.

## Boundaries

- **Suggestions only.** The intelligence context produces suggestions with
  calibrated confidence. Finance mutations (transactions, transfers, wallets)
  are committed exclusively by the finance context after explicit user
  confirmation.
- **The external Hermes finance agent is separate** (see `docs/finance-agent.md`).
  This context is the in-dashboard reviewer for imports.

## Enabling

```bash
# Required for encrypted credentials (provider/connector secrets)
export MY_LLM_MASTER_KEY="$(openssl rand -base64 32)"

# Feature gate for analysis endpoints
export MY_LLM_ENABLED=true

# Optional: sandboxed Codex CLI adapter (local, default off)
export MY_LLM_CODEX_PATH="$(command -v codex)"
```

Without a master key the encrypted credential store is unavailable and
credential writes fail closed; provider profiles and status stay usable.
Analysis additionally requires `MY_LLM_ENABLED=true` — the analysis endpoints
answer 503 with a clear message when either condition is missing.

Settings live in the UI under **Settings → AI analysis** (`/settings`); the
import page shows an availability card linking there. `GET
/api/v1/intelligence/status` reports `enabled`, `masterKeyConfigured`,
`providerCount`, and the active provider without revealing secrets.

## Providers

`POST /api/v1/intelligence/providers` with:

| type | base URL | credential | notes |
|---|---|---|---|
| `openai` | optional (default `https://api.openai.com/v1`) | API key | Responses API; Codex-capable model IDs |
| `openai_compatible` | required, https | API key optional | Yunwu and other gateways; `/chat/completions` |
| `ollama` | required (loopback) | none | `allowLocal: true`; `/chat/completions` shape |
| `codex_cli` | ignored | none (Codex home auth) | sandboxed subprocess; see below |

Credentials are encrypted with AES-256-GCM under `MY_LLM_MASTER_KEY` (key
version 1). The API is write-only: plaintext is never returned, never logged.
Provider profiles may set `allowLocal` to permit loopback endpoints; private
and link-local addresses are always rejected (SSRF guard).

### Codex CLI adapter

Runs `codex exec --json --output-last-message` with:

- `--sandbox read-only`, fresh empty working directory (repository never
  accessible), ephemeral session (`--no-rollout-persistence`)
- MCP, hooks, memory, and web search disabled
- environment sanitized of `MY_LLM_MASTER_KEY`, `MY_MCP_TOKEN`, and common
  cloud secrets
- strict timeout and 1 MB output cap

Never used as an automatic fallback for API providers.

## Web search providers

Outbound search for merchant corroboration, configured under
`/api/v1/intelligence/connectors`. Providers are native HTTP adapters with
fixed endpoints — no self-hosted MCP sidecars needed:

| kind | endpoint (fixed) | auth header | free tier (2026-08) |
|---|---|---|---|
| `tavily` | `https://api.tavily.com/search` | `Authorization: Bearer` | 1,000 credits/month, no card |
| `brave` | `https://api.search.brave.com/res/v1/web/search` | `X-Subscription-Token` | ~1,000 searches/month via $5 credit; card required |
| `exa` | `https://api.exa.ai/search` | `x-api-key` (optional — keyless tier) | monthly credits + signup credits |
| `custom_mcp` | user-provided Streamable HTTP endpoint | none / Bearer / `X-Api-Key` | provider-dependent (advanced) |

Behavior:

- **Query-all**: every enabled provider is queried for each weak merchant
  claim (max 10 redacted queries per run; results capped at 5 per provider;
  max 3 providers). One provider failing never fails analysis.
- **Credentials** are encrypted (AES-256-GCM under `MY_LLM_MASTER_KEY`),
  write-only, and travel only in headers — never in URLs. Keyless
  connectors simply omit the header.
- **Corroboration is evidence-based**: a response only counts as web evidence
  when a result's title or URL host actually matches a merchant token
  (snippets and provider relevance scores are ignored). Only matched
  provider names are persisted as evidence — never queries, snippets, or
  responses. Brave's storage terms are respected: no snippets are stored.
- Web confidence ceiling stays `0.85`; transfer ownership never uses web
  corroboration.
- Custom MCP endpoints are SSRF-checked at configuration and every resolved
  IP is re-validated at dial time; redirects are disabled everywhere.
- Queries are redacted (digits — references, phones, amounts — stripped)
  before leaving the device, and each query is sent to every enabled
  provider.

## Confidence

`application/confidence.go` calibrates scores from evidence sources:

| field | user rule | history | web | model-only |
|---|---|---|---|---|
| category | 0.98 | 0.90 (+0.03 with corroboration) | 0.85 | 0.70 |
| merchant | 0.98 | 0.90 (+0.03) | 0.85 | 0.70 |
| relationship | 0.98 | 0.90 (+0.03) | — | 0.70 |
| transfer (ownership) | 0.98 | 0.85 (+0.03) | — | 0.59 |

Buckets shared with the UI: `>= 0.90` preselect (applied, still editable),
`0.60–0.89` review, `< 0.60` unresolved. A model alone can never preselect a
transfer ownership claim.

## Analysis flow

`POST /api/v1/finance/imports/analyses` (≤ 50 rows) → provider call →
strict-JSON parse (unknown references dropped) → optional redacted web
corroboration → calibration → persisted `intelligence_agent_runs` +
`intelligence_suggestions`. The wizard's "Analyze with AI" applies preselect
fields and shows confidence badges for the rest.

## Schema

Migration `012_intelligence.sql` (+ `013_intelligence_search_connectors.sql`):
`intelligence_provider_profiles`, `intelligence_credentials` (ciphertext
only), `intelligence_mcp_connectors` (kind + auth type), `intelligence_agent_runs`,
`intelligence_suggestions`.

## Safety checklist

- Secrets: env-held master key only; ciphertext in DB; write-only API.
- No PDFs, passwords, raw prompts, or chain-of-thought persisted; agent-run
  input summaries hold counts only — never raw descriptions or amounts.
- SSRF guard on every endpoint (URL + DNS resolution + redirect policy);
  https required off-loopback; private/link-local/metadata targets always
  rejected.
- Suggestions never commit finance data; final import confirmation mandatory.
- Model output treated as untrusted; confidence ceilings enforced server-side;
  search decisions use calibrated scores, bounded to 10 redacted queries/run.
