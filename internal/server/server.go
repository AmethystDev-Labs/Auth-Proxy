package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"authproxy/internal/config"
	"authproxy/internal/logging"
	"authproxy/internal/proxy"
	"authproxy/internal/session"
	"go.uber.org/zap"
)

const internalPrefix = "/__auth_proxy__"

type Options struct {
	Config    config.Config
	Sessions  *session.Store
	Logger    logging.Logger
	LoginPage []byte
	Assets    http.Handler
	HTTPProxy http.Handler
	WSProxy   http.Handler
}

type Server struct {
	config    config.Config
	sessions  *session.Store
	logger    logging.Logger
	loginPage []byte
	assets    http.Handler
	httpProxy http.Handler
	wsProxy   http.Handler
}

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionPayload struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
}

func New(options Options) (*Server, error) {
	if len(options.LoginPage) == 0 {
		return nil, errors.New("login page is required")
	}

	if options.Sessions == nil {
		options.Sessions = session.NewStore(time.Now)
	}
	if options.Logger == nil {
		options.Logger = logging.Nop()
	}
	if options.Assets == nil {
		options.Assets = http.NotFoundHandler()
	}
	if options.HTTPProxy == nil {
		options.HTTPProxy = http.NotFoundHandler()
	}
	if options.WSProxy == nil {
		options.WSProxy = http.NotFoundHandler()
	}

	return &Server{
		config:    options.Config,
		sessions:  options.Sessions,
		logger:    options.Logger,
		loginPage: append([]byte(nil), options.LoginPage...),
		assets:    options.Assets,
		httpProxy: options.HTTPProxy,
		wsProxy:   options.WSProxy,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isInternalPath(r.URL.Path) {
		logging.SetRouteKind(r.Context(), "internal")
		if s.isAuthenticated(r) {
			logging.SetAuthenticated(r.Context(), true)
		}
		s.serveInternal(w, r)
		return
	}

	requestID := logging.RequestID(r.Context())
	if proxy.IsWebSocketRequest(r) {
		if !s.isAuthenticated(r) {
			logging.SetRouteKind(r.Context(), "blocked_ws")
			s.logger.Warn("auth", requestID, "unauthorized websocket request blocked",
				zap.String("path", r.URL.Path),
			)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		logging.SetRouteKind(r.Context(), "upstream")
		logging.SetAuthenticated(r.Context(), true)
		s.wsProxy.ServeHTTP(w, r)
		return
	}

	if !s.isAuthenticated(r) {
		logging.SetRouteKind(r.Context(), "blocked_http")
		s.logger.Warn("auth", requestID, "unauthorized http request blocked",
			zap.String("path", r.URL.Path),
		)
		s.writeLoginPage(w, http.StatusUnauthorized)
		return
	}

	logging.SetRouteKind(r.Context(), "upstream")
	logging.SetAuthenticated(r.Context(), true)
	s.httpProxy.ServeHTTP(w, r)
}

func (s *Server) serveInternal(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == internalPrefix+"/healthz":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	case r.URL.Path == internalPrefix+"/pages/login":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.writeLoginPage(w, http.StatusOK)
	case r.URL.Path == internalPrefix+"/api/login":
		s.handleLogin(w, r)
	case r.URL.Path == internalPrefix+"/api/logout":
		s.handleLogout(w, r)
	case r.URL.Path == internalPrefix+"/api/session":
		s.handleSession(w, r)
	case strings.HasPrefix(r.URL.Path, internalPrefix+"/assets/"):
		s.serveAssets(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload loginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	requestID := logging.RequestID(r.Context())
	if payload.Username != s.config.AuthUsername || payload.Password != s.config.AuthPassword {
		s.logger.Warn("auth", requestID, fmt.Sprintf("login failure username=%q", payload.Username),
			zap.String("username", payload.Username),
		)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	created := s.sessions.Create(payload.Username, s.config.SessionTTL)
	s.logger.Info("auth", requestID, fmt.Sprintf("login success username=%q", created.Username),
		zap.String("username", created.Username),
	)
	session.WriteSessionCookie(w, s.config.SessionCookieName, created.ID, s.config.SessionTTL, s.config.CookieSecure)
	writeJSON(w, http.StatusOK, sessionPayload{
		Authenticated: true,
		Username:      created.Username,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if cookie, err := r.Cookie(s.config.SessionCookieName); err == nil {
		s.sessions.Delete(cookie.Value)
	}

	s.logger.Info("auth", logging.RequestID(r.Context()), "logout")
	session.ClearSessionCookie(w, s.config.SessionCookieName, s.config.CookieSecure)
	writeJSON(w, http.StatusOK, sessionPayload{Authenticated: false})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	current, ok := s.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusOK, sessionPayload{Authenticated: false})
		return
	}

	writeJSON(w, http.StatusOK, sessionPayload{
		Authenticated: true,
		Username:      current.Username,
	})
}

func (s *Server) serveAssets(w http.ResponseWriter, r *http.Request) {
	trimmed := r.Clone(r.Context())
	trimmed.URL.Path = strings.TrimPrefix(r.URL.Path, internalPrefix)
	if trimmed.URL.Path == "" {
		trimmed.URL.Path = "/"
	}
	s.assets.ServeHTTP(w, trimmed)
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	_, ok := s.currentSession(r)
	return ok
}

func (s *Server) currentSession(r *http.Request) (session.Session, bool) {
	cookie, err := r.Cookie(s.config.SessionCookieName)
	if err != nil {
		return session.Session{}, false
	}
	return s.sessions.Get(cookie.Value)
}

func (s *Server) writeLoginPage(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(s.loginPage)
}

func isInternalPath(path string) bool {
	return path == internalPrefix || strings.HasPrefix(path, internalPrefix+"/")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
