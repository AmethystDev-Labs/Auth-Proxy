package logging

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewarePreservesHijacker(t *testing.T) {
	t.Parallel()

	handler := Middleware(Nop(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatalf("response writer does not implement http.Hijacker")
		}
		if _, _, err := hijacker.Hijack(); err != nil {
			t.Fatalf("Hijack returned error: %v", err)
		}
	}))

	request := httptest.NewRequest(http.MethodGet, "/socket", nil)
	handler.ServeHTTP(&hijackableResponseWriter{header: make(http.Header)}, request)
}

type hijackableResponseWriter struct {
	header http.Header
}

func (w *hijackableResponseWriter) Header() http.Header {
	return w.header
}

func (w *hijackableResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (w *hijackableResponseWriter) WriteHeader(statusCode int) {}

func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, bufio.NewReadWriter(bufio.NewReader(nil), bufio.NewWriter(nil)), nil
}
