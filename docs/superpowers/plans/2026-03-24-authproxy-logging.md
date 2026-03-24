# AuthProxy Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a zap-backed logging system to AuthProxy with configurable `text/json` output, request access logs, and useful auth/proxy/server event logging.

**Architecture:** A new `internal/logging` package wraps `zap`, owns logger construction and request logging middleware, and exposes a narrow interface to the server and proxy layers. Config parsing is extended with logging settings, then the main entrypoint wires the logger through the server, proxy, and access middleware.

**Tech Stack:** Go 1.26, `go.uber.org/zap`, standard library HTTP middleware, existing AuthProxy packages

---

### Task 1: Extend config and document the logging feature

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write failing config tests for log defaults and flag-over-env precedence**
- [ ] **Step 2: Run `go test ./internal/config` and confirm failure**
- [ ] **Step 3: Implement minimal logging config parsing and validation**
- [ ] **Step 4: Re-run `go test ./internal/config` and confirm pass**

### Task 2: Build the logging package with TDD

**Files:**
- Create: `internal/logging/logger.go`
- Create: `internal/logging/middleware.go`
- Create: `internal/logging/logger_test.go`

- [ ] **Step 1: Write failing tests for text formatting, JSON formatting, and access-log output**
- [ ] **Step 2: Run `go test ./internal/logging` and confirm failure**
- [ ] **Step 3: Implement the minimal zap-backed logger and access middleware**
- [ ] **Step 4: Re-run `go test ./internal/logging` and confirm pass**

### Task 3: Integrate logging into server and proxy layers with TDD

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`
- Modify: `internal/proxy/http.go`
- Modify: `internal/proxy/websocket.go`
- Modify: `internal/proxy/proxy_test.go`

- [ ] **Step 1: Write failing tests for auth logs, unauthenticated-block logs, and proxy error logs**
- [ ] **Step 2: Run `go test ./internal/server ./internal/proxy` and confirm failure**
- [ ] **Step 3: Inject the logger into server and proxy packages and add the event logs**
- [ ] **Step 4: Re-run `go test ./internal/server ./internal/proxy` and confirm pass**

### Task 4: Wire logging into the entrypoint and verify

**Files:**
- Modify: `cmd/authproxy/main.go`
- Modify: `cmd/authproxy/main_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write or update tests for startup wiring and help output if needed**
- [ ] **Step 2: Create the logger in `main`, add startup logging, and wrap the handler with access-log middleware**
- [ ] **Step 3: Run `go test ./... -count=1` and fix failures**
- [ ] **Step 4: Run `go build -o .\\bin\\authproxy.exe .\\cmd\\authproxy` and confirm the binary builds**
