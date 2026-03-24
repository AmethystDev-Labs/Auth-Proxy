# AuthProxy Design

## Goal

Build a Go-based reverse proxy that only forwards non-internal traffic when the request carries a valid AuthProxy session cookie. Unauthenticated HTTP requests should receive the embedded login page as a `401` response body, while unauthenticated WebSocket upgrades should receive a bare `401`.

## Scope

The proxy targets a single upstream origin and reserves only the ` /__auth_proxy__/` namespace for its own pages, APIs, assets, and health endpoint. All other paths are treated as upstream-owned routes and must never be shadowed by AuthProxy pages or APIs.

## Confirmed Decisions

- Single upstream base URL.
- Configuration comes from flags and environment variables, with flags taking precedence over environment values.
- Authentication uses a server-side in-memory session store with an `HttpOnly` cookie.
- Internal routes live only under ` /__auth_proxy__/`.
- Login page is served from `GET /__auth_proxy__/pages/login`.
- Internal APIs live under ` /__auth_proxy__/api/`.
- Unauthenticated non-internal HTTP requests return `401` plus the login page HTML.
- Unauthenticated non-internal WebSocket handshakes return `401` and are not upgraded.
- Frontend is implemented with React + shadcn/ui, built separately, and embedded into the Go binary.

## Architecture

### Request routing

Requests are handled in this order:

1. Match internal ` /__auth_proxy__/` routes.
2. For non-internal requests, check whether the request is a WebSocket upgrade.
3. Validate the AuthProxy session cookie.
4. If unauthenticated:
   - HTTP: return `401` with the embedded login page HTML.
   - WebSocket: return `401` without upgrading.
5. If authenticated, proxy the request to the configured upstream.

### Backend packages

- `cmd/authproxy`: process entrypoint.
- `internal/config`: flag and environment parsing, validation, and defaults.
- `internal/session`: in-memory session store and cookie helpers.
- `internal/server`: top-level HTTP handler tree, auth gate, internal APIs, and login page handling.
- `internal/proxy`: HTTP reverse proxy and WebSocket tunneling.
- `internal/web`: embedded frontend assets and login page rendering helpers.

### Authentication flow

1. Client accesses any non-internal path without a valid AuthProxy cookie.
2. AuthProxy returns `401` with the login page HTML.
3. Frontend posts credentials to `POST /__auth_proxy__/api/login`.
4. Backend validates credentials against configured static username/password and issues a session cookie.
5. Frontend reloads the current page or navigates to the `next` target when using the explicit login page route.
6. Authenticated requests are proxied upstream.
7. Logout clears both the cookie and any in-memory session entry.

### Upstream forwarding

- Normal HTTP proxying uses `httputil.ReverseProxy`.
- WebSocket proxying uses hijacking plus raw bidirectional streaming so upgrade traffic is preserved.
- AuthProxy strips only its own session cookie before forwarding to upstream.
- Existing upstream cookies and headers are preserved.
- Forwarding headers include `X-Forwarded-For`, `X-Forwarded-Host`, and `X-Forwarded-Proto`.

## Internal endpoints

- `GET /__auth_proxy__/pages/login` -> login page HTML
- `POST /__auth_proxy__/api/login` -> validates credentials, sets session cookie
- `POST /__auth_proxy__/api/logout` -> idempotently clears session and cookie
- `GET /__auth_proxy__/api/session` -> returns authenticated state
- `GET /__auth_proxy__/healthz` -> liveness response
- `GET /__auth_proxy__/assets/*` -> embedded frontend assets

## Frontend behavior

- Vite `base` is ` /__auth_proxy__/`.
- When rendered as a `401` body on an upstream path, successful login reloads the current page.
- When rendered from ` /__auth_proxy__/pages/login`, the frontend reads `next` from the query string and falls back to `/`.
- The page uses a compact shadcn-style card form and talks only to AuthProxy internal APIs.

## Error handling

- Invalid configuration fails fast at startup.
- Login failures return `401` JSON without revealing whether username or password was wrong.
- Invalid or expired sessions behave the same as missing sessions.
- Proxy dial or upstream errors return `502 Bad Gateway`.

## Test strategy

- Config parsing tests for defaulting and flag-over-env precedence.
- Session and auth endpoint tests for login, session lookup, logout, and cookie issuance.
- Server tests for `401` login-page responses on unauthenticated HTTP requests.
- HTTP proxy tests for authenticated forwarding and cookie stripping.
- WebSocket tests for authenticated upgrade forwarding and unauthenticated `401` rejection.
