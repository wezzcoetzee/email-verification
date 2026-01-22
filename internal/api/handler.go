package api

import (
	"encoding/json"
	"net/http"
	"time"

	"email-verification/internal/verifier"
)

const maxBatchSize = 100

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	emails := buildEmailList(req)
	if len(emails) == 0 {
		writeError(w, http.StatusBadRequest, "no email addresses provided")
		return
	}

	if len(emails) > maxBatchSize {
		writeError(w, http.StatusBadRequest, "batch size exceeds maximum of 100 emails")
		return
	}

	opts := verifier.VerifyOptions{
		EnableSMTP: true,
	}
	if req.Options != nil {
		opts.EnableSMTP = req.Options.EnableSMTP
	}

	start := time.Now()
	results := verifier.VerifyMultiple(emails, opts)
	elapsed := time.Since(start)

	validCount := 0
	for _, r := range results {
		if r.Valid {
			validCount++
		}
	}

	resp := VerifyResponse{
		Results: results,
		Summary: &Summary{
			Total:            len(results),
			Valid:            validCount,
			Invalid:          len(results) - validCount,
			ProcessingTimeMs: elapsed.Milliseconds(),
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

func buildEmailList(req VerifyRequest) []string {
	if len(req.Emails) > 0 {
		return req.Emails
	}
	if req.Email != "" {
		return []string{req.Email}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
