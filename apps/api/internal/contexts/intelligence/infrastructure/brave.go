package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

var braveEndpoint = "https://api.search.brave.com/res/v1/web/search"

// braveSearch calls the Brave Web Search API with the X-Subscription-Token
// header. Only title/URL/description are retained; Brave's storage terms are
// respected by persisting no snippets at all.
func braveSearch(ctx context.Context, c *domain.MCPConnector, credential, query string) ([]domain.SearchResult, error) {
	u, _ := url.Parse(braveEndpoint)
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprint(maxSearchResults))
	q.Set("safesearch", "moderate")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if credential != "" {
		req.Header.Set("X-Subscription-Token", credential)
	}

	resp, err := newHardenedClient(c.Timeout).Do(req)
	if err != nil {
		return nil, fmt.Errorf("brave: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("brave: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpErr("brave", resp.StatusCode, string(raw))
	}

	var out struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("brave: malformed response: %w", err)
	}
	res := make([]domain.SearchResult, 0, len(out.Web.Results))
	for _, r := range out.Web.Results {
		res = append(res, domain.SearchResult{Title: r.Title, URL: r.URL, Snippet: r.Description})
	}
	return limitResults(res), nil
}
