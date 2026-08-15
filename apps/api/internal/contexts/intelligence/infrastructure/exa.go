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

var exaEndpoint = "https://api.exa.ai/search"

// exaSearch calls the Exa Search API with the x-api-key header. Fast, cheap
// search mode; no deep research, no contents. Keyless (empty credential) is
// allowed and simply omits the header.
func exaSearch(ctx context.Context, c *domain.MCPConnector, credential, query string) ([]domain.SearchResult, error) {
	body, _ := json.Marshal(map[string]any{
		"query":      query,
		"type":       "fast",
		"numResults": maxSearchResults,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, exaEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if credential != "" {
		req.Header.Set("x-api-key", credential)
	}

	resp, err := newHardenedClient(c.Timeout).Do(req)
	if err != nil {
		return nil, fmt.Errorf("exa: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("exa: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpErr("exa", resp.StatusCode, string(raw))
	}

	var out struct {
		Results []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Text  string `json:"text"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("exa: malformed response: %w", err)
	}
	res := make([]domain.SearchResult, 0, len(out.Results))
	for _, r := range out.Results {
		res = append(res, domain.SearchResult{Title: r.Title, URL: r.URL, Snippet: r.Text})
	}
	return limitResults(res), nil
}
