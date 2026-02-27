# Design: Skip-Flag Reason Tracking

**TLDR:** Add `--reason` requirement to the 4 override flags that lack it (`--bypass-triage`, `--force` on complete, `--force-hotspot`, `--no-track`), store reasons in events.jsonl, and extend `orch stats` with a new "Override Reasons" section. Follow the existing `--skip-reason` pattern from targeted completion gates.

**Status:** Active
**Phase:** Synthesis
**Date:** 2026-02-27
**Beads:** orch-go-f06i

## Problem Statement

Skip flags are pressure release valves that let operators override safety gates. Currently we track *counts* of overrides but not *why* they happened. Without reasons:
- Can't distinguish systemic issues (daemon unreliable → bypass-triage) from legitimate overrides (urgent fix)
- Can't identify which gates generate false-positive friction (the friction-bypass probe found 66% of test_evidence bypasses were "docs-only change")
- No feedback loop to improve gates — just accumulating bypass counts

### Success Criteria
1. Every safety-override flag requires a reason before proceeding
2. Reasons are persisted in events.jsonl alongside existing events
3. `orch stats` surfaces reason frequency to reveal patterns
4. Implementation follows the existing `--skip-reason` pattern (min 10 chars)

### Constraints
- Gate Over Remind principle: "Gates must be passable by the gated party" — reason requirement must not create unpassable friction
- Observation Infrastructure principle: "Every state transition emits an event. Observation gaps are P1 bugs."
- User Interaction Model: CLI flags are for orchestrator to pass programmatically, not for Dylan to type
- No UI changes (out of scope)
- No changes to when/whether skip flags are available (out of scope)

### Scope
**IN:** Reason capture on override flags, events.jsonl schema extension, orch stats reason reporting
**OUT:** Changing skip flag availability, UI changes, daemon behavior changes

## Current State Inventory

### Flags WITH reason tracking (already covered)
| Flag | Event Type | Reason Mechanism |
|------|-----------|-----------------|
| `--skip-*` (complete, 14 gates) | `verification.bypassed` | `--skip-reason` (min 10 chars) |
| `--bypass-verification` (spawn) | `spawn.verification_bypassed` | `--bypass-reason` |

### Flags WITHOUT reason tracking (the gap)
| Flag | Command | Event Emitted | Reason Field |
|------|---------|--------------|-------------|
| `--bypass-triage` | spawn | `spawn.triage_bypassed` (skill, task only) | **NONE** |
| `--force` | complete | embedded in `agent.completed` as `forced:true` | **NONE** |
| `--force-hotspot` | spawn | **NO EVENT AT ALL** | **NONE** |
| `--no-track` | spawn | embedded in `session.spawned` as `no_track:true` | **NONE** |

### Additional uncovered flags discovered during exploration
| Flag | Command | Event | Notes |
|------|---------|-------|-------|
| `--force` (workspace overwrite) | spawn | None | Low-risk operational flag |
| `--skip-gap-gate` | spawn | None | Only active with `--gate-on-gap` |
| `--skip-artifact-check` | spawn | Embedded in `session.spawned` | Pre-spawn artifact check |
| `rework --force` | rework | None | Issue/workspace checks |

### Stats gap
`verification.bypassed` events (from `--skip-*` flags) are logged to events.jsonl but **NOT consumed by `orch stats`**. Neither are `spawn.verification_bypassed` events. This means the existing reason-tracked bypasses aren't surfaced in stats either.

## Decision Forks

### Fork 1: Which flags get --reason?

**Options:**
- A: Only the 4 specified flags (--bypass-triage, --force on complete, --force-hotspot, --no-track)
- B: All 4 specified + the additional 4 uncovered flags (--force on spawn, --skip-gap-gate, --skip-artifact-check, rework --force)

**Substrate says:**
- Observation Infrastructure principle: "Every state transition emits an event. Observation gaps are P1 bugs." → Favors covering all
- Gate Over Remind: "Gates must be passable by the gated party" → Don't add friction to low-risk operational flags unnecessarily
- Scope definition: Task explicitly lists --bypass-triage, --force, --force-hotspot, --no-track

**RECOMMENDATION:** Option A — the 4 specified flags. The additional flags are either low-risk operational (workspace overwrite), conditional (--skip-gap-gate only with --gate-on-gap), or already somewhat covered (--skip-artifact-check is embedded in session.spawned). They can be added in a follow-up if patterns warrant it.

**Trade-off accepted:** Incomplete coverage of all override flags. When this would change: if stats show the uncovered flags are used frequently.

---

### Fork 2: Flag naming — --reason for all or per-flag?

**Options:**
- A: Single `--reason` flag that applies context-sensitively to whichever override flag is active
- B: Per-flag names: `--bypass-triage-reason`, `--force-reason`, `--hotspot-reason`, `--no-track-reason`
- C: Per-flag names matching existing patterns: `--bypass-reason` (already exists for verification), `--skip-reason` (already exists for gates)

**Substrate says:**
- Existing pattern: `--bypass-verification` uses `--bypass-reason`, `--skip-*` uses `--skip-reason`
- User Interaction Model: Orchestrator passes flags programmatically — ergonomics matter for CLI consistency, not memorability
- Evolve by Distinction: Each flag serves a different purpose in a different command — conflating them into one name risks ambiguity

**RECOMMENDATION:** Option A — single `--reason` flag per command. Reasoning:
- `orch spawn` already has `--bypass-reason` for one flag; adding `--reason` serves the other spawn flags
- `orch complete` already has `--skip-reason` for targeted gates; `--reason` serves `--force`
- Avoids proliferation of `--X-reason` flags (4 new flag names vs 1 per command)
- Context is unambiguous: if `--bypass-triage` is set, `--reason` obviously applies to the triage bypass

**Implementation detail:** `--reason` on spawn applies to `--bypass-triage`, `--force-hotspot`, and `--no-track` (whichever is active). If `--bypass-verification` is also set, it still uses its own `--bypass-reason`. On complete, `--reason` applies to `--force` (the `--skip-*` flags keep `--skip-reason`).

**Trade-off accepted:** Slight naming inconsistency with existing `--bypass-reason` and `--skip-reason`. When this would change: if users/agents report confusion about which reason flag applies to which override.

---

### Fork 3: Event schema — enrich existing or new event types?

**Options:**
- A: Add `reason` field to existing event data (e.g., add `reason` to `spawn.triage_bypassed` data, add `force_reason` to `agent.completed` data)
- B: Create new dedicated event types (e.g., `spawn.bypass_reason`, `completion.force_reason`)
- C: Both — enrich existing AND emit separate reason events

**Substrate says:**
- Observation Infrastructure: Observation gaps are P1 bugs — the fix is enriching existing events, not creating parallel tracking
- Existing pattern: `verification.bypassed` is a DEDICATED event type with `gate` + `reason` fields. `agent.completed` embeds `forced:true` and `gates_bypassed`. Both patterns coexist.
- Local-First: Simpler schema = easier to parse and maintain

**RECOMMENDATION:** Option A — enrich existing events with `reason` field. Specifically:
- `spawn.triage_bypassed` data: add `"reason": "..."` alongside existing `skill` and `task`
- `agent.completed` data: add `"force_reason": "..."` when `forced: true` (new field name avoids conflict with existing `reason` field which is the completion reason)
- New event `spawn.hotspot_bypassed`: add as new event type (currently NO event exists) with `skill`, `task`, `architect_ref`, `reason`
- `session.spawned` data: add `"no_track_reason": "..."` when `no_track: true`

**Why new event for hotspot:** hotspot bypass currently has zero observability (no event emitted at all). Creating a new event follows the same pattern as `spawn.triage_bypassed` and `spawn.verification_bypassed`.

**Trade-off accepted:** Mixing enrichment (triage, force, no-track) with a new event type (hotspot). This is pragmatic — the other flags already have events to enrich, hotspot doesn't.

---

### Fork 4: orch stats presentation — how to surface reason clusters?

**Options:**
- A: Simple frequency table — top N reasons per override flag type
- B: NLP/fuzzy clustering of similar reasons (group "docs-only" and "documentation change" together)
- C: Manual category tags — predefined reason categories users select from
- D: Exact string frequency + a regex-based grouping for known patterns

**Substrate says:**
- Reflection Before Action principle: "Build the process that surfaces patterns, not the solution to this instance"
- Evolve by Distinction: Start simple, add complexity when patterns recur
- The friction-bypass probe already showed that simple reason strings reveal patterns (66% "docs-only change")

**RECOMMENDATION:** Option A — simple frequency table. Reasons:
- Reasons are provided by AI orchestrators programmatically — they'll naturally use consistent phrasing
- Exact string frequency will reveal clusters immediately (same reason = same underlying issue)
- Fuzzy clustering adds complexity for marginal benefit given the programmatic input source
- Can evolve to Option D if frequency tables show near-duplicate strings

**Stats output design:**

```
🔓 OVERRIDE REASONS (skip flag usage)
  Spawn Overrides:
    bypass-triage:     12 events
      "urgent production fix"                     5 (42%)
      "daemon not running"                        4 (33%)
      "custom skill context needed"               3 (25%)
    force-hotspot:      3 events
      "architect reviewed in orch-go-abc1"        3 (100%)
    no-track:           8 events
      "testing spawn configuration"               5 (63%)
      "one-off exploration"                       3 (37%)

  Complete Overrides:
    force (deprecated):  2 events
      "agent died, partial work salvaged"         2 (100%)

  Targeted Skip Reasons (--skip-*):
    test_evidence:      18 events
      "docs-only change, no tests needed"        12 (67%)
      "tests run in CI, not locally"              4 (22%)
      "config change only"                        2 (11%)
    synthesis:           6 events
      "light-tier agent, synthesis not required"  6 (100%)
```

This consumes BOTH the new `--reason` data AND the existing `verification.bypassed` events that stats currently ignores.

---

### Fork 5: What about --force (complete) deprecation?

**Options:**
- A: Add --reason to --force despite deprecation
- B: Skip --force, only add --reason to non-deprecated flags
- C: Accelerate deprecation by making --force require --reason (increases migration pressure)

**Substrate says:**
- Coherence Over Patches: Don't invest in deprecated paths
- Existing pattern: --force already shows deprecation warning suggesting --skip-* + --skip-reason
- Friction-bypass probe: --force usage dropped from 72.8% to 16.7% after targeted skips were added

**RECOMMENDATION:** Option A — add --reason to --force. Despite deprecation:
- --force is still actively used (16.7% of completions per probe data)
- Adding --reason creates parity across all override flags (consistent mental model)
- The reason data will help identify remaining --force use cases, informing WHEN to remove it
- Low implementation cost (one flag, one field)

**Trade-off accepted:** Investment in deprecated code path. When this would change: when --force usage drops to <5%.

## Recommendations

### Recommended: Unified --reason flag with event enrichment

**What to build:**

1. **Add `--reason` flag to `orch spawn`** (string, min 10 chars)
   - Required when `--bypass-triage`, `--force-hotspot`, or `--no-track` is used
   - NOT required for `--bypass-verification` (which already has `--bypass-reason`)
   - Validation: if any of those 3 flags are set AND `--reason` is empty → error

2. **Add `--reason` flag to `orch complete`** (string, min 10 chars)
   - Required when `--force` is used
   - NOT required for `--skip-*` flags (which already have `--skip-reason`)
   - Validation: if `--force` is set AND `--reason` is empty → error

3. **Enrich existing events:**
   - `spawn.triage_bypassed` data: add `reason` field
   - `agent.completed` data: add `force_reason` field (when `forced: true`)
   - `session.spawned` data: add `no_track_reason` field (when `no_track: true`)

4. **New event type:**
   - `spawn.hotspot_bypassed` with fields: `skill`, `task`, `architect_ref`, `reason`, `critical_files`
   - Add constant `EventTypeHotspotBypassed` to pkg/events/logger.go

5. **Extend `orch stats` with Override Reasons section:**
   - Consume `spawn.triage_bypassed` reasons (new)
   - Consume `spawn.hotspot_bypassed` events (new)
   - Consume `session.spawned` with `no_track_reason` (new)
   - Consume `agent.completed` with `force_reason` (new)
   - Consume `verification.bypassed` events with reasons (EXISTING but not yet consumed)
   - Present as frequency table per override type

### File Targets

| File | Change |
|------|--------|
| `cmd/orch/spawn_cmd.go` | Add `--reason` flag, validation, pass to gates |
| `cmd/orch/complete_cmd.go` | Add `--reason` flag, validation, pass to event |
| `pkg/spawn/gates/triage.go` | Accept reason param, include in event |
| `pkg/spawn/gates/hotspot.go` | Emit new `spawn.hotspot_bypassed` event |
| `pkg/events/logger.go` | Add `EventTypeHotspotBypassed` constant |
| `pkg/spawn/backends/common.go` | Pass no_track_reason to session.spawned event |
| `cmd/orch/stats_cmd.go` | New Override Reasons section, consume verification.bypassed |

### Acceptance Criteria

- [ ] `orch spawn --bypass-triage investigation "test"` fails with "requires --reason"
- [ ] `orch spawn --bypass-triage --reason "urgent fix" investigation "test"` succeeds and emits `spawn.triage_bypassed` with reason
- [ ] `orch spawn --force-hotspot --architect-ref X --reason "reviewed" feature-impl "test"` emits `spawn.hotspot_bypassed` with reason
- [ ] `orch spawn --no-track --reason "testing config" investigation "test"` includes `no_track_reason` in `session.spawned`
- [ ] `orch complete X --force --reason "agent died"` includes `force_reason` in `agent.completed`
- [ ] `orch complete X --force` (no --reason) fails with "requires --reason"
- [ ] `orch stats` shows Override Reasons section when bypass events exist
- [ ] `orch stats` now consumes `verification.bypassed` events (closing existing gap)
- [ ] All reasons have min 10 char validation

### Out of Scope

- Changing when/whether skip flags are available
- UI changes to display reasons
- Daemon behavior changes
- Covering additional flags (--force on spawn workspace, --skip-gap-gate, --skip-artifact-check, rework --force)
- NLP/fuzzy clustering of reason strings

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:** This design resolves the observation gap for override flags and establishes the pattern for future override flags.

**Suggested blocks keywords:** "skip flag", "override reason", "bypass tracking", "events schema"

## Conclusion

The design follows existing patterns (`--skip-reason` on complete, `--bypass-reason` on spawn verification) and extends them to the 4 remaining uncovered override flags. The key insight is that 2 of the 4 flags already emit events but lack reason fields (enrich), 1 flag has no event at all (create), and 1 embeds its state in another event (enrich). Stats already ignores `verification.bypassed` events, so the stats extension closes both the new and existing gaps simultaneously.
