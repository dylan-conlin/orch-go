<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** serve.go (2921 lines) should be split into 6-7 domain-based handler files via 4 phases (~500-800 lines each).

**Evidence:** Identified 9 handler groups with distinct pkg/ dependencies; existing shared utilities span shared.go, review.go, wait.go.

**Knowledge:** Domain-based file organization (agents, beads, system, learn, errors, reviews) aligns with pkg/ dependency boundaries and enables parallel development.

**Next:** Spawn feature-impl agents for each phase, starting with Phase 1 (serve_agents.go - highest complexity, ~600 lines).

---

# Investigation: Map Serve Go Api Handler

**Question:** How should serve.go (2921 lines) be split into handler groupings with shared middleware/utilities, and what phases should the refactoring follow?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Agent (og-inv-map-serve-go-03jan)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: serve.go is 2921 lines (not 4125 as initially stated)

**Evidence:** `wc -l` shows serve.go is 2921 lines. Still substantial, but smaller than expected.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go`

**Significance:** Slightly reduces scope of refactoring. Based on main.go learnings (~500-800 lines per phase), this is roughly 4-5 phases of work.

---

### Finding 2: Initial handler groupings identified

**Evidence:** From analyzing the file, handlers fall into these logical groupings:

1. **Agents/Sessions** (~440 lines: 560-999)
   - `handleAgents` - Core agent list with OpenCode integration
   - `handleEvents` - SSE event proxy
   - `handleAgentlog`, `handleAgentlogSSE`, `handleAgentlogJSON` - Agent lifecycle events
   - Helper: `workspaceCache`, `buildWorkspaceCache`, `buildMultiProjectWorkspaceCache`

2. **Beads** (~230 lines: 1392-1622)
   - `handleBeads` - Stats endpoint
   - `handleBeadsReady` - Ready issues queue
   - `handleIssues` - Create issue POST endpoint
   - Related types: `BeadsAPIResponse`, `BeadsReadyAPIResponse`, `ReadyIssueResponse`, `CreateIssueRequest`, `CreateIssueResponse`

3. **Usage/Focus/Config** (~250 lines: 1268-1391, 2786-2882)
   - `handleUsage` - Claude Max usage stats
   - `handleFocus` - Focus/drift status
   - `handleConfig`, `handleConfigGet`, `handleConfigPut` - User config CRUD
   - Helper: `lookupAccountName`

4. **Servers/Daemon** (~180 lines: 1523-1700)
   - `handleServers` - Project server status
   - `handleDaemon` - Daemon status
   - Helper: `formatDurationAgo`
   - Related types: `ServerPortInfo`, `ServerProjectInfo`, `ServersAPIResponse`, `DaemonAPIResponse`

5. **Gaps/Reflect/Learn** (~270 lines: 1800-2110)
   - `handleGaps` - Gap tracker stats
   - `handleReflect` - Reflect suggestions
   - `getGapAnalysisFromEvents`, `extractGapAnalysisFromEvent`
   - Related types: `GapsAPIResponse`, `GapSuggestionSummary`, `GapAPIResponse`, `ReflectAPIResponse`, etc.

6. **Errors** (~170 lines: 2112-2381)
   - `handleErrors` - Error pattern analysis
   - Helpers: `extractSkillFromAgentID`, `normalizeErrorMessage`, `suggestRemediation`, `containsString`
   - Related types: `ErrorEvent`, `ErrorPattern`, `ErrorsAPIResponse`

7. **Pending Reviews** (~400 lines: 2382-2784)
   - `handlePendingReviews` - Synthesis review queue
   - `handleDismissReview` - Dismiss recommendations
   - Helpers: `isLightTierWorkspace`, `isLightTierComplete`, `contains`
   - Related types: `PendingReviewItem`, `PendingReviewAgent`, `PendingReviewsAPIResponse`, `DismissReviewRequest`, `DismissReviewResponse`

8. **Changelog** (~40 lines: 2884-2921)
   - `handleChangelog` - Aggregated changelog
   - Defers to `GetChangelog()` in changelog.go

9. **Server Setup/Shared** (~560 lines: 1-560)
   - `runServe`, `runServeStatus` - Server startup
   - `corsHandler` - CORS middleware
   - Route registration
   - Response types: `AgentAPIResponse`, `SynthesisResponse`
   - Shared helpers: `checkWorkspaceSynthesis`, `extractDateFromWorkspaceName` (imported from status_cmd.go)

**Source:** Line-by-line analysis of `cmd/orch/serve.go:1-2921`

**Significance:** These 9 groupings form natural file boundaries. Some helpers are already external (in shared.go, review.go, wait.go).

---

### Finding 3: External dependencies from other files

**Evidence:** The following helpers are used by serve.go but defined elsewhere:

| Helper Function | Defined In | Used In serve.go |
|----------------|------------|------------------|
| `extractBeadsIDFromTitle` | shared.go:27 | handleAgents, handleFocus |
| `extractSkillFromTitle` | shared.go:39 | handleAgents |
| `extractProjectFromBeadsID` | shared.go:90 | handleAgents |
| `extractBeadsIDFromWindowName` | shared.go:65 | handleAgents |
| `extractSkillFromWindowName` | shared.go:77 | handleAgents |
| `truncate` | shared.go:18 | handleAgents |
| `formatDuration` | wait.go:115 | handleAgents |
| `extractDateFromWorkspaceName` | status_cmd.go:422 | handleAgents |
| `extractBeadsIDFromWorkspace` | review.go:303 | handleAgents, handlePendingReviews |
| `extractProjectDirFromWorkspace` | review.go:337 | handleAgents |
| `summarizeDelta` | review.go:962 | handleAgents |

**Source:** grep analysis of function references

**Significance:** These helpers form the "shared utilities" layer that must be accessible to all handler files after the split.

---

### Finding 4: Package-level dependencies by handler group

**Evidence:**

| Handler Group | pkg/ Imports |
|---------------|--------------|
| Agents/Sessions | `opencode`, `events`, `spawn`, `tmux`, `verify`, `beads` |
| Beads | `beads` |
| Usage/Focus/Config | `usage`, `focus`, `account`, `userconfig` |
| Servers/Daemon | `port`, `tmux`, `daemon` |
| Gaps/Reflect | `spawn` (learning), `events` |
| Errors | `events` |
| Pending Reviews | `verify`, `spawn` |
| Changelog | (defers to changelog.go) |

**Source:** Import analysis of serve.go:3-29

**Significance:** Handler groups have distinct dependency profiles. agents/sessions is the most complex; beads is cleanest.

---

### Finding 5: Test coverage is concentrated in serve_test.go (1016 lines)

**Evidence:** Tests cover:
- Handler method validation (TestHandleAgentsMethodNotAllowed, etc.)
- JSON serialization (TestAgentAPIResponseJSONFormat, etc.)
- Helper functions (TestCheckWorkspaceSynthesis, TestBuildWorkspaceCache)
- Workspace cache logic (TestExtractUniqueProjectDirs, TestBuildMultiProjectWorkspaceCache)
- Error handling helpers (TestExtractSkillFromAgentID, TestNormalizeErrorMessage)

**Source:** `cmd/orch/serve_test.go:1-1016`

**Significance:** Tests are well-structured. When splitting serve.go, corresponding tests must move to new test files (e.g., serve_agents_test.go).

---

### Finding 6: CORS middleware is shared across all handlers

**Evidence:** `corsHandler` wrapper (lines 180-202) is applied to every route in runServe:
```go
mux.HandleFunc("/api/agents", corsHandler(handleAgents))
mux.HandleFunc("/api/events", corsHandler(handleEvents))
// ... all 18 endpoints
```

**Source:** serve.go:180-265

**Significance:** CORS middleware must stay in the main serve.go (server setup) file or be moved to a shared middleware file.

---

### Finding 7: Response types could be consolidated into an API types file

**Evidence:** 20+ response/request types are defined inline:
- `AgentAPIResponse`, `GapAPIResponse`, `SynthesisResponse` (core)
- `UsageAPIResponse`, `FocusAPIResponse`, `BeadsAPIResponse` (resource-specific)
- `CreateIssueRequest`, `CreateIssueResponse` (request/response pairs)
- `ErrorEvent`, `ErrorPattern`, `ErrorsAPIResponse` (error domain)
- `PendingReviewItem`, `PendingReviewAgent`, `DismissReviewRequest` (reviews)
- `ConfigAPIResponse`, `ConfigUpdateRequest` (config)

**Source:** Extracted from grep of `^type` declarations

**Significance:** Types could live with their handlers OR be consolidated into a `serve_types.go` for shared use. Recommendation: keep types with handlers for cohesion.

---

## Synthesis

**Key Insights:**

1. **Natural file boundaries exist along domain lines** - The 9 handler groupings (Finding 2) map cleanly to separate files. Each group has distinct pkg/ dependencies (Finding 4), making them logically independent.

2. **Shared utilities already exist but need consolidation** - Helper functions span 3 files (shared.go, review.go, wait.go). The split should NOT create new inter-handler dependencies; instead, shared code should migrate to appropriate locations.

3. **The agents handler is the "god handler"** - At ~440 lines with 7 pkg/ dependencies, handleAgents is the most complex. It includes workspace caching logic that benefits other handlers but is only used here. This should stay in the agents file.

4. **Tests follow handler structure** - serve_test.go tests are organized by handler, making test file splitting straightforward.

**Answer to Investigation Question:**

serve.go should be split into **6-7 handler files** plus a **server setup file**. Based on the main.go learnings (~500-800 lines per phase), this is roughly **4 phases of work**:

### Recommended File Structure

```
cmd/orch/
├── serve.go          (~300 lines) Server setup, CORS, route registration, cobra command
├── serve_agents.go   (~600 lines) handleAgents, handleEvents, handleAgentlog*, workspace cache
├── serve_beads.go    (~300 lines) handleBeads, handleBeadsReady, handleIssues
├── serve_system.go   (~400 lines) handleUsage, handleFocus, handleServers, handleDaemon, handleConfig*
├── serve_learn.go    (~350 lines) handleGaps, handleReflect, getGapAnalysisFromEvents
├── serve_errors.go   (~250 lines) handleErrors, error pattern helpers
├── serve_reviews.go  (~400 lines) handlePendingReviews, handleDismissReview, light tier helpers
├── serve_changelog.go  (exists)   Already split out
│
├── serve_test.go        (split into per-handler test files)
├── serve_agents_test.go
├── serve_beads_test.go
├── serve_system_test.go
├── serve_learn_test.go
├── serve_errors_test.go
├── serve_reviews_test.go
```

### Phase Breakdown (Avoiding Context Exhaustion)

**Phase 1: Extract serve_agents.go (~600 lines)**
- Highest complexity, most dependencies
- Includes workspace cache (used only by handleAgents)
- Move related tests to serve_agents_test.go
- Estimated: 2-3 hours

**Phase 2: Extract serve_beads.go + serve_reviews.go (~700 lines combined)**
- Clean domain boundaries
- handleIssues uses beadsClient (global in serve.go)
- Pending reviews uses extractBeadsIDFromWorkspace (review.go)
- Estimated: 2 hours

**Phase 3: Extract serve_system.go (~400 lines)**
- handleUsage, handleFocus, handleServers, handleDaemon, handleConfig
- Uses multiple pkg/ (usage, focus, port, daemon, userconfig)
- formatDurationAgo stays local to this file
- Estimated: 1.5 hours

**Phase 4: Extract serve_learn.go + serve_errors.go (~600 lines combined)**
- handleGaps uses spawn.LoadTracker
- handleErrors uses events package
- handleReflect reads JSON file (standalone)
- Estimated: 1.5 hours

### Shared Utilities Strategy

**Don't create new shared utilities file.** Instead:

1. `corsHandler` stays in serve.go (used at route registration)
2. `beadsClient` global stays in serve.go (initialized once)
3. Extract helpers stay in their current files:
   - shared.go: `truncate`, `extractBeadsIDFromTitle`, `extractSkillFromTitle`, `extractProjectFromBeadsID`
   - review.go: `extractBeadsIDFromWorkspace`, `extractProjectDirFromWorkspace`, `summarizeDelta`
   - wait.go: `formatDuration`
   - status_cmd.go: `extractDateFromWorkspaceName`

4. Handler-local helpers stay with their handler:
   - `extractSkillFromAgentID` → serve_errors.go
   - `normalizeErrorMessage`, `suggestRemediation`, `containsString` → serve_errors.go
   - `formatDurationAgo` → serve_system.go
   - `isLightTierWorkspace`, `isLightTierComplete`, `contains` → serve_reviews.go
   - `getGapAnalysisFromEvents`, `extractGapAnalysisFromEvent` → serve_learn.go

### Migration Safety

Each phase should:
1. Create new file with extracted content
2. Update imports (ensure pkg/ imports are correct)
3. Run `go build ./cmd/orch/` to verify compilation
4. Run `go test ./cmd/orch/` to verify tests pass
5. Commit before next phase

This approach keeps each phase under 800 lines of changes, within the safe zone for agent context.

---

## Structured Uncertainty

**What's tested:**

- ✅ Line counts verified via `wc -l` (serve.go is 2921 lines, not 4125)
- ✅ Handler groupings verified via `grep "^func\|^type"` declarations
- ✅ External dependencies verified via grep for function calls
- ✅ Test coverage verified by reading serve_test.go (1016 lines)

**What's untested:**

- ⚠️ Phase time estimates are based on main.go learnings (not benchmarked for serve.go)
- ⚠️ Import cycles not verified (assumed clean based on analysis)
- ⚠️ Test file split not validated (tests may have cross-handler dependencies)

**What would change this:**

- If serve.go imports create cycles when split, shared code may need extraction to pkg/
- If tests share fixtures across handlers, a shared test helpers file may be needed
- If future handlers need workspace caching, that logic may need promotion to pkg/workspace/

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Domain-based file split with 4 phases** - Split serve.go into 6-7 handler files organized by domain (agents, beads, system, learn, errors, reviews).

**Why this approach:**
- Aligns with natural domain boundaries identified in Finding 2
- Keeps pkg/ dependencies isolated per handler group (Finding 4)
- Enables parallel development - different engineers can work on different handlers
- Matches existing pattern (changelog.go already split out)

**Trade-offs accepted:**
- Slight increase in file count (1 → 7 files)
- Some code duplication for response types (kept with handlers for cohesion)
- Workspace caching stays in serve_agents.go (not promoted to pkg/ yet)

**Implementation sequence:**
1. Phase 1: serve_agents.go (highest complexity, establishes pattern)
2. Phase 2: serve_beads.go + serve_reviews.go (clean domains, similar complexity)
3. Phase 3: serve_system.go (multiple pkg/ deps, moderate complexity)
4. Phase 4: serve_learn.go + serve_errors.go (simpler handlers, finishing touches)

### Alternative Approaches Considered

**Option B: Move handlers to pkg/server/**
- **Pros:** Cleaner architecture, handlers become testable in isolation
- **Cons:** Major refactor, changes import structure, more risk
- **When to use instead:** If serve handlers need to be reused by other binaries

**Option C: Single serve_handlers.go file**
- **Pros:** Minimal file count, all handlers in one place
- **Cons:** Still ~2600 lines, doesn't solve the god object problem
- **When to use instead:** Never - defeats the purpose

**Rationale for recommendation:** Domain-based split balances maintainability with minimal structural change. The existing shared.go/review.go pattern proves this approach works in the codebase.

---

### Implementation Details

**What to implement first:**
- Phase 1 (serve_agents.go) - most complex, establishes pattern for other phases
- Create serve_agents_test.go simultaneously to validate approach
- Run `go build` and `go test` after each file extraction

**Things to watch out for:**
- ⚠️ `beadsClient` global must remain in serve.go (initialized in runServe)
- ⚠️ `corsHandler` must remain in serve.go (used at route registration)
- ⚠️ Test fixtures may share setup - check for cross-handler dependencies
- ⚠️ Some response types reference each other (AgentAPIResponse → SynthesisResponse)

**Areas needing further investigation:**
- Whether workspace caching should be promoted to pkg/workspace/ (currently handleAgents-only)
- Whether response types should have JSON schema validation
- Performance of multi-project workspace cache scanning

**Success criteria:**
- ✅ `go build ./cmd/orch/` succeeds after each phase
- ✅ `go test ./cmd/orch/` passes after each phase
- ✅ No file exceeds 800 lines (phase scope)
- ✅ Each handler file has corresponding test file
- ✅ serve.go reduced to ~300 lines (server setup only)

---

## References

**Files Examined:**
- `cmd/orch/serve.go` - Primary investigation target (2921 lines)
- `cmd/orch/serve_test.go` - Test coverage analysis (1016 lines)
- `cmd/orch/shared.go` - External helper functions (269 lines)
- `cmd/orch/review.go` - External helpers used by serve.go
- `cmd/orch/wait.go` - formatDuration helper
- `cmd/orch/status_cmd.go` - extractDateFromWorkspaceName helper

**Commands Run:**
```bash
# Line count of largest Go files
wc -l /Users/dylanconlin/Documents/personal/orch-go/**/*.go | sort -rn | head -20

# Top-level declarations in serve.go
grep -n "^func\|^type\|^var\|^const" cmd/orch/serve.go | head -100

# External helper references
grep -n "extractBeadsIDFromTitle\|extractSkillFromTitle\|..." cmd/orch/serve.go
```

**External Documentation:**
- main.go refactoring learnings (~500-800 lines per phase) referenced in SPAWN_CONTEXT.md

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-map-serve-go-03jan/` - This investigation workspace

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: How should serve.go be split for maintainability?
- Context: serve.go is a 2921-line file with 20+ handlers

**2026-01-03:** Handler groupings identified
- Found 9 logical handler groups with distinct pkg/ dependencies
- Identified shared utilities across shared.go, review.go, wait.go

**2026-01-03:** Investigation completed
- Status: Complete
- Key outcome: 4-phase split strategy with domain-based file organization (agents, beads, system, learn, errors, reviews)

---

## Self-Review

- [x] Real test performed (line count, grep analysis, file reading)
- [x] Conclusion from evidence (handler groupings from actual code)
- [x] Question answered (split strategy with phases)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED
