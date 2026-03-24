package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTextLoggerFormatsModuleAndRequestID(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := New(Config{
		Level:  "info",
		Format: "text",
	}, &output)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	logger.Info("auth", "qwacm84y", `authorize -> https://example.com/path`)

	line := output.String()
	if !strings.Contains(line, "INFO") {
		t.Fatalf("line = %q, want INFO", line)
	}
	if !strings.Contains(line, "[auth]") {
		t.Fatalf("line = %q, want [auth]", line)
	}
	if !strings.Contains(line, "[qwacm84y]") {
		t.Fatalf("line = %q, want [qwacm84y]", line)
	}
	if !strings.Contains(line, "authorize -> https://example.com/path") {
		t.Fatalf("line = %q, want message", line)
	}
}

func TestJSONLoggerEmitsStructuredFields(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := New(Config{
		Level:  "info",
		Format: "json",
	}, &output)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	logger.Warn("proxy", "qnbbw06q", "upstream http proxy error")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if got, want := payload["level"], "warn"; got != want {
		t.Fatalf("level = %#v, want %#v", got, want)
	}
	if got, want := payload["module"], "proxy"; got != want {
		t.Fatalf("module = %#v, want %#v", got, want)
	}
	if got, want := payload["request_id"], "qnbbw06q"; got != want {
		t.Fatalf("request_id = %#v, want %#v", got, want)
	}
}

func TestMiddlewareEmitsAccessLogAtInfo(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := New(Config{
		Level:  "info",
		Format: "text",
	}, &output)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	handler := Middleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRouteKind(r.Context(), "internal")
		SetAuthenticated(r.Context(), false)
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodPost, "/__auth_proxy__/api/login", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	line := output.String()
	if !strings.Contains(line, "INFO") {
		t.Fatalf("line = %q, want INFO", line)
	}
	if !strings.Contains(line, "POST") {
		t.Fatalf("line = %q, want POST", line)
	}
	if !strings.Contains(line, "/__auth_proxy__/api/login") {
		t.Fatalf("line = %q, want path", line)
	}
	if !strings.Contains(line, "internal") {
		t.Fatalf("line = %q, want route kind", line)
	}
	if !strings.Contains(line, "201") {
		t.Fatalf("line = %q, want status", line)
	}
}
