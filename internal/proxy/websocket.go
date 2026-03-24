package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"authproxy/internal/logging"
	"go.uber.org/zap"
)

func NewWebSocket(target *url.URL, cookieName string, logger logging.Logger) http.Handler {
	if logger == nil {
		logger = logging.Nop()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsWebSocketRequest(r) {
			http.Error(w, "websocket upgrade required", http.StatusBadRequest)
			return
		}

		upstreamConn, err := dialTarget(r.Context(), target)
		if err != nil {
			logger.Error("ws", logging.RequestID(r.Context()), fmt.Sprintf("upstream websocket dial failed: %v", err),
				zap.Error(err),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		outbound := cloneWebSocketRequest(r, target, cookieName)
		if err := outbound.Write(upstreamConn); err != nil {
			logger.Error("ws", logging.RequestID(r.Context()), fmt.Sprintf("upstream websocket write failed: %v", err),
				zap.Error(err),
			)
			upstreamConn.Close()
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			logger.Error("ws", logging.RequestID(r.Context()), "websocket hijacking unsupported")
			upstreamConn.Close()
			http.Error(w, "websocket hijacking unsupported", http.StatusInternalServerError)
			return
		}

		clientConn, clientRW, err := hijacker.Hijack()
		if err != nil {
			logger.Error("ws", logging.RequestID(r.Context()), fmt.Sprintf("websocket hijack failed: %v", err),
				zap.Error(err),
			)
			upstreamConn.Close()
			return
		}
		defer clientConn.Close()
		defer upstreamConn.Close()

		done := make(chan struct{}, 2)
		go copyBidirectional(done, clientConn, upstreamConn)
		go copyBidirectional(done, upstreamConn, clientRW)
		<-done
	})
}

func IsWebSocketRequest(r *http.Request) bool {
	return headerContainsToken(r.Header, "Connection", "upgrade") &&
		headerContainsToken(r.Header, "Upgrade", "websocket")
}

func cloneWebSocketRequest(r *http.Request, target *url.URL, cookieName string) *http.Request {
	outbound := r.Clone(r.Context())
	outbound.URL = &url.URL{
		Scheme:   target.Scheme,
		Host:     target.Host,
		Path:     joinPath(target.Path, r.URL.Path),
		RawQuery: joinQuery(target.RawQuery, r.URL.RawQuery),
	}
	outbound.Host = target.Host
	outbound.RequestURI = ""
	outbound.Header = cloneHeader(r.Header)
	outbound.Header.Del("Proxy-Connection")
	outbound.Header.Set("X-Forwarded-Host", r.Host)
	outbound.Header.Set("X-Forwarded-Proto", forwardedProto(r))
	appendForwardedFor(outbound)
	stripAuthProxyCookie(outbound.Header, cookieName)
	return outbound
}

func cloneHeader(header http.Header) http.Header {
	cloned := make(http.Header, len(header))
	for key, values := range header {
		copyValues := make([]string, len(values))
		copy(copyValues, values)
		cloned[key] = copyValues
	}
	return cloned
}

func dialTarget(ctx context.Context, target *url.URL) (net.Conn, error) {
	dialer := &net.Dialer{}
	switch target.Scheme {
	case "http", "ws", "":
		return dialer.DialContext(ctx, "tcp", target.Host)
	case "https", "wss":
		serverName := target.Hostname()
		return tls.DialWithDialer(dialer, "tcp", target.Host, &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: serverName,
		})
	default:
		return nil, &url.Error{Op: "dial", URL: target.String(), Err: errUnsupportedScheme(target.Scheme)}
	}
}

func copyBidirectional(done chan<- struct{}, dst io.WriteCloser, src io.Reader) {
	_, _ = io.Copy(dst, src)
	done <- struct{}{}
}

func headerContainsToken(header http.Header, key, token string) bool {
	for _, value := range header.Values(key) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

type unsupportedSchemeError string

func (e unsupportedSchemeError) Error() string {
	return "unsupported scheme: " + string(e)
}

func errUnsupportedScheme(scheme string) error {
	return unsupportedSchemeError(scheme)
}
