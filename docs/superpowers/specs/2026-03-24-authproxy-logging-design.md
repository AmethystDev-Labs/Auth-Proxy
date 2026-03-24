# AuthProxy Logging Design

## Goal

Add a production-grade logging system to AuthProxy that supports configurable log level and output format, emits human-readable terminal logs in a fixed-width style, and records access, auth, and proxy events at useful points in the request lifecycle.

## Confirmed Decisions

- Use an existing logging library instead of a handwritten logger.
- Use `zap` as the logging backend.
- Support both `text` and `json` output formats.
- Log output goes to the console only.
- Add CLI and environment configuration for:
  - `log-level`
  - `log-format`
  - `log-add-source`
- Requests themselves should also be logged at `INFO`.

## Architecture

### Logging package

Create a dedicated `internal/logging` package that wraps `zap` and exposes a narrow application-facing interface. This package owns:

- logger construction from config
- text vs json output behavior
- request ID generation
- access logging middleware
- request-scoped metadata helpers for route kind and authentication state

### Output modes

#### Text mode

Text mode is optimized for terminal readability and should resemble:

```text
02:01:31  INFO   [auth]   [0000000a] login success username="alice"
02:01:33  INFO   POST   /protected                   upstream     auth=yes  200   3.60ms  127.0.0.1
02:01:59  WARN   [proxy]  [0000000b] upstream http proxy error: dial tcp 127.0.0.1:1: connectex: connection refused
```

Rules:

- time uses `HH:MM:SS`
- level is upper-case and fixed-width
- module tags appear as `[auth]`, `[proxy]`, `[ws]`, `[main]`
- request-scoped logs may include `[request-id]`
- access log lines are aligned by columns
- when `log-add-source=true`, append `file:line`

#### JSON mode

JSON mode keeps the same event coverage but emits structured records for external log collectors. Include fields such as:

- `ts`
- `level`
- `msg`
- `module`
- `request_id`
- `route_kind`
- `authenticated`
- `status`
- `duration`
- `source` when enabled

### Request logging

Wrap the top-level server handler in access-log middleware. The middleware should:

1. create a short request ID
2. attach per-request state to context
3. measure duration and capture response status
4. emit one `INFO` access log after the request completes

The request state tracks:

- request ID
- route kind: `internal`, `upstream`, `blocked_http`, `blocked_ws`
- authenticated state

### Application events

Log these events:

- startup summary
- login success
- login failure
- logout
- unauthenticated HTTP block
- unauthenticated WebSocket block
- HTTP proxy upstream errors
- WebSocket dial/write/hijack errors

## Config

Add to existing config loading:

- `--log-level` / `AUTH_PROXY_LOG_LEVEL`, default `info`
- `--log-format` / `AUTH_PROXY_LOG_FORMAT`, default `text`
- `--log-add-source` / `AUTH_PROXY_LOG_ADD_SOURCE`, default `false`

Flags still override environment variables.

## Test Strategy

- config tests for defaults and flag-over-env precedence
- logger tests for text formatting and JSON formatting
- logger middleware tests for access-log output shape
- server tests for login and unauthenticated-block log events
- proxy tests for HTTP and WebSocket error logging
