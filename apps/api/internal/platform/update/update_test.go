package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompare(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		want        int
	}{
		{"v2.0.0", "v1.9.9", 1},
		{"v1.2.0", "v1.2.0", 0},
		{"v1.1.0", "v1.2.0", -1},
		{"dev", "v1.0.0", -1},
		{"v1.2.3-rc.1", "v1.2.3", 0},
	} {
		if got := Compare(tc.left, tc.right); got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestResultMessage(t *testing.T) {
	if got := (Result{Current: "v1.0.0", Latest: "v1.1.0"}).Message(); got == "" {
		t.Fatal("Message() empty")
	}
	if got := (Result{Current: "v1.0.0", Latest: "v1.0.0"}).Message(); got != "up to date: v1.0.0" {
		t.Fatalf("Message() = %q", got)
	}
}

func TestLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer server.Close()

	got, err := latestVersion(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.3" {
		t.Fatalf("latestVersion() = %q, want v1.2.3", got)
	}
}
