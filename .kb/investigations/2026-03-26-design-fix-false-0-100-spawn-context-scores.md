## Summary (D.E.K.N.)

**Delta:** False 0/100 context scores are caused by `runKBContextQuery` returning `nil, nil` on timeout — identical to "no matches" — making timeout indistinguishable from genuine absence.

**Evidence:** Code trace: `kbcontext.go:155-158` catches all errors (including `context.DeadlineExceeded`) and returns `nil, nil`; `gap.go:104` treats nil result as `GapTypeNoContext` with quality=0. Real queries ('isreasoningmodel test gpt', 'verify dao claim') take 5.8-8.8s, exceeding the 5s hardcoded timeout.

**Knowledge:** The fix requires changes in 3 layers: (1) timeout detection in `runKBContextQuery`, (2) a new `GapTypeTimeout` gap type that doesn't trigger gate blocking, (3) observability to track timeout frequency. Raising the timeout from 5s to 10s addresses the immediate cause, but without timeout detection the same class of false negatives will recur at different latency thresholds.

**Next:** Implement (3 files, ~50 lines net change). Created implementation issue.

**Authority:** architectural — changes cross the spawn/gap boundary and alter gate behavior semantics

---

# Investigation: Design Fix for False 0/100 Spawn Context Scores

**Question:** How should we fix the false 0/100 spawn context scores caused by KB context timeout, distinguishing timeout from genuine absence while adding observability?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** architect (orch-go-k6c0v)
**Phase:** Complete
**Next Step:** None — implement from recommendations
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| orch-go-304ta (GPT-5.4 headless 0/100 scores) | extends | Yes — traced same timeout→nil→0 chain | None |
| kn constraint: "kb context command hangs on some queries" | confirms | Yes — constraint exists, workaround is `--skip-artifact-check` | None — but workaround skips ALL context, not just slow queries |

---

## Findings

### Finding 1: Timeout is indistinguishable from no-matches

**Evidence:** `runKBContextQuery` (pkg/spawn/kbcontext.go:155-158):
```go
output, err := cmd.Output()
if err != nil {
    // If kb command fails (not found, no matches, timeout, etc.), return nil
    return nil, nil
}
```

All error types — timeout, command not found, actual failures — return `(nil, nil)`. There is no check for `ctx.Err() == context.DeadlineExceeded`. No logging, no event, no distinction.

**Source:** `pkg/spawn/kbcontext.go:154-159`

**Significance:** This is the root cause. Every timeout appears identical to "no context exists." The gap analysis at `pkg/spawn/gap.go:104` then assigns `GapTypeNoContext` (critical severity) with quality=0, triggering false alarms.

---

### Finding 2: 5s timeout is too tight for orch-go's knowledge base

**Evidence:** Per investigation orch-go-304ta, real queries on orch-go (280+ investigations, rich KB) take 5.8-8.8s:
- 'isreasoningmodel test gpt' → 5.8s (just over timeout)
- 'comment explaining skillmodelmapping' → 7.2s
- 'verify dao claim' → 8.8s (well over timeout)

All return real, relevant matches when given sufficient time.

**Source:** orch-go-304ta findings, manual `time kb context` runs

**Significance:** The timeout value itself is the immediate trigger, but fixing only the value without fixing detection means the problem returns at any new latency threshold.

---

### Finding 3: Timeout can cascade through multiple search stages

**Evidence:** The KB context search has up to 4 sequential queries in worst case:

1. `RunKBContextCheckForDir` tries local search (5s timeout)
2. If sparse, tries global search (5s timeout)
3. `runPreSpawnKBCheckFull` may retry with broader keywords (another `RunKBContextCheckForDir` = 2 more queries)

Each creates its own `context.WithTimeout(5s)`. If local times out at 5s, global never runs. If both primary and fallback time out, total wait is 20s with zero useful information.

**Source:** `pkg/spawn/kbcontext.go:137-139` (per-query timeout), `pkg/spawn/kbcontext.go:82-89` (two-stage search), `pkg/orch/spawn_kb_context.go:89-98` (fallback search)

**Significance:** The per-query timeout architecture means budget isn't shared. A single slow local query blocks the global fallback that might have been fast.

---

### Finding 4: Fallback search overwrites timeout signal

**Evidence:** In `runPreSpawnKBCheckFull` (pkg/orch/spawn_kb_context.go:89-98):
```go
if result == nil || !result.HasMatches {
    ...
    result, err = spawn.RunKBContextCheckForDir(firstKeyword, projectDir)
```

If primary search returned a timeout signal and fallback returns `nil` (no matches, no timeout), the `result` variable is overwritten, losing the timeout information. The subsequent `AnalyzeGaps(result, ...)` call then sees nil and classifies as `GapTypeNoContext` — the same false signal.

**Source:** `pkg/orch/spawn_kb_context.go:89-98`

**Significance:** Defect class 1 (Filter Amnesia) — the timeout signal exists in one path but is lost in the fallback path. Implementation must preserve timeout across the fallback chain.

---

## Synthesis

**Key Insights:**

1. **Absence of evidence ≠ evidence of absence** — The core epistemic error: timeout means "we don't know," but the system reports it as "definitely no context." This directly violates the Provenance principle — claiming knowledge (absence of context) without evidence.

2. **Two fixes needed, not one** — Raising the timeout (from 5s to 10s) addresses the immediate symptoms but not the structural defect. Timeout detection (new GapTypeTimeout) fixes the class of problem. Both are needed.

3. **Gate behavior must respect epistemic state** — The gap gate (ShouldBlockSpawn) should not block spawns based on unknowns. Blocking is only justified when we have evidence of gap, not when we failed to check.

**Answer to Investigation Question:**

The fix requires 3 coordinated changes: (1) raise per-query timeout from 5s to 10s, (2) detect timeout vs. other errors and propagate through a `TimedOut` flag on `KBContextResult`, (3) add `GapTypeTimeout` gap type that produces a warning (not critical) and doesn't trigger the gap gate. ~50 lines net across 3 files.

---

## Structured Uncertainty

**What's tested:**

- ✅ Timeout is the cause of false 0/100 scores (verified: queries return matches at 8.8s, timeout is 5s)
- ✅ `runKBContextQuery` returns nil on timeout (verified: code trace, no DeadlineExceeded check)
- ✅ `AnalyzeGaps(nil, ...)` produces quality=0 with `GapTypeNoContext` (verified: code at gap.go:104-113)
- ✅ Fallback search overwrites timeout signal (verified: code at spawn_kb_context.go:93)

**What's untested:**

- ⚠️ 10s timeout is sufficient for all production queries (based on 8.8s maximum observed, not exhaustive)
- ⚠️ Raising timeout won't cause spawn latency complaints (max 10s per query, up from 5s)
- ⚠️ Go's `exec.CommandContext` returns partial stdout on timeout (Go docs suggest yes, but not verified for `kb` specifically)

**What would change this:**

- If queries routinely exceed 10s, a budget-based approach (shared context across queries) would be needed instead
- If partial stdout recovery works reliably, timeout results could include whatever matches were found before deadline

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Raise timeout from 5s to 10s | implementation | Single constant change, stays within spawn subsystem |
| Add TimedOut flag + GapTypeTimeout | architectural | Changes the KBContextResult/GapAnalysis contract across spawn+orch packages |
| Don't block gap gate on timeout | architectural | Changes gate semantics — affects spawn policy |

### Recommended Approach ⭐

**Timeout-aware gap analysis with raised timeout** — Detect timeouts as a distinct signal (not null), propagate through gap analysis as `GapTypeTimeout`, and raise the timeout to 10s.

**Why this approach:**
- Fixes the root cause (timeout ≠ no-matches) not just the symptom (timeout too short)
- Minimal change surface (~50 lines across 3 files, no new files)
- Preserves honest quality scoring (0 = zero matches found, but gap type explains WHY)
- Gate behavior changes only for timeout, not for genuine no-context

**Trade-offs accepted:**
- Per-query timeout (not budget-based): worst case rises to 40s from 20s. Acceptable because typical case is 1-2 queries (10-20s) and budget-based approach is a larger refactor.
- No partial result capture on timeout: potential improvement deferred.

**Implementation sequence:**

1. **Add `TimedOut bool` to `KBContextResult`** (pkg/spawn/kbcontext.go)
   - Foundation — all other changes depend on this signal

2. **Detect timeout in `runKBContextQuery`** (pkg/spawn/kbcontext.go:155-158)
   - Check `ctx.Err() == context.DeadlineExceeded`, return `KBContextResult{TimedOut: true}` instead of nil
   - Raise timeout constant from 5s to 10s

3. **Propagate timeout through `RunKBContextCheckForDir`** (pkg/spawn/kbcontext.go:80-130)
   - Track `anyTimedOut` across local and global queries
   - Clear timeout flag when either query returns real matches
   - Return timeout result only when no matches found AND at least one query timed out

4. **Add `GapTypeTimeout` to gap analysis** (pkg/spawn/gap.go)
   - New gap type with `GapSeverityWarning` (not critical)
   - `AnalyzeGaps`: when result.TimedOut && no matches → GapTypeTimeout instead of GapTypeNoContext
   - `ShouldBlockSpawn`: return false when any gap is GapTypeTimeout
   - `FormatGapSummary`: "KB context check timed out — agent may be missing historical context"

5. **Preserve timeout signal in fallback search** (pkg/orch/spawn_kb_context.go:89-98)
   - Don't overwrite timeout result with nil from fallback
   - Only overwrite if fallback returns real matches

6. **Observability: log timeout events** (pkg/orch/spawn_kb_context.go)
   - When result.TimedOut, log event type `kb.context.timeout` with query and project_dir
   - Print warning to stderr: "KB context check timed out for %q"

### Alternative Approaches Considered

**Option B: Budget-based timeout (shared context)**
- **Pros:** Prevents budget waste (slow local doesn't starve global). Total time capped at budget regardless of query count. More principled timeout management.
- **Cons:** Larger change — requires threading `context.Context` through `RunKBContextCheckForDir` API (breaking public signature) or adding new `WithContext` variant. More testing surface.
- **When to use instead:** If 10s per-query leads to observed 40s+ spawn times. Track via `kb.context.timeout` events.

**Option C: Only raise timeout (no detection)**
- **Pros:** 1-line change. Zero risk.
- **Cons:** Doesn't fix the structural defect. Any future query that exceeds 10s will produce the same false 0/100. No observability.
- **When to use instead:** Never as sole fix. Could be interim hotfix while detection is implemented.

**Rationale for recommendation:** Option A fixes both the symptom (tight timeout) and the disease (timeout ≟ no-matches). Option B is better architecturally but the added complexity isn't justified until we observe the problem with Option A. Option C is insufficient.

---

### Implementation Details

**What to implement first:**
- The `TimedOut` flag on `KBContextResult` (everything else depends on it)
- Then timeout detection in `runKBContextQuery` (the root fix)
- Then gap analysis changes (the user-visible fix)

**Things to watch out for:**
- ⚠️ **Defect Class 1 (Filter Amnesia):** The fallback search in `runPreSpawnKBCheckFull` must not overwrite a timeout result with nil. Use a separate variable for fallback results.
- ⚠️ **Defect Class 5 (Contradictory Authority):** Quality score 0 + GapTypeTimeout must not conflict. The 0 is honest (zero matches), the type explains why (timeout, not absence). Display text must make this clear.
- ⚠️ **Test coverage:** The existing test file `pkg/spawn/gap_test.go` (if it exists) needs timeout test cases. Also `kbcontext_test.go`.

**Areas needing further investigation:**
- Partial stdout capture on timeout — could recover matches that were output before deadline
- Budget-based timeout as follow-up if per-query approach causes latency issues
- Why `kb context` takes 5.8-8.8s on orch-go (280+ investigations may need index optimization)

**Success criteria:**
- ✅ Queries that timeout produce `GapTypeTimeout` warning, not `GapTypeNoContext` critical
- ✅ Gap gate does NOT block spawn when context check times out
- ✅ `kb.context.timeout` events are logged for observability
- ✅ Queries taking 5.8-8.8s now complete (10s timeout gives headroom)
- ✅ All existing gap tests continue to pass

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` — Timeout creation (L139), error handling (L155-158), two-stage search
- `pkg/spawn/gap.go` — `AnalyzeGaps` nil handling (L104-113), quality scoring, gate blocking
- `pkg/orch/spawn_kb_context.go` — `runPreSpawnKBCheckFull` orchestration, fallback search (L89-98)
- `pkg/orch/spawn_types.go` — `GapCheckResult` struct

**Commands Run:**
```bash
# Check for existing timeout handling
grep -r "DeadlineExceeded\|ctx.Err\|deadline" pkg/spawn/kbcontext.go
# Result: No matches — no timeout detection exists

# Check file sizes for hotspot compliance
wc -l pkg/spawn/kbcontext.go pkg/orch/spawn_kb_context.go pkg/spawn/gap.go
# 182, 171, 623 — all well under 1500-line threshold
```

**Related Artifacts:**
- **Investigation:** orch-go-304ta — GPT-5.4/headless 0/100 score tracing
- **Constraint:** "kb context command hangs on some queries" (kn)

---

## Investigation History

**2026-03-26:** Investigation started
- Initial question: How to fix false 0/100 scores from KB context timeout
- Context: orch-go-304ta traced the symptom to pre-spawn KB lookup timing

**2026-03-26:** Root cause confirmed — 3 design forks identified
- Fork 1: Timeout value (5s → 10s)
- Fork 2: Timeout detection (TimedOut flag + GapTypeTimeout)
- Fork 3: Gate behavior (don't block on timeout)

**2026-03-26:** Investigation completed
- Status: Complete
- Key outcome: Design for 3-layer fix (~50 lines, 3 files) that distinguishes timeout from genuine no-context
