package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"authproxy/internal/logging"
	"go.uber.org/zap"
)

func NewHTTP(target *url.URL, cookieName string, logger logging.Logger) http.Handler {
	if logger == nil {
		logger = logging.Nop()
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			originalHost := req.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = joinPath(target.Path, req.URL.Path)
			req.URL.RawPath = req.URL.Path
			req.URL.RawQuery = joinQuery(target.RawQuery, req.URL.RawQuery)
			req.Host = target.Host

			req.Header.Set("X-Forwarded-Host", originalHost)
			req.Header.Set("X-Forwarded-Proto", forwardedProto(req))
			appendForwardedFor(req)

			stripAuthProxyCookie(req.Header, cookieName)
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			logger.Error("proxy", logging.RequestID(req.Context()), fmt.Sprintf("upstream http proxy error: %v", err),
				zap.Error(err),
				zap.String("path", req.URL.Path),
			)
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
	}

	return proxy
}

func StripCookieHeader(headerValue, cookieName string) string {
	if headerValue == "" {
		return ""
	}

	parts := strings.Split(headerValue, ";")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name := part
		if idx := strings.Index(part, "="); idx >= 0 {
			name = part[:idx]
		}
		if name == cookieName {
			continue
		}

		kept = append(kept, part)
	}

	return strings.Join(kept, "; ")
}

func stripAuthProxyCookie(header http.Header, cookieName string) {
	stripped := StripCookieHeader(header.Get("Cookie"), cookieName)
	if stripped == "" {
		header.Del("Cookie")
		return
	}
	header.Set("Cookie", stripped)
}

func appendForwardedFor(req *http.Request) {
	clientIP := clientIPFromRemoteAddr(req.RemoteAddr)
	if clientIP == "" {
		return
	}

	existing := req.Header.Get("X-Forwarded-For")
	if existing == "" {
		req.Header.Set("X-Forwarded-For", clientIP)
		return
	}
	req.Header.Set("X-Forwarded-For", existing+", "+clientIP)
}

func clientIPFromRemoteAddr(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

func forwardedProto(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

func joinPath(basePath, requestPath string) string {
	switch {
	case basePath == "":
		if requestPath == "" {
			return "/"
		}
		return requestPath
	case requestPath == "":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + requestPath[1:]
	case strings.HasSuffix(basePath, "/") || strings.HasPrefix(requestPath, "/"):
		return basePath + requestPath
	default:
		return basePath + "/" + requestPath
	}
}

func joinQuery(baseQuery, requestQuery string) string {
	switch {
	case baseQuery == "":
		return requestQuery
	case requestQuery == "":
		return baseQuery
	default:
		return baseQuery + "&" + requestQuery
	}
}
