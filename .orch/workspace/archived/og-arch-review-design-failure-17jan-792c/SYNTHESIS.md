# Session Synthesis

**Agent:** og-arch-review-design-failure-17jan-792c
**Issue:** orch-go-k3xr1
**Duration:** 2026-01-17 10:31 → 2026-01-17 10:55
**Outcome:** success

---

## TLDR

Identified root cause of silent coaching plugin: `detectWorkerSession()` caches `false` prematurely on first tool call, and the bash workdir detection signal never fires because bash has no `workdir` argument. Worker health metrics exist in code but are never reached because all workers are misclassified as orchestrators.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md` - Root cause analysis of worker detection failure

### Files Modified
- None (investigation only, no code changes)

### Commits
- Pending (investigation file to be committed)

---

## Evidence (What Was Observed)

- Metrics file (`~/.orch/coaching-metrics.jsonl`, 78KB) contains orchestrator metrics (action_ratio, analysis_paralysis) but ZERO worker metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap)
- `detectWorkerSession()` line 1256 caches result on EVERY tool call, even when `isWorker = false`
- bash tool schema shows args: `command`, `timeout`, `dangerouslyDisableSandbox`, `run_in_background` - NO `workdir`
- Commit b82715c1 ("fix: enable plugin loading and refine worker detection") removed detection signal 3 (filePath in .orch/workspace/)
- Prior investigation (Jan 11) recommended daemon architecture separation but was never implemented

### Tests Run
```bash
# Check for worker metrics
grep -E "tool_failure_rate|context_usage" ~/.orch/coaching-metrics.jsonl
# Result: EMPTY (zero hits)

# Check what metrics exist
tail -20 ~/.orch/coaching-metrics.jsonl
# Result: action_ratio, analysis_paralysis, compensation_pattern - all orchestrator metrics

# Check detection signal removal
git show b82715c1
# Result: Confirmed removal of filePath detection signal
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md` - Comprehensive root cause analysis with 5 findings and fix recommendations

### Decisions Made
- **Pattern established:** Never cache negative results in per-session detection - applies to any future session-type detection logic
- **Root cause identified:** Premature caching + invalid bash workdir assumption + removed detection signal 3

### Constraints Discovered
- **Bash tool has no `workdir` argument** - Detection signal 1 is fundamentally broken
- **Caching on first miss creates race condition** - If ANY tool call happens before SPAWN_CONTEXT.md read, detection fails permanently
- **"Fix" commits without testing** - Changes to detection logic aren't being validated against actual worker sessions

### Externalized via `kn`
- N/A (findings captured in investigation file with recommend-yes for decision promotion)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix worker detection in coaching.ts
**Skill:** feature-impl
**Context:**
```
detectWorkerSession() in plugins/coaching.ts has 3 bugs causing all workers to be misclassified as orchestrators:
1. Line 1256 caches `false` prematurely - should only cache `true`
2. bash workdir check (lines 1238-1244) never fires - bash has no workdir arg
3. Commit b82715c1 removed filePath detection signal - needs restoration

See .kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md for full analysis.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orchestrator frame collapse detection use similar heuristics? (potential similar bugs)
- Should the daemon architecture (from Jan 11 investigation) be prioritized now that we understand the coupling better?
- Why was detection signal 3 removed in commit b82715c1? (rationale not documented)

**Areas worth exploring further:**
- Whether explicit worker flag at spawn time would be more reliable than heuristics
- Performance impact of not caching detection results

**What remains unclear:**
- Whether the bash workdir arg was ever valid (or always broken from start)
- How many historic worker sessions were misclassified before this fix

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20250929
**Workspace:** `.orch/workspace/og-arch-review-design-failure-17jan-792c/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md`
**Beads:** `bd show orch-go-k3xr1`
