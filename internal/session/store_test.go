package session

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestStoreCreateGetDeleteAndExpiry(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	store := NewStore(func() time.Time { return current })

	created := store.Create("alice", 30*time.Minute)
	if created.ID == "" {
		t.Fatal("Create returned empty session ID")
	}

	session, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("Get returned no session after Create")
	}
	if got, want := session.Username, "alice"; got != want {
		t.Fatalf("Username = %q, want %q", got, want)
	}

	store.Delete(created.ID)
	if _, ok := store.Get(created.ID); ok {
		t.Fatal("Get returned deleted session")
	}

	expiring := store.Create("bob", time.Minute)
	current = current.Add(2 * time.Minute)
	if _, ok := store.Get(expiring.ID); ok {
		t.Fatal("Get returned expired session")
	}
}

func TestCookieHelpersWriteAndClear(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	WriteSessionCookie(recorder, "auth_proxy_session", "session-123", time.Hour, true)
	ClearSessionCookie(recorder, "auth_proxy_session", true)

	result := recorder.Result()
	cookies := result.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2", len(cookies))
	}

	if got, want := cookies[0].Name, "auth_proxy_session"; got != want {
		t.Fatalf("set cookie name = %q, want %q", got, want)
	}
	if got, want := cookies[0].Value, "session-123"; got != want {
		t.Fatalf("set cookie value = %q, want %q", got, want)
	}
	if !cookies[0].HttpOnly {
		t.Fatal("set cookie HttpOnly = false, want true")
	}
	if !cookies[0].Secure {
		t.Fatal("set cookie Secure = false, want true")
	}

	if got, want := cookies[1].MaxAge, -1; got != want {
		t.Fatalf("cleared cookie MaxAge = %d, want %d", got, want)
	}
	if got, want := cookies[1].Value, ""; got != want {
		t.Fatalf("cleared cookie value = %q, want empty", got)
	}
}
