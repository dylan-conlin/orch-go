# Session Synthesis

**Agent:** og-inv-investigate-operationalizing-ralph-25mar-66f8
**Issue:** orch-go-pk7ds
**Duration:** 2026-03-25
**Outcome:** success

---

## Plain-Language Summary

The Ralph Wiggum loop (`while :; do cat PROMPT.md | claude-code; done`) is a pattern for running an AI agent in continuous improvement cycles — autoresearch uses it for ML research, our iterate-design skill uses it for OpenSCAD. This investigation asked: should orch-go make this a first-class feature, and how?

The answer is yes, as a spawn flag (`orch spawn --loop`). The surprising finding: orch-go already has all the building blocks. The rework command creates fresh agent contexts with prior results injected. The wait command blocks until an agent finishes. The exploration events track iteration metadata. What's missing is a 200-line controller that composes these: spawn an agent, wait for it to finish, run an evaluation command (like `go test -cover`), check if the result improved, and either stop or re-spawn with the results fed back in. This "automated rework loop" gives orch-go a clear advantage over raw while loops — structured knowledge transfer between iterations so each agent is smarter than the last.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

**Key outcomes:**
- Investigation file complete with 6 findings, D.E.K.N. summary, and implementation recommendation
- Completion criteria taxonomy mapped across 9 domains
- Architectural recommendation: `--loop` spawn flag composing rework + wait + eval primitives
- No code changes (investigation only) — architect session recommended as next step

---

## TLDR

Investigated how to operationalize the Ralph loop (continuous agent iteration) in orch-go. Found that existing rework primitives + wait command + exploration events compose into a loop controller without new subsystems. Recommended: `--loop` spawn flag with pluggable eval command (`--loop-cmd "go test -cover" --loop-target 80`).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-25-inv-investigate-operationalizing-ralph-loop-orch.md` — Full investigation with 6 findings
- `.orch/workspace/og-inv-investigate-operationalizing-ralph-25mar-66f8/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-investigate-operationalizing-ralph-25mar-66f8/BRIEF.md` — Comprehension brief
- `.orch/workspace/og-inv-investigate-operationalizing-ralph-25mar-66f8/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- None (investigation only)

---

## Evidence (What Was Observed)

- `pkg/spawn/config.go` has `ReworkFeedback`, `PriorSynthesis`, `ReworkNumber` fields — cross-iteration knowledge transfer already supported
- `pkg/orch/spawn_modes.go:66-109` inline mode uses `WaitForSessionIdle()` — blocking completion detection exists
- `pkg/events/logger.go:899-928` has `ExplorationIteratedData` struct — iteration event tracking exists
- `cmd/orch/rework_cmd.go` creates fresh workspace with prior context — the rework command IS a manual single-step loop
- `cmd/orch/spawn_cmd.go:246-267` `--explore` flag swaps skill and adds fields — precedent for spawn-flag-driven behavior transformation
- autoresearch (`program.md`) uses results.tsv as unstructured cross-iteration memory; orch-go's SPAWN_CONTEXT is the structured equivalent

---

## Architectural Choices

### Loop mode as spawn flag vs new skill type vs daemon behavior
- **What I chose:** `--loop` spawn flag (Option A)
- **What I rejected:** New skill type (Option B — skill explosion, doesn't leverage rework), daemon behavior (Option C — conflates routing and execution, violates daemon-routes-not-executes decision)
- **Why:** Follows `--explore` precedent, composes existing primitives, works with any skill
- **Risk accepted:** Rework overhead per iteration may be 10-30s; acceptable for improvement loops but too slow for tight inner loops

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-25-inv-investigate-operationalizing-ralph-loop-orch.md` — Full analysis of loop operationalization

### Decisions Made
- Decision 1: `--loop` spawn flag over new skill type — because it composes existing primitives and follows `--explore` precedent
- Decision 2: Pluggable eval command over built-in metrics — because domain portability is more valuable than convenience presets

### Constraints Discovered
- Rework creates full workspace + context each iteration — overhead may matter for tight loops
- Daemon may try to claim loop-spawned issues — will need `loop:managed` exemption
- Only scalar-metric domains are good first candidates — judgment-requiring domains need different patterns

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Design --loop spawn flag architecture
**Skill:** architect
**Context:**
```
Investigation orch-go-pk7ds found that --loop should compose rework + wait + eval primitives.
Design: loop controller placement (pkg/orch/loop.go), flag set (--loop-cmd, --loop-target,
--loop-max, --loop-direction), eval interface, daemon exemption. Start with test coverage domain.
```

---

## Unexplored Questions

- Whether PriorSynthesis alone is rich enough for effective cross-iteration learning, or if a dedicated `ITERATION_LOG.md` artifact is needed
- How loop mode interacts with git state — should each iteration commit? Branch per iteration like autoresearch?
- Whether parallel loops (multiple agents iterating on different aspects) compose or interfere
- Cost implications — each iteration is a full agent session, 10 iterations of Opus could cost $20-50

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-operationalizing-ralph-25mar-66f8/`
**Investigation:** `.kb/investigations/2026-03-25-inv-investigate-operationalizing-ralph-loop-orch.md`
**Beads:** `bd show orch-go-pk7ds`
