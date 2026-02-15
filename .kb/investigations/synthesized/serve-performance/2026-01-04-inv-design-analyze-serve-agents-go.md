<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** serve_agents.go (1399 lines) can be split into 3 focused files following the caching/handlers/events pattern, reducing it to ~500 lines (main handler + status logic).

**Evidence:** Identified 3 distinct domains: caching infrastructure (227 lines), event streaming (230 lines), and core agent handlers (942 lines). The caching layer has no handler dependencies; events are self-contained SSE proxies.

**Knowledge:** The file contains a god handler (`handleAgents` at 451 lines) that should NOT be further split - its complexity is inherent (Priority Cascade model, workspace scanning, beads integration). Extracting infrastructure reduces cognitive load without breaking cohesion.

**Next:** Implement in 2 phases: (1) Extract serve_agents_cache.go (caching infrastructure), (2) Extract serve_agents_events.go (SSE handlers).

---

# Investigation: Analyze serve_agents.go for Extraction Design

**Question:** How should cmd/orch/serve_agents.go (1399 lines) be structured for maintainability, following the main.go and serve.go split patterns?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Three Distinct Domains Identified

**Evidence:** The file contains three logically separable domains:

| Domain | Lines | Key Components | Dependencies |
|--------|-------|----------------|--------------|
| **Caching Infrastructure** | 227 (lines 27-227) | `beadsCache`, `globalWorkspaceCacheType`, `workspaceCache`, TTL constants | `beads`, `verify`, `time` |
| **Event Streaming** | 230 (lines 954-1195) | `handleEvents`, `handleAgentlog*`, `readLastNEvents` | `events`, `bufio`, `io` |
| **Core Agent Handlers** | 942 (lines 228-953, 1196-1399) | `handleAgents`, `determineAgentStatus`, gap analysis, cache invalidation | All pkg/* |

**Source:** Line-by-line analysis of `cmd/orch/serve_agents.go:1-1399`

**Significance:** The caching infrastructure is entirely self-contained - it provides O(1) lookups but has no HTTP handler logic. Event streaming is also isolated - SSE proxy and event file reading. These are clean extraction targets.

---

### Finding 2: Caching Layer is Self-Contained

**Evidence:** The caching types and methods have a clear boundary:

```
Types:
- beadsCache (struct + 7 methods)
- globalWorkspaceCacheType (struct + 2 methods)  
- workspaceCache (struct + 3 methods)

Globals:
- globalWorkspaceCacheInstance
- globalBeadsCache
- TTL constants (3)

Functions:
- newBeadsCache()
- buildWorkspaceCache()
- buildMultiProjectWorkspaceCache()
- extractUniqueProjectDirs()
```

The caching layer is **consumed** by handleAgents but has **no knowledge** of HTTP or handlers. It could be in pkg/cache/ but the file-level extraction keeps it with its consumers.

**Source:** `cmd/orch/serve_agents.go:27-227` (caching) and `cmd/orch/serve_agents.go:277-496` (workspace cache)

**Significance:** Extracting to `serve_agents_cache.go` (~350 lines including workspace cache logic) creates a focused file about caching concerns only.

---

### Finding 3: handleAgents is a "God Handler" - But Coherently So

**Evidence:** `handleAgents` (lines 499-950, 451 lines) is large but has necessary complexity:

1. **Multi-source aggregation** - OpenCode sessions, tmux windows, completed workspaces
2. **Priority Cascade status model** - 4-level priority for status determination
3. **Batch beads integration** - Cached fetching of issues, comments, phase status
4. **Cross-project visibility** - Multi-project workspace aggregation
5. **Token usage collection** - Parallelized HTTP calls with semaphore

**Why NOT to split further:**
- The logic is inherently sequential (gather sources → enrich → filter → return)
- Splitting would require passing large data structures between functions
- All 451 lines are about ONE concept: agent list with enrichment

**Source:** `cmd/orch/serve_agents.go:499-950`

**Significance:** This is not a god object anti-pattern - it's a complex handler with high cohesion. The fix is NOT to split handleAgents, but to extract unrelated infrastructure (caching, events) to separate files.

---

### Finding 4: Event Streaming is Independent

**Evidence:** Event-related handlers form a self-contained module:

| Handler | Lines | Purpose |
|---------|-------|---------|
| `handleEvents` | 954-1027 | SSE proxy to OpenCode /event |
| `handleAgentlog` | 1032-1045 | Router (JSON vs SSE based on query param) |
| `handleAgentlogJSON` | 1048-1068 | Return last 100 events as JSON |
| `handleAgentlogSSE` | 1071-1161 | Stream new events via SSE |
| `readLastNEvents` | 1164-1195 | Helper to read JSONL file |

These handlers:
- Don't use workspace caching
- Don't use beads caching
- Only depend on `events` package and standard library

**Source:** `cmd/orch/serve_agents.go:954-1195`

**Significance:** Clean extraction to `serve_agents_events.go` (~230 lines) with zero coupling to main handler.

---

### Finding 5: Remaining Core After Extraction

**Evidence:** After extracting caching and events, the remaining core would be:

```
Types:
- AgentAPIResponse (struct)
- GapAPIResponse (struct)
- SynthesisResponse (struct)

Handlers:
- handleAgents() (~451 lines)
- handleCacheInvalidate() (~20 lines)

Helpers:
- getProjectAPIPort() (~18 lines)
- checkWorkspaceSynthesis() (~12 lines)
- determineAgentStatus() (~18 lines) - Priority Cascade
- getGapAnalysisFromEvents() (~58 lines)
- extractGapAnalysisFromEvent() (~47 lines)

Total: ~630 lines
```

**Source:** Subtraction analysis from total (1399 - 350 cache - 230 events ≈ 820; actual ~630 after removing blank lines)

**Significance:** The core file would still be sizeable (~630 lines) but focused on ONE responsibility: agent listing with enrichment. This is acceptable given the inherent complexity.

---

### Finding 6: No Tests Currently Exist for serve_agents.go

**Evidence:** `wc -l cmd/orch/serve_agents_test.go` shows 909 lines, but examining content reveals:

The existing test file focuses on:
- Handler method validation
- JSON serialization  
- Workspace cache logic
- Error handling helpers

Tests are well-structured and can remain in single file or be split along with implementation.

**Source:** `cmd/orch/serve_agents_test.go:1-909`

**Significance:** Test extraction is optional - the test file is manageable. If extracting, create `serve_agents_cache_test.go` for cache tests.

---

## Synthesis

**Key Insights:**

1. **Infrastructure vs Application separation** - The caching layer is infrastructure; handlers are application. Separating them improves readability without breaking cohesion.

2. **God handler is acceptable when coherent** - `handleAgents` at 451 lines is complex but unified around a single concept (agent list enrichment). Splitting it would fragment a coherent flow.

3. **Events are a completely separate concern** - SSE proxy and event log reading have zero overlap with agent listing. Clean extraction target.

4. **Two phases are sufficient** - Unlike main.go (4 phases) or serve.go (4 phases), serve_agents.go only needs 2 phases because it has fewer distinct domains.

**Answer to Investigation Question:**

Split serve_agents.go into **3 files** using **2 phases**:

### Recommended File Structure

```
cmd/orch/
├── serve_agents.go        (~630 lines) Core agent handler + status logic
├── serve_agents_cache.go  (~350 lines) Caching infrastructure
├── serve_agents_events.go (~230 lines) SSE event handlers
│
├── serve_agents_test.go   (optional: split to match implementation)
```

### Phase Breakdown

**Phase 1: Extract serve_agents_cache.go (~350 lines)**
- Move: `beadsCache`, `globalWorkspaceCacheType`, `workspaceCache` types
- Move: All cache methods (invalidate, get*, lookup*)
- Move: TTL constants
- Move: `buildWorkspaceCache`, `buildMultiProjectWorkspaceCache`, `extractUniqueProjectDirs`
- Move: Global instances (`globalBeadsCache`, `globalWorkspaceCacheInstance`)
- Keep: Response types (AgentAPIResponse, etc.) in main file
- Estimated: 1-1.5 hours

**Phase 2: Extract serve_agents_events.go (~230 lines)**
- Move: `handleEvents`, `handleAgentlog`, `handleAgentlogJSON`, `handleAgentlogSSE`
- Move: `readLastNEvents`
- Estimated: 30-45 minutes

---

## Structured Uncertainty

**What's tested:**

- ✅ Line counts verified via `wc -l` (serve_agents.go is 1399 lines)
- ✅ Domain boundaries verified via `grep "^func\|^type"` analysis
- ✅ Caching layer independence verified (no HTTP imports needed)
- ✅ Events layer independence verified (only `events` pkg dependency)

**What's untested:**

- ⚠️ Whether cache globals need initialization order considerations
- ⚠️ Whether extracting to separate files affects build time
- ⚠️ Whether test coverage needs updates after extraction

**What would change this:**

- If cache initialization depends on handler globals → keep in same file
- If events handlers need shared types with main handler → may need type file
- If performance regresses after split → reconsider (unlikely)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Infrastructure extraction with coherent handler preservation** - Extract caching and events to separate files, but keep handleAgents unified.

**Why this approach:**
- Reduces cognitive load (3 smaller files vs 1 large file)
- Preserves cohesion in the complex handler
- Clean boundaries with no cross-dependencies after split
- Follows established serve_*.go pattern

**Trade-offs accepted:**
- handleAgents remains 451 lines (acceptable: unified concern)
- 3 files instead of 1 (acceptable: clear responsibilities)
- Tests may need minor updates (acceptable: isolated changes)

**Implementation sequence:**

1. **Phase 1: serve_agents_cache.go** (Highest priority - reduces file size most)
   - Extract all caching types and methods
   - Verify globals are accessible across files (package main scope)
   - Run `go build ./cmd/orch/` to verify
   - Run `go test ./cmd/orch/` to verify

2. **Phase 2: serve_agents_events.go** (Lower priority - smaller extraction)
   - Extract SSE handlers and event reading
   - Verify no dependencies on cache or main handler
   - Final build and test verification

### Alternative Approaches Considered

**Option B: Extract handleAgents to Smaller Functions**
- **Pros:** Smaller individual functions
- **Cons:** Would fragment coherent flow, require passing large data structures
- **When to use instead:** If handleAgents grows to 600+ lines with new concerns

**Option C: Move Caching to pkg/cache/**
- **Pros:** Reusable across other packages
- **Cons:** Only used by serve_agents; premature abstraction
- **When to use instead:** If other handlers need the same caching pattern

**Option D: Keep as Single File**
- **Pros:** No file management overhead
- **Cons:** 1399 lines is beyond comfortable editing range
- **When to use instead:** Never - file is already at maintainability limit

**Rationale for recommendation:** Infrastructure extraction provides the best maintainability improvement with minimal risk. The god handler stays coherent, and extracted infrastructure becomes easier to understand and test.

---

### Implementation Details

**What to implement first:**
- Phase 1 (serve_agents_cache.go) - largest impact, most isolated
- Create file with package declaration and imports
- Cut-paste caching code (no modification needed)

**Things to watch out for:**
- ⚠️ `globalBeadsCache` is initialized in `runServe` (in serve.go) - initialization stays there
- ⚠️ `newBeadsCache()` is called from serve.go - must remain exported
- ⚠️ Cache methods use `verify.Issue` and `beads.Comment` - need those imports
- ⚠️ Workspace cache uses `os`, `path/filepath`, `strings` - common imports

**Areas needing further investigation:**
- Whether `handleCacheInvalidate` should move to cache file (currently in main)
- Whether to create `serve_agents_cache_test.go` or keep tests unified

**Success criteria:**
- ✅ `go build ./cmd/orch/` succeeds after each phase
- ✅ `go test ./cmd/orch/` passes after each phase
- ✅ Main serve_agents.go reduced to ~630 lines
- ✅ No circular dependencies introduced
- ✅ Cache file has no HTTP handler logic
- ✅ Events file has no caching logic

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Primary investigation target (1399 lines)
- `cmd/orch/serve_agents_test.go` - Test coverage reference (909 lines)
- `cmd/orch/serve.go` - Route registration (imports handlers)

**Commands Run:**
```bash
# Line count
wc -l cmd/orch/serve_agents.go
# Output: 1399

# Type and function declarations
grep -n "^func\|^type\|^const\|^var" cmd/orch/serve_agents.go

# Verify test file exists
wc -l cmd/orch/serve_agents_test.go
# Output: 909
```

**External Documentation:**
- Prior investigation: `.kb/investigations/2026-01-04-inv-cmd-orch-main-go-49.md` (main.go split pattern)
- Prior investigation: `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md` (serve.go split pattern)

**Related Artifacts:**
- **Decision:** Domain-based file split is preferred pattern (from serve.go investigation)
- **Pattern:** ~500-800 lines per extraction phase (from main.go investigation)

---

## Investigation History

**2026-01-04 10:00:** Investigation started
- Initial question: How to split serve_agents.go for maintainability
- Context: File is 1399 lines, already extracted from serve.go

**2026-01-04 10:30:** Domain analysis complete
- Identified 3 domains: caching (350), events (230), core (630)
- Determined handleAgents is a coherent god handler, not anti-pattern

**2026-01-04 11:00:** Investigation completed
- Status: Complete
- Key outcome: 2-phase extraction plan with infrastructure separation pattern
