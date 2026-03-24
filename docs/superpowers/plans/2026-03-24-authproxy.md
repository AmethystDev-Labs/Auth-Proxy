# AuthProxy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go AuthProxy that gates all non-internal HTTP and WebSocket traffic behind a server-side session cookie and serves an embedded React login experience under ` /__auth_proxy__/`.

**Architecture:** The server owns only ` /__auth_proxy__/` routes and treats everything else as upstream-owned traffic. Requests pass through internal routing first, then an auth gate, then either the HTTP reverse proxy or the WebSocket tunnel. Frontend assets are built by Vite and embedded into the Go binary.

**Tech Stack:** Go 1.26, standard-library reverse proxying plus raw WebSocket tunneling, React, Vite, Tailwind CSS, shadcn-style UI components

---

### Task 1: Scaffold the repository

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `README.md`
- Create: `cmd/authproxy/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/session/store.go`
- Create: `internal/session/store_test.go`
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`
- Create: `internal/proxy/http.go`
- Create: `internal/proxy/websocket.go`
- Create: `internal/proxy/proxy_test.go`
- Create: `internal/web/embed.go`
- Create: `internal/web/render.go`
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/app.tsx`
- Create: `web/src/components/ui/button.tsx`
- Create: `web/src/components/ui/card.tsx`
- Create: `web/src/components/ui/input.tsx`
- Create: `web/src/components/ui/label.tsx`
- Create: `web/src/index.css`

- [ ] **Step 1: Create repo metadata and empty module skeleton**
- [ ] **Step 2: Add placeholder tests for config, session, server, and proxy packages**
- [ ] **Step 3: Run `go test ./...` and confirm the new tests fail for missing behavior**

### Task 2: Implement configuration and session primitives with TDD

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/session/store.go`
- Modify: `internal/session/store_test.go`

- [ ] **Step 1: Write failing config tests for defaults, required upstream URL, and flag-over-env precedence**
- [ ] **Step 2: Run `go test ./internal/config` and confirm failure**
- [ ] **Step 3: Implement minimal config parsing and validation**
- [ ] **Step 4: Re-run `go test ./internal/config` and confirm pass**
- [ ] **Step 5: Write failing session tests for create/get/delete and cookie clearing**
- [ ] **Step 6: Run `go test ./internal/session` and confirm failure**
- [ ] **Step 7: Implement minimal in-memory store and cookie helpers**
- [ ] **Step 8: Re-run `go test ./internal/session` and confirm pass**

### Task 3: Implement internal routes and auth gate with TDD

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`
- Modify: `internal/web/embed.go`
- Modify: `internal/web/render.go`

- [ ] **Step 1: Write failing tests for `healthz`, `api/session`, `api/login`, `api/logout`, and unauthenticated `401` login-page responses**
- [ ] **Step 2: Run `go test ./internal/server` and confirm failure**
- [ ] **Step 3: Implement the internal router, cookie-backed auth gate, and login-page rendering**
- [ ] **Step 4: Re-run `go test ./internal/server` and confirm pass**

### Task 4: Implement authenticated HTTP and WebSocket proxying with TDD

**Files:**
- Modify: `internal/proxy/http.go`
- Modify: `internal/proxy/websocket.go`
- Modify: `internal/proxy/proxy_test.go`
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`

- [ ] **Step 1: Write failing tests for authenticated HTTP forwarding, AuthProxy-cookie stripping, authenticated WebSocket tunneling, and unauthenticated WebSocket rejection**
- [ ] **Step 2: Run `go test ./internal/proxy ./internal/server` and confirm failure**
- [ ] **Step 3: Implement minimal HTTP reverse proxy and raw WebSocket tunneling**
- [ ] **Step 4: Re-run `go test ./internal/proxy ./internal/server` and confirm pass**

### Task 5: Build and embed the React login frontend

**Files:**
- Modify: `web/package.json`
- Modify: `web/vite.config.ts`
- Modify: `web/src/main.tsx`
- Modify: `web/src/app.tsx`
- Modify: `web/src/components/ui/button.tsx`
- Modify: `web/src/components/ui/card.tsx`
- Modify: `web/src/components/ui/input.tsx`
- Modify: `web/src/components/ui/label.tsx`
- Modify: `web/src/index.css`
- Modify: `internal/web/embed.go`

- [ ] **Step 1: Write or update backend tests that assert login-page HTML references embedded assets under ` /__auth_proxy__/assets/`**
- [ ] **Step 2: Implement the React + shadcn-style login form, session bootstrap, and login submission flow**
- [ ] **Step 3: Run `npm install` in `web/`**
- [ ] **Step 4: Run `npm run build` in `web/` and confirm assets land in the embedded dist directory**
- [ ] **Step 5: Re-run backend tests to confirm embedded asset serving still passes**

### Task 6: Verify the full project

**Files:**
- Modify: `README.md`
- Modify: any touched source or test files required by verification fixes

- [ ] **Step 1: Document configuration, routes, and local dev workflow in `README.md`**
- [ ] **Step 2: Run `go test ./...` and fix any failures**
- [ ] **Step 3: Run `npm run build` in `web/` and fix any frontend build failures**
- [ ] **Step 4: Re-run `go test ./...` to ensure the embedded frontend still passes**
