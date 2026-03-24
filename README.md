# AuthProxy

AuthProxy is a Go reverse proxy that only forwards non-internal HTTP and WebSocket traffic when the browser carries a valid AuthProxy session cookie. Its own routes are reserved under ` /__auth_proxy__/`, and the login UI is a React + shadcn-style frontend embedded into the Go binary.

## Features

- Reverse proxies both normal HTTP traffic and WebSocket upgrades.
- Blocks unauthenticated upstream requests and serves the embedded login page as the `401` body for HTTP.
- Keeps all internal AuthProxy pages, APIs, assets, and health checks under ` /__auth_proxy__/`.
- Uses an in-memory server-side session store with an `HttpOnly` cookie.
- Embeds the React + shadcn/ui login frontend into the Go binary.
- Supports structured console logging in `text` and `json` modes.

## Behavior

- Non-internal HTTP request without a valid AuthProxy cookie: returns `401` with the login page HTML as the response body.
- Non-internal WebSocket upgrade without a valid AuthProxy cookie: returns bare `401`.
- Authenticated non-internal requests: proxied to the configured upstream.
- Internal AuthProxy routes:
  - `GET /__auth_proxy__/pages/login`
  - `POST /__auth_proxy__/api/login`
  - `POST /__auth_proxy__/api/logout`
  - `GET /__auth_proxy__/api/session`
  - `GET /__auth_proxy__/healthz`
  - `GET /__auth_proxy__/assets/*`

## Configuration

Configuration is loaded from flags and environment variables, with flags taking precedence.

| Flag | Environment Variable | Required | Default |
| --- | --- | --- | --- |
| `--listen-addr` | `AUTH_PROXY_LISTEN_ADDR` | No | `:8080` |
| `--upstream-url` | `AUTH_PROXY_UPSTREAM_URL` | Yes | |
| `--username` | `AUTH_PROXY_USERNAME` | Yes | |
| `--password` | `AUTH_PROXY_PASSWORD` | Yes | |
| `--session-cookie-name` | `AUTH_PROXY_SESSION_COOKIE_NAME` | No | `auth_proxy_session` |
| `--session-ttl` | `AUTH_PROXY_SESSION_TTL` | No | `24h` |
| `--cookie-secure` | `AUTH_PROXY_COOKIE_SECURE` | No | `false` |
| `--log-level` | `AUTH_PROXY_LOG_LEVEL` | No | `info` |
| `--log-format` | `AUTH_PROXY_LOG_FORMAT` | No | `text` |
| `--log-add-source` | `AUTH_PROXY_LOG_ADD_SOURCE` | No | `false` |

Example:

```powershell
go run ./cmd/authproxy `
  --upstream-url=http://127.0.0.1:3000 `
  --username=admin `
  --password=secret `
  --log-level=info `
  --log-format=text
```

## Logging

AuthProxy uses a zap-backed logging layer with two output modes:

- `text`: fixed-width terminal-friendly lines for local runs and simple process managers
- `json`: structured output for container or external log collection

The server emits:

- startup logs
- auth logs for login success/failure, logout, and unauthenticated request blocking
- proxy error logs for HTTP and WebSocket upstream failures
- one `INFO` access log line per request

Typical text output looks like:

```text
15:04:05  INFO  [main]  starting authproxy listen=":8080" upstream="http://127.0.0.1:3000" log_level="info" log_format="text"
15:04:11  WARN  [auth]  [00000002]  unauthorized http request blocked
15:04:11  INFO  GET    /protected                   blocked_http auth=no  401  1.213ms  127.0.0.1
15:04:20  INFO  [auth]  [00000003]  login success username="admin"
15:04:20  INFO  POST   /__auth_proxy__/api/login    internal     auth=no  200  3.174ms  127.0.0.1
```

## Local Development

Install frontend dependencies and build the embedded site:

```powershell
cd web
npm install
npm run build
```

Run tests and build the binary from the repository root:

```powershell
$env:GOCACHE="$PWD\\.gocache"
$env:GOMODCACHE="$PWD\\.gomodcache"
go test ./...
go build -o .\\bin\\authproxy.exe .\\cmd\\authproxy
```

## CI/CD

GitHub Actions now runs on every branch `push` and performs the full preview pipeline:

- `npm ci`, `npm test`, and `npm run build` in [`web`](web)
- `go test ./... -count=1`
- cross-compiles release archives for:
  - Linux `amd64` and `arm64`
  - macOS `amd64` and `arm64`
  - Windows `amd64` and `arm64`
- publishes a GitHub prerelease named like `preview-main-r12-a1-abcdef0`

The preview release workflow is defined in [preview-release.yml](.github/workflows/preview-release.yml). Every generated release is marked as a prerelease and explicitly created with `--latest=false`, so it does not take over the repository's Latest release slot.

## Repository Layout

- `cmd/authproxy`: executable entrypoint
- `internal/config`: CLI and environment config loading
- `internal/logging`: zap-backed logger and access-log middleware
- `internal/proxy`: HTTP reverse proxy and WebSocket tunneling
- `internal/server`: auth routes, gating logic, and internal handlers
- `internal/session`: in-memory session store and cookie helpers
- `internal/web`: embedded frontend assets
- `web`: React + Vite frontend source

## License

This project is licensed under the GNU General Public License v3.0. See [LICENSE](LICENSE).
