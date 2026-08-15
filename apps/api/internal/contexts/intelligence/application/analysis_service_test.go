package application

import (
	"context"
	"sync"
	"testing"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSearch implements infrastructure.SearchGateway for application tests.
// It records every call and answers with configurable results per connector.
type fakeSearch struct {
	mu      sync.Mutex
	calls   []string // query per call
	results map[string][]domain.SearchResult
	errs    map[string]error
}

func (f *fakeSearch) Search(_ context.Context, connector *domain.MCPConnector, _ string, query string) ([]domain.SearchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, query)
	if err := f.errs[connector.Name]; err != nil {
		return nil, err
	}
	return f.results[connector.Name], nil
}

func (f *fakeSearch) callCount(query string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, q := range f.calls {
		if q == query {
			n++
		}
	}
	return n
}

func TestMerchantTokens(t *testing.T) {
	tokens := merchantTokens("jollibee dela rosa payment")
	assert.Equal(t, []string{"jollibee", "dela", "rosa"}, tokens)

	tokens = merchantTokens("pay 12345 ref 6789")
	assert.Empty(t, tokens)
}

func TestHitsMerchant(t *testing.T) {
	res := []domain.SearchResult{
		{Title: "Jollibee Philippines", URL: "https://www.jollibee.com.ph/", Snippet: "fast food"},
		{Title: "Other", URL: "https://other.example/", Snippet: "jollibee mentions"},
	}
	// Title/URL match is required; a snippet-only mention must not count.
	assert.True(t, hitsMerchant([]string{"jollibee"}, res[:1]))
	assert.False(t, hitsMerchant([]string{"jollibee"}, []domain.SearchResult{{Title: "x", URL: "https://x.example/", Snippet: "Jollibee"}}))

	// Host match counts even when the title is generic.
	assert.True(t, hitsMerchant([]string{"lbcexpress"}, []domain.SearchResult{{Title: "Home", URL: "https://www.lbcexpress.com/", Snippet: ""}}))
}

func TestCorroborateQueriesAllProvidersAndMatches(t *testing.T) {
	fs := &fakeSearch{results: map[string][]domain.SearchResult{
		"Tavily": {{Title: "Jollibee", URL: "https://www.jollibee.com.ph/"}},
		"Brave":  {{Title: "Jollibee Delivery", URL: "https://jollibee.example/"}},
	}}
	svc := &AnalysisService{search: fs}

	names := svc.corroborate(context.Background(), []ActiveConnector{
		{Connector: &domain.MCPConnector{Name: "Tavily"}},
		{Connector: &domain.MCPConnector{Name: "Brave"}},
	}, "jollibee dela rosa")

	assert.Equal(t, []string{"Brave", "Tavily"}, names)
	assert.Equal(t, 2, fs.callCount("jollibee dela rosa"))
}

func TestCorroborateRequiresMatchNotMereSuccess(t *testing.T) {
	fs := &fakeSearch{results: map[string][]domain.SearchResult{
		"Tavily": {{Title: "Unrelated", URL: "https://other.example/", Snippet: "jollibee in snippet"}},
	}}
	svc := &AnalysisService{search: fs}

	names := svc.corroborate(context.Background(), []ActiveConnector{
		{Connector: &domain.MCPConnector{Name: "Tavily"}},
	}, "jollibee")

	assert.Empty(t, names, "a response that does not match must not count as evidence")
}

func TestCorroborateSurvivesProviderFailure(t *testing.T) {
	fs := &fakeSearch{
		results: map[string][]domain.SearchResult{"Brave": {{Title: "Jollibee", URL: "https://jollibee.example/"}}},
		errs:    map[string]error{"Tavily": assert.AnError},
	}
	svc := &AnalysisService{search: fs}

	names := svc.corroborate(context.Background(), []ActiveConnector{
		{Connector: &domain.MCPConnector{Name: "Tavily"}},
		{Connector: &domain.MCPConnector{Name: "Brave"}},
	}, "jollibee")

	assert.Equal(t, []string{"Brave"}, names)
}

func TestCorroborateEmptyTokensSkipsCalls(t *testing.T) {
	fs := &fakeSearch{}
	svc := &AnalysisService{search: fs}
	names := svc.corroborate(context.Background(), []ActiveConnector{
		{Connector: &domain.MCPConnector{Name: "Tavily"}},
	}, "ref")
	assert.Empty(t, names)
	assert.Empty(t, fs.calls)
}

func requireConnector(t *testing.T, svc *SettingsService, email string, in CreateConnectorInput) *domain.MCPConnector {
	t.Helper()
	c, err := svc.CreateConnector(context.Background(), email, in)
	require.NoError(t, err)
	return c
}
