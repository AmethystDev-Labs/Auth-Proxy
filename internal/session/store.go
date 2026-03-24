package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	ID        string
	Username  string
	ExpiresAt time.Time
}

type Store struct {
	mu       sync.RWMutex
	sessions map[string]Session
	now      func() time.Time
}

func NewStore(now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}

	return &Store{
		sessions: make(map[string]Session),
		now:      now,
	}
}

func (s *Store) Create(username string, ttl time.Duration) Session {
	session := Session{
		ID:        newSessionID(),
		Username:  username,
		ExpiresAt: s.now().Add(ttl),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return session
}

func (s *Store) Get(id string) (Session, bool) {
	if id == "" {
		return Session{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	if !session.ExpiresAt.After(s.now()) {
		delete(s.sessions, id)
		return Session{}, false
	}

	return session, true
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func WriteSessionCookie(w http.ResponseWriter, name, value string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
	})
}

func ClearSessionCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func newSessionID() string {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw[:])
}
