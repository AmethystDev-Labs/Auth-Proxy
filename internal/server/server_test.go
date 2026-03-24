package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"authproxy/internal/config"
	"authproxy/internal/logging"
	"authproxy/internal/session"
)

func TestServerLoginSessionLogoutFlow(t *testing.T) {
	t.Parallel()

	store := session.NewStore(time.Now)
	handler, err := New(Options{
		Config: config.Config{
			AuthUsername:      "alice",
			AuthPassword:      "secret",
			SessionCookieName: "auth_proxy_session",
			SessionTTL:        time.Hour,
		},
		Sessions:  store,
		LoginPage: []byte("<html><body>login</body></html>"),
		Assets:    http.NotFoundHandler(),
		HTTPProxy: http.NotFoundHandler(),
		WSProxy:   http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/__auth_proxy__/api/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(loginRecorder, loginRequest)

	if got, want := loginRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("login status = %d, want %d", got, want)
	}

	loginCookie := loginRecorder.Result().Cookies()
	if len(loginCookie) == 0 {
		t.Fatal("login did not set a session cookie")
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/__auth_proxy__/api/session", nil)
	sessionRequest.AddCookie(loginCookie[0])
	sessionRecorder := httptest.NewRecorder()
	handler.ServeHTTP(sessionRecorder, sessionRequest)

	if got, want := sessionRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("session status = %d, want %d", got, want)
	}
	if !strings.Contains(sessionRecorder.Body.String(), `"authenticated":true`) {
		t.Fatalf("session body = %q, want authenticated true", sessionRecorder.Body.String())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/__auth_proxy__/api/logout", nil)
	logoutRequest.AddCookie(loginCookie[0])
	logoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(logoutRecorder, logoutRequest)

	if got, want := logoutRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("logout status = %d, want %d", got, want)
	}
}

func TestServerReturnsLoginPageForUnauthenticatedHTTP(t *testing.T) {
	t.Parallel()

	proxied := false
	handler, err := New(Options{
		Config: config.Config{
			AuthUsername:      "alice",
			AuthPassword:      "secret",
			SessionCookieName: "auth_proxy_session",
			SessionTTL:        time.Hour,
		},
		Sessions:  session.NewStore(time.Now),
		LoginPage: []byte("<html><body>login form</body></html>"),
		Assets:    http.NotFoundHandler(),
		HTTPProxy: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { proxied = true }),
		WSProxy:   http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusUnauthorized; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if proxied {
		t.Fatal("request was proxied without a valid session")
	}
	if got, want := recorder.Body.String(), "<html><body>login form</body></html>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestServerRejectsUnauthenticatedWebSocketRequests(t *testing.T) {
	t.Parallel()

	wsProxyCalled := false
	handler, err := New(Options{
		Config: config.Config{
			AuthUsername:      "alice",
			AuthPassword:      "secret",
			SessionCookieName: "auth_proxy_session",
			SessionTTL:        time.Hour,
		},
		Sessions:  session.NewStore(time.Now),
		LoginPage: []byte("<html><body>login</body></html>"),
		Assets:    http.NotFoundHandler(),
		HTTPProxy: http.NotFoundHandler(),
		WSProxy: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			wsProxyCalled = true
		}),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/socket", nil)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Sec-WebSocket-Version", "13")
	request.Header.Set("Sec-WebSocket-Key", "dGVzdGtleQ==")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusUnauthorized; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if wsProxyCalled {
		t.Fatal("websocket proxy was called without a valid session")
	}
}

func TestServerLogsLoginSuccessAndAccessAtInfo(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := logging.New(logging.Config{
		Level:  "info",
		Format: "text",
	}, &output)
	if err != nil {
		t.Fatalf("logging.New returned error: %v", err)
	}

	store := session.NewStore(time.Now)
	handler, err := New(Options{
		Config: config.Config{
			AuthUsername:      "alice",
			AuthPassword:      "secret",
			SessionCookieName: "auth_proxy_session",
			SessionTTL:        time.Hour,
		},
		Sessions:  store,
		Logger:    logger,
		LoginPage: []byte("<html><body>login</body></html>"),
		Assets:    http.NotFoundHandler(),
		HTTPProxy: http.NotFoundHandler(),
		WSProxy:   http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	wrapped := logging.Middleware(logger, handler)
	loginRequest := httptest.NewRequest(http.MethodPost, "/__auth_proxy__/api/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	wrapped.ServeHTTP(loginRecorder, loginRequest)

	logs := output.String()
	if !strings.Contains(logs, "[auth]") {
		t.Fatalf("logs = %q, want [auth]", logs)
	}
	if !strings.Contains(logs, "login success") {
		t.Fatalf("logs = %q, want login success", logs)
	}
	if !strings.Contains(logs, "INFO") || !strings.Contains(logs, "POST") {
		t.Fatalf("logs = %q, want INFO access log", logs)
	}
}

func TestServerLogsUnauthorizedHTTPBlock(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := logging.New(logging.Config{
		Level:  "info",
		Format: "text",
	}, &output)
	if err != nil {
		t.Fatalf("logging.New returned error: %v", err)
	}

	handler, err := New(Options{
		Config: config.Config{
			AuthUsername:      "alice",
			AuthPassword:      "secret",
			SessionCookieName: "auth_proxy_session",
			SessionTTL:        time.Hour,
		},
		Sessions:  session.NewStore(time.Now),
		Logger:    logger,
		LoginPage: []byte("<html><body>login</body></html>"),
		Assets:    http.NotFoundHandler(),
		HTTPProxy: http.NotFoundHandler(),
		WSProxy:   http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	wrapped := logging.Middleware(logger, handler)
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()
	wrapped.ServeHTTP(recorder, request)

	logs := output.String()
	if !strings.Contains(logs, "[auth]") {
		t.Fatalf("logs = %q, want [auth]", logs)
	}
	if !strings.Contains(logs, "unauthorized http request blocked") {
		t.Fatalf("logs = %q, want unauthorized http request blocked", logs)
	}
}
