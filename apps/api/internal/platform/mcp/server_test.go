package mcp

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/jjspscl/my/internal/platform/bootstrap"
	"github.com/jjspscl/my/internal/platform/config"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var expectedReadTools = []string{
	"finance_list_transactions", "finance_today_total", "finance_budget_summary",
	"finance_list_bills", "finance_upcoming_bills", "finance_list_goals",
	"finance_list_wallets", "finance_list_transfers", "habits_list", "habits_completions",
}

var expectedWriteTools = []string{
	"finance_create_transaction", "finance_delete_transaction", "finance_upsert_budget",
	"finance_create_bill", "finance_update_bill", "finance_delete_bill", "finance_pay_bill",
	"finance_create_goal", "finance_update_goal", "finance_delete_goal", "finance_add_goal_contribution",
	"finance_create_wallet", "finance_update_wallet", "finance_archive_wallet", "finance_create_transfer",
	"habits_create", "habits_toggle", "habits_archive",
}

func TestToolRegistry(t *testing.T) {
	server := NewServer(&bootstrap.App{Cfg: &config.Config{UserEmail: "user@example.com"}}, Options{})
	result := listTools(t, server)

	want := append(append([]string{}, expectedReadTools...), expectedWriteTools...)
	sort.Strings(want)
	assertToolNames(t, result.Tools, want)
	for _, tool := range result.Tools {
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("tool %q has no input schema", tool.Name)
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal schema for %q: %v", tool.Name, err)
		}
		if strings.Contains(string(schema), `"user"`) || strings.Contains(string(schema), `"email"`) || strings.Contains(string(schema), `user_email`) {
			t.Errorf("tool %q exposes user identity input: %s", tool.Name, schema)
		}
	}
}

func TestReadOnlyToolRegistry(t *testing.T) {
	server := NewServer(&bootstrap.App{Cfg: &config.Config{UserEmail: "user@example.com"}}, Options{ReadOnly: true})
	result := listTools(t, server)
	want := append([]string{}, expectedReadTools...)
	sort.Strings(want)
	assertToolNames(t, result.Tools, want)
}

func TestToolAnnotations(t *testing.T) {
	server := NewServer(&bootstrap.App{Cfg: &config.Config{UserEmail: "user@example.com"}}, Options{})
	result := listTools(t, server)
	for _, tool := range result.Tools {
		if tool.Annotations == nil {
			t.Fatalf("tool %q has no annotations", tool.Name)
		}
		if contains(expectedReadTools, tool.Name) && !tool.Annotations.ReadOnlyHint {
			t.Errorf("read tool %q lacks read-only hint", tool.Name)
		}
		if contains([]string{"finance_delete_transaction", "finance_delete_bill", "finance_pay_bill", "finance_delete_goal", "finance_archive_wallet", "habits_archive"}, tool.Name) && (tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint) {
			t.Errorf("destructive tool %q lacks destructive hint", tool.Name)
		}
	}
}

func listTools(t *testing.T, server *mcpsdk.Server) *mcpsdk.ListToolsResult {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertToolNames(t *testing.T, tools []*mcpsdk.Tool, want []string) {
	t.Helper()
	if len(tools) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(tools), len(want))
	}
	for i, tool := range tools {
		if tool.Name != want[i] {
			t.Fatalf("tool %d = %q, want %q; registry order or names changed", i, tool.Name, want[i])
		}
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func TestNewServerRejectsEmptyUserEmail(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewServer() did not panic on empty UserEmail")
		}
	}()
	NewServer(&bootstrap.App{Cfg: &config.Config{}}, Options{})
}
