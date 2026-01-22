package api

import "email-verification/internal/verifier"

type VerifyRequest struct {
	Email   string         `json:"email,omitempty"`
	Emails  []string       `json:"emails,omitempty"`
	Options *VerifyOptions `json:"options,omitempty"`
}

type VerifyOptions struct {
	EnableSMTP bool `json:"enable_smtp"`
}

type VerifyResponse struct {
	Results []verifier.VerifyResult `json:"results"`
	Summary *Summary                `json:"summary,omitempty"`
}

type Summary struct {
	Total            int   `json:"total"`
	Valid            int   `json:"valid"`
	Invalid          int   `json:"invalid"`
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
