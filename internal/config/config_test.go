package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLoadRequiresUpstreamAndCredentials(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected missing configuration error")
	}
}

func TestLoadUsesEnvDefaultsAndFlagOverrides(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"AUTH_PROXY_LISTEN_ADDR":         ":9090",
		"AUTH_PROXY_UPSTREAM_URL":        "http://env-upstream.example",
		"AUTH_PROXY_USERNAME":            "env-user",
		"AUTH_PROXY_PASSWORD":            "env-pass",
		"AUTH_PROXY_SESSION_COOKIE_NAME": "env_cookie",
		"AUTH_PROXY_SESSION_TTL":         "45m",
		"AUTH_PROXY_COOKIE_SECURE":       "true",
		"AUTH_PROXY_LOG_LEVEL":           "warn",
		"AUTH_PROXY_LOG_FORMAT":          "text",
		"AUTH_PROXY_LOG_ADD_SOURCE":      "false",
	}

	cfg, err := Load([]string{
		"--listen-addr=:8088",
		"--upstream-url=http://flag-upstream.example",
		"--username=flag-user",
		"--password=flag-pass",
		"--session-cookie-name=flag_cookie",
		"--session-ttl=2h",
		"--cookie-secure=false",
		"--log-level=debug",
		"--log-format=json",
		"--log-add-source=true",
	}, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := cfg.ListenAddr, ":8088"; got != want {
		t.Fatalf("ListenAddr = %q, want %q", got, want)
	}
	if got, want := cfg.Upstream.String(), "http://flag-upstream.example"; got != want {
		t.Fatalf("Upstream = %q, want %q", got, want)
	}
	if got, want := cfg.AuthUsername, "flag-user"; got != want {
		t.Fatalf("AuthUsername = %q, want %q", got, want)
	}
	if got, want := cfg.AuthPassword, "flag-pass"; got != want {
		t.Fatalf("AuthPassword = %q, want %q", got, want)
	}
	if got, want := cfg.SessionCookieName, "flag_cookie"; got != want {
		t.Fatalf("SessionCookieName = %q, want %q", got, want)
	}
	if got, want := cfg.SessionTTL, 2*time.Hour; got != want {
		t.Fatalf("SessionTTL = %s, want %s", got, want)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure = true, want false")
	}
	if got, want := cfg.LogLevel, "debug"; got != want {
		t.Fatalf("LogLevel = %q, want %q", got, want)
	}
	if got, want := cfg.LogFormat, "json"; got != want {
		t.Fatalf("LogFormat = %q, want %q", got, want)
	}
	if !cfg.LogAddSource {
		t.Fatal("LogAddSource = false, want true")
	}
}

func TestLoadAcceptsDoubleDashLongFlags(t *testing.T) {
	t.Parallel()

	cfg, err := Load([]string{
		"--listen-addr=:8088",
		"--upstream-url=http://flag-upstream.example",
		"--username=flag-user",
		"--password=flag-pass",
	}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := cfg.ListenAddr, ":8088"; got != want {
		t.Fatalf("ListenAddr = %q, want %q", got, want)
	}
	if got, want := cfg.Upstream.String(), "http://flag-upstream.example"; got != want {
		t.Fatalf("Upstream = %q, want %q", got, want)
	}
}

func TestLoadRejectsSingleDashLongFlags(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{
		"-upstream-url=http://flag-upstream.example",
		"--username=flag-user",
		"--password=flag-pass",
	}, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected long flag syntax error")
	}
	if !strings.Contains(err.Error(), "long flags must use --") {
		t.Fatalf("error = %q, want long flag syntax message", err)
	}
}

func TestLoadHelpFlagsReturnUsage(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"--help"}, {"-h"}} {
		_, err := Load(args, func(string) (string, bool) {
			return "", false
		})
		if err == nil {
			t.Fatalf("Load(%v) returned nil error, want help", args)
		}

		var helpErr *HelpError
		if !errors.As(err, &helpErr) {
			t.Fatalf("Load(%v) error = %T, want *HelpError", args, err)
		}
		if !strings.Contains(helpErr.Usage, "--upstream-url") {
			t.Fatalf("usage = %q, want --upstream-url", helpErr.Usage)
		}
		if !strings.Contains(helpErr.Usage, "-h, --help") {
			t.Fatalf("usage = %q, want -h, --help", helpErr.Usage)
		}
	}
}

func TestLoadLoggingDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load([]string{
		"--upstream-url=http://flag-upstream.example",
		"--username=flag-user",
		"--password=flag-pass",
	}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := cfg.LogLevel, "info"; got != want {
		t.Fatalf("LogLevel = %q, want %q", got, want)
	}
	if got, want := cfg.LogFormat, "text"; got != want {
		t.Fatalf("LogFormat = %q, want %q", got, want)
	}
	if cfg.LogAddSource {
		t.Fatal("LogAddSource = true, want false")
	}
}
