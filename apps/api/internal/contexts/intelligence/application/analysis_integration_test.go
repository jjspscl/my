package application

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	infra "github.com/jjspscl/my/internal/contexts/intelligence/infrastructure"
	"github.com/jjspscl/my/internal/platform/database"
	"github.com/jjspscl/my/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newIntelDB(t *testing.T) *infra.IntelligenceRepoLibSQL {
	t.Helper()
	path := filepath.Join(t.TempDir(), "intel.db")
	db, err := database.Open("file:" + path)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.Migrate(db, migrations.FS, quiet); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return infra.NewIntelligenceRepoLibSQL(db)
}

// fakeOpenAICompatible is a minimal chat-completions server returning a fixed
// structured analysis.
func fakeOpenAICompatible(t *testing.T, suggestionsJSON string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected request body")
		}
		if r.Header.Get("Authorization") != "Bearer sk-fake" {
			t.Error("expected bearer auth")
		}
		resp, _ := json.Marshal(map[string]any{
			"model": "fake-model",
			"choices": []map[string]any{
				{"message": map[string]any{"content": suggestionsJSON}},
			},
			"usage": map[string]int{"prompt_tokens": 30, "completion_tokens": 20},
		})
		_, _ = w.Write(resp)
	}))
}

func TestAnalysisEndToEndWithRealProvider(t *testing.T) {
	srv := fakeOpenAICompatible(t, `[
	  {"sourceReference":"REF1","category":"Food","merchant":"Jollibee","transferWallet":null,"refundOf":null,
	   "confidenceCategory":0.8,"confidenceMerchant":0.75,"confidenceTransfer":0.5,"confidenceRelationship":0.5,
	   "rationale":"fast food description"},
	  {"sourceReference":"REF2","category":"Transfer","merchant":"BDO Unibank","transferWallet":"BDO","refundOf":null,
	   "confidenceCategory":0.9,"confidenceMerchant":0.8,"confidenceTransfer":0.7,"confidenceRelationship":0.5,
	   "rationale":"bank transfer"}
	]`)
	defer srv.Close()

	ctx := context.Background()
	repo := newIntelDB(t)
	box, err := infra.NewSecretBox("test-master-key-32-bytes-minimum!")
	require.NoError(t, err)
	settings := NewSettingsService(repo, box, RuntimeConfig{LLMEnabled: true})
	analysis := NewAnalysisService(repo, settings, NewConfidenceService(), infra.NewMCPGateway(), true)

	// Configure a provider + credential (the fake endpoint is loopback, so
	// AllowLocal must be true).
	profile, err := settings.CreateProvider(ctx, "you@example.com", CreateProviderInput{
		Name:         "Fake",
		ProviderType: domain.ProviderOpenAICompatible,
		BaseURL:      srv.URL,
		Model:        "fake-model",
		AllowLocal:   true,
		APIKey:       "sk-fake",
	})
	require.NoError(t, err)
	require.NotEmpty(t, profile.ID)

	result, err := analysis.AnalyzeImport(ctx, "you@example.com", "fingerprint-1", []AnalysisRow{
		{SourceReference: "REF1", Description: "Payment to Jollibee", AmountCents: 15000, Kind: "expense"},
		{SourceReference: "REF2", Description: "BDO Bank Transfer", AmountCents: 50000, Kind: "transfer_out"},
	})
	require.NoError(t, err)
	require.Equal(t, domain.RunSucceeded, result.Run.Status)

	// 4 suggestions (category+merchant for each; REF2 also transfer).
	require.Len(t, result.Suggestions, 5)

	// Category with model-only evidence caps at 0.70 -> review bucket.
	catFood := findSuggestion(result.Suggestions, "REF1", domain.FieldCategory)
	require.NotNil(t, catFood)
	assert.Equal(t, 0.70, catFood.Confidence)
	assert.Equal(t, "Food", catFood.Value)

	// Transfer ownership stays below preselect (0.59 + 0.03? no: transfer
	// model-only = 0.59).
	transfer := findSuggestion(result.Suggestions, "REF2", domain.FieldTransfer)
	require.NotNil(t, transfer)
	assert.Equal(t, 0.59, transfer.Confidence)
	assert.Equal(t, "BDO", transfer.Value)

	// Persisted: reload by scope and confirm the run survived.
	runs, err := analysis.ListByScope(ctx, "you@example.com", "fingerprint-1", 5)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "fake-model", runs[0].Model)

	// Privacy contract: the input summary never contains raw descriptions or
	// amounts — only counts and hints.
	assert.NotContains(t, runs[0].InputSummary, "Jollibee")
	assert.NotContains(t, runs[0].InputSummary, "BDO Bank Transfer")
	assert.Contains(t, runs[0].InputSummary, "\"rows\":2")

	got, err := analysis.Get(ctx, "you@example.com", result.Run.ID)
	require.NoError(t, err)
	assert.Len(t, got.Suggestions, 5)

	// Cancelling a completed run is a no-op.
	require.NoError(t, analysis.Cancel(ctx, "you@example.com", result.Run.ID))
}

func TestAnalysisRejectsUnknownReferences(t *testing.T) {
	srv := fakeOpenAICompatible(t, `[
	  {"sourceReference":"HALLUCINATED","category":"Food","merchant":null,"transferWallet":null,"refundOf":null,
	   "confidenceCategory":0.9,"confidenceMerchant":0.5,"confidenceTransfer":0.5,"confidenceRelationship":0.5,"rationale":"x"}
	]`)
	defer srv.Close()

	ctx := context.Background()
	repo := newIntelDB(t)
	box, _ := infra.NewSecretBox("test-master-key-32-bytes-minimum!")
	settings := NewSettingsService(repo, box, RuntimeConfig{LLMEnabled: true})
	analysis := NewAnalysisService(repo, settings, NewConfidenceService(), infra.NewMCPGateway(), true)

	_, err := settings.CreateProvider(ctx, "you@example.com", CreateProviderInput{
		Name: "Fake", ProviderType: domain.ProviderOpenAICompatible, BaseURL: srv.URL,
		Model: "fake-model", AllowLocal: true, APIKey: "sk-fake",
	})
	require.NoError(t, err)

	result, err := analysis.AnalyzeImport(ctx, "you@example.com", "fingerprint-2", []AnalysisRow{
		{SourceReference: "REAL1", Description: "Payment to Jollibee", AmountCents: 15000, Kind: "expense"},
	})
	require.NoError(t, err)
	require.Equal(t, domain.RunSucceeded, result.Run.Status)
	// Hallucinated reference must be dropped.
	assert.Empty(t, result.Suggestions)
}

func TestAnalysisFailsClosedWithoutProvider(t *testing.T) {
	ctx := context.Background()
	repo := newIntelDB(t)
	box, _ := infra.NewSecretBox("test-master-key-32-bytes-minimum!")
	settings := NewSettingsService(repo, box, RuntimeConfig{LLMEnabled: true})
	analysis := NewAnalysisService(repo, settings, NewConfidenceService(), infra.NewMCPGateway(), true)

	_, err := analysis.AnalyzeImport(ctx, "you@example.com", "fingerprint-3", []AnalysisRow{
		{SourceReference: "REF1", Description: "x", AmountCents: 1},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no enabled provider")
}

func findSuggestion(list []*domain.Suggestion, target, field string) *domain.Suggestion {
	for _, s := range list {
		if s.TargetKey == target && s.Field == field {
			return s
		}
	}
	return nil
}
