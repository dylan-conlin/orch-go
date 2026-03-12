## Summary (D.E.K.N.)

**Delta:** Exploration orchestrators are structurally hybrid — they spawn sub-agents (orchestrator behavior) but run autonomously without interactive engagement (worker behavior). The current system misclassifies them as full orchestrators, causing 5 concrete problems: wrong tmux placement, stop hook misfires, missing Phase: Complete, review tier over-escalation, and architect_handoff false rejects.

**Evidence:** Code audit of `pkg/spawn/claude.go:153`, `enforce-phase-complete.py:45-50`, `pkg/spawn/verify_level.go`, `pkg/spawn/review_tier.go`, `pkg/verify/architect_handoff.go`. Cross-referenced with exploration-orchestrator skill at `skills/src/meta/exploration-orchestrator/SKILL.md` and tmux session management at `pkg/tmux/tmux.go:296-356`.

**Knowledge:** The exploration orchestrator needs a new classification: "autonomous orchestrator" — a third category alongside interactive orchestrator and worker. This category inherits tmux placement from workers, stop hook behavior from workers, Phase: Complete emission from workers, but retains orchestrator tool disallow list and context generation.

**Next:** Implementation across 5 files. Each fix is independent and can be shipped incrementally.

**Authority:** architectural — Crosses tmux placement, hook infrastructure, verification pipeline, and skill classification.

---

# Investigation: Exploration Orchestrator Lifecycle Design

**Question:** How should exploration orchestrators interact with tmux placement, phase reporting, stop hook, review tier, and completion gates?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Worker agent
**Phase:** Complete
**Status:** Complete

**Patches-Decision:** N/A (design-only, implementation issues to follow)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-11-inv-task-orchestrator-skill-design-tension.md | related — exploration orch is a new orchestrator variant | Yes | None |
| spawned-orchestrator-pattern.md (guide) | foundational — defines spawned orch lifecycle | Yes | Exploration orch violates several assumptions |

---

## Problem Statement

Exploration orchestrators (`orch spawn --explore`) have a lifecycle shaped like **decompose -> fan-out -> judge -> iterate -> synthesize**. They are:

- **Structurally orchestrators:** They spawn sub-agents (investigation workers, judge agents)
- **Operationally workers:** They run autonomously to completion, no interactive engagement needed

The current system treats them identically to interactive/spawned orchestrators, causing 5 problems:

### Problem 1: Wrong Tmux Placement

**Code:** `pkg/spawn/claude.go:153`
```go
if cfg.IsMetaOrchestrator || cfg.IsOrchestrator {
    sessionName, err = tmux.EnsureOrchestratorSession()
}
```

Exploration orchestrators land in the shared `"orchestrator"` tmux session alongside interactive orchestrators. This creates noise — exploration orchestrators run for 30-90min unattended, while interactive orchestrators need Dylan's attention. Having both in the same session makes it harder to find the interactive session that needs input.

**Fix:** Route exploration orchestrators to the workers session instead. They're autonomous — visibility in the orchestrator session has no benefit.

### Problem 2: Stop Hook Misfires

**Code:** `enforce-phase-complete.py:45-50`
```python
def is_spawned_worker() -> bool:
    return (
        os.environ.get("ORCH_SPAWNED") == "1"
        and os.environ.get("CLAUDE_CONTEXT") == "worker"
    )
```

The stop hook ONLY gates `CLAUDE_CONTEXT=worker`. Exploration orchestrators have `CLAUDE_CONTEXT=orchestrator`, so the hook **does not fire** for them. This is actually the opposite problem from the task description — the hook doesn't fire at all, which means exploration orchestrators can exit without Phase: Complete.

However, during the wait phases (Phase 3: waiting for workers, Phase 4b: waiting for judge), the exploration orchestrator spawns sub-agents and then polls/waits. If the orchestrator hits max-turns or rate-limits during a wait, it exits without Phase: Complete and the stop hook doesn't catch it.

**Fix:** Set `CLAUDE_CONTEXT=exploration-orchestrator` and update the stop hook to also gate this context type. Or simpler: set `CLAUDE_CONTEXT=worker` for exploration orchestrators since they follow worker lifecycle.

### Problem 3: No Standard Phase: Complete Emission

**Code:** The exploration-orchestrator skill (`skills/src/meta/exploration-orchestrator/SKILL.md:158`) does include a Phase: Complete instruction:
```
Report: `bd comment <beads-id> "Phase: Complete - Exploration synthesis: ..."`
```

But since orchestrators are spawned with `CLAUDE_CONTEXT=orchestrator`, the worker-base skill's phase reporting protocol isn't loaded. The exploration-orchestrator skill includes its own completion instruction, but it's not reinforced by the stop hook (Problem 2). The skill guidance is the only layer — no infrastructure enforcement.

**Fix:** Either:
- (a) Set `CLAUDE_CONTEXT=worker` so worker-base protocols apply, OR
- (b) Extend the stop hook to also gate `CLAUDE_CONTEXT=exploration-orchestrator`

Option (a) is simpler and more robust — it gives exploration orchestrators the full worker lifecycle protocol including phase reporting, friction capture, and discovered work.

### Problem 4: Review Tier Over-Escalation

**Code:** `pkg/spawn/review_tier.go:22-43`

The `exploration-orchestrator` skill is not in `SkillReviewTierDefaults`, so it falls through to the conservative default: `ReviewReview` (full review). Exploration orchestrators produce a synthesis document — they're knowledge-producing, similar to investigation/research. They should default to `ReviewScan`.

Similarly, `SkillVerifyLevelDefaults` doesn't include `exploration-orchestrator`, defaulting to `VerifyV1` (conservative). Since exploration orchestrators produce investigation artifacts and SYNTHESIS.md, `VerifyV1` is actually correct — but it should be explicit.

**Fix:** Add to both maps:
```go
"exploration-orchestrator": ReviewScan   // review_tier.go
"exploration-orchestrator": VerifyV1     // verify_level.go
```

### Problem 5: Architect Handoff Gate (Resolved)

The task description mentions "architect_handoff gate rejects spawn-follow-up." This was already fixed in commit `b0c113434` — `spawn-follow-up` is now a valid architect recommendation. No further work needed.

---

## Design: The "Autonomous Orchestrator" Classification

### Current Classification

```
CLAUDE_CONTEXT values:
  "worker"             → Workers: autonomous, beads-tracked, stop-hook gated
  "orchestrator"       → Orchestrators: autonomous, session-registry, no stop-hook
  "meta-orchestrator"  → Meta-orchestrators: interactive, session-registry, no stop-hook
```

### Proposed Classification

Two options:

**Option A: New CLAUDE_CONTEXT value** (structural clarity, more code changes)
```
CLAUDE_CONTEXT values:
  "worker"                      → Workers: autonomous, beads-tracked, stop-hook gated
  "autonomous-orchestrator"     → Exploration orch: autonomous, beads-tracked, stop-hook gated, tool-restricted
  "orchestrator"                → Spawned orchestrators: autonomous, session-registry, no stop-hook
  "meta-orchestrator"           → Meta-orchestrators: interactive, session-registry, no stop-hook
```

**Option B: Reclassify as worker** (minimal changes, pragmatic)
```
Set CLAUDE_CONTEXT=worker for exploration-orchestrator skill.
Keep IsOrchestrator=false despite skill-type: orchestrator.
Override tmux routing to workers session.
```

### Recommendation: Option B (Reclassify as Worker)

Rationale:
- Exploration orchestrators follow the worker lifecycle (beads-tracked, stop-hook gated, Phase: Complete)
- The only orchestrator-specific behavior they need is the tool disallow list (`--disallowedTools 'Agent,Edit,Write,NotebookEdit'`)
- A new CLAUDE_CONTEXT value requires changes across: stop hook, claude.go, config detection, skill loading, spawn_cmd.go
- Option B requires changes in: spawn_cmd.go (override IsOrchestrator for explore mode), review_tier.go, verify_level.go

**The tool disallow list** can be handled separately — add a `DisallowTools` field to Config that's set for exploration orchestrators regardless of CLAUDE_CONTEXT.

---

## Implementation Plan

### Change 1: Tmux Placement (pkg/spawn/claude.go)

Override IsOrchestrator for exploration orchestrators so they route to workers session.

**Where:** The caller (`spawn_cmd.go`) should set `cfg.IsOrchestrator = false` when `cfg.Explore == true`, while preserving the tool disallow list.

**Alternative:** In `claude.go:153`, add a check:
```go
if (cfg.IsMetaOrchestrator || cfg.IsOrchestrator) && !cfg.Explore {
    sessionName, err = tmux.EnsureOrchestratorSession()
} else {
    sessionName, err = tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
}
```

### Change 2: CLAUDE_CONTEXT (pkg/spawn/config.go or claude.go)

For exploration orchestrators, set `CLAUDE_CONTEXT=worker` so:
- Stop hook fires and enforces Phase: Complete
- Worker-base skill protocols apply

**Where:** In the `ClaudeContext()` method or in spawn_cmd.go when setting up config.

**Tool disallow list preservation:** Add explicit `DisallowTools` field:
```go
if cfg.Explore {
    cfg.DisallowTools = "Agent,Edit,Write,NotebookEdit"
}
```

Then in `BuildClaudeLaunchCommand`, use `DisallowTools` instead of checking `claudeContext`.

### Change 3: Review Tier & Verify Level Defaults (pkg/spawn/)

Add exploration-orchestrator to the defaults maps:

```go
// review_tier.go
"exploration-orchestrator": ReviewScan,

// verify_level.go
"exploration-orchestrator": VerifyV1,

// config.go - SkillTierDefaults
"exploration-orchestrator": TierFull,  // Produces SYNTHESIS.md

// config.go - SkillProducesInvestigation
"exploration-orchestrator": true,  // Produces investigation-style output
```

### Change 4: Exploration-Orchestrator Skill Update (skills/src/meta/)

The skill already includes Phase: Complete instructions. No changes needed to the skill itself — the infrastructure changes (stop hook gating, worker lifecycle) will provide the enforcement layer.

### Change 5: Tests

- Update `TestExplorationOrchestratorSkillLoads` — verify it's detected as orchestrator type but treated as worker lifecycle
- Add test verifying exploration orchestrators route to workers tmux session
- Add test verifying CLAUDE_CONTEXT is "worker" for explore mode
- Add review tier/verify level default tests for exploration-orchestrator

---

## Structured Uncertainty

**What's tested:**
- All 5 problems confirmed via code inspection
- Problem 5 (architect_handoff) confirmed resolved
- Current tmux routing, stop hook, and verification pipeline code paths verified

**What's untested:**
- Whether setting CLAUDE_CONTEXT=worker causes any downstream issues for exploration orchestrators (e.g., skill loading, context generation)
- Whether the tool disallow list refactor (from CLAUDE_CONTEXT check to explicit DisallowTools field) has edge cases
- Whether exploration orchestrators actually hit the stop hook in practice (they might complete gracefully every time)

**What would change this:**
- If exploration orchestrators need tool access that workers have but orchestrators don't (currently orchestrators are MORE restricted, so this is unlikely)
- If we add more orchestrator-type skills that also need worker lifecycle — at that point, Option A (new CLAUDE_CONTEXT) might become necessary

---

## References

**Files Examined:**
- `pkg/spawn/claude.go` — SpawnClaude, BuildClaudeLaunchCommand, tmux routing
- `pkg/spawn/config.go` — Config struct, skill defaults
- `pkg/spawn/verify_level.go` — Verification level defaults
- `pkg/spawn/review_tier.go` — Review tier defaults
- `pkg/verify/level.go` — Gate-by-level mapping
- `pkg/verify/architect_handoff.go` — Architect handoff gate
- `pkg/tmux/tmux.go` — Session management (orchestrator vs workers)
- `~/.orch/hooks/enforce-phase-complete.py` — Stop hook implementation
- `skills/src/meta/exploration-orchestrator/SKILL.md` — Exploration orchestrator skill
- `pkg/spawn/explore_test.go` — Existing explore tests
- `pkg/spawn/explore_skill_test.go` — Skill loading tests

**Related Artifacts:**
- Guide: `.kb/guides/spawned-orchestrator-pattern.md`
- Investigation: `.kb/investigations/2026-03-11-inv-task-orchestrator-skill-design-tension.md`

---

## Investigation History

**2026-03-11 T1:** Investigation started. Read spawned-orchestrator-pattern guide, design tension mapping, orchestrator skill probes.
**2026-03-11 T2:** Code audit of all 5 problem areas. Confirmed Problem 5 already resolved. Identified root cause as classification mismatch.
**2026-03-11 T3:** Designed Option A (new CLAUDE_CONTEXT) vs Option B (reclassify as worker). Recommended Option B for minimal changes.
**2026-03-11 T4:** Wrote implementation plan with 5 changes. Investigation complete.
