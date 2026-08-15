package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	"github.com/jjspscl/my/internal/contexts/intelligence/infrastructure"
	"github.com/jjspscl/my/internal/contexts/intelligence/infrastructure/providers"
)

// AnalysisService runs confidence-gated analysis over import rows: it asks the
// configured provider for structured suggestions, calibrates every score with
// the ConfidenceService, optionally corroborates merchant names with web
// search connectors, and persists the run + suggestions. It never writes
// finance data itself — it only suggests.
type AnalysisService struct {
	repo       domain.IntelligenceRepository
	settings   *SettingsService
	confidence *ConfidenceService
	search     infrastructure.SearchGateway
	enabled    bool // MY_LLM_ENABLED && master key configured
}

func NewAnalysisService(repo domain.IntelligenceRepository, settings *SettingsService, confidence *ConfidenceService, search infrastructure.SearchGateway, enabled bool) *AnalysisService {
	return &AnalysisService{repo: repo, settings: settings, confidence: confidence, search: search, enabled: enabled}
}

// Enabled reports whether analysis is available at runtime.
func (s *AnalysisService) Enabled() bool { return s.enabled }

const promptVersion = "finance-import-v1"

// MaxAnalysisRows caps one analysis request; the client chunks larger
// statements.
const MaxAnalysisRows = 50

// MaxSearchesPerRun bounds web corroboration cost per analysis run.
const MaxSearchesPerRun = 10

// AnalysisRow is one statement row the agent may analyze.
type AnalysisRow struct {
	SourceReference string
	Description     string
	AmountCents     int64
	Kind            string // deterministic first-pass kind (expense|income|transfer_out|transfer_in)
}

// AnalysisResult is what the HTTP layer returns.
type AnalysisResult struct {
	Run         *domain.AgentRun
	Suggestions []*domain.Suggestion
}

// AnalyzeImport runs the full pipeline synchronously (provider calls are fast
// and bounded) and persists run + suggestions.
func (s *AnalysisService) AnalyzeImport(ctx context.Context, userEmail, scopeID string, rows []AnalysisRow) (*AnalysisResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("LLM analysis is not enabled (set MY_LLM_ENABLED=true and MY_LLM_MASTER_KEY)")
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows to analyze")
	}
	if len(rows) > MaxAnalysisRows {
		return nil, fmt.Errorf("too many rows (max %d per request)", MaxAnalysisRows)
	}
	for _, r := range rows {
		if r.SourceReference == "" || r.Description == "" {
			return nil, fmt.Errorf("every row needs a sourceReference and description")
		}
	}

	provider, credential, err := s.settings.ResolveProvider(ctx, userEmail)
	if err != nil {
		return nil, err
	}
	client, err := s.settings.BuildProvider(ctx, provider, credential)
	if err != nil {
		return nil, err
	}

	run, err := domain.NewAgentRun(uuid.New().String(), userEmail, domain.ScopeFinanceImportAnalysis, scopeID, provider.ID)
	if err != nil {
		return nil, err
	}
	started := time.Now()

	// Search connectors are consulted for every enabled provider (query-all);
	// each connector carries its own encrypted credential, or none for
	// keyless tiers.
	conns := s.settings.ActiveConnectors(ctx, userEmail)

	// Persist as running so interrupted runs are visible as such. The input
	// summary holds ONLY counts and hints — never the raw descriptions or
	// amounts, which stay in the provider request and are not persisted.
	inputSummary, _ := json.Marshal(map[string]any{
		"rows":        len(rows),
		"walletHints": len(walletHints(rows)),
	})
	if err := s.repo.SaveRun(ctx, run); err != nil {
		return nil, err
	}

	var suggestions []*domain.Suggestion
	usage := map[string]int{}
	searchedKeys := []string{}

	complete := func(status string, errMsg string) error {
		run.Status = status
		run.Model = provider.Model
		run.PromptVersion = promptVersion
		run.InputSummary = string(inputSummary)
		run.DurationMS = time.Since(started).Milliseconds()
		run.Error = truncateS(errMsg, domain.MaxRunErrorLen)
		usageJSON, _ := json.Marshal(usage)
		run.TokenUsage = string(usageJSON)
		outJSON, _ := json.Marshal(map[string]any{
			"suggestions": len(suggestions),
			"searched":    len(searchedKeys),
		})
		run.OutputSummary = string(outJSON)
		now := time.Now().UTC()
		run.CompletedAt = &now
		return s.repo.SaveRun(ctx, run)
	}

	chatReq := providers.ChatRequest{
		System:    analysisSystemPrompt,
		User:      analysisUserPrompt(rows),
		MaxTokens: minOf(provider.MaxTokens, 2000),
	}
	res, err := client.Complete(ctx, chatReq)
	if err != nil {
		_ = complete(domain.RunFailed, err.Error())
		return nil, fmt.Errorf("analysis failed: %w", err)
	}
	usage = res.Usage

	parsed, err := parseSuggestions(res.Content, rows)
	if err != nil {
		_ = complete(domain.RunFailed, err.Error())
		return nil, fmt.Errorf("parse provider output: %w", err)
	}

	// Web corroboration for weak merchant claims (redacted queries only).
	// Every enabled provider is queried concurrently for each claim; the
	// decision is based on the CALIBRATED score (model evidence only at this
	// point), never the model's self-rated confidence. A successful call that
	// returns no matching results is NOT evidence.
	if len(conns) > 0 {
		queries := 0
		for i := range parsed {
			if parsed[i].merchant == "" || queries >= MaxSearchesPerRun {
				continue
			}
			modelOnly := s.confidence.Calibrate(domain.FieldMerchant, []domain.Evidence{
				{Source: domain.EvidenceModel},
			})
			if modelOnly >= PreselectThreshold {
				continue
			}
			query := redactQuery(parsed[i].merchant)
			if query == "" {
				continue
			}
			queries++
			matched := s.corroborate(ctx, conns, query)
			if len(matched) > 0 {
				parsed[i].evidence = append(parsed[i].evidence, domain.Evidence{
					Source: domain.EvidenceWeb,
					Detail: "web corroboration: " + strings.Join(matched, ", "),
				})
				searchedKeys = append(searchedKeys, query)
			}
		}
	}

	// Calibrate + persist.
	for _, p := range parsed {
		if p.category != "" {
			suggestions = append(suggestions, s.suggestion(run, p.sourceReference, domain.FieldCategory, p.category, p.categoryConfidence, p.rationale, p.evidence))
		}
		if p.merchant != "" {
			suggestions = append(suggestions, s.suggestion(run, p.sourceReference, domain.FieldMerchant, p.merchant, p.merchantConfidence, p.rationale, p.evidence))
		}
		if p.transferWallet != "" {
			suggestions = append(suggestions, s.suggestion(run, p.sourceReference, domain.FieldTransfer, p.transferWallet, p.transferConfidence, p.rationale, p.evidence))
		}
		if p.refundOf != "" {
			suggestions = append(suggestions, s.suggestion(run, p.sourceReference, domain.FieldRelationship, p.refundOf, p.relationshipConfidence, p.rationale, p.evidence))
		}
	}

	if err := s.repo.SaveSuggestions(ctx, suggestions); err != nil {
		_ = complete(domain.RunFailed, err.Error())
		return nil, err
	}
	if err := complete(domain.RunSucceeded, ""); err != nil {
		return nil, err
	}

	return &AnalysisResult{Run: run, Suggestions: suggestions}, nil
}

// suggestion builds a calibrated, validated suggestion (or skips invalid).
func (s *AnalysisService) suggestion(run *domain.AgentRun, target, field, value string, modelScore float64, rationale string, evidence []domain.Evidence) *domain.Suggestion {
	evidence = append(evidence, domain.Evidence{Source: domain.EvidenceModel, Detail: "model claim"})
	calibrated := s.confidence.Calibrate(field, evidence)
	if calibrated <= 0 {
		return nil
	}
	sugg, err := domain.NewSuggestion(
		uuid.New().String(), run.ID, run.ScopeID, target, field, value, truncateS(rationale, 300),
		calibrated, evidence,
	)
	if err != nil {
		return nil
	}
	_ = modelScore
	return sugg
}

// Get returns a run with its suggestions.
func (s *AnalysisService) Get(ctx context.Context, userEmail, id string) (*AnalysisResult, error) {
	run, err := s.repo.FindRun(ctx, id, userEmail)
	if err != nil {
		return nil, err
	}
	suggestions, err := s.repo.ListSuggestionsByRun(ctx, id)
	if err != nil {
		return nil, err
	}
	return &AnalysisResult{Run: run, Suggestions: suggestions}, nil
}

// ListByScope returns the latest runs for a fingerprint (re-analysis dedupe).
func (s *AnalysisService) ListByScope(ctx context.Context, userEmail, scopeID string, limit int) ([]*domain.AgentRun, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	return s.repo.ListRunsByScope(ctx, userEmail, domain.ScopeFinanceImportAnalysis, scopeID, limit)
}

// Cancel marks a running analysis as cancelled.
func (s *AnalysisService) Cancel(ctx context.Context, userEmail, id string) error {
	run, err := s.repo.FindRun(ctx, id, userEmail)
	if err != nil {
		return err
	}
	if run.Status != domain.RunRunning {
		return nil
	}
	run.Status = domain.RunCancelled
	now := time.Now().UTC()
	run.CompletedAt = &now
	return s.repo.SaveRun(ctx, run)
}

// ---- prompt + parsing ----

const analysisSystemPrompt = `You are a conservative personal-finance analyst reviewing GCash e-wallet statement rows. The user imports a PDF statement; rows may be ambiguous.

Rules:
- Categorize each row into a short category (max 3 words, e.g. "Food", "Transport", "Shopping", "Utilities", "Rent", "Health", "Subscriptions", "Load & Data", "Income", "Transfer", "Unclassified").
- merchant: normalized merchant/payee name when confident, else null.
- transferWallet: ONLY when the counterparty is clearly the user's OWN wallet or bank account (a bank name, e-wallet name, or the user's own phone number pattern). Otherwise null. Never guess ownership.
- refundOf: for refunds/reversals, the likely original description (short), else null.
- Per-field confidence 0..1 reflecting YOUR uncertainty. Be conservative: 0.5-0.6 when unsure.
- rationale: one short sentence per row.

Respond with STRICT JSON only — an array of objects:
[{"sourceReference":"...","category":"...","merchant":"..."|null,"transferWallet":"..."|null,"refundOf":"..."|null,"confidenceCategory":0.0,"confidenceMerchant":0.0,"confidenceTransfer":0.0,"confidenceRelationship":0.0,"rationale":"..."}]
No markdown fences, no prose outside the JSON. Never invent references; one object per input row.`

func analysisUserPrompt(rows []AnalysisRow) string {
	type row struct {
		SourceReference string `json:"sourceReference"`
		Description     string `json:"description"`
		AmountCents     int64  `json:"amountCents"`
		Kind            string `json:"kind"`
	}
	out := make([]row, 0, len(rows))
	for _, r := range rows {
		out = append(out, row{r.SourceReference, r.Description, r.AmountCents, r.Kind})
	}
	payload, _ := json.Marshal(out)
	return "Analyze these rows:\n" + string(payload)
}

type parsedSuggestion struct {
	sourceReference        string
	category               string
	merchant               string
	transferWallet         string
	refundOf               string
	categoryConfidence     float64
	merchantConfidence     float64
	transferConfidence     float64
	relationshipConfidence float64
	rationale              string
	evidence               []domain.Evidence
}

type rawSuggestion struct {
	SourceReference        string  `json:"sourceReference"`
	Category               string  `json:"category"`
	Merchant               *string `json:"merchant"`
	TransferWallet         *string `json:"transferWallet"`
	RefundOf               *string `json:"refundOf"`
	ConfidenceCategory     float64 `json:"confidenceCategory"`
	ConfidenceMerchant     float64 `json:"confidenceMerchant"`
	ConfidenceTransfer     float64 `json:"confidenceTransfer"`
	ConfidenceRelationship float64 `json:"confidenceRelationship"`
	Rationale              string  `json:"rationale"`
}

func parseSuggestions(content string, rows []AnalysisRow) ([]*parsedSuggestion, error) {
	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if !strings.HasPrefix(cleaned, "[") {
		// Try to find the array anywhere in the output.
		if idx := strings.Index(cleaned, "["); idx >= 0 {
			cleaned = cleaned[idx:]
		}
	}

	var raw []rawSuggestion
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, fmt.Errorf("output is not a valid JSON array: %w", err)
	}

	valid := map[string]bool{}
	for _, r := range rows {
		valid[r.SourceReference] = true
	}

	out := make([]*parsedSuggestion, 0, len(raw))
	for _, r := range raw {
		if !valid[r.SourceReference] {
			continue // never trust unknown references
		}
		p := &parsedSuggestion{
			sourceReference:        r.SourceReference,
			category:               strings.TrimSpace(r.Category),
			rationale:              strings.TrimSpace(r.Rationale),
			categoryConfidence:     r.ConfidenceCategory,
			merchantConfidence:     r.ConfidenceMerchant,
			transferConfidence:     r.ConfidenceTransfer,
			relationshipConfidence: r.ConfidenceRelationship,
		}
		if r.Merchant != nil {
			p.merchant = truncateS(strings.TrimSpace(*r.Merchant), 100)
		}
		if r.TransferWallet != nil {
			p.transferWallet = truncateS(strings.TrimSpace(*r.TransferWallet), 100)
		}
		if r.RefundOf != nil {
			p.refundOf = truncateS(strings.TrimSpace(*r.RefundOf), 200)
		}
		out = append(out, p)
	}
	return out, nil
}

var digitRun = regexp.MustCompile(`[0-9]+`)

// redactQuery strips digits (references, phones, amounts) and keeps only
// merchant-like words, so searches never leak personal identifiers.
func redactQuery(merchant string) string {
	q := digitRun.ReplaceAllString(merchant, "")
	q = strings.TrimSpace(strings.Join(strings.Fields(q), " "))
	if len(q) < 2 {
		return ""
	}
	if len(q) > 60 {
		q = q[:60]
	}
	return q
}

// corroborate queries every enabled connector concurrently and returns the
// names of the connectors whose results actually match the merchant tokens.
// Provider failures are swallowed: one provider must never fail analysis,
// and a response that merely exists is not corroboration.
func (s *AnalysisService) corroborate(ctx context.Context, conns []ActiveConnector, query string) []string {
	tokens := merchantTokens(query)
	if len(tokens) == 0 {
		return nil
	}
	var mu sync.Mutex
	matched := map[string]bool{}
	var wg sync.WaitGroup
	for _, ac := range conns {
		wg.Add(1)
		go func(ac ActiveConnector) {
			defer wg.Done()
			res, err := s.search.Search(ctx, ac.Connector, ac.Credential, query)
			if err != nil {
				return
			}
			if hitsMerchant(tokens, res) {
				mu.Lock()
				matched[ac.Connector.Name] = true
				mu.Unlock()
			}
		}(ac)
	}
	wg.Wait()
	names := make([]string, 0, len(matched))
	for name := range matched {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var merchantTokenRe = regexp.MustCompile(`[^a-z0-9]+`)

// merchantStopwords are words too generic to corroborate a merchant claim.
var merchantStopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "via": true, "pay": true,
	"payment": true, "you": true, "your": true, "from": true, "this": true,
	"transaction": true, "ref": true, "bank": true, "transfer": true, "fee": true,
}

// merchantTokens normalizes a redacted query into distinct, meaningful tokens.
// Digit-only tokens are dropped (defense in depth: redaction already strips
// digits, but a token must never look like an identifier).
func merchantTokens(query string) []string {
	words := merchantTokenRe.Split(strings.ToLower(query), -1)
	var out []string
	seen := map[string]bool{}
	for _, w := range words {
		if len(w) < 3 || merchantStopwords[w] || seen[w] || isDigits(w) {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// hitsMerchant reports whether any result's title or URL host contains a
// merchant token. Snippets are deliberately not used for the match decision
// — they are too noisy — and provider-reported relevance is ignored entirely
// (it never raises calibrated confidence by itself).
func hitsMerchant(tokens []string, results []domain.SearchResult) bool {
	for _, r := range results {
		title := strings.ToLower(r.Title)
		host := hostnameOf(r.URL)
		for _, t := range tokens {
			if strings.Contains(title, t) || strings.Contains(host, t) {
				return true
			}
		}
	}
	return false
}

func hostnameOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// walletHints surfaces owned-wallet names when they appear in descriptions so
// the model can match them without guessing.
func walletHints(rows []AnalysisRow) []string {
	var hints []string
	seen := map[string]bool{}
	for _, r := range rows {
		lower := strings.ToLower(r.Description)
		for _, bank := range []string{"BDO", "BPI", "Maya", "ShopeePay", "UnionBank", "GCash", "GrabPay", "Coins.ph", "Metrobank", "RCBC", "PNB", "LandBank", "EastWest", "Security Bank"} {
			if !seen[bank] && strings.Contains(lower, strings.ToLower(bank)) {
				seen[bank] = true
				hints = append(hints, bank)
			}
		}
	}
	return hints
}

func truncateS(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func minOf(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}
