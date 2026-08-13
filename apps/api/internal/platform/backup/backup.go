// Package backup provides snapshot and export facilities for the SQLite
// database.
//
// Snapshot uses VACUUM INTO, which is safe on a live database (it reads
// through the WAL) and produces a defragmented, byte-exact file that can be
// restored by replacing the database. Never back up by copying the .db file:
// with WAL enabled a naive copy can be torn.
package backup

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	platformversion "github.com/jjspscl/my/internal/platform/version"
)

// SnapshotTo writes a consistent snapshot of db to dest using VACUUM INTO.
// dest must not exist (VACUUM INTO refuses to overwrite) and its directory
// must exist and be writable.
func SnapshotTo(db *sql.DB, dest string) error {
	// VACUUM INTO's target is a literal in the SQL statement — the driver
	// cannot bind it. dest comes from the CLI flag or a path we generate
	// server-side, never from HTTP input; escape single quotes defensively.
	escaped := strings.ReplaceAll(dest, "'", "''")
	if _, err := db.Exec("VACUUM INTO '" + escaped + "'"); err != nil {
		return fmt.Errorf("vacuum into %s: %w", dest, err)
	}
	return nil
}

// exportTables is the set of user-data tables included in exports. magic_tokens
// is deliberately excluded: it holds single-use authentication credentials,
// not user data.
var exportTables = []string{
	"transactions",
	"wallets",
	"wallet_transfers",
	"finance_categories",
	"budgets",
	"budget_categories",
	"recurring_bills",
	"bill_payments",
	"savings_goals",
	"goal_contributions",
	"habits",
	"habit_completions",
}

// Export is the JSON export payload: one entry per user-data table.
type Export struct {
	ExportedAt time.Time                   `json:"exportedAt"`
	AppVersion string                      `json:"appVersion"`
	Tables     map[string][]map[string]any `json:"tables"`
}

// ExportTo reads every user-data table into memory and returns it as a
// portable JSON document. Intended for inspection and migration, not as the
// primary restore path — that is SnapshotTo.
func ExportTo(db *sql.DB) (*Export, error) {
	out := &Export{
		ExportedAt: time.Now().UTC(),
		AppVersion: platformversion.String(),
		Tables:     make(map[string][]map[string]any, len(exportTables)),
	}

	for _, table := range exportTables {
		out.Tables[table] = []map[string]any{}
		rows, err := db.Query("SELECT * FROM " + table)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", table, err)
		}

		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("columns %s: %w", table, err)
		}

		for rows.Next() {
			raw := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range raw {
				ptrs[i] = &raw[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan %s: %w", table, err)
			}
			row := make(map[string]any, len(cols))
			for i, col := range cols {
				row[col] = normalize(raw[i])
			}
			out.Tables[table] = append(out.Tables[table], row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate %s: %w", table, err)
		}
		rows.Close()
	}

	return out, nil
}

// normalize converts driver values to JSON-friendly shapes: SQLite stores
// DATETIME columns as text in this schema, so most values are already strings;
// []byte becomes string, time.Time becomes RFC3339.
func normalize(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return v
	}
}
