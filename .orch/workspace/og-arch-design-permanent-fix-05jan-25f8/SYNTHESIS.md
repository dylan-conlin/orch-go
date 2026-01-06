# Session Synthesis

**Agent:** og-arch-design-permanent-fix-05jan-25f8
**Issue:** ad-hoc (--no-track)
**Duration:** 2026-01-05 → 2026-01-05
**Outcome:** success

---

## TLDR

Designed permanent fix for recurring HTTP/1.1 connection pool exhaustion. **HTTP/2 with TLS is recommended** - it eliminates the 6-connection limit at the protocol level with a one-line server change plus TLS setup. Single multiplexed SSE is a viable alternative with lower complexity.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-05-design-permanent-fix-http-connection-pool.md` - Full design investigation with recommendations

### Files Modified
- None (design-only session)

### Commits
- Pending commit for investigation file

---

## Evidence (What Was Observed)

- HTTP/1.1 limits browsers to 6 connections per origin (RFC 7230 Section 6.4)
- Dashboard uses two SSE endpoints: `/api/events` and `/api/agentlog`
- This is the 2nd or 3rd time connection pool exhaustion has occurred
- Prior workarounds made agentlog SSE opt-in (band-aid, not fix)
- Go 1.24 (`go version` shows 1.24.11) supports HTTP/2 natively
- Current server uses `http.ListenAndServe(addr, mux)` at `cmd/orch/serve.go:289`
- SSE handlers use standard ResponseWriter (HTTP/2 compatible)

### Tests Run
```bash
# Verified Go version supports HTTP/2
go version
# go version go1.24.11 darwin/arm64

# Found server setup location
grep -r "http.ListenAndServe" --include="*.go"
# ./cmd/orch/serve.go:289
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-design-permanent-fix-http-connection-pool.md` - Full design analysis with 5 options evaluated

### Decisions Made
- Decision 1: HTTP/2 is recommended over single SSE multiplex because it eliminates the constraint entirely rather than working around it
- Decision 2: WebSocket rejected as overkill for server→client only use case
- Decision 3: Different port/origin rejected as adding complexity without solving root cause

### Constraints Discovered
- HTTP/2 requires TLS for browsers (h2c not supported by browsers)
- Self-signed certs are acceptable for localhost development tooling
- Go's ServeMux and SSE handlers are HTTP/2 compatible without modification

### Externalized via `kn`
- None (design session, constraints are in investigation)

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate

**Question:** Which approach should be implemented for permanent fix?

**Options:**

1. **HTTP/2 with TLS** (Recommended)
   - **Pros:** Eliminates connection limit at protocol level, no JS changes, no handler changes, solves problem class
   - **Cons:** Requires TLS setup (self-signed cert), browser shows https://
   - **Effort:** ~2-4 hours (mostly TLS setup + testing)

2. **Single Multiplexed SSE**
   - **Pros:** Simpler (no TLS), reduces from 2 to 1 connection
   - **Cons:** Still limited to 5 remaining connections, doesn't scale to more SSE types
   - **Effort:** ~4-6 hours (backend + frontend changes)

3. **Keep current workaround** (Not recommended)
   - **Pros:** No work needed
   - **Cons:** Problem will recur when more SSE streams are added
   - **Effort:** 0

**Recommendation:** Option 1 (HTTP/2 with TLS)

This is the architectural fix vs workaround. The recurring nature of this bug (2nd or 3rd time) matches the spawn context constraint: "High patch density signals missing coherent model." HTTP/2 provides that coherent model.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- Should `mkcert` be used for cert generation (easier UX) vs `openssl` (no extra dependency)?
- Should the cert be committed to repo or generated at build time?
- Does CI/CD need HTTP/2 for integration tests or can tests stay HTTP/1.1?

**Areas worth exploring further:**

- HTTP/2 server push for preloading critical resources
- gRPC for agent communication (also uses HTTP/2)

**What remains unclear:**

- Whether SSE reconnection behavior differs over HTTP/2 (needs manual testing)
- Whether there are any Firefox/Safari edge cases with HTTP/2 SSE

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-permanent-fix-05jan-25f8/`
**Investigation:** `.kb/investigations/2026-01-05-design-permanent-fix-http-connection-pool.md`
**Beads:** ad-hoc (--no-track)
