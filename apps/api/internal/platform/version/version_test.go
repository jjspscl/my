package version

import "testing"

func TestString(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = originalVersion, originalCommit, originalDate
	})

	Version = "v9.9.9"
	Commit = "abc1234"
	Date = "2026-08-08T00:00:00Z"

	if got, want := String(), "v9.9.9 (abc1234, 2026-08-08T00:00:00Z)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
