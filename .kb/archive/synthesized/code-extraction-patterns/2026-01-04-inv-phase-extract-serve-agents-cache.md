<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully extracted ~441 lines of cache infrastructure from serve_agents.go to serve_agents_cache.go, reducing main file from 1400 to 970 lines.

**Evidence:** Build and tests pass: `go build ./cmd/orch/` succeeded, `go test ./cmd/orch/` passed.

**Knowledge:** Cache infrastructure (beadsCache, workspaceCache, globalWorkspaceCacheType) forms a cohesive unit that belongs together; keeping HTTP handlers separate improves maintainability.

**Next:** Phase 2: Extract serve_agents_events.go (~230 lines) containing SSE event handlers.

---

# Investigation: Phase Extract Serve Agents Cache

**Question:** Extract cache infrastructure from serve_agents.go to reduce file size

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Feature Agent
**Phase:** Complete
**Next Step:** None - Phase 1 complete
**Status:** Complete

**Prior Investigation:** `.kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md`

---

## Findings

### Finding 1: Extraction Successful

**Evidence:** 
- `serve_agents.go`: 970 lines (down from 1400)
- `serve_agents_cache.go`: 441 lines (new file)
- Build passes: `go build ./cmd/orch/`
- Tests pass: `go test ./cmd/orch/`

**Source:** `wc -l cmd/orch/serve_agents.go cmd/orch/serve_agents_cache.go`

**Significance:** Phase 1 of the extraction plan is complete. Main file reduced by ~430 lines.

---

### Finding 2: Components Extracted

**Evidence:** Moved to serve_agents_cache.go:
- beadsCache struct + 7 methods (TTL-based beads data caching)
- globalWorkspaceCacheType struct + 2 methods (workspace metadata caching)
- workspaceCache struct + 4 methods (workspace lookup maps)
- buildWorkspaceCache, buildMultiProjectWorkspaceCache functions
- extractUniqueProjectDirs function
- TTL constants (defaultOpenIssuesTTL, defaultAllIssuesTTL, defaultCommentsTTL)
- Global instances (globalBeadsCache, globalWorkspaceCacheInstance)

**Source:** `cmd/orch/serve_agents_cache.go`

**Significance:** All caching primitives now in one file, HTTP handlers remain in serve_agents.go.

---

## Synthesis

**Key Insights:**

1. **Clean separation achieved** - Cache infrastructure has no HTTP handler logic; handlers have no caching primitives.

2. **No import changes needed in handlers** - Both files are in package main, so cache types/functions are accessible without import changes.

3. **handleCacheInvalidate stays in serve_agents.go** - Per prior investigation, it's an HTTP handler that invalidates BOTH caches, so it belongs with handlers.

**Answer to Investigation Question:**

Extraction successful. Cache infrastructure (~441 lines) moved to serve_agents_cache.go. Build and tests pass. Main file reduced to 970 lines.

---

## Structured Uncertainty

**What's tested:**

- Build passes (verified: `go build ./cmd/orch/`)
- Tests pass (verified: `go test ./cmd/orch/` in 48.288s)
- No compilation errors

**What's untested:**

- Runtime behavior (not smoke-tested via dashboard)
- Performance impact (not benchmarked)

**What would change this:**

- Runtime errors would indicate missed dependencies
- Performance regression would require investigation

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Primary target, now 970 lines
- `cmd/orch/serve_agents_cache.go` - New file, 441 lines

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/

# Test verification
go test ./cmd/orch/

# Line count
wc -l cmd/orch/serve_agents.go cmd/orch/serve_agents_cache.go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md` - Design analysis for extraction plan

---

## Investigation History

**2026-01-04:** Investigation started
- Initial question: Extract cache infrastructure from serve_agents.go
- Context: Part of 2-phase extraction plan to reduce 1399-line file

**2026-01-04:** Investigation completed
- Status: Complete
- Key outcome: 441 lines extracted, build and tests pass
