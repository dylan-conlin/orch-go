# Session Synthesis

**Agent:** og-arch-design-fix-false-26mar-b628
**Issue:** orch-go-k6c0v
**Duration:** 2026-03-26
**Outcome:** success

---

## TLDR

Designed a 3-layer fix for false 0/100 spawn context scores: the root cause is `runKBContextQuery` returning nil on timeout (indistinguishable from "no matches"), which gap analysis misclassifies as `GapTypeNoContext`. The fix adds a `TimedOut` flag to `KBContextResult`, a new `GapTypeTimeout` gap type that doesn't trigger gate blocking, and raises the per-query timeout from 5s to 10s. ~50 lines across 3 files.

---

## Plain-Language Summary

When orch spawns an agent, it checks the knowledge base for relevant context and scores the quality 0-100. A score of 0 means "no prior knowledge found" and can block the spawn. The problem: the KB check has a 5-second timeout, but on orch-go's large knowledge base, many queries take 6-9 seconds. When they timeout, the system reports "no context found (0/100)" — which is wrong. The context exists, we just didn't wait long enough. The fix makes timeout a distinct signal from "genuinely no context": it raises the timeout to 10 seconds, flags timed-out queries so they produce a warning instead of a critical alarm, and prevents the gap gate from blocking spawns based on what is fundamentally "we don't know" rather than "we know there's nothing."

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-fix-false-0-100-spawn-context-scores.md` - Full design investigation

### Files Modified
- None (design only)

### Commits
- Investigation + workspace artifacts committed together

---

## Evidence (What Was Observed)

- `runKBContextQuery` (kbcontext.go:155-158) returns `nil, nil` for ALL errors including `context.DeadlineExceeded` — no timeout detection exists
- `AnalyzeGaps` (gap.go:104) treats nil result as `GapTypeNoContext` with quality=0 and severity=critical
- Per-query timeout is hardcoded to 5s (kbcontext.go:139), real queries on orch-go take 5.8-8.8s
- Fallback search in `runPreSpawnKBCheckFull` (spawn_kb_context.go:93) overwrites timeout result with nil, losing the signal (Defect Class 1: Filter Amnesia)
- Worst-case query chain: 4 sequential queries × 5s = 20s with zero useful output

---

## Architectural Choices

### Timeout-aware gap type vs. sentinel quality score
- **What I chose:** New `GapTypeTimeout` with quality=0 (honest) + gate exemption
- **What I rejected:** Setting quality=50 (neutral/unknown) to avoid gate trigger
- **Why:** Quality score should reflect actual data (0 matches found). Lying about quality creates Defect Class 5 (Contradictory Authority). Better to keep quality honest and change gate behavior based on gap type.
- **Risk accepted:** Consumers that check `quality == 0` without checking gap type will still see "bad" quality. Mitigated by changing display text.

### Per-query timeout vs. budget-based timeout
- **What I chose:** Raise per-query from 5s to 10s, keep per-query architecture
- **What I rejected:** Shared budget context threaded through all queries
- **Why:** Budget-based approach requires API changes (new `WithContext` function or `context.Context` parameter on public functions). Per-query raise is minimal change and addresses observed latency (8.8s max). Budget approach documented as follow-up if 10s per-query proves insufficient.
- **Risk accepted:** Worst case rises from 20s to 40s. Typical case (1-2 queries) is 10-20s.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-design-fix-false-0-100-spawn-context-scores.md` - Full design with 3 forks navigated

### Decisions Made
- Decision: timeout ≠ no-context — these are epistemically different states requiring different treatment
- Decision: gap gate should not block on "unknown" — absence of evidence is not evidence of absence

### Constraints Discovered
- `kbcontext.go` lacks any `context.DeadlineExceeded` check — Go's context cancellation signal is entirely ignored
- Fallback search pattern (`result, err = fn()`) silently overwrites prior signals — fragile for multi-stage pipelines

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for implementation acceptance criteria.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** orch-go-paatt (Implement timeout-aware gap analysis for KB context)
**Skill:** feature-impl
**Context:**
```
Design in .kb/investigations/2026-03-26-design-fix-false-0-100-spawn-context-scores.md.
3 files to modify: pkg/spawn/kbcontext.go (TimedOut flag + timeout detection + raise to 10s),
pkg/spawn/gap.go (GapTypeTimeout + gate exemption), pkg/orch/spawn_kb_context.go (preserve
timeout signal in fallback, add observability event). ~50 lines net. Watch for Defect Class 1
in fallback search.
```

---

## Unexplored Questions

- Why does `kb context` take 5.8-8.8s on orch-go? 280+ investigations may need index optimization in kb-cli.
- Could partial stdout be recovered from timed-out `exec.CommandContext`? Go docs suggest output captured before SIGKILL is returned.
- Budget-based timeout (shared `context.Context`) is architecturally cleaner — worth implementing if per-query approach shows latency issues.

---

## Friction

- No friction — smooth session

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-fix-false-26mar-b628/`
**Investigation:** `.kb/investigations/2026-03-26-design-fix-false-0-100-spawn-context-scores.md`
**Beads:** `bd show orch-go-k6c0v`
