package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildEmailList(t *testing.T) {
	tests := []struct {
		name     string
		req      VerifyRequest
		expected []string
	}{
		{
			name:     "returns emails when Emails is set",
			req:      VerifyRequest{Emails: []string{"a@example.com", "b@example.com"}},
			expected: []string{"a@example.com", "b@example.com"},
		},
		{
			name:     "returns single email when Email is set",
			req:      VerifyRequest{Email: "single@example.com"},
			expected: []string{"single@example.com"},
		},
		{
			name:     "prefers Emails over Email",
			req:      VerifyRequest{Email: "single@example.com", Emails: []string{"batch@example.com"}},
			expected: []string{"batch@example.com"},
		},
		{
			name:     "returns nil when both empty",
			req:      VerifyRequest{},
			expected: nil,
		},
		{
			name:     "returns nil when Emails is empty slice",
			req:      VerifyRequest{Emails: []string{}},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEmailList(tt.req)
			if len(result) != len(tt.expected) {
				t.Errorf("buildEmailList() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i, email := range result {
				if email != tt.expected[i] {
					t.Errorf("buildEmailList()[%d] = %q, want %q", i, email, tt.expected[i])
				}
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	t.Run("returns OK on GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		HealthHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HealthHandler() status = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if health.Status != "ok" {
			t.Errorf("HealthHandler() status = %q, want %q", health.Status, "ok")
		}
	})

	t.Run("returns 405 on POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/health", nil)
		w := httptest.NewRecorder()

		HealthHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("HealthHandler() status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})

	t.Run("returns 405 on PUT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/health", nil)
		w := httptest.NewRecorder()

		HealthHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("HealthHandler() status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})
}

func TestVerifyHandler(t *testing.T) {
	t.Run("returns 405 on GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/verify", nil)
		w := httptest.NewRecorder()

		VerifyHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("VerifyHandler() status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader("not json"))
		w := httptest.NewRecorder()

		VerifyHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("VerifyHandler() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if errResp.Error != "invalid JSON request body" {
			t.Errorf("VerifyHandler() error = %q, want %q", errResp.Error, "invalid JSON request body")
		}
	})

	t.Run("returns 400 when no emails provided", func(t *testing.T) {
		body := VerifyRequest{}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()

		VerifyHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("VerifyHandler() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if errResp.Error != "no email addresses provided" {
			t.Errorf("VerifyHandler() error = %q, want %q", errResp.Error, "no email addresses provided")
		}
	})

	t.Run("returns 400 when batch size exceeds limit", func(t *testing.T) {
		emails := make([]string, 101)
		for i := range emails {
			emails[i] = "test@example.com"
		}
		body := VerifyRequest{Emails: emails}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()

		VerifyHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("VerifyHandler() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if errResp.Error != "batch size exceeds maximum of 100 emails" {
			t.Errorf("VerifyHandler() error = %q, want %q", errResp.Error, "batch size exceeds maximum of 100 emails")
		}
	})

	t.Run("accepts exactly 100 emails", func(t *testing.T) {
		emails := make([]string, 100)
		for i := range emails {
			emails[i] = "test@example.com"
		}
		body := VerifyRequest{Emails: emails}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()

		VerifyHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("VerifyHandler() status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("sets content type header", func(t *testing.T) {
		w := httptest.NewRecorder()

		writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

		if ct := w.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}
	})

	t.Run("sets status code", func(t *testing.T) {
		w := httptest.NewRecorder()

		writeJSON(w, http.StatusCreated, map[string]string{"key": "value"})

		if w.Code != http.StatusCreated {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("writes error response", func(t *testing.T) {
		w := httptest.NewRecorder()

		writeError(w, http.StatusBadRequest, "test error message")

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if errResp.Error != "test error message" {
			t.Errorf("error = %q, want %q", errResp.Error, "test error message")
		}
	})
}

func TestMaxBatchSize(t *testing.T) {
	if maxBatchSize != 100 {
		t.Errorf("maxBatchSize = %d, want %d", maxBatchSize, 100)
	}
}
