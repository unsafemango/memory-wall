package http

import (
	"encoding/json"
	"net/http"
)

// writeError sends every API error in the same shape:
//
//	{"error": {"code": "not_found", "message": "invitation not found"}}
//
// The code is stable for programs to check; the message is for people.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
