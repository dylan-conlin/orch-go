# Probe: Human Feedback Channel Structural Disuse

**Model:** completion-verification
**Date:** 2026-03-20
**Status:** Complete
**claim:** CV-07 (Rework Path Gap)
**verdict:** extends

---

## Question

The completion-verification model's "Why This Fails §7" describes a "Rework Path Gap" — claiming no automated rework path, no rework count tracking, and no `agent.reworked` event. But `orch rework` now exists and emits `agent.reworked` events. Why does the human negative-feedback channel (rework, abandon, manual review) show 0 reworks and only 11 abandons across 1,102+ completions? Is this a UX friction problem, a structural incentive problem, or both?

---

## What I Tested

### 1. Event Data — Ground Truth

```bash
grep '"agent.completed"' ~/.orch/events.jsonl | wc -l        # 1102
grep '"session.auto_completed"' ~/.orch/events.jsonl | wc -l  # 406
grep '"agent.reworked"' ~/.orch/events.jsonl | wc -l           # 0
grep '"agent.abandoned"' ~/.orch/events.jsonl | wc -l          # 11
grep '"agent.force_completed"' ~/.orch/events.jsonl | wc -l    # 0
grep '"agent.force_abandoned"' ~/.orch/events.jsonl | wc -l    # 0
```

### 2. Rework Command UX — Friction Audit

Read `cmd/orch/rework_cmd.go` (356 lines). Counted mandatory inputs and blocking checks:

1. **Beads ID** (mandatory arg) — must know the issue ID
2. **Feedback text** (mandatory arg) — must articulate what went wrong
3. **`--bypass-triage`** (mandatory flag) — without it, command fails immediately (line 69-71)
4. Issue must be **closed** — open issues rejected unless `--force` (line 86-87)
5. Prior workspace must be **findable** — archived workspace lookup required (line 96-106)
6. Runs full preflight checks: hotspot, agreements, open questions (line 154)
7. Gathers full spawn context: KB context, gap analysis, model injection (line 216)
8. Only supports **worker skills** — orchestrator rework rejected (line 163-165)

**Total: 3 mandatory inputs + 5 blocking preconditions = 8 friction points before any agent spawns.**

Compare to `orch work <issue-id>` (re-spawn from issue): **1 input, 0 preconditions** (minus `--bypass-triage`).

### 3. Abandon Command UX — Friction Audit

Read `cmd/orch/abandon_cmd.go` (362 lines). Simpler than rework but still guarded:

1. **Beads ID** (mandatory arg)
2. Activity check: blocks if agent reported phase within **30 minutes** (line 293-306)
3. Requires `--force` to override activity check
4. Optional `--reason` for failure report

All 11 historical abandon reasons examined:
```
"Silent stall at Planning. Overlapping work already completed..."
"Orchestrator implemented the gate directly — worker was blocked by governance hooks"
"Agent BLOCKED by governance hook on pkg/spawn/gates/..."
"Daemon auto-spawned on bug issue — not ready for implementation..."
""  (empty)
"Orphaned agent from earlier session, no tmux window or active process"  (x3)
"unresponsive during lunch, will respawn"
"silent stall, no phase ever reported, respawning"
```

**Every single abandon is operational** (stuck/orphaned/duplicate). Zero are quality judgments ("agent did bad work").

### 4. Daemon Auto-Completion — The Bypass Path

Read `pkg/daemon/auto_complete.go` and `pkg/daemon/coordination.go`. The daemon routes completions based on review tier:

```go
// coordination.go:RouteCompletion()
if IsEffortSmall(agent.Labels) { return "auto-complete-light" }
if reviewTier == "auto"        { return "auto-complete" }
if reviewTier == "scan"        { return "auto-complete" }
return "label-ready-review"  // default
```

Auto-complete implementation (`OrcCompleter.Complete`):
```go
cmd := exec.Command("orch", "complete", beadsID, "--force")
```

**`--force` bypasses ALL interactive verification gates.** 406 of 1,102 completions (37%) were auto-completed by daemon — never reviewed by a human.

### 5. bd close — The Zero-Check Path

Read `.beads/hooks/on_close`. Direct `bd close` has:
- Zero quality checks
- Zero verification gates
- Emits a sparse `agent.completed` event (beads_id + reason only)
- When called from `orch complete`, the hook is suppressed (`ORCH_COMPLETING=1`)

This means `bd close` is an unguarded completion path — any issue can be closed with 1 command and 0 quality signals.

### 6. review done — The Batch Path

Read `cmd/orch/review_done.go`. Batch-closes all verified agents per project:
- Prompts for follow-up issues from synthesis recommendations (y/n/skip-all)
- `--no-prompt` skips all interactive checks
- Closes beads issues, archives workspaces, logs events
- No explain-back, no `--verified`, no behavioral check

---

## What I Observed

### The Asymmetry

| Action | Steps | Effort | Usage |
|--------|-------|--------|-------|
| Auto-complete (daemon) | 0 (fully automated) | None | 406 (37%) |
| `review done` (batch close) | 1-2 (confirmation + prompt) | ~30 seconds | Bulk of remaining |
| `orch complete` (manual) | 5-10 (flags, explain, verify) | 5-10 minutes | Some |
| `orch abandon` | 1-2 (ID + optional reason) | ~1 minute | 11 (all operational) |
| `orch rework` | 3 mandatory + 5 preconditions | 5-15 minutes | **0** |

### Why Zero Reworks

1. **Friction asymmetry:** Rework requires 8 friction points; re-spawning the same issue via `orch work` requires 1. Rational actors choose re-spawn.
2. **Rework requires a closed issue:** You must complete the bad work BEFORE reworking it. This creates a paradox — if you know the work is bad, completing it (to then rework) feels like validating it.
3. **No rework button in the review flow:** `orch review` shows agents and suggests `orch complete`. There is no "rework" action in the review UX. The orchestrator must exit review, run a separate rework command, remember the beads ID, and formulate feedback.
4. **Prior workspace dependency:** Rework requires the archived workspace to exist. If `orch clean` ran first, rework fails.

### Why Zero Quality Abandons

1. **Abandon = "agent is broken"**, not "work is bad." The command's UX (kill tmux, export transcript, generate FAILURE_REPORT.md) is oriented around stuck/crashed agents.
2. **No "reject" verb exists.** There is no `orch reject <id> "this work is wrong"` that would reopen the issue for re-assignment.
3. **Abandoning feels like waste.** Even if work is low quality, the agent consumed tokens and time. Users default to completing rather than abandoning.

### The False Ground Truth Problem

The system records 1,102 completions and interprets them as "work successfully done." But the 0-rework, 0-quality-abandon rate means:
- **No negative signal enters the learning loop.** `pkg/daemon/learning.go` computes skill success rates from completed/abandoned ratios — but if abandoned=0, success rate=100% for all skills.
- **Gate effectiveness metrics are inflated.** `orch stats` shows per-gate pass/fail/bypass rates, but "pass" means "mechanical gate passed," not "work was good."
- **The daemon auto-completes 37% of agents** without any quality check. These completions are counted as successes indistinguishable from verified ones.

### Model Staleness in §7

The model's "Why This Fails §7: Rework Path Gap" states:
> "Rework count is not tracked, no agent.reworked event exists"

This is stale. `orch rework` exists (cmd/orch/rework_cmd.go, 356 lines). `agent.reworked` events are defined and logged (pkg/events/). The rework command was built — it just has never been used. The gap is now UX friction, not missing infrastructure.

---

## Model Impact

- [ ] **Confirms** invariant: N/A
- [ ] **Contradicts** invariant: §7 claim that "no agent.reworked event exists" — the event exists, rework count is tracked, the command was built. The model should be updated to reflect that the infrastructure exists but is structurally unusable due to friction.
- [x] **Extends** model with: **Human negative-feedback channel is structurally broken** not because the infrastructure is missing but because:
  1. **Completing is frictionless** (daemon auto-completes 37%, batch close handles the rest)
  2. **Reworking is 8x harder** than re-spawning from the same issue
  3. **No "reject" verb exists** in the completion vocabulary
  4. **All 11 abandons are operational**, not quality-based — showing the abandon UX is oriented around broken agents, not bad work
  5. **The learning loop has no negative signal** — skill success rates show 100% because failed-quality-work paths don't emit distinct events

---

## Notes

### Recommendations (for architect)

1. **Add `orch reject <id> "reason"`** — a friction-equivalent-to-complete negative signal that reopens the issue and emits an `agent.rejected` event. No workspace lookup, no mandatory triage bypass.
2. **Add reject action to `orch review`** — when reviewing completed agents, allow `r` (reject) alongside `y` (complete) and `n` (skip).
3. **Distinguish auto-completed from verified in stats** — `orch stats` should show auto-completed (37%) separately from human-verified completions.
4. **Track quality signal separately from operational signal** — separate `agent.abandoned` (operational: stuck/orphaned) from `agent.rejected` (quality: work was bad).
5. **Fix model §7 staleness** — update to reflect that rework infrastructure exists but is unused due to friction.

### Evidence Quality

- **Primary:** Event data from `~/.orch/events.jsonl` (actual system behavior)
- **Primary:** Source code read of `rework_cmd.go`, `abandon_cmd.go`, `complete_cmd.go`, `coordination.go`, `auto_complete.go`, `review_done.go`
- **Primary:** All 11 abandon reasons examined and categorized
