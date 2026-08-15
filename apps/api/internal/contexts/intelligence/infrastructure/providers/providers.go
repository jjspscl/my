package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

// ChatRequest is the transport-neutral completion request.
type ChatRequest struct {
	System    string
	User      string
	MaxTokens int
}

// ChatResult is the transport-neutral completion result.
type ChatResult struct {
	Content string
	Model   string
	Usage   map[string]int
}

// Provider completes chat-style requests. Implementations must bound their
// timeouts and never log request bodies or credentials.
type Provider interface {
	Complete(ctx context.Context, req ChatRequest) (*ChatResult, error)
}

// Config carries runtime values adapters need that are not stored in the DB.
type Config struct {
	CodexPath string // absolute path to the codex binary (CLI adapter)
}

// New builds the adapter for a provider profile. Credential is the decrypted
// secret (may be empty for local/CLI providers).
func New(p *domain.ProviderProfile, credential string, cfg Config) (Provider, error) {
	switch p.ProviderType {
	case domain.ProviderOpenAI:
		base := p.BaseURL
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		return &openAIResponses{baseURL: base, apiKey: credential, model: p.Model, maxTokens: p.MaxTokens, timeout: p.Timeout, allowLocal: p.AllowLocal}, nil
	case domain.ProviderOpenAICompatible, domain.ProviderOllama:
		base := p.BaseURL
		if base == "" {
			return nil, fmt.Errorf("base url is required for %s providers", p.ProviderType)
		}
		return &openAICompatible{baseURL: base, apiKey: credential, model: p.Model, maxTokens: p.MaxTokens, timeout: p.Timeout, allowLocal: p.AllowLocal}, nil
	case domain.ProviderCodexCLI:
		if cfg.CodexPath == "" {
			return nil, fmt.Errorf("codex CLI provider requires a configured binary path")
		}
		return &codexCLI{binPath: cfg.CodexPath, model: p.Model, maxTokens: p.MaxTokens, timeout: p.Timeout}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", p.ProviderType)
	}
}

// ---- OpenAI-compatible chat completions (Ollama, Yunwu, gateways) ----

type openAICompatible struct {
	baseURL    string
	apiKey     string
	model      string
	maxTokens  int
	timeout    time.Duration
	allowLocal bool
}

func (p *openAICompatible) Complete(ctx context.Context, req ChatRequest) (*ChatResult, error) {
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.User},
		},
		"temperature": 0,
		"stream":      false,
	}
	if p.maxTokens > 0 {
		body["max_tokens"] = p.maxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	client := &http.Client{
		Timeout:       p.timeout,
		CheckRedirect: checkRedirect(p.allowLocal),
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse provider response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("provider returned no choices")
	}

	return &ChatResult{
		Content: parsed.Choices[0].Message.Content,
		Model:   parsed.Model,
		Usage:   map[string]int{"prompt": parsed.Usage.PromptTokens, "completion": parsed.Usage.CompletionTokens},
	}, nil
}

// checkRedirect blocks redirect hops that would escape the endpoint policy
// (SSRF defense-in-depth: URL validation alone is not enough).
func checkRedirect(allowLocal bool) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("stopped after 5 redirects")
		}
		host := strings.ToLower(req.URL.Hostname())
		if host == "" {
			return fmt.Errorf("redirect target has no host")
		}
		if ip := net.ParseIP(host); ip != nil && (ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast()) {
			return fmt.Errorf("redirect blocked: private or link-local target %q", host)
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() && !allowLocal {
			return fmt.Errorf("redirect blocked: loopback target requires local mode")
		}
		if !isLoopbackHost(host) && req.URL.Scheme != "https" {
			return fmt.Errorf("redirect blocked: non-https target %q", req.URL.String())
		}
		return nil
	}
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ---- OpenAI Responses API (OpenAI + Codex-capable model IDs) ----

type openAIResponses struct {
	baseURL    string
	apiKey     string
	model      string
	maxTokens  int
	timeout    time.Duration
	allowLocal bool
}

func (p *openAIResponses) Complete(ctx context.Context, req ChatRequest) (*ChatResult, error) {
	body := map[string]any{
		"model": p.model,
		"input": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.User},
		},
		"text": map[string]any{
			"format": map[string]string{"type": "text"},
		},
	}
	if p.maxTokens > 0 {
		body["max_output_tokens"] = p.maxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	client := &http.Client{
		Timeout:       p.timeout,
		CheckRedirect: checkRedirect(p.allowLocal),
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed struct {
		Model      string `json:"model"`
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			Text string `json:"text"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse provider response: %w", err)
	}
	content := parsed.OutputText
	if content == "" {
		for _, o := range parsed.Output {
			if o.Type != "message" {
				continue
			}
			for _, c := range o.Content {
				if (c.Type == "output_text" || c.Type == "text") && c.Text != "" {
					content += c.Text
				}
			}
			// Some implementations put the text directly on the item.
			if content == "" && o.Text != "" {
				content = o.Text
			}
		}
	}
	if content == "" {
		return nil, fmt.Errorf("provider returned no output")
	}

	return &ChatResult{
		Content: content,
		Model:   parsed.Model,
		Usage:   map[string]int{"prompt": parsed.Usage.InputTokens, "completion": parsed.Usage.OutputTokens},
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
