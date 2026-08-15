package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

// SearchGateway performs one normalized search through a connector.
//
// Security posture (shared by every adapter):
//   - native providers use fixed HTTPS endpoints with redirects disabled
//   - custom MCP endpoints are validated at configuration time and every
//     resolved IP is re-checked at dial time (private/loopback/link-local/
//     metadata/multicast never dialed)
//   - credentials travel only in explicit headers, never in URLs
//   - responses are treated as untrusted evidence; only title/URL/snippet
//     are retained, bounded in size
type SearchGateway interface {
	Search(ctx context.Context, connector *domain.MCPConnector, credential, query string) ([]domain.SearchResult, error)
}

// SearchService dispatches a connector to its kind-specific adapter.
type SearchService struct{}

func NewSearchService() *SearchService { return &SearchService{} }

var errNoAdapter = errors.New("no adapter for connector kind")

// Search calls the adapter for the connector's kind. The MCP adapter keeps
// the allowlist gate; native adapters are fixed to a single search tool.
func (s *SearchService) Search(ctx context.Context, connector *domain.MCPConnector, credential, query string) ([]domain.SearchResult, error) {
	switch connector.Kind {
	case domain.ConnectorKindTavily:
		return tavilySearch(ctx, connector, credential, query)
	case domain.ConnectorKindBrave:
		return braveSearch(ctx, connector, credential, query)
	case domain.ConnectorKindExa:
		return exaSearch(ctx, connector, credential, query)
	case domain.ConnectorKindCustomMCP:
		g := &MCPGateway{}
		text, err := g.Call(ctx, connector, credential, ToolCall{
			Name:      connector.Allowlist[0],
			Arguments: map[string]any{"query": query},
		})
		if err != nil {
			return nil, err
		}
		return []domain.SearchResult{{Title: "", URL: "", Snippet: text}}, nil
	default:
		return nil, fmt.Errorf("%w: %s", errNoAdapter, connector.Kind)
	}
}

// testDialAllowLocal permits loopback dials for httptest fixtures. NEVER set
// in production code paths; only unit tests may flip it.
var testDialAllowLocal bool

// newHardenedClient builds an HTTP client that:
//   - validates every resolved IP at dial time (SSRF re-check per connection)
//   - never follows redirects (credentials must not leak to another origin)
//   - applies a strict per-request timeout
func newHardenedClient(timeout time.Duration) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", host, err)
		}
		for _, ip := range ips {
			if err := validateIP(ip.IP, host, testDialAllowLocal); err != nil {
				return nil, err
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
	}
	tr.DisableKeepAlives = true // credentials in transport; avoid idle reuse surprises
	return &http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("redirects are disabled for search endpoints")
		},
	}
}

// maxSearchResults bounds how many normalized hits an adapter retains.
const maxSearchResults = 5

// limitResults truncates and bounds a slice of hits.
func limitResults(res []domain.SearchResult) []domain.SearchResult {
	if len(res) > maxSearchResults {
		res = res[:maxSearchResults]
	}
	for i := range res {
		res[i].Title = truncateUTF8(res[i].Title, 300)
		res[i].URL = truncateUTF8(res[i].URL, 500)
		res[i].Snippet = truncateUTF8(res[i].Snippet, 800)
	}
	return res
}

// truncateUTF8 cuts a string at n bytes without splitting a rune.
func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}

// httpErr classifies provider HTTP failures into readable errors.
func httpErr(provider string, status int, body string) error {
	hint := "provider error"
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		hint = "authentication failed — check the API key"
	case status == http.StatusTooManyRequests:
		hint = "rate limit reached"
	case status == http.StatusPaymentRequired || status == 432 || status == 433:
		hint = "quota exceeded — check the plan"
	case status >= 500:
		hint = "provider server error"
	}
	if body = truncateUTF8(body, 200); body != "" {
		return fmt.Errorf("%s: %d %s: %s", provider, status, hint, body)
	}
	return fmt.Errorf("%s: %d %s", provider, status, hint)
}
