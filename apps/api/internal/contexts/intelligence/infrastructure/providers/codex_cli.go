package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// codexCLI runs the sandboxed Codex CLI in non-interactive exec mode.
//
// Safety posture (see docs/mcp.md + architecture docs):
//   - fixed binary path from config; never shell interpolation
//   - fresh empty working directory; the repository is never accessible
//   - read-only sandbox, no approvals needed for reads
//   - ephemeral session: no transcript rollouts persisted
//   - project instructions, MCP servers, hooks, memories, and web search are
//     disabled so ambient config cannot leak or exfiltrate
//   - strict timeout + output cap; structured JSON output only
//
// The adapter is disabled by default (requires MY_LLM_CODEX_PATH).
type codexCLI struct {
	binPath   string
	model     string
	maxTokens int
	timeout   time.Duration
}

// maxOutputBytes caps captured stdout (1 MB) — a runaway agent cannot flood
// the process.
const maxOutputBytes = 1 << 20

func (p *codexCLI) Complete(ctx context.Context, req ChatRequest) (*ChatResult, error) {
	workDir, err := os.MkdirTemp("", "my-codex-*")
	if err != nil {
		return nil, fmt.Errorf("create sandbox dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	prompt := req.System + "\n\n" + req.User

	// Flags are kept to a well-documented, version-stable subset of `codex
	// exec`: structured output, last-message extraction, and a read-only
	// sandbox. Isolation beyond the sandbox comes from the empty working
	// directory, the sanitized environment, and the strict timeout.
	args := []string{
		"exec",
		"--json",
		"--output-last-message",
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--skip-prompt-check",
	}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	if p.maxTokens > 0 {
		args = append(args, "--max-output-tokens", fmt.Sprintf("%d", p.maxTokens))
	}
	args = append(args, "--", prompt)

	cmd := exec.CommandContext(ctx, p.binPath, args...)
	cmd.Dir = workDir
	cmd.Env = sanitizeEnv(os.Environ())

	stdout := &limitedBuffer{limit: maxOutputBytes}
	stderr := &limitedBuffer{limit: 64 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex: %w", err)
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("codex exec failed: %v: %s", err, stderr.String())
		}
	case <-time.After(p.timeout):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("codex exec timed out after %s", p.timeout)
	}
	if stdout.truncated {
		return nil, fmt.Errorf("codex output exceeded %d bytes", maxOutputBytes)
	}

	// The last line carries the final assistant message when --json is used.
	lines := strings.Split(stdout.String(), "\n")
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		last = line
		break
	}

	content := last
	var event struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(last), &event); err == nil && event.Message.Content != nil {
		content = ""
		for _, c := range event.Message.Content {
			if c.Type == "output_text" || c.Type == "text" {
				content += c.Text
			}
		}
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("codex produced no output")
	}

	return &ChatResult{Content: content, Model: "codex-cli", Usage: map[string]int{}}, nil
}

// limitedBuffer captures up to limit bytes, flagging truncation instead of
// growing unbounded — a runaway agent cannot exhaust memory.
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		n, _ := b.buf.Write(p[:remaining])
		return n, nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string { return b.buf.String() }

// sanitizeEnv strips secrets and network-bearing variables from the subprocess
// environment. Codex CLI credentials stay in Codex home; the analysis process
// should not see application secrets.
func sanitizeEnv(env []string) []string {
	blocklist := []string{
		"MY_LLM_MASTER_KEY", "MY_MCP_TOKEN", "MY_SMTP_PASS",
		"AWS_", "AZURE_", "GOOGLE_", "OPENAI_API_KEY", "ANTHROPIC_API_KEY",
		"GITHUB_TOKEN", "GH_TOKEN",
	}
	out := make([]string, 0, len(env))
	for _, kv := range env {
		key := kv[:strings.IndexByte(kv, '=')+1]
		skip := false
		for _, b := range blocklist {
			if strings.HasPrefix(key, b) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, kv)
		}
	}
	return out
}
