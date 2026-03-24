package proxy

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"authproxy/internal/logging"
)

func TestHTTPProxyForwardsAndStripsAuthCookie(t *testing.T) {
	t.Parallel()

	var upstreamCookie string
	var forwardedHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCookie = r.Header.Get("Cookie")
		forwardedHost = r.Header.Get("X-Forwarded-Host")
		fmt.Fprintf(w, "%s?%s", r.URL.Path, r.URL.RawQuery)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	handler := NewHTTP(target, "auth_proxy_session", logging.Nop())
	request := httptest.NewRequest(http.MethodGet, "http://proxy.local/protected?hello=world", nil)
	request.Host = "proxy.local"
	request.Header.Set("Cookie", "auth_proxy_session=secret; upstream_cookie=keep-me")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := recorder.Body.String(), "/protected?hello=world"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if strings.Contains(upstreamCookie, "auth_proxy_session") {
		t.Fatalf("upstream cookie = %q, auth cookie should have been stripped", upstreamCookie)
	}
	if !strings.Contains(upstreamCookie, "upstream_cookie=keep-me") {
		t.Fatalf("upstream cookie = %q, upstream cookie should have been preserved", upstreamCookie)
	}
	if got, want := forwardedHost, "proxy.local"; got != want {
		t.Fatalf("X-Forwarded-Host = %q, want %q", got, want)
	}
}

func TestWebSocketProxyForwardsUpgradeAndFrames(t *testing.T) {
	t.Parallel()

	var upstreamCookie string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("upstream response writer is not a hijacker")
		}

		upstreamCookie = r.Header.Get("Cookie")
		conn, rw, err := hijacker.Hijack()
		if err != nil {
			t.Fatalf("Hijack returned error: %v", err)
		}
		defer conn.Close()

		accept := websocketAccept(r.Header.Get("Sec-WebSocket-Key"))
		fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\n")
		fmt.Fprintf(rw, "Upgrade: websocket\r\n")
		fmt.Fprintf(rw, "Connection: Upgrade\r\n")
		fmt.Fprintf(rw, "Sec-WebSocket-Accept: %s\r\n\r\n", accept)
		if err := rw.Flush(); err != nil {
			t.Fatalf("Flush returned error: %v", err)
		}

		payload, err := readMaskedTextFrame(rw.Reader)
		if err != nil {
			t.Fatalf("readMaskedTextFrame returned error: %v", err)
		}
		if err := writeTextFrame(rw, payload); err != nil {
			t.Fatalf("writeTextFrame returned error: %v", err)
		}
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	proxyServer := httptest.NewServer(NewWebSocket(target, "auth_proxy_session", logging.Nop()))
	defer proxyServer.Close()

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	conn, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET /socket HTTP/1.1\r\n")
	fmt.Fprintf(conn, "Host: %s\r\n", proxyURL.Host)
	fmt.Fprintf(conn, "Connection: Upgrade\r\n")
	fmt.Fprintf(conn, "Upgrade: websocket\r\n")
	fmt.Fprintf(conn, "Sec-WebSocket-Version: 13\r\n")
	fmt.Fprintf(conn, "Sec-WebSocket-Key: dGVzdGtleQ==\r\n")
	fmt.Fprintf(conn, "Cookie: auth_proxy_session=secret; upstream_cookie=keep-me\r\n\r\n")

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString returned error: %v", err)
	}
	if !strings.Contains(statusLine, "101") {
		t.Fatalf("status line = %q, want 101 Switching Protocols", statusLine)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString returned error while reading headers: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	if err := writeMaskedTextFrame(conn, "hello over proxy"); err != nil {
		t.Fatalf("writeMaskedTextFrame returned error: %v", err)
	}

	echo, err := readTextFrame(reader)
	if err != nil {
		t.Fatalf("readTextFrame returned error: %v", err)
	}
	if got, want := echo, "hello over proxy"; got != want {
		t.Fatalf("echo = %q, want %q", got, want)
	}
	if strings.Contains(upstreamCookie, "auth_proxy_session") {
		t.Fatalf("upstream cookie = %q, auth cookie should have been stripped", upstreamCookie)
	}
	if !strings.Contains(upstreamCookie, "upstream_cookie=keep-me") {
		t.Fatalf("upstream cookie = %q, upstream cookie should have been preserved", upstreamCookie)
	}
}

func TestHTTPProxyLogsUpstreamErrors(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := logging.New(logging.Config{
		Level:  "info",
		Format: "text",
	}, &output)
	if err != nil {
		t.Fatalf("logging.New returned error: %v", err)
	}

	target, err := url.Parse("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	handler := NewHTTP(target, "auth_proxy_session", logger)
	request := httptest.NewRequest(http.MethodGet, "http://proxy.local/protected", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusBadGateway; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if logs := output.String(); !strings.Contains(logs, "upstream http proxy error") || !strings.Contains(logs, "[proxy]") {
		t.Fatalf("logs = %q, want proxy error log", logs)
	}
}

func TestWebSocketProxyLogsDialErrors(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger, err := logging.New(logging.Config{
		Level:  "info",
		Format: "text",
	}, &output)
	if err != nil {
		t.Fatalf("logging.New returned error: %v", err)
	}

	target, err := url.Parse("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	handler := NewWebSocket(target, "auth_proxy_session", logger)
	request := httptest.NewRequest(http.MethodGet, "/socket", nil)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Sec-WebSocket-Version", "13")
	request.Header.Set("Sec-WebSocket-Key", "dGVzdGtleQ==")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusBadGateway; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if logs := output.String(); !strings.Contains(logs, "upstream websocket dial failed") || !strings.Contains(logs, "[ws]") {
		t.Fatalf("logs = %q, want websocket error log", logs)
	}
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func writeMaskedTextFrame(w io.Writer, payload string) error {
	data := []byte(payload)
	mask := []byte{0x37, 0xfa, 0x21, 0x3d}

	frame := []byte{0x81, 0x80 | byte(len(data))}
	frame = append(frame, mask...)
	for i, b := range data {
		frame = append(frame, b^mask[i%len(mask)])
	}

	_, err := w.Write(frame)
	return err
}

func readMaskedTextFrame(r *bufio.Reader) (string, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return "", err
	}

	length := int(header[1] & 0x7f)
	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(r, maskKey); err != nil {
		return "", err
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return "", err
	}

	for i := range payload {
		payload[i] ^= maskKey[i%len(maskKey)]
	}

	return string(payload), nil
}

func writeTextFrame(w *bufio.ReadWriter, payload string) error {
	data := []byte(payload)
	frame := []byte{0x81, byte(len(data))}
	frame = append(frame, data...)
	if _, err := w.Write(frame); err != nil {
		return err
	}
	return w.Flush()
}

func readTextFrame(r *bufio.Reader) (string, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return "", err
	}

	length := int(header[1] & 0x7f)
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return "", err
	}
	return string(payload), nil
}
