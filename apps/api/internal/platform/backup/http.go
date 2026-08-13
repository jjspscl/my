package backup

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jjspscl/my/internal/shared/response"
)

// Handler exposes the backup and export endpoints. Both are mounted behind
// RequireAuth in the protected route group.
type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// Snapshot streams a VACUUM INTO snapshot as a downloadable file. The
// snapshot path is generated server-side and removed afterwards; VACUUM INTO
// refuses to overwrite an existing file, so the temp file is created and
// deleted first.
func (h *Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	tmp, err := os.CreateTemp("", "my-backup-*.db")
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "could not create backup", err)
		return
	}
	path := tmp.Name()
	tmp.Close()
	// VACUUM INTO must create the destination itself.
	_ = os.Remove(path)
	defer os.Remove(path)

	if err := SnapshotTo(h.db, path); err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "backup failed", err)
		return
	}

	name := "my-backup-" + time.Now().Format("20060102-150405") + ".db"
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	http.ServeFile(w, r, path)
}

// Export streams the full JSON export. Built in memory first so a failure
// cannot produce a truncated document.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	export, err := ExportTo(h.db)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "export failed", err)
		return
	}

	name := "my-export-" + time.Now().Format("20060102-150405") + ".json"
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	response.WriteJSON(w, http.StatusOK, export)
}
