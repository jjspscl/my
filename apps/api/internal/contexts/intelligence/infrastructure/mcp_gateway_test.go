package infrastructure

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchArgs struct {
	Query string `json:"query"`
}

// newFakeMCPServer builds an in-process MCP server exposing one search tool.
func newFakeMCPServer(t *testing.T, requireAuth string) *httptest.Server {
	t.Helper()
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "fake-search", Version: "1.0.0"}, nil)
	mcpsdk.AddTool[searchArgs, string](server, &mcpsdk.Tool{
		Name:        "brave_web_search",
		Description: "Search the web",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, args searchArgs) (*mcpsdk.CallToolResult, string, error) {
		if strings.TrimSpace(args.Query) == "" {
			return nil, "", fmt.Errorf("query required")
		}
		out := `[{"title":"Result for ` + args.Query + `","url":"https://example.com","snippet":"snippet"}]`
		return &mcpsdk.CallToolResult{}, out, nil
	})

	inner := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return server },
		&mcpsdk.StreamableHTTPOptions{JSONResponse: true},
	)
	var handler http.Handler = inner
	if requireAuth != "" {
		base := inner
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+requireAuth {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			base.ServeHTTP(w, r)
		})
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func testConnector(endpoint string, allowlist ...string) *domain.MCPConnector {
	return &domain.MCPConnector{
		ID: "c1", UserEmail: "you@example.com", Name: "Brave",
		Endpoint: endpoint, Enabled: true, Allowlist: allowlist, Timeout: 5 * time.Second,
	}
}

func TestMCPGatewayRejectsNonAllowlistedTool(t *testing.T) {
	srv := newFakeMCPServer(t, "")
	gateway := NewMCPGateway()
	_, err := gateway.Call(context.Background(), testConnector(srv.URL, "brave_web_search"), "", ToolCall{
		Name:      "filesystem_write",
		Arguments: map[string]any{"path": "/etc/passwd", "content": "pwned"},
	})
	if err == nil || !strings.Contains(err.Error(), "not allowlisted") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestMCPGatewayCallsAllowlistedTool(t *testing.T) {
	srv := newFakeMCPServer(t, "")
	gateway := NewMCPGateway()
	out, err := gateway.Call(context.Background(), testConnector(srv.URL, "brave_web_search"), "", ToolCall{
		Name:      "brave_web_search",
		Arguments: map[string]any{"query": "jollibee"},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(out, "Result for jollibee") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestMCPGatewaySendsBearerToken(t *testing.T) {
	srv := newFakeMCPServer(t, "mcp-secret")
	gateway := NewMCPGateway()
	_, err := gateway.Call(context.Background(), testConnector(srv.URL, "brave_web_search"), "mcp-secret", ToolCall{
		Name:      "brave_web_search",
		Arguments: map[string]any{"query": "jollibee"},
	})
	if err != nil {
		t.Fatalf("call with token: %v", err)
	}
}

func TestMCPGatewayRejectsWrongToken(t *testing.T) {
	srv := newFakeMCPServer(t, "mcp-secret")
	gateway := NewMCPGateway()
	_, err := gateway.Call(context.Background(), testConnector(srv.URL, "brave_web_search"), "wrong", ToolCall{
		Name:      "brave_web_search",
		Arguments: map[string]any{"query": "jollibee"},
	})
	if err == nil {
		t.Fatal("expected auth failure")
	}
}
