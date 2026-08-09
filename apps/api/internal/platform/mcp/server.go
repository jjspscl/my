package mcp

import (
	"context"
	"log/slog"
	"time"

	"github.com/jjspscl/my/internal/platform/bootstrap"
	platformversion "github.com/jjspscl/my/internal/platform/version"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Options struct {
	ReadOnly bool
}

func NewServer(app *bootstrap.App, opts Options) *mcpsdk.Server {
	// Every tool and resource uses UserEmail as the data ownership key. An empty
	// value would read and write against a phantom tenant, so refuse to build a
	// server that could do that. config.Load rejects this first; this guard
	// protects callers that construct Config directly.
	if app.Cfg.UserEmail == "" {
		panic("mcp: NewServer requires a non-empty Cfg.UserEmail")
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "my",
		Version: platformversion.Version,
	}, &mcpsdk.ServerOptions{
		Logger: app.Log,
		Instructions: "Personal dashboard for finance, habits, and daily life. " +
			"All data belongs to the configured single user. Confirm destructive actions before calling them.",
	})

	registerFinanceReadTools(server, app)
	registerHabitsReadTools(server, app)
	if !opts.ReadOnly {
		registerFinanceWriteTools(server, app)
		registerHabitsWriteTools(server, app)
	}
	registerResources(server, app)
	registerPrompts(server)
	return server
}

type toolHandler[In any] func(context.Context, In) (any, error)

func registerTool[In any](server *mcpsdk.Server, logger *slog.Logger, tool *mcpsdk.Tool, handler toolHandler[In]) {
	mcpsdk.AddTool[In, any](server, tool, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input In) (_ *mcpsdk.CallToolResult, output any, err error) {
		started := time.Now()
		defer func() {
			if logger != nil {
				attrs := []any{
					slog.String("tool", tool.Name),
					slog.Duration("duration", time.Since(started)),
					slog.String("outcome", outcome(err)),
				}
				if err != nil {
					attrs = append(attrs, slog.String("error", err.Error()))
				}
				logger.Info("mcp tool call", attrs...)
			}
		}()
		output, err = handler(ctx, input)
		return nil, output, err
	})
}

func outcome(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

// idInput is shared by finance and habits tools that address a single record.
type idInput struct {
	ID string `json:"id"`
}

// writable marks a tool that mutates state but is not destructive.
var writable = &mcpsdk.ToolAnnotations{DestructiveHint: boolPointer(false)}

// destructive marks a tool whose effect cannot be undone through MCP.
func destructive() *mcpsdk.ToolAnnotations {
	return &mcpsdk.ToolAnnotations{DestructiveHint: boolPointer(true), IdempotentHint: false}
}

func boolPointer(value bool) *bool {
	return &value
}
