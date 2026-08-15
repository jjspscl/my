package infrastructure

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPGateway calls allowlisted tools on outbound MCP search connectors
// (Brave, Exa, …). It is the OUTBOUND counterpart of the inbound `my` MCP
// server; they share no code path.
//
// Security posture:
//   - only tools named in the connector allowlist may be called
//   - responses are treated as untrusted evidence, never instructions
//   - only text content is returned, truncated and bounded
//   - timeouts are strict; credentials travel only in Authorization headers
type MCPGateway struct{}

func NewMCPGateway() *MCPGateway {
	return &MCPGateway{}
}

type ToolCall struct {
	Name      string
	Arguments map[string]any
}

// Call invokes one allowlisted tool and returns joined text content.
func (g *MCPGateway) Call(ctx context.Context, connector *domain.MCPConnector, credential string, call ToolCall) (string, error) {
	allowed := false
	for _, name := range connector.Allowlist {
		if name == call.Name {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("tool %q is not allowlisted on connector %q", call.Name, connector.Name)
	}

	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: connector.Endpoint,
		HTTPClient: &http.Client{
			Timeout: connector.Timeout,
			Transport: &authHeaderTransport{
				base:       http.DefaultTransport,
				credential: credential,
				authType:   connector.AuthType,
				timeout:    connector.Timeout,
			},
			CheckRedirect: CheckRedirectFor(false), // connectors are never local
		},
		MaxRetries:           1,
		DisableStandaloneSSE: true, // request/response only; no server-initiated streams
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "my-intelligence", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return "", fmt.Errorf("connect to %s: %w", connector.Name, err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      call.Name,
		Arguments: call.Arguments,
	})
	if err != nil {
		return "", fmt.Errorf("call %s: %w", call.Name, err)
	}
	if res.IsError {
		return "", fmt.Errorf("tool %s reported an error", call.Name)
	}

	var parts []string
	for _, content := range res.Content {
		if text, ok := content.(*mcpsdk.TextContent); ok && strings.TrimSpace(text.Text) != "" {
			parts = append(parts, text.Text)
		}
	}
	out := strings.Join(parts, "\n")
	if len(out) > 8000 {
		out = out[:8000] + "…"
	}
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("tool %s returned no text", call.Name)
	}
	return out, nil
}

// authHeaderTransport injects the connector credential into the header chosen
// by the connector's auth type (none | bearer | x-api-key). The credential
// never travels in the URL.
type authHeaderTransport struct {
	base       http.RoundTripper
	credential string
	authType   string
	timeout    time.Duration
}

func (t *authHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.credential != "" {
		switch t.authType {
		case domain.ConnectorAuthXKey:
			req.Header.Set("X-Api-Key", t.credential)
		default: // bearer (also legacy connectors without an explicit auth type)
			req.Header.Set("Authorization", "Bearer "+t.credential)
		}
	}
	return t.base.RoundTrip(req)
}
