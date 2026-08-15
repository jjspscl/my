package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConnectorWith(kind string) *domain.MCPConnector {
	return &domain.MCPConnector{
		ID: "c1", UserEmail: "you@example.com", Name: "Test " + kind,
		Kind: kind, AuthType: domain.ConnectorAuthBearer,
		Allowlist: []string{"web_search"}, Timeout: 5 * time.Second,
	}
}

// searchFixture spins up a fake provider endpoint and routes the adapter at
// it, returning the server (for body assertions) and a search func.
func searchFixture(t *testing.T, kind string, handler http.HandlerFunc) (func(query string) ([]domain.SearchResult, error), *httptest.Server) {
	t.Helper()
	testDialAllowLocal = true
	t.Cleanup(func() { testDialAllowLocal = false })
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	svc := NewSearchService()
	conn := testConnectorWith(kind)
	return func(query string) ([]domain.SearchResult, error) {
		return svc.Search(context.Background(), conn, "secret", query)
	}, srv
}

func TestTavilyAdapterAuthAndParse(t *testing.T) {
	old := tavilyEndpoint
	tavilyEndpoint = ""
	t.Cleanup(func() { tavilyEndpoint = old })

	search, srv := searchFixture(t, domain.ConnectorKindTavily, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		// Credential must never ride in the URL.
		assert.NotContains(t, r.URL.RawQuery, "tvly")
		_, _ = w.Write([]byte(`{"results":[{"title":"Jollibee Official","url":"https://www.jollibee.com.ph/","content":"menu and branches"}]}`))
	})
	tavilyEndpoint = srv.URL + "/search"

	res, err := search("jollibee")
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Jollibee Official", res[0].Title)
	assert.Equal(t, "https://www.jollibee.com.ph/", res[0].URL)
}

func TestTavilyAdapterQuotaError(t *testing.T) {
	old := tavilyEndpoint
	tavilyEndpoint = ""
	t.Cleanup(func() { tavilyEndpoint = old })

	search, srv := searchFixture(t, domain.ConnectorKindTavily, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(432)
		_, _ = w.Write([]byte(`{"detail":{"error":"plan limit"}}`))
	})
	tavilyEndpoint = srv.URL + "/search"

	_, err := search("jollibee")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota")
}

func TestBraveAdapterAuthAndParse(t *testing.T) {
	old := braveEndpoint
	braveEndpoint = ""
	t.Cleanup(func() { braveEndpoint = old })

	search, srv := searchFixture(t, domain.ConnectorKindBrave, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "secret", r.Header.Get("X-Subscription-Token"))
		assert.Equal(t, "5", r.URL.Query().Get("count"))
		assert.Equal(t, "moderate", r.URL.Query().Get("safesearch"))
		assert.NotContains(t, r.URL.RawQuery, "secret", "credential must never ride in the URL")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"SM Malls","url":"https://www.smsupermalls.com/","description":"shopping"}]}}`))
	})
	braveEndpoint = srv.URL + "/web/search"

	res, err := search("sm malls")
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "SM Malls", res[0].Title)
}

func TestExaAdapterAuthAndKeyless(t *testing.T) {
	old := exaEndpoint
	exaEndpoint = ""
	t.Cleanup(func() { exaEndpoint = old })

	gotKey := ""
	search, srv := searchFixture(t, domain.ConnectorKindExa, func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		_, _ = w.Write([]byte(`{"results":[{"title":"LBC","url":"https://www.lbcexpress.com/","text":"courier"}]}`))
	})
	exaEndpoint = srv.URL + "/search"

	res, err := search("lbc")
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "secret", gotKey)

	// Keyless: no header, no error.
	svc := NewSearchService()
	conn := testConnectorWith(domain.ConnectorKindExa)
	res, err = svc.Search(context.Background(), conn, "", "lbc")
	require.NoError(t, err)
	assert.Len(t, res, 1)
}

func TestAdapterResultLimitsAndTruncation(t *testing.T) {
	old := exaEndpoint
	exaEndpoint = ""
	t.Cleanup(func() { exaEndpoint = old })

	search, srv := searchFixture(t, domain.ConnectorKindExa, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[` + strings.Repeat(`{"title":"T","url":"https://x.example/","text":"s"},`, 20) + `{"title":"T","url":"https://y.example/","text":"s"}]}`))
	})
	exaEndpoint = srv.URL + "/search"

	res, err := search("query")
	require.NoError(t, err)
	assert.Len(t, res, maxSearchResults)
}

func TestSearchServiceRejectsUnknownKind(t *testing.T) {
	svc := NewSearchService()
	conn := testConnectorWith("unknown-kind")
	_, err := svc.Search(context.Background(), conn, "", "q")
	require.Error(t, err)
}

func TestHardenedClientBlocksPrivateDial(t *testing.T) {
	// A custom MCP connector pointing at a private IP must fail at dial time.
	client := newHardenedClient(3 * time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://10.0.0.1/mcp", nil)
	require.NoError(t, err)
	_, err = client.Do(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private or link-local")
}

func TestHardenedClientRejectsRedirects(t *testing.T) {
	client := newHardenedClient(3 * time.Second)
	err := client.CheckRedirect(&http.Request{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirects are disabled")
}
