package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)

		if rw.status != http.StatusNotFound {
			t.Errorf("responseWriter.status = %d, want %d", rw.status, http.StatusNotFound)
		}
	})

	t.Run("forwards WriteHeader to underlying writer", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		rw.WriteHeader(http.StatusCreated)

		if w.Code != http.StatusCreated {
			t.Errorf("underlying writer code = %d, want %d", w.Code, http.StatusCreated)
		}
	})

	t.Run("default status is preserved", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		if rw.status != http.StatusOK {
			t.Errorf("initial status = %d, want %d", rw.status, http.StatusOK)
		}
	})
}

func TestLogging(t *testing.T) {
	t.Run("passes request to next handler", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := Logging(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !called {
			t.Error("Logging() did not call next handler")
		}
	})

	t.Run("preserves response from next handler", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("test body"))
		})

		handler := Logging(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
		}
		if w.Body.String() != "test body" {
			t.Errorf("body = %q, want %q", w.Body.String(), "test body")
		}
	})

	t.Run("captures status code from handler", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		handler := Logging(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestRecovery(t *testing.T) {
	t.Run("passes request to next handler when no panic", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := Recovery(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !called {
			t.Error("Recovery() did not call next handler")
		}
		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("recovers from panic and returns 500", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		handler := Recovery(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if errResp.Error != "internal server error" {
			t.Errorf("error = %q, want %q", errResp.Error, "internal server error")
		}
	})

	t.Run("recovers from error panic", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(&customError{msg: "custom error"})
		})

		handler := Recovery(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
