<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Successfully extracted agent-related handlers from serve.go into serve_agents.go, reducing serve.go from ~2921 to ~1815 lines.

**Evidence:** Build passes (`go build ./cmd/orch/`), all tests pass (`go test ./cmd/orch/`), committed as 763cb55d.

**Knowledge:** Agent handlers form a cohesive group (workspace cache, gap analysis, event streaming) that can be cleanly separated without breaking dependencies.

**Next:** Phase 2 - Extract serve_beads.go and serve_reviews.go (see `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md`).

---

# Investigation: Extract serve_agents.go from serve.go (Phase 1)

**Question:** How to extract agent-related handlers from serve.go into a dedicated file?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Related:** `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md` (parent strategy)

---

## Findings

### Finding 1: Agent handlers share workspace cache infrastructure

**Evidence:** `handleAgents`, `handleAgentlog`, `handleAgentlogJSON`, `handleAgentlogSSE` all use `workspaceCache` type and its lookup methods.

**Source:** `cmd/orch/serve.go` lines ~600-1100 (pre-extraction)

**Significance:** These handlers are tightly coupled through shared types, making them a natural extraction unit.

---

### Finding 2: Gap analysis and synthesis checking are agent-specific

**Evidence:** `checkWorkspaceSynthesis`, `getGapAnalysisFromEvents`, `extractGapAnalysisFromEvent` are only used by agent handlers.

**Source:** `cmd/orch/serve.go` lines ~1100-1300 (pre-extraction)

**Significance:** These helpers can be moved with the handlers without affecting other serve.go functionality.

---

### Finding 3: Event reading is shared but primarily agent-focused

**Evidence:** `readLastNEvents` is used by agent log handlers. The events package import was already present.

**Source:** `cmd/orch/serve.go`, `pkg/events/`

**Significance:** Moving `readLastNEvents` to serve_agents.go keeps related functionality together.

---

## Synthesis

**Key Insights:**

1. **Cohesive extraction unit** - Agent handlers + types + helpers form ~1100 lines that cleanly separate

2. **No cross-dependencies broken** - Remaining serve.go handlers don't depend on extracted code

3. **Test coverage moves cleanly** - 13 test functions moved to serve_agents_test.go with no modifications needed

**Answer to Investigation Question:**

Agent handlers can be extracted by moving the following groups:
- Types: `AgentAPIResponse`, `GapAPIResponse`, `SynthesisResponse`, `workspaceCache`
- Handlers: `handleAgents`, `handleEvents`, `handleAgentlog*`
- Helpers: workspace cache methods, `readLastNEvents`, gap analysis functions

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes (verified: `go build ./cmd/orch/`)
- ✅ All tests pass (verified: `go test ./cmd/orch/`)
- ✅ No import cycles introduced (verified: successful compilation)

**What's untested:**

- ⚠️ Runtime behavior with live agents (not tested in this session)
- ⚠️ Dashboard SSE streaming (not manually verified)

**What would change this:**

- Integration test failures in CI would indicate missed dependencies
- Runtime errors when serving agent data would indicate incomplete extraction

---

## Implementation Summary

### What was done:

1. **Created `cmd/orch/serve_agents.go`** (~1125 lines)
   - Types: `AgentAPIResponse`, `GapAPIResponse`, `SynthesisResponse`, `workspaceCache`
   - Handlers: `handleAgents`, `handleEvents`, `handleAgentlog`, `handleAgentlogJSON`, `handleAgentlogSSE`
   - Helpers: `extractUniqueProjectDirs`, `buildMultiProjectWorkspaceCache`, `buildWorkspaceCache`, workspace cache lookup methods, `readLastNEvents`, `getProjectAPIPort`, `checkWorkspaceSynthesis`, `getGapAnalysisFromEvents`, `extractGapAnalysisFromEvent`

2. **Created `cmd/orch/serve_agents_test.go`** (~620 lines)
   - Moved 13 test functions covering all extracted code

3. **Updated `cmd/orch/serve.go`** (reduced to ~1815 lines)
   - Removed extracted code
   - Added missing imports (`bufio`, `events`) still needed by remaining handlers

4. **Updated `cmd/orch/serve_test.go`** (reduced to ~409 lines)
   - Removed tests that moved to serve_agents_test.go

### Line count summary:

| File | Before | After | Delta |
|------|--------|-------|-------|
| serve.go | ~2921 | ~1815 | -1106 |
| serve_test.go | ~1016 | ~409 | -607 |
| serve_agents.go | 0 | ~1125 | +1125 |
| serve_agents_test.go | 0 | ~620 | +620 |

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Source of extraction
- `cmd/orch/serve_test.go` - Tests moved out
- `cmd/orch/serve_agents.go` - New file created
- `cmd/orch/serve_agents_test.go` - New test file created

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/

# Test verification
go test ./cmd/orch/ -count=1
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md` - 4-phase strategy
- **Commit:** 763cb55d

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: How to extract agent handlers from serve.go?
- Context: Phase 1 of 4-phase serve.go refactoring strategy

**2026-01-03:** Extraction completed
- Created serve_agents.go and serve_agents_test.go
- All tests pass, build succeeds

**2026-01-03:** Investigation completed
- Status: Complete
- Key outcome: Agent handlers successfully extracted, serve.go reduced by ~1100 lines
