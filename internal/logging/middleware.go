package logging

import (
	"context"
	"net"
	"net/http"
	"time"
)

type requestStateKey struct{}

type requestState struct {
	RequestID     string
	RouteKind     string
	Authenticated bool
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func Middleware(logger Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = Nop()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := &requestState{
			RequestID: NextRequestID(),
			RouteKind: "unknown",
		}
		ctx := context.WithValue(r.Context(), requestStateKey{}, state)
		r = r.WithContext(ctx)

		recorder := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		start := time.Now()
		next.ServeHTTP(recorder, r)

		logger.Access(AccessLogEntry{
			Method:        r.Method,
			Path:          requestPath(r),
			RouteKind:     state.RouteKind,
			Authenticated: state.Authenticated,
			Status:        recorder.status,
			Duration:      time.Since(start),
			ClientIP:      clientIP(r.RemoteAddr),
		})
	})
}

func RequestID(ctx context.Context) string {
	if state, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		return state.RequestID
	}
	return ""
}

func SetRouteKind(ctx context.Context, routeKind string) {
	if state, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		state.RouteKind = routeKind
	}
}

func SetAuthenticated(ctx context.Context, authenticated bool) {
	if state, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		state.Authenticated = authenticated
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func requestPath(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return r.URL.Path
	}
	return r.URL.Path + "?" + r.URL.RawQuery
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}
