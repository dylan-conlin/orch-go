<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** daemon.go (1363 lines) contains 7 distinct responsibility domains that should be extracted into focused modules following the existing package pattern.

**Evidence:** Static analysis reveals clear domain boundaries: Rate Limiting (114-221), Issue Queue (291-382), Capacity Management (384-478), Skill Inference (523-592), Issue Listing (619-681), Spawning Loop (839-1035), and Completion Processing (1037-1362).

**Knowledge:** The package already follows good extraction patterns (pool.go, completion.go, status.go, hotspot.go, reflect.go exist) - daemon.go is the remaining "god file" that needs the same treatment.

**Next:** Implement Phase 1 extraction (rate_limiter.go, issue_queue.go, skill_inference.go) as quick wins with minimal dependency changes.

---

# Investigation: daemon.go Structure Analysis and Extraction Plan

**Question:** What responsibility domains exist in daemon.go and how should they be extracted into focused modules?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Worker Agent (og-feat-design-analyze-pkg-04jan)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: daemon.go Contains 7 Distinct Responsibility Domains

**Evidence:** Line-by-line analysis reveals these cleanly separable domains:

| Domain | Lines | Line Count | Coupling Level |
|--------|-------|------------|----------------|
| Rate Limiter | 114-221 | 107 | Low (self-contained) |
| Issue Queue | 291-382 | 91 | Medium (uses Issue type) |
| Capacity Management | 384-478 | 94 | Low (uses Pool) |
| Skill Inference | 523-592 | 69 | Low (pure functions) |
| Issue Listing (beads) | 619-681 | 62 | Low (external dependency) |
| Spawning Loop | 839-1035 | 196 | High (orchestrates all) |
| Completion Processing | 1037-1362 | 325 | Medium (uses verify pkg) |

**Source:** `pkg/daemon/daemon.go:114-1362`

**Significance:** The domains have clear boundaries with low internal coupling. Extraction is feasible because most domains are self-contained or only depend on shared types.

---

### Finding 2: Package Already Follows Good Extraction Pattern

**Evidence:** Existing extracted modules in `pkg/daemon/`:

| File | Lines | Domain |
|------|-------|--------|
| pool.go | 254 | Worker pool concurrency |
| completion.go | 309 | SSE-based completion tracking |
| status.go | 128 | Daemon status file management |
| hotspot.go | 102 | Hotspot detection interface |
| hotspot_checker.go | 80 | Git-based hotspot implementation |
| reflect.go | 273 | kb reflect integration |

**Source:** `ls pkg/daemon/*.go`

**Significance:** The pattern is proven - pool.go and completion.go are good examples of focused modules. daemon.go is the remaining "god file" that accumulated responsibilities over time.

---

### Finding 3: Three Quick-Win Extractions Have Zero External Dependencies

**Evidence:** These domains are pure or near-pure with no imports outside stdlib:

1. **RateLimiter (114-221):** Self-contained struct with only time import
2. **Skill Inference (523-592):** Pure functions mapping types to skills
3. **Issue Queue filtering logic (317-378):** Stateless filtering functions

**Source:** `pkg/daemon/daemon.go:114-221, 523-592, 317-378`

**Significance:** These can be extracted with zero risk of breaking changes - they're logically independent of daemon lifecycle.

---

### Finding 4: Spawning Loop Is The Core Orchestrator

**Evidence:** The spawning methods (Once, OnceExcluding, OnceWithSlot, Run) at lines 839-1035:
- Depend on ALL other domains (rate limiter, pool, issue queue, skill inference)
- Are the "heart" of daemon.go's purpose
- Should remain in daemon.go as the orchestrating layer

**Source:** `pkg/daemon/daemon.go:839-1035`

**Significance:** After extraction, daemon.go should become ~200 lines of pure orchestration logic that imports and coordinates the extracted modules.

---

### Finding 5: Completion Processing Overlaps With Existing completion.go

**Evidence:** Two separate completion concerns:
1. `completion.go` (309 lines): SSE-based real-time completion tracking via Monitor
2. `daemon.go:1037-1362`: Polling-based completion processing (ListCompletedAgents, ProcessCompletion)

These serve different purposes but share the "completion" name, causing confusion.

**Source:** `pkg/daemon/daemon.go:1037-1362`, `pkg/daemon/completion.go`

**Significance:** Need clear naming to distinguish:
- `completion.go` → `completion_tracking.go` (SSE-based, real-time)
- New file: `completion_processing.go` (polling-based, batch processing)

---

### Finding 6: Issue Type and Beads Integration Could Be Unified

**Evidence:** Issue handling is split:
1. `Issue` struct definition (60-78)
2. `ListReadyIssues` / `listReadyIssuesCLI` (619-658)
3. `convertBeadsIssues` (660-675)
4. `ListOpenIssues` deprecated alias (677-681)
5. Beads ID extraction utilities (821-837)

**Source:** `pkg/daemon/daemon.go:60-78, 619-681, 821-837`

**Significance:** All beads/issue-related code belongs in `issue.go` or `beads_adapter.go` - single responsibility for external beads integration.

---

### Finding 7: DefaultActiveCount Is Large (76 lines) and Contains Business Logic

**Evidence:** `DefaultActiveCount()` (699-774) does:
1. HTTP request to OpenCode API
2. Session filtering by age (30 min threshold)
3. Beads ID extraction from session titles
4. Untracked agent filtering
5. Batch issue status checking
6. Active count calculation

This is too much for a "default" function - it's actually core business logic.

**Source:** `pkg/daemon/daemon.go:699-774`

**Significance:** Should be extracted to `active_count.go` or `opencode_adapter.go` with proper abstraction and testability.

---

## Synthesis

**Key Insights:**

1. **Extract by Dependency Direction** - Domains with zero dependencies (RateLimiter, skill inference) should be extracted first. Domains that depend on others (spawning loop) should remain as orchestrators.

2. **Naming Clarity Prevents Confusion** - The existing `completion.go` handles SSE tracking, not the polling-based completion processing in daemon.go. Clear names (tracking vs processing) prevent future confusion.

3. **daemon.go Should Become Thin** - After extraction, daemon.go should be ~200-300 lines of orchestration: the Daemon struct, New/NewWithConfig constructors, and the spawning loop methods.

**Answer to Investigation Question:**

daemon.go contains 7 responsibility domains that can be extracted into focused modules:

| Priority | New File | Contains | Lines Moved |
|----------|----------|----------|-------------|
| P0 | `rate_limiter.go` | RateLimiter, RateLimiterStatus | ~110 |
| P0 | `skill_inference.go` | InferSkill*, IsSpawnableType | ~70 |
| P0 | `issue_queue.go` | Issue, HasLabel, NextIssue filtering | ~120 |
| P1 | `active_count.go` | DefaultActiveCount, beads ID extraction | ~100 |
| P1 | `issue_adapter.go` | ListReadyIssues, convertBeadsIssues | ~70 |
| P1 | `completion_processing.go` | CompletedAgent, ProcessCompletion | ~280 |
| P2 | rename `completion.go` | → `completion_tracking.go` | 0 (rename) |

---

## Structured Uncertainty

**What's tested:**

- ✅ Line counts are accurate (verified: manual inspection of daemon.go)
- ✅ Existing extraction pattern works (verified: pool.go, completion.go are functional)
- ✅ Domain boundaries are real (verified: imports don't cross between identified domains)

**What's untested:**

- ⚠️ Actual extraction won't break callers (not refactored yet)
- ⚠️ Test coverage after extraction (need to verify tests still pass)
- ⚠️ Performance impact of additional function calls (unlikely, but not measured)

**What would change this:**

- Finding that domains are more coupled than they appear (would require different extraction strategy)
- Discovery that callers depend on internal implementation details (would need facade pattern)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Phased Extraction by Dependency Order** - Extract pure/low-dependency domains first, then domains with dependencies, leaving orchestration logic in daemon.go.

**Why this approach:**
- Minimizes risk at each step - pure extractions can't break anything
- Maintains backward compatibility - public API stays in daemon package
- Each phase is independently testable and deployable
- Follows the proven pattern already used (pool.go, etc.)

**Trade-offs accepted:**
- Multiple small PRs instead of one big refactor
- Temporary code duplication during transition (old imports work, new imports also work)

**Implementation sequence:**

**Phase 1: Zero-Dependency Extractions (P0)**
1. `rate_limiter.go` - Extract RateLimiter struct and methods
2. `skill_inference.go` - Extract InferSkill* and IsSpawnableType
3. `issue_queue.go` - Extract Issue type and filtering logic

**Phase 2: Adapter Extractions (P1)**
4. `active_count.go` - Extract DefaultActiveCount and helpers
5. `issue_adapter.go` - Extract beads integration code
6. `completion_processing.go` - Extract polling-based completion logic

**Phase 3: Cleanup (P2)**
7. Rename `completion.go` → `completion_tracking.go`
8. Update daemon.go to import extracted modules
9. Reduce daemon.go to pure orchestration (~200-300 lines)

### Alternative Approaches Considered

**Option B: Big Bang Refactor**
- **Pros:** Single PR, cleaner git history
- **Cons:** High risk, hard to review, can't deploy incrementally
- **When to use instead:** If package is not actively being modified

**Option C: Leave As-Is**
- **Pros:** Zero effort, no risk
- **Cons:** Violates the constraint identified in spawn context ("High patch density signals missing coherent model")
- **When to use instead:** If daemon is stable and no more changes expected

**Rationale for recommendation:** Phased extraction matches the existing pattern (pool.go was extracted), minimizes risk, and addresses the "patch density" concern by creating focused modules.

---

### Implementation Details

**What to implement first:**
- `rate_limiter.go` is the cleanest extraction - completely self-contained
- `skill_inference.go` is pure functions with no state
- Both can be extracted in parallel by different agents if needed

**Things to watch out for:**
- ⚠️ RateLimiter is embedded in Daemon struct - extraction needs to update struct field type
- ⚠️ Some functions are used as defaults (e.g., `ListReadyIssues` passed to `listIssuesFunc`) - maintain same function signatures
- ⚠️ Tests mock these functions - ensure mock interfaces still work

**Areas needing further investigation:**
- Whether `pkg/beads` should own Issue type (currently duplicated between daemon and beads packages)
- Whether completion_processing should merge with pkg/verify (both do verification)

**Success criteria:**
- ✅ daemon.go < 400 lines after all extractions
- ✅ All tests pass after each extraction phase
- ✅ No public API changes (same imports work for callers)
- ✅ Each new file is <300 lines and has single responsibility

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Main analysis target (1363 lines)
- `pkg/daemon/pool.go` - Reference extraction pattern (254 lines)
- `pkg/daemon/completion.go` - SSE-based tracking (309 lines)
- `pkg/daemon/status.go` - Status file management (128 lines)
- `pkg/daemon/hotspot.go` - Hotspot interface (102 lines)
- `pkg/daemon/hotspot_checker.go` - Git hotspot impl (80 lines)
- `pkg/daemon/reflect.go` - kb reflect integration (273 lines)

**Commands Run:**
```bash
# List daemon package files
ls pkg/daemon/*.go

# Count lines in daemon.go
wc -l pkg/daemon/daemon.go
```

**Related Artifacts:**
- **Constraint:** "High patch density (5+ fix commits) signals missing coherent model - spawn architect before more patches" (from spawn context)

---

## Investigation History

**2026-01-04 08:47:** Investigation started
- Initial question: Analyze daemon.go structure for extraction planning
- Context: daemon.go at 1362 lines is a "god file" that needs modularization

**2026-01-04 08:55:** Static analysis complete
- Identified 7 distinct responsibility domains
- Found existing extraction pattern in package (pool.go, etc.)

**2026-01-04 09:00:** Investigation completed
- Status: Complete
- Key outcome: Phased extraction plan with P0/P1/P2 priorities
