# Session Synthesis

**Agent:** og-inv-audit-spawn-context-27feb-936a
**Issue:** orch-go-1goo
**Outcome:** success

---

## Plain-Language Summary

Cross-repo spawns (daemon spawning agents for toolshed/price-watch from orch-go) inject the WRONG project's knowledge into SPAWN_CONTEXT.md. The root cause is `runKBContextQuery` in kbcontext.go:250 — it runs `kb context` from the daemon's working directory (orch-go) instead of the target project directory. This means toolshed agents receive orch-go Dashboard Architecture knowledge instead of Toolshed Architecture knowledge. The Feb 25 fix for group-based filtering only fixed the global search path (Step 2), but if the local search (Step 1) finds enough results from the wrong project, Step 2 is never reached and the fix is bypassed entirely. Additionally, the gap analysis quality scoring reports 90-95% quality on these degraded spawns because it checks category population, not content relevance.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- Probe confirms `runKBContextQuery` CWD dependency via code trace AND real production evidence (3 toolshed spawns from Feb 27)
- Real SPAWN_CONTEXT.md files show orch-go knowledge injected into toolshed agents
- `kb context` output comparison from different CWDs confirms different results
- Two bugs filed: orch-go-t3ll (P1, CWD fix) and orch-go-3go1 (P2, gap analysis false positive)

---

## Delta (What Changed)

### Files Created
- `.kb/models/spawn-architecture/probes/2026-02-27-probe-cross-repo-spawn-context-quality-audit.md` - Probe documenting the CWD bug in runKBContextQuery and its impact on cross-repo spawn context quality
- `.orch/workspace/og-inv-audit-spawn-context-27feb-936a/SYNTHESIS.md` - This file
- `.orch/workspace/og-inv-audit-spawn-context-27feb-936a/VERIFICATION_SPEC.yaml` - Verification contract

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `kbcontext.go:250` — `runKBContextQuery` creates exec.Command without setting `cmd.Dir`, always uses process CWD
- `kbcontext.go:202` — `RunKBContextCheckForDir` passes `projectDir` to group filter (line 216) but NOT to `runKBContextQuery` (line 202)
- `kb context "pricing strategy"` from orch-go CWD returns orch-go guides (Model Selection, Two-Tier Sensing); from toolshed CWD returns toolshed models (Toolshed Architecture, Toolshed ↔ Price Watch Integration)
- Real toolshed SPAWN_CONTEXT.md at `to-feat-ai-pricing-strategy-27feb-c64d/SPAWN_CONTEXT.md` contains orch-go Dashboard Architecture invariants (connection pool, progressive disclosure) — 0% relevant to toolshed pricing panel
- events.jsonl gap scores: toolshed-74=90%, toolshed-98=95% despite 100% wrong-project knowledge
- events.jsonl toolshed-128: verification git_diff gate failed because changes were in price-watch repo but gate checked toolshed

### Tests Run
```bash
# Direct comparison of kb context output from different CWDs
kb context "pricing strategy"  # from orch-go → orch-go knowledge
cd ~/Documents/work/.../toolshed && kb context "pricing strategy"  # → toolshed knowledge
```

---

## Architectural Choices

No architectural choices — this was an audit/investigation, not implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/spawn-architecture/probes/2026-02-27-probe-cross-repo-spawn-context-quality-audit.md` - Extends spawn architecture model with CWD bug finding

### Constraints Discovered
- Local `kb context` search is CWD-dependent — `runKBContextQuery` must set `cmd.Dir` for cross-repo accuracy
- Gap analysis measures category fill, not content relevance — can't detect wrong-project injection
- Feb 25 group-filter fix only helps when Step 2 (global search) is reached — bypassed when Step 1 returns ≥3 results

### Issues Created
- `orch-go-t3ll` (P1 bug) — Fix runKBContextQuery CWD bug
- `orch-go-3go1` (P2 bug) — Gap analysis false positive on wrong-project context

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** orch-go-t3ll
**Skill:** systematic-debugging (or feature-impl for the fix)
**Context:**
```
Fix runKBContextQuery in kbcontext.go:250 to accept projectDir and set cmd.Dir.
Update RunKBContextCheckForDir to pass projectDir to both Step 1 and Step 2 calls.
This is a 3-line fix with high cross-repo spawn quality impact.
```

---

## Unexplored Questions

- Does the same CWD bug affect `orch spawn --workdir` (manual spawn) or only daemon spawns? Both call the same code path, so likely both affected.
- Are there other `exec.Command` calls in the spawn pipeline that should use `cmd.Dir = projectDir`?
- How many past cross-repo spawns have been affected? Could analyze events.jsonl for all non-orch-go beads-id spawns.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-audit-spawn-context-27feb-936a/`
**Probe:** `.kb/models/spawn-architecture/probes/2026-02-27-probe-cross-repo-spawn-context-quality-audit.md`
**Beads:** `bd show orch-go-1goo`
