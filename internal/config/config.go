package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenAddr        = ":8080"
	defaultSessionCookieName = "auth_proxy_session"
	defaultSessionTTL        = 24 * time.Hour
)

type LookupEnv func(string) (string, bool)

type Config struct {
	ListenAddr        string
	Upstream          *url.URL
	AuthUsername      string
	AuthPassword      string
	SessionCookieName string
	SessionTTL        time.Duration
	CookieSecure      bool
	LogLevel          string
	LogFormat         string
	LogAddSource      bool
}

type HelpError struct {
	Usage string
}

func (e *HelpError) Error() string {
	return "help requested"
}

func Load(args []string, lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}

	listenAddr := envOrDefault(lookup, "AUTH_PROXY_LISTEN_ADDR", defaultListenAddr)
	upstreamRaw := envOrDefault(lookup, "AUTH_PROXY_UPSTREAM_URL", "")
	username := envOrDefault(lookup, "AUTH_PROXY_USERNAME", "")
	password := envOrDefault(lookup, "AUTH_PROXY_PASSWORD", "")
	cookieName := envOrDefault(lookup, "AUTH_PROXY_SESSION_COOKIE_NAME", defaultSessionCookieName)
	sessionTTL := envOrDefault(lookup, "AUTH_PROXY_SESSION_TTL", defaultSessionTTL.String())
	logLevel := envOrDefault(lookup, "AUTH_PROXY_LOG_LEVEL", "info")
	logFormat := envOrDefault(lookup, "AUTH_PROXY_LOG_FORMAT", "text")

	cookieSecure, err := envBoolOrDefault(lookup, "AUTH_PROXY_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	logAddSource, err := envBoolOrDefault(lookup, "AUTH_PROXY_LOG_ADD_SOURCE", false)
	if err != nil {
		return Config{}, err
	}

	normalizedArgs, err := normalizeArgs(args)
	if err != nil {
		return Config{}, err
	}

	fs := flag.NewFlagSet("authproxy", flag.ContinueOnError)
	var showHelp bool
	fs.StringVar(&listenAddr, "listen-addr", listenAddr, "listen address")
	fs.StringVar(&upstreamRaw, "upstream-url", upstreamRaw, "upstream base URL")
	fs.StringVar(&username, "username", username, "login username")
	fs.StringVar(&password, "password", password, "login password")
	fs.StringVar(&cookieName, "session-cookie-name", cookieName, "session cookie name")
	fs.StringVar(&sessionTTL, "session-ttl", sessionTTL, "session TTL")
	fs.BoolVar(&cookieSecure, "cookie-secure", cookieSecure, "mark session cookies secure")
	fs.StringVar(&logLevel, "log-level", logLevel, "log level")
	fs.StringVar(&logFormat, "log-format", logFormat, "log format")
	fs.BoolVar(&logAddSource, "log-add-source", logAddSource, "include source location in logs")
	fs.BoolVar(&showHelp, "help", false, "show help and exit")
	fs.BoolVar(&showHelp, "h", false, "show help and exit")

	if err := fs.Parse(normalizedArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, &HelpError{Usage: Usage()}
		}
		return Config{}, err
	}
	if showHelp {
		return Config{}, &HelpError{Usage: Usage()}
	}

	if upstreamRaw == "" {
		return Config{}, errors.New("missing upstream URL")
	}
	if username == "" || password == "" {
		return Config{}, errors.New("missing username or password")
	}

	upstream, err := url.Parse(upstreamRaw)
	if err != nil {
		return Config{}, fmt.Errorf("parse upstream URL: %w", err)
	}
	if upstream.Scheme == "" || upstream.Host == "" {
		return Config{}, errors.New("upstream URL must include scheme and host")
	}

	ttl, err := time.ParseDuration(sessionTTL)
	if err != nil {
		return Config{}, fmt.Errorf("parse session TTL: %w", err)
	}
	if ttl <= 0 {
		return Config{}, errors.New("session TTL must be greater than zero")
	}

	return Config{
		ListenAddr:        listenAddr,
		Upstream:          upstream,
		AuthUsername:      username,
		AuthPassword:      password,
		SessionCookieName: cookieName,
		SessionTTL:        ttl,
		CookieSecure:      cookieSecure,
		LogLevel:          strings.ToLower(logLevel),
		LogFormat:         strings.ToLower(logFormat),
		LogAddSource:      logAddSource,
	}, nil
}

func Usage() string {
	return strings.TrimSpace(`
Usage:
  authproxy [options]

Options:
  --listen-addr <addr>             Listen address. Env: AUTH_PROXY_LISTEN_ADDR. Default: :8080
  --upstream-url <url>             Upstream base URL. Env: AUTH_PROXY_UPSTREAM_URL. Required.
  --username <name>                Login username. Env: AUTH_PROXY_USERNAME. Required.
  --password <value>               Login password. Env: AUTH_PROXY_PASSWORD. Required.
  --session-cookie-name <name>     Session cookie name. Env: AUTH_PROXY_SESSION_COOKIE_NAME. Default: auth_proxy_session
  --session-ttl <duration>         Session TTL. Env: AUTH_PROXY_SESSION_TTL. Default: 24h
  --cookie-secure <bool>           Mark session cookies Secure. Env: AUTH_PROXY_COOKIE_SECURE. Default: false
  --log-level <level>              Log level. Env: AUTH_PROXY_LOG_LEVEL. Default: info
  --log-format <format>            Log format (text|json). Env: AUTH_PROXY_LOG_FORMAT. Default: text
  --log-add-source <bool>          Include source location in logs. Env: AUTH_PROXY_LOG_ADD_SOURCE. Default: false
  -h, --help                       Show help and exit

Flag values override environment variables.
`) + "\n"
}

func envOrDefault(lookup LookupEnv, key, fallback string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}
	return fallback
}

func envBoolOrDefault(lookup LookupEnv, key string, fallback bool) (bool, error) {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s as bool: %w", key, err)
	}
	return parsed, nil
}

func normalizeArgs(args []string) ([]string, error) {
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		switch {
		case arg == "-h":
			normalized = append(normalized, "-h")
		case arg == "--help":
			normalized = append(normalized, "-help")
		case arg == "--":
			normalized = append(normalized, arg)
		case strings.HasPrefix(arg, "--"):
			normalized = append(normalized, "-"+strings.TrimPrefix(arg, "--"))
		case strings.HasPrefix(arg, "-") && len(arg) > 2:
			return nil, fmt.Errorf("long flags must use --, got %q", arg)
		default:
			normalized = append(normalized, arg)
		}
	}
	return normalized, nil
}
