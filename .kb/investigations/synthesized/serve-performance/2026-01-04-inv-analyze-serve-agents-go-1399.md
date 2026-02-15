<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Validated prior investigation and refined extraction plan with exact line counts - 3 files (cache ~470, events ~230, core ~700) vs original estimate (350, 230, 630).

**Evidence:** Line-by-line analysis confirms caching infrastructure is 468 lines (lines 27-497), events are 242 lines (lines 954-1195), and core is remaining 689 lines.

**Knowledge:** The `workspaceCache` type (lines 277-496) was underestimated in prior investigation - it's 220 lines including builders and lookup methods. This should go with caching infrastructure as a cohesive unit.

**Next:** Proceed to Phase 1: Extract serve_agents_cache.go (~470 lines) including both beadsCache and workspaceCache.

---

# Investigation: Analyze serve_agents.go (1399 lines) Extraction Design

**Question:** How should cmd/orch/serve_agents.go be split into focused handler files, building on the prior investigation's 2-phase plan?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

**Prior Investigation:** `.kb/investigations/2026-01-04-inv-design-analyze-serve-agents-go.md`

---

## Findings

### Finding 1: Prior Investigation Line Counts Need Minor Revision

**Evidence:** The prior investigation estimated:
- Caching Infrastructure: 227 lines (lines 27-227)
- Events: 230 lines (lines 954-1195)
- Core: 942 lines

Current analysis shows:
- beadsCache types/methods: lines 27-227 (201 lines)
- workspaceCache types/methods: lines 277-496 (220 lines) - **This was partially counted in "core"**
- Response types: lines 230-274 (45 lines) - Stay in core
- Events: lines 954-1195 (242 lines)
- Core handlers: lines 499-953, 1199-1399 (~455 + 201 = ~656 lines)

**Source:** `grep -n "^func\|^type\|^const\|^var" cmd/orch/serve_agents.go`

**Significance:** The workspaceCache (220 lines) was underestimated. It belongs with caching infrastructure, not core, because it's about caching workspace metadata.

---

### Finding 2: Exact File Boundaries Identified

**Evidence:** Three logical domains with precise line ranges:

| Domain | Lines | Components |
|--------|-------|------------|
| **serve_agents_cache.go** | 27-227, 277-496 | beadsCache (struct + 7 methods), globalWorkspaceCacheType (struct + 2 methods), workspaceCache (struct + 4 methods), buildWorkspaceCache, buildMultiProjectWorkspaceCache, extractUniqueProjectDirs, TTL constants, global instances |
| **serve_agents_events.go** | 954-1195 | handleEvents, handleAgentlog, handleAgentlogJSON, handleAgentlogSSE, readLastNEvents |
| **serve_agents.go (core)** | 230-274, 499-953, 1199-1399 | AgentAPIResponse, GapAPIResponse, SynthesisResponse, handleAgents, handleCacheInvalidate, determineAgentStatus, getGapAnalysisFromEvents, extractGapAnalysisFromEvent, getProjectAPIPort, checkWorkspaceSynthesis |

**Source:** Line-by-line analysis of function and type declarations

**Significance:** Clear boundaries enable clean extraction without circular dependencies.

---

### Finding 3: Import Requirements Verified

**Evidence:** Import analysis by file:

**serve_agents_cache.go imports:**
- `os`, `path/filepath`, `strings` (workspace scanning)
- `sync`, `time` (caching with TTL)
- `github.com/dylan-conlin/orch-go/pkg/beads` (Comment type)
- `github.com/dylan-conlin/orch-go/pkg/verify` (Issue type)
- `github.com/dylan-conlin/orch-go/pkg/opencode` (Session type for extractUniqueProjectDirs)

**serve_agents_events.go imports:**
- `bufio`, `encoding/json`, `fmt`, `io`, `net/http`, `os`, `strings`, `time`
- `github.com/dylan-conlin/orch-go/pkg/events` (Event type, DefaultLogPath)

**serve_agents.go (core) imports:**
- All remaining imports from current file

**Source:** Analysis of function bodies for package references

**Significance:** No circular dependencies - cache is consumed by handlers, events are independent.

---

### Finding 4: handleCacheInvalidate Placement Decision

**Evidence:** Prior investigation noted uncertainty about whether `handleCacheInvalidate` (lines 1380-1399) should go in cache file or stay in main.

Analysis:
- It's an HTTP handler (takes `http.ResponseWriter, *http.Request`)
- It invalidates both beadsCache AND workspaceCache
- Other handlers (handleAgents, handleEvents) stay in their respective files

**Recommendation:** Keep `handleCacheInvalidate` in core file (`serve_agents.go`) because:
1. It's an HTTP handler, not a caching primitive
2. It needs to know about BOTH cache types (coupling)
3. Matches pattern from serve.go where route handlers stay together

**Source:** `cmd/orch/serve_agents.go:1380-1399`

**Significance:** Avoids putting an HTTP handler in a file focused on caching primitives.

---

## Synthesis

**Key Insights:**

1. **Workspace cache is caching infrastructure, not core** - The 220-line workspaceCache block belongs with beadsCache because both are about performance optimization through caching, not about agent handling logic.

2. **Three files with clear single responsibility** - Each file owns ONE concern: caching primitives (serve_agents_cache.go), SSE streaming (serve_agents_events.go), agent handling (serve_agents.go).

3. **handleAgents remains large but coherent** - The ~455-line handleAgents function is a "god handler" that shouldn't be split further. Its complexity is inherent (multi-source aggregation, Priority Cascade model, cross-project visibility).

**Answer to Investigation Question:**

Split serve_agents.go into **3 files** using **2 phases**:

```
cmd/orch/
├── serve_agents.go        (~700 lines) Core agent handler + status logic + response types
├── serve_agents_cache.go  (~470 lines) All caching infrastructure (beads + workspace)
├── serve_agents_events.go (~230 lines) SSE event handlers
```

Phase 1 (cache extraction) has higher impact because it removes the largest cohesive block. Phase 2 (events extraction) is simpler but still valuable for separation of concerns.

---

## Structured Uncertainty

**What's tested:**

- ✅ Line count verified: `wc -l cmd/orch/serve_agents.go` returns 1399
- ✅ Function boundaries verified via `grep -n "^func\|^type"`
- ✅ Import analysis done for each proposed file
- ✅ Prior investigation findings validated against current code

**What's untested:**

- ⚠️ Whether Go compiler accepts the split (compile verification needed during implementation)
- ⚠️ Whether tests still pass after extraction (run `go test ./cmd/orch/`)
- ⚠️ Whether any unexported helpers are referenced across file boundaries

**What would change this:**

- If any unexported function is called across proposed file boundaries → need to export or move
- If tests specifically mock caching functions → may need to update test imports
- If runtime initialization order matters → may need init() function

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Two-Phase Infrastructure Extraction

**Phase 1: Extract serve_agents_cache.go (~470 lines)**

Move in order:
1. Constants: `defaultOpenIssuesTTL`, `defaultAllIssuesTTL`, `defaultCommentsTTL` (lines 88-92)
2. Global variables: `globalBeadsCache`, `globalWorkspaceCacheInstance` (lines 62-64, 124)
3. Type `beadsCache` + all methods (lines 27-227)
4. Type `globalWorkspaceCacheType` + methods (lines 50-122)
5. Type `workspaceCache` + all methods (lines 277-496)
6. Functions: `newBeadsCache()`, `extractUniqueProjectDirs()`, `buildMultiProjectWorkspaceCache()`, `buildWorkspaceCache()`

After extraction, verify:
```bash
go build ./cmd/orch/
go test ./cmd/orch/
```

**Phase 2: Extract serve_agents_events.go (~230 lines)**

Move:
1. `handleEvents` (lines 954-1027)
2. `handleAgentlog` (lines 1032-1045)
3. `handleAgentlogJSON` (lines 1048-1068)
4. `handleAgentlogSSE` (lines 1071-1161)
5. `readLastNEvents` (lines 1164-1195)

After extraction, verify:
```bash
go build ./cmd/orch/
go test ./cmd/orch/
```

### Alternative Approaches Considered

**Option B: Move to pkg/cache/**
- **Pros:** Reusable across other packages
- **Cons:** Only used by serve_agents; premature abstraction
- **When to use:** If other handlers need the same caching pattern

**Option C: Extract handleAgents sub-functions**
- **Pros:** Smaller individual functions
- **Cons:** Fragments coherent flow, requires passing large data structures
- **When to use:** If handleAgents grows beyond 600 lines

**Rationale for recommendation:** File-level extraction provides best maintainability improvement with minimal risk. Keeps caching and events as self-contained modules while preserving the coherent agent handler.

---

### Implementation Details

**What to implement first:**
- Phase 1 (serve_agents_cache.go) - largest impact, removes ~470 lines from main file

**Things to watch out for:**
- ⚠️ `globalBeadsCache` is initialized in `runServe()` in serve.go - initialization stays there
- ⚠️ `newBeadsCache()` is called from serve.go - must remain exported (it already is)
- ⚠️ `globalWorkspaceCacheInstance` uses package-level var init - verify it works across files
- ⚠️ opencode.Session type needed for extractUniqueProjectDirs - add import

**Areas needing further investigation:**
- Whether to create `serve_agents_cache_test.go` or keep tests unified
- Whether `handleCacheInvalidate` should be in serve.go or serve_agents.go

**Success criteria:**
- ✅ `go build ./cmd/orch/` succeeds after each phase
- ✅ `go test ./cmd/orch/` passes after each phase
- ✅ Main serve_agents.go reduced to ~700 lines
- ✅ No circular dependencies introduced
- ✅ Cache file has no HTTP handler logic
- ✅ Events file has no caching logic

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Primary target (1399 lines)
- `.kb/investigations/2026-01-04-inv-design-analyze-serve-agents-go.md` - Prior investigation

**Commands Run:**
```bash
# Verify line count
wc -l cmd/orch/serve_agents.go
# Output: 1399

# List declarations
grep -n "^func\|^type\|^const\|^var" cmd/orch/serve_agents.go
```

**Related Artifacts:**
- **Prior Investigation:** `.kb/investigations/2026-01-04-inv-design-analyze-serve-agents-go.md` - Original analysis

---

## Investigation History

**2026-01-04 10:30:** Investigation started
- Initial question: Validate prior investigation and refine extraction plan
- Context: Prior investigation complete, need to verify line counts and produce implementation-ready details

**2026-01-04 11:00:** Investigation completed
- Status: Complete
- Key outcome: Validated 2-phase extraction with revised line counts (~470 cache, ~230 events, ~700 core)
