// Package migrations embeds the SQL migration files so the binary is
// self-contained: no filesystem dependency at runtime, no relative-path
// footguns, and the same migration set regardless of working directory.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
