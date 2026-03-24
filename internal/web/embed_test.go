package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginPageHTMLIncludesAssetPaths(t *testing.T) {
	t.Parallel()

	page, err := LoginPage()
	if err != nil {
		t.Fatalf("LoginPage returned error: %v", err)
	}

	html := string(page)
	if !strings.Contains(html, "/__auth_proxy__/assets/app.js") {
		t.Fatalf("login page HTML does not reference app.js: %q", html)
	}
	if !strings.Contains(html, "/__auth_proxy__/assets/app.css") {
		t.Fatalf("login page HTML does not reference app.css: %q", html)
	}
}

func TestAssetsHandlerServesEmbeddedFiles(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	recorder := httptest.NewRecorder()

	AssetsHandler().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("embedded asset body is empty")
	}
}
