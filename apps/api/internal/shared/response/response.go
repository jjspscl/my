package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// WriteJSON serializes v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError sends clientMsg as JSON error to the client and logs the
// underlying err (if non-nil) server-side with request context.
// Use for all 4xx/5xx responses — gives server visibility without leaking
// internals to the client.
func WriteError(w http.ResponseWriter, r *http.Request, status int, clientMsg string, err error) {
	if err != nil {
		log.Printf("ERROR %d [%s %s]: %s | cause: %s", status, r.Method, r.URL.Path, clientMsg, err)
	} else {
		log.Printf("ERROR %d [%s %s]: %s", status, r.Method, r.URL.Path, clientMsg)
	}
	WriteJSON(w, status, map[string]string{"error": clientMsg})
}
