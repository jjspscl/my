package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

var tavilyEndpoint = "https://api.tavily.com/search"

// tavilySearch calls the Tavily Search API (cost-friendly defaults: basic
// depth, no generated answer, no raw content, 5 results). Bearer auth is
// used; the documented ?tavilyApiKey= query form is never used.
func tavilySearch(ctx context.Context, c *domain.MCPConnector, credential, query string) ([]domain.SearchResult, error) {
	body, _ := json.Marshal(map[string]any{
		"query":               query,
		"search_depth":        "basic",
		"max_results":         maxSearchResults,
		"include_answer":      false,
		"include_raw_content": false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}

	resp, err := newHardenedClient(c.Timeout).Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("tavily: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpErr("tavily", resp.StatusCode, string(raw))
	}

	var out struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("tavily: malformed response: %w", err)
	}
	res := make([]domain.SearchResult, 0, len(out.Results))
	for _, r := range out.Results {
		res = append(res, domain.SearchResult{Title: r.Title, URL: r.URL, Snippet: r.Content})
	}
	return limitResults(res), nil
}
