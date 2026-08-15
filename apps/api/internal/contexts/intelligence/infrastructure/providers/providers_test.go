package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

func compatibleServer(t *testing.T, content string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if auth := r.Header.Get("Authorization"); auth == "" {
			t.Errorf("expected Authorization header")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(content))
	}))
}

func responsesServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" && r.URL.Path != "/responses" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(content))
	}))
}

func TestOpenAICompatibleComplete(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"model": "test-model",
		"choices": []map[string]any{
			{"message": map[string]any{"content": "hello from provider"}},
		},
		"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5},
	})
	srv := compatibleServer(t, string(body), http.StatusOK)
	defer srv.Close()

	p, err := New(&domain.ProviderProfile{
		ProviderType: domain.ProviderOpenAICompatible,
		BaseURL:      srv.URL,
		Model:        "test-model",
		Timeout:      5 * time.Second,
	}, "sk-test", Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	res, err := p.Complete(context.Background(), ChatRequest{System: "s", User: "u"})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if res.Content != "hello from provider" {
		t.Fatalf("content = %q", res.Content)
	}
	if res.Usage["prompt"] != 10 {
		t.Fatalf("usage = %v", res.Usage)
	}
}

func TestOpenAICompatibleErrorStatus(t *testing.T) {
	srv := compatibleServer(t, `{"error":{"message":"rate limited"}}`, http.StatusTooManyRequests)
	defer srv.Close()

	p, err := New(&domain.ProviderProfile{
		ProviderType: domain.ProviderOpenAICompatible,
		BaseURL:      srv.URL,
		Model:        "m",
		Timeout:      5 * time.Second,
	}, "sk-test", Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	_, err = p.Complete(context.Background(), ChatRequest{User: "u"})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected 429 error, got %v", err)
	}
}

func TestOpenAIResponsesComplete(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"model":      "gpt-5.2-codex",
		"output_text": "structured answer",
		"usage":      map[string]int{"input_tokens": 20, "output_tokens": 8},
	})
	srv := responsesServer(t, string(body))
	defer srv.Close()

	p, err := New(&domain.ProviderProfile{
		ProviderType: domain.ProviderOpenAI,
		BaseURL:      srv.URL,
		Model:        "gpt-5.2-codex",
		Timeout:      5 * time.Second,
	}, "sk-test", Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	res, err := p.Complete(context.Background(), ChatRequest{User: "u"})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if res.Content != "structured answer" {
		t.Fatalf("content = %q", res.Content)
	}
	if res.Usage["prompt"] != 20 {
		t.Fatalf("usage = %v", res.Usage)
	}
}

func TestOpenAIResponsesDefaultsToOpenAIFallback(t *testing.T) {
	// Empty BaseURL for openai should not be reachable in tests; instead
	// verify the default base is applied (documented behavior).
	p, err := New(&domain.ProviderProfile{
		ProviderType: domain.ProviderOpenAI,
		Model:        "gpt-5.2-codex",
		Timeout:      5 * time.Second,
	}, "sk-test", Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	resp, ok := p.(*openAIResponses)
	if !ok {
		t.Fatalf("expected *openAIResponses, got %T", p)
	}
	if resp.baseURL != "https://api.openai.com/v1" {
		t.Fatalf("baseURL = %q", resp.baseURL)
	}
}

// fakeCodex writes a minimal codex-like binary that emits one JSON event and
// exits zero, proving the adapter parses --json --output-last-message output.
func fakeCodex(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	script := `#!/bin/sh
printf '{"type":"agent_message","message":{"content":[{"type":"output_text","text":"fake codex answer"}]}}\n'
exit 0
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return bin
}

func TestCodexCLIComplete(t *testing.T) {
	bin := fakeCodex(t)
	p := &codexCLI{binPath: bin, timeout: 10 * time.Second}

	res, err := p.Complete(context.Background(), ChatRequest{System: "sys", User: "user"})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if res.Content != "fake codex answer" {
		t.Fatalf("content = %q", res.Content)
	}
}

func TestSanitizeEnvStripsSecrets(t *testing.T) {
	env := []string{
		"MY_LLM_MASTER_KEY=supersecret",
		"MY_MCP_TOKEN=tok",
		"OPENAI_API_KEY=sk-abc",
		"PATH=/usr/bin",
		"HOME=/home/user",
	}
	out := sanitizeEnv(env)
	joined := strings.Join(out, "\n")
	for _, banned := range []string{"MY_LLM_MASTER_KEY", "MY_MCP_TOKEN", "OPENAI_API_KEY"} {
		if strings.Contains(joined, banned) {
			t.Errorf("sanitized env still contains %s: %v", banned, out)
		}
	}
	if !strings.Contains(joined, "PATH=/usr/bin") {
		t.Errorf("sanitized env dropped PATH: %v", out)
	}
}
