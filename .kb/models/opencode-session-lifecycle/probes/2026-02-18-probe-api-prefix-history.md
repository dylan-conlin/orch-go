# Probe: OpenCode API Prefix History (orch-go vs server)

**Model:** opencode-session-lifecycle
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Did orch-go ever use `/api/sessions` in `pkg/opencode/client.go`, or has it always used `/session`? Did the OpenCode server ever mount API routes under an `/api` prefix, and does the SPA frontend proxy strip `/api`?

---

## What I Tested

1. **orch-go client history**
   - `git log -S "/api" -- pkg/opencode/client.go`
   - `git log -S "/session" --oneline -- pkg/opencode/client.go`
   - `git show 26f9acba:pkg/opencode/client.go | rg "/session"`

2. **OpenCode server route history** (opencode repo)
   - `git log -S "route(\"/api\"" --oneline -- packages/opencode/src/server/server.ts`
   - `git show HEAD:packages/opencode/src/server/server.ts | sed -n '545,565p'`

3. **Current OpenCode server routing + SPA proxy config**
   - Read `packages/opencode/src/server/server.ts` (working tree) and `packages/opencode/src/server/routes/session.ts`
   - Read Vite configs:
     - `packages/app/vite.config.ts`
     - `packages/console/app/vite.config.ts`

---

## What I Observed

### orch-go client has always used `/session`

- `git log -S "/api" -- pkg/opencode/client.go` returned no commits.
- The earliest client commit (`26f9acba`) already uses `/session` (`http.Get(c.ServerURL + "/session")`).
- Current `pkg/opencode/client.go` uses `/session`, `/session/:id`, `/session/:id/message`, and `/event` — no `/api` prefix.

### OpenCode server history shows no committed `/api` mount

- `git log -S "route(\"/api\"" -- packages/opencode/src/server/server.ts` returned no commits.
- The committed `HEAD` version of `packages/opencode/src/server/server.ts` shows the API router mounted without a `/api` prefix (direct `.all("/*", ...)` after API setup).

### Working tree in opencode repo *does* add `/api` route, but it's uncommitted

- `packages/opencode/src/server/server.ts` in the working tree contains:
  - `.route("/", api)`
  - `.route("/api", api)`
  - followed by the SPA catch-all.
- `git status --short` in the opencode repo shows `server.ts` modified and not committed.

### SPA frontend does not proxy/strip `/api`

- `packages/app/vite.config.ts` and `packages/console/app/vite.config.ts` contain no `server.proxy` or rewrite rules.
- No evidence of `/api` stripping or proxying in the SPA configs.

---

## Model Impact

- [x] **Contradicts** model statement: "session created (POST /api/sessions)" — orch-go client has used `/session` since its initial commit, and OpenCode server routes are defined without an `/api` prefix in committed history.
- [x] **Extends** model with: **/api prefix only appears in local (uncommitted) OpenCode server changes**, not in repo history.
- [x] **Extends** model with: **No SPA proxy stripping `/api`** in current Vite configs.
