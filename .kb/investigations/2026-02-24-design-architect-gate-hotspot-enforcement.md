# Architect: Design Enforcement for Investigation → Architect → Implementation Sequence

**Date:** 2026-02-24
**Phase:** Complete
**Triggered by:** orch-go-1187
**Related:** orch-go-1177 (investigation), orch-go-1182 (reverted impl), orch-go-1183 (reverted impl), orch-go-1184 (architect that caught violations)

---

## Design Question

How should orch-go enforce the investigation → architect → implementation sequence for hotspot areas, so that workers cannot jump from investigation findings directly to code changes without architectural review?

## Problem Framing

### Context: What went wrong

The orch-go-1182/1183 incident:
1. **Investigation** (orch-go-1177) correctly found root cause: `runSpawnClaude` skips `AtomicSpawnPhase2`
2. **Worker** (orch-go-1182) went directly to implementation — added tmux liveness check, violating the two-lane decision
3. **Second worker** (orch-go-1183) added a 10s TTL cache to fix the regression — deepened the violation
4. **Architect** (orch-go-1184) finally caught both violations, recommended a completely different approach (phase-based liveness)

The hotspot gate fired on orch-go-1182 but was overridden with `--force-hotspot`. Cost: 3 spawn cycles, a regression, and reverted work.

### Success Criteria

A good design:
1. **Prevents the orch-go-1182 failure mode** — feature-impl can't spawn in hotspot areas without architect review
2. **Preserves escape hatches** — critical infrastructure work can still proceed in emergencies
3. **Low friction for non-hotspot work** — 90%+ of spawns should not be affected
4. **Infrastructure-enforced** — not dependent on orchestrator remembering to route correctly
5. **Works for both manual and daemon-driven spawns**

### Constraints

- **"Skills own domain behavior, spawn owns orchestration infrastructure"** — enforcement belongs in spawn gates, not skill templates
- **"Gate Over Remind"** — advisory guidance is insufficient; needs hard gates
- **"Infrastructure Over Instruction"** — code/tools enforce behavior, instructions don't
- **"Escape Hatches"** — critical paths need independent secondary paths
- **User Interaction Model** — CLI flags are for orchestrator programmatic use, not Dylan typing
- **Existing infrastructure** — spawn gates, hotspot detection, beads issue system all exist and work

### Scope

**IN:** Spawn gate modifications, investigation skill follow-up routing, daemon hotspot awareness, --force-hotspot requirements
**OUT:** New hotspot detection signals, completion gate changes, coaching plugin changes

---

## Exploration: Decision Forks

### Fork 1: Where should primary enforcement live?

**Options:**
- A: **Spawn gate only** — strengthen --force-hotspot to require architect reference
- B: **Investigation completion gate** — orch complete for investigations auto-creates architect issues when hotspot files are in scope
- C: **Multi-layer** — spawn gate (primary) + investigation skill guidance (advisory) + daemon routing (gap closure)

**Substrate says:**
- Principle: "Gate Over Remind" → Hard gate required, not just advisory
- Principle: "Infrastructure Over Instruction" → Code enforcement, not prompt guidance
- Decision: "Skills own domain behavior, spawn owns orchestration infrastructure" → Primary enforcement in spawn infrastructure
- Model: Completion verification operates through gates organized by level → Pattern supports gate-based enforcement

**Recommendation:** Option C (Multi-layer) because:
- Spawn gate is the hard enforcement (principles demand it)
- Investigation skill guidance prevents the situation from reaching the gate in the first place
- Daemon routing closes a gap the spawn gate alone can't address (daemon-driven spawns skip the hotspot gate)

**Trade-off accepted:** More implementation surface area than option A, but option A alone leaves the daemon gap open.

### Fork 2: What should --force-hotspot require?

**Options:**
- A: **Architect issue reference** — `--force-hotspot --architect-ref orch-go-XXXX` where the referenced issue is type=architect and status=closed
- B: **Remove --force-hotspot entirely** — all hotspot implementation must go through architect first (no bypass)
- C: **Mandatory reason** — `--force-hotspot --reason "emergency fix for production outage"`
- D: **Keep as-is** — rely on other layers for enforcement

**Substrate says:**
- Principle: "Escape Hatches" → Critical paths need independent secondary paths → Removing --force-hotspot violates this
- Principle: "Gate Over Remind" → A mere reason string is a reminder, not a gate → Option C insufficient
- Evidence: orch-go-1182 used --force-hotspot with no accountability → Option D failed in practice

**Recommendation:** Option A (Architect issue reference) because:
- Preserves escape hatch (--force-hotspot still exists)
- Adds verifiable accountability (system checks if architect issue exists and is closed)
- Architect spawns are always available (Claude instances, not human bottleneck)
- Follows existing pattern: gates that verify preconditions (like verification gate checking Tier 1 work)

**Trade-off accepted:** Slightly more friction on legitimate hotspot overrides. Mitigated by: architect spawns complete in ~30 minutes.

**When this would change:** If architect spawns become unavailable (all LLM providers down simultaneously — extremely unlikely).

### Fork 3: Should daemon-driven spawns check hotspots?

**Options:**
- A: **Yes, add hotspot check to daemon** — daemon checks if issue targets hotspot files and prefers architect over feature-impl
- B: **No, rely on triage** — triage already happened, daemon trusts it
- C: **Partial** — daemon checks hotspots but only warns (doesn't block), emits event for orchestrator review

**Substrate says:**
- Principle: "Gate Over Remind" → Warn-only insufficient → Against option C
- Evidence: Daemon currently silently skips hotspot gate (`pkg/spawn/gates/hotspot.go:50-53`) → Gap exists
- Daemon already defaults bugs to architect → Pattern of skill routing in daemon exists
- But: daemon processes `triage:ready` issues that have already been reviewed by orchestrator → Triage should catch hotspot routing

**Recommendation:** Option A (Add hotspot check) because:
- The daemon's "triage already happened" assumption doesn't hold for hotspot routing — triage checks issue validity, not architectural appropriateness of the inferred skill
- The daemon already has skill inference logic that can incorporate hotspot awareness
- Feature/task issues always infer to `feature-impl` regardless of target area — this is the gap

**Implementation approach:** In `InferSkillWithContext()`, if issue targets files in a hotspot area AND inferred skill is `feature-impl`, escalate to `architect` instead. This mirrors the existing bug → architect mapping.

**Trade-off accepted:** Daemon needs hotspot detection dependency (currently isolated to cmd/orch). May need to expose hotspot check as a pkg-level function.

### Fork 4: Should the investigation skill be modified?

**Options:**
- A: **Add hotspot awareness to investigation skill** — investigation detects when findings affect hotspot files, recommends architect in "Next" section
- B: **Keep investigation skill generic** — routing is infrastructure's job, not the skill's
- C: **Add skill template variable** — spawn context injects hotspot info, investigation skill uses it

**Substrate says:**
- Decision: "Skills own domain behavior, spawn owns orchestration infrastructure" → Option B seems correct at first
- BUT: The investigation skill's "Next" section recommends follow-up actions → This IS domain behavior (what the investigation found, what should happen next)
- Principle: "Session amnesia" → Investigation agent has context about what it found that spawn infrastructure doesn't → Investigation is uniquely positioned to recommend architect

**Recommendation:** Option C (Spawn context injection) because:
- Skills own domain behavior — the investigation skill should react to hotspot context
- Spawn owns infrastructure — the spawn command injects hotspot status into SPAWN_CONTEXT.md
- Investigation agent sees the flag and includes hotspot-aware routing in its "Next" recommendation
- No skill template changes needed — the hotspot context is injected by spawn infrastructure

**Implementation approach:**
1. `orch spawn` detects hotspot status for the task area
2. Adds `HOTSPOT_AREA: true` and `HOTSPOT_FILES: [list]` to SPAWN_CONTEXT.md
3. Investigation skill (already instrumented) sees this and recommends: "Findings affect hotspot files — spawn architect before implementation"

**Trade-off accepted:** Adds spawn context complexity. But this follows the existing pattern of spawn injecting context (skill content, beads context, server config).

---

## Synthesis: Recommended Approach

### Three-Layer Enforcement

**Layer 1: Spawn Gate Enhancement (Hard Enforcement)**

Modify `pkg/spawn/gates/hotspot.go:CheckHotspot()`:

When `--force-hotspot` is passed AND the skill is a blocking skill (feature-impl, systematic-debugging):
1. Require `--architect-ref <issue-id>` flag
2. Verify the referenced issue exists (via `bd show`)
3. Verify the issue type includes architect skill OR is type=architect
4. Verify the issue is status=closed (architect completed review)
5. If any check fails → block spawn with descriptive error

**Error messages:**
- Missing flag: `"--force-hotspot requires --architect-ref <issue-id> to prove architect reviewed the area. Spawn architect first, then reference its issue."`
- Issue not found: `"--architect-ref orch-go-XXXX: issue not found"`
- Not an architect issue: `"--architect-ref orch-go-XXXX: not an architect issue (type={type})"`
- Not closed: `"--architect-ref orch-go-XXXX: architect review not complete (status={status})"`

**File targets:**
- `pkg/spawn/gates/hotspot.go` — add architect reference verification to CheckHotspot()
- `cmd/orch/spawn_cmd.go` — add `--architect-ref` flag definition
- `pkg/orch/extraction.go` — pass architect-ref through RunPreFlightChecks()

**Layer 2: Daemon Hotspot Routing (Gap Closure)**

Modify daemon skill inference to check hotspot status:

When daemon infers skill for a feature/task issue:
1. Run hotspot check for the issue's task description
2. If task targets hotspot files AND inferred skill is feature-impl → escalate to architect
3. Log the escalation: "Daemon: escalated {issue-id} to architect (hotspot area)"

**File targets:**
- `pkg/daemon/skill_inference.go` — add `InferSkillWithHotspotCheck()` or modify `InferSkill()` to accept hotspot context
- `cmd/orch/daemon.go` — pass hotspot checker to skill inference

**Layer 3: Spawn Context Hotspot Injection (Advisory)**

When `orch spawn` runs for any skill:
1. Run hotspot check (already runs)
2. If hotspot detected, inject `HOTSPOT_AREA: true` and `HOTSPOT_FILES: [list]` into SPAWN_CONTEXT.md
3. Investigation agents see this and include hotspot-aware routing in their "Next" recommendation

**File targets:**
- `pkg/spawn/context.go` — add hotspot fields to SpawnConfig
- `cmd/orch/spawn_cmd.go` — pass hotspot result to context generation

---

## Implementation Plan

### Phase 1: Spawn Gate Enhancement (Layer 1)

**Priority: Highest — addresses the direct failure mode**

1. Add `--architect-ref` flag to spawn command
2. Modify `CheckHotspot()` to accept and verify architect reference
3. Add `bd show` integration to verify architect issue status
4. Add tests for all verification paths (missing ref, wrong type, not closed, valid ref)

**Estimated scope:** ~100 lines of gate logic + ~50 lines of flag plumbing + ~100 lines of tests

### Phase 2: Daemon Routing (Layer 2)

**Priority: Medium — closes gap for automated spawns**

1. Expose hotspot check as pkg-level function (currently in cmd/orch)
2. Add hotspot context to daemon skill inference
3. Add tests for daemon skill escalation behavior

**Estimated scope:** ~50 lines of inference logic + ~30 lines of integration + ~80 lines of tests

### Phase 3: Spawn Context Injection (Layer 3)

**Priority: Lower — advisory, supports correct routing**

1. Add hotspot fields to SpawnConfig
2. Inject into SPAWN_CONTEXT.md template
3. No skill template changes needed (agents read SPAWN_CONTEXT)

**Estimated scope:** ~20 lines of context generation + ~10 lines of template

---

## Acceptance Criteria

1. `orch spawn --bypass-triage --force-hotspot feature-impl "task"` → ERROR: requires --architect-ref
2. `orch spawn --bypass-triage --force-hotspot --architect-ref orch-go-1184 feature-impl "task"` → succeeds (1184 is closed architect)
3. `orch spawn --bypass-triage --force-hotspot --architect-ref orch-go-1182 feature-impl "task"` → ERROR: not an architect issue
4. Daemon infers architect (not feature-impl) when issue targets hotspot files
5. Investigation SPAWN_CONTEXT.md includes hotspot info when targeting hotspot area

## Out of Scope

- Changing hotspot detection thresholds (800/1500 are well-calibrated)
- Modifying completion accretion gate (already works correctly)
- Coaching plugin changes (working but different enforcement layer)
- Changing skill templates directly (spawn owns infrastructure)

---

## Recommendations

⭐ **RECOMMENDED:** Three-Layer Enforcement with Architect-Gated Override

- **Why:** Addresses the direct failure mode (--force-hotspot too easy) while closing the daemon gap and providing advisory routing. Follows "Gate Over Remind" and "Infrastructure Over Instruction" principles.
- **Trade-off:** More implementation surface than single-layer approach, but single-layer leaves daemon gap open.
- **Expected outcome:** The orch-go-1182 failure mode becomes impossible without an architect having reviewed the area first.

**Alternative: Spawn Gate Only (Layer 1 only)**
- **Pros:** Simplest to implement, addresses the direct failure mode
- **Cons:** Daemon gap remains open; no advisory routing for investigations
- **When to choose:** If implementation bandwidth is limited, Layer 1 alone still prevents the direct failure mode

**Alternative: Remove --force-hotspot entirely**
- **Pros:** Simplest possible enforcement; no way to bypass
- **Cons:** Violates "Escape Hatches" principle; removes ability to handle emergencies
- **When to choose:** Never (violates architectural principle)

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves recurring violations (3+ spawn cycles wasted on orch-go-1182/1183)
- This decision establishes constraints future agents might violate
- Future spawns targeting hotspot files will hit this gate

**Suggested blocks keywords:**
- "hotspot enforcement"
- "force-hotspot"
- "architect gate"
- "investigation implementation sequence"
- "accretion enforcement"
