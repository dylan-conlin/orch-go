# Probe: Spawn-Time Staleness Detection Effectiveness

**Model:** drift-taxonomy
**Date:** 2026-03-01
**Status:** Complete

---

## Question

Does spawn-time staleness detection actually catch meaningful drift at consumption and produce better agent behavior? The drift-taxonomy model claims: "Detection at consumption beats detection at production. Spawn-time staleness detection is the primary consumption-time mechanism."

---

## What I Tested

1. Checked `~/.orch/model-staleness-events.jsonl` for volume, coverage, and patterns (1,343 events across 546 spawns, 12-day window Feb 18 - Mar 1)
2. Read `pkg/spawn/kbcontext.go:1422-1497` — the `checkModelStaleness()` function that detects staleness by comparing model `Last Updated` dates against `git log --since=` on `code_refs` files
3. Read `pkg/spawn/staleness_events.go` — event recording and 30-day retention
4. Read `pkg/spawn/kbcontext.go:1111-1182` — `formatModelMatchForSpawn()` which injects `**STALENESS WARNING:**` into SPAWN_CONTEXT.md
5. Searched 1,682 archived SPAWN_CONTEXT.md files for actual warning injection
6. Searched agent SYNTHESIS.md files for evidence agents used staleness warnings
7. Verified whether files flagged as "deleted" are actually deleted

```bash
# Event volume
wc -l ~/.orch/model-staleness-events.jsonl
# → 1343 events

# Warning injection rate
find .orch/workspace/archived -name "SPAWN_CONTEXT.md" | wc -l  # → 1682
find .orch/workspace/archived -name "SPAWN_CONTEXT.md" -exec grep -l "STALENESS WARNING" {} \; | wc -l  # → 506

# False positive analysis: noise file triggers
# .beads/issues.jsonl → 282 events (21%), CLAUDE.md → 101 events (7.5%)

# Agent awareness: SYNTHESIS.md files referencing staleness
find .orch/workspace/archived -name "SYNTHESIS.md" -exec grep -l -i "staleness\|stale model" {} \;
# → Only found staleness-specific audit tasks, no spontaneous agent action from warnings

# Deleted file verification
ls pkg/session/registry.go cmd/orch/serve_agents.go cmd/orch/abandon.go 2>&1
# → All genuinely deleted (TRUE positives for structural staleness)
```

---

## What I Observed

### Detection Mechanism (Works Correctly)

The mechanism at `kbcontext.go:1422` is sound:
1. Extracts `Last Updated: YYYY-MM-DD` from model content
2. Extracts file paths from `<!-- code_refs: ... -->` blocks in the model's Primary Evidence section
3. For each file: checks `os.Stat()` (deleted?) and `git log --since=<date+1d>` (changed?)
4. If stale: injects a `**STALENESS WARNING:**` block into SPAWN_CONTEXT.md with specific changed/deleted files
5. Records event to `~/.orch/model-staleness-events.jsonl`

### Volume (High Activity)

| Metric | Value |
|--------|-------|
| Total events (12 days) | 1,343 |
| Unique spawns with stale models | 546 |
| Unique models flagged | 25 |
| SPAWN_CONTEXT.md files with warnings | 506/1,682 (30.1%) |
| Events per spawn (avg) | 2.5 stale models per spawn |

### Signal vs Noise Analysis

| Category | Count | % | Assessment |
|----------|-------|---|------------|
| Events with genuinely deleted files | 963 | 71.7% | TRUE positives — model references files that no longer exist |
| Events with only changed files | 380 | 28.3% | Mixed — some meaningful, some noise |
| Events where ALL triggers are noise files | 7 | 0.5% | Pure false positives |
| Events involving `.beads/issues.jsonl` | 282 | 21.0% | NOISE — changes on every beads operation |
| Events involving `CLAUDE.md` | 101 | 7.5% | NOISE — changes frequently, rarely invalidates model claims |

**Key false positive patterns:**
- `.beads/issues.jsonl` changes 499 times in the same period — any model listing it in `code_refs` is perpetually "stale" (noise, not signal)
- `CLAUDE.md` changes frequently but rarely invalidates model architectural claims
- Home-dir files (`~/.orch/sessions.json`, `~/.orch/coaching-metrics.jsonl`) flagged as deleted 69-75 times but actually exist — suggests they were temporarily absent or recreated

**True positives (top 3):**
- `pkg/session/registry.go` — genuinely deleted (137 events). Models referencing it describe architecture that no longer exists.
- `cmd/orch/serve_agents.go` — genuinely deleted (84 events). Extracted/renamed.
- `cmd/orch/abandon.go` — genuinely deleted (60 events). Command removed.

### Agent Behavioral Impact (None Observed)

**Critical finding:** No evidence that agents spontaneously change their behavior because of a staleness warning.

- `og-debug-bug-workspace-manifest-01mar-1629` had 3 staleness warnings but its SYNTHESIS.md discussed only the manifest priority bug. The warnings about `workspace-lifecycle-model`, `beads-database-corruption`, and `beads-integration-architecture` were irrelevant to the agent's debugging task.
- `og-feat-update-daemon-guide-01mar-f0ef` had a staleness warning about the daemon model but its workspace had empty SYNTHESIS.md — unclear if the warning helped.
- The ONLY case where staleness produced actionable work was `og-audit-staleness-audit-orchestrator-18feb-8da1` — but that was an **explicitly assigned staleness audit task**, not an agent spontaneously acting on a warning.
- No investigation files reference "STALENESS WARNING" as a trigger for corrective action.

### Why Agents Ignore Warnings

The warning text says "Verify model claims about these files against current code" — but agents are spawned with a specific task (fix bug, implement feature, investigate topic). Verifying model staleness is orthogonal to their assigned work. The warning adds context noise without a clear action path.

---

## Model Impact

- [x] **Extends** model with: The detection mechanism works correctly at the mechanical level, but the model's implicit assumption that consumption-time detection produces better agent behavior is unvalidated. Detection creates awareness (30% of spawns see warnings), but awareness without action is noise.

**What the model gets right:**
- "Detection at consumption beats detection at production" — the timing is correct (spawn-time checks catch stale models before agents use them)
- The staleness detection correctly identifies genuinely deleted files (71.7% of events)
- The mechanism scales (1,343 events across 546 spawns with minimal latency overhead)

**What the model should acknowledge:**
1. **Awareness ≠ Action.** Agents see the warning but don't act on it because staleness remediation isn't their assigned task. The model says detection at consumption catches drift "at the moment of harm" — but without behavioral change, the harm still occurs.
2. **False positive rate degrades trust.** 21% of events involve `.beads/issues.jsonl` (a file that changes on every beads operation). Agents that see perpetually-stale models learn to ignore all staleness warnings.
3. **The actual value is in the event log, not the annotation.** The `model-staleness-events.jsonl` file is useful for the daemon/periodic reflection to prioritize model updates. The inline SPAWN_CONTEXT.md annotation adds ~100-200 characters of noise per stale model with no observed behavioral benefit.
4. **Missing: a feedback loop from detection to remediation.** The model describes Detect-Annotate-Queue but the Queue step (daemon creating model-drift issues) is the actual value-producing step, not the Annotate step.

---

## Notes

**Recommendations (for architect review, not direct implementation):**
1. **Filter noise files from code_refs:** `.beads/issues.jsonl`, `CLAUDE.md`, and other frequently-changing files that don't invalidate model claims should be excluded or tagged as `volatile` in `code_refs` blocks.
2. **Consider removing inline annotations:** The SPAWN_CONTEXT.md warnings are ~30% of spawn contexts but produce no observed agent behavioral change. The event log alone may be sufficient for the daemon/reflection pipeline.
3. **Model update:** Add a constraint to the drift-taxonomy model: "Consumption-time detection creates awareness, not action. The value is in the event log feeding the remediation pipeline, not in the agent-facing annotation."
