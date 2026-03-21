---
status: proposed
blocks:
  - keywords:
      - accretion gate
      - accretion enforcement
      - hotspot blocking
      - gate blocking vs signaling
    patterns:
      - "**/spawn/gates/hotspot*"
      - "**/verify/accretion*"
      - "**/verify/accretion_precommit*"
---

## Summary (D.E.K.N.)

**Delta:** Convert all accretion gates from blocking to advisory (warn + emit event, never block). Daemon extraction response unchanged.

**Evidence:** 2-week probe (orch-go-74rhs) shows 55 gate firings, 2 blocks, both bypassed in seconds (100% bypass rate). No quality difference between enforced/bypassed cohorts. Hotspot reduction (12→3 files, 75%) driven entirely by daemon extraction cascades triggered by gate *events*, not by gate *blocks*.

**Knowledge:** Gates work through signaling, not blocking. The blocking path adds friction agents route around instantly, producing zero behavioral change. The event emission path triggers daemon responses that produce the actual structural improvement. Blocking is ceremony; signaling is mechanism.

**Next:** Implement in orchestrator direct session (all target files are governance-protected). Update CLAUDE.md Accretion Boundaries section to reflect advisory model.

---

# Decision: Accretion Gates Advisory, Not Blocking

**Date:** 2026-03-17
**Status:** Proposed
**Enforcement:** gate
**Deciders:** Dylan (via orchestrator)
**Evidence Issues:** orch-go-74rhs (accretion gate effectiveness probe), orch-go-00r9c (gate effectiveness query)
**Source Probe:** `.kb/models/harness-engineering/probes/2026-03-17-probe-pre-commit-accretion-gate-2-week-effectiveness.md`
**Amends:** `2026-02-26-three-layer-hotspot-enforcement.md` (Layer 1 and Layer 0 change from blocking to advisory)

---

## Context

The three-layer hotspot enforcement system (decision 2026-02-26) was designed with "Gate Over Remind" as a driving principle. After 2 weeks of measurement:

| Gate | Fires | Blocks | Bypasses | Bypass Rate | Behavioral Effect |
|------|-------|--------|----------|-------------|-------------------|
| Spawn hotspot (Layer 1) | — | 2 | 2 | 100% | None observed |
| Pre-commit accretion (Layer 0) | 55 | 2 | 2 | 100% | None observed |
| Completion accretion (Layer 2) | — | — | — | — | Warnings only (already advisory for pre-existing bloat) |

**What actually reduced hotspots (12→3 files, 75%):** Gate event emission → daemon extraction triggers → extraction cascades. The blocking was bypassed instantly; the signaling drove the real work.

**Key probe finding:** "The gate's primary mechanism is indirect pressure, not direct blocking... The gate's value is in its warnings and visibility, not its blocking power."

---

## Options Considered

### Option A: Keep Blocking (Status Quo)
- **Pros:** "Gate Over Remind" principle; audit trail of bypasses; friction may have unmeasured indirect value
- **Cons:** 100% bypass rate means blocking is purely ceremonial; agents learn to add `FORCE_ACCRETION=1` or `--force-hotspot` reflexively; bypass mechanics add code complexity

### Option B: Signal-Only (Remove Gates Entirely)
- **Pros:** Zero friction; simplest
- **Cons:** Loses event emission that drives daemon extraction; loses warning visibility

### Option C: Advisory — Warn + Emit Event, Never Block
- **Pros:** Preserves the mechanism that works (event emission → daemon response); removes the mechanism that doesn't work (blocking); reduces code complexity (remove bypass logic); honest about what the system actually does
- **Cons:** Loses the theoretical value of "forcing agents to consciously bypass"

---

## Decision

**Chosen:** Option C — Advisory (warn + emit event, never block)

### Rationale

"Gate Over Remind" was the right instinct but the measurement shows blocking doesn't gate — agents route around it in seconds. The actual gate is the daemon extraction cascade, which is triggered by events, not blocks. Converting to advisory aligns the code with what the system already does in practice.

### What Changes

**Layer 0 — Pre-commit accretion (`pkg/verify/accretion_precommit.go`):**
- `CheckStagedAccretion()`: When agent-caused bloat pushes file over 1500, move from `BlockedFiles` to `WarningFiles` instead. Set `Passed = true` always.
- Remove `FORCE_ACCRETION=1` bypass logic from `cmd/orch/precommit_cmd.go` (no longer needed — gate never blocks)
- Keep warning output and event emission unchanged

**Layer 1 — Spawn hotspot gate (`pkg/spawn/gates/hotspot.go`):**
- `CheckHotspot()`: When CRITICAL hotspot + blocking skill, emit warning + event but return `nil` error instead of blocking error
- Remove `--force-hotspot`, `--architect-ref`, `--reason` flag handling (no longer needed)
- Remove `ArchitectVerifier`, `ArchitectFinder` interfaces (no longer needed for bypass logic)
- Keep `LogHotspotBypass()` renamed to `LogHotspotAdvisory()` for event emission
- Keep `IsBlockingSkill()` for routing decisions (daemon still uses it for escalation)

**Layer 2 — Daemon escalation (`pkg/daemon/architect_escalation.go`):**
- No change. Daemon still escalates feature-impl to architect when targeting hotspot areas. This is routing, not blocking.

**Layer 3 — Spawn context injection:**
- No change. Advisory info still injected into SPAWN_CONTEXT.md.

**Completion accretion (`pkg/verify/accretion.go`):**
- `VerifyAccretionForCompletion()`: When agent-caused bloat pushes file over 1500, downgrade from error to warning. Set `Passed = true` always.
- Keep warning messages and file info unchanged

**CLAUDE.md:**
- Update Accretion Boundaries section: "gates warn (advisory) instead of blocking"
- Remove references to `--force-hotspot`, `--architect-ref`, `FORCE_ACCRETION=1`

**Spawn command (`cmd/orch/spawn_cmd.go`):**
- Remove `--force-hotspot`, `--architect-ref`, `--reason` flags
- Simplify `CheckHotspot()` call (fewer parameters)

### What Stays the Same

- Event emission (spawn.gate_decision, spawn.hotspot_bypassed → spawn.hotspot_advisory)
- Daemon extraction cascades triggered by events
- Warning messages displayed to agents
- Hotspot detection and thresholds (800/1500 lines)
- Layer 2 daemon escalation routing
- Layer 3 spawn context injection
- `IsBlockingSkill()` for daemon routing decisions

---

## Structured Uncertainty

**What's tested:**
- 2-week measurement showing 100% bypass rate on blocks
- No quality difference between enforced/bypassed cohorts
- Hotspot reduction driven by daemon extraction, not blocking

**What's untested:**
- Whether removing the *possibility* of blocking changes agent behavior (unlikely given reflexive bypassing, but unmeasured)
- Long-term effect — could hotspots return without the blocking deterrent? (Mitigated: daemon extraction is the actual mechanism and is unchanged)

**What would change this:**
- If hotspot count trends back upward after conversion to advisory, re-evaluate
- If a new agent population emerges that doesn't reflexively bypass (would make blocking meaningful again)

---

## Implementation Notes

**All target files are governance-protected** (`pkg/spawn/gates/*`, `pkg/verify/*`). Must be implemented in an orchestrator direct session.

### Exact Code Changes

#### 1. `pkg/verify/accretion_precommit.go` — CheckStagedAccretion()
Lines 89-98: Move agent-caused bloat from `BlockedFiles` to `WarningFiles`, remove `result.Passed = false`:
```go
// Before (blocking):
result.Passed = false
result.BlockedFiles = append(result.BlockedFiles, StagedFileInfo{...})

// After (advisory):
result.WarningFiles = append(result.WarningFiles, StagedFileInfo{
    Path:      file,
    Lines:     stagedLines,
    NetDelta:  netDelta,
    Threshold: AccretionCriticalThreshold,
})
```

#### 2. `pkg/verify/accretion.go` — VerifyAccretionForCompletion()
Lines 137-146: Downgrade from error to warning:
```go
// Before (blocking):
result.Passed = false
result.Errors = append(result.Errors, fmt.Sprintf("CRITICAL accretion: ..."))

// After (advisory):
result.Warnings = append(result.Warnings, fmt.Sprintf("CRITICAL accretion (advisory): ..."))
```

#### 3. `pkg/spawn/gates/hotspot.go` — CheckHotspot()
Lines 76-123: Replace blocking error with warning + event emission:
```go
// Before: returns error
return result, fmt.Errorf("CRITICAL hotspot: ...")

// After: emits advisory event, returns nil
LogHotspotAdvisory(skillName, task, result.CriticalFiles)
return result, nil
```
Remove `forceHotspot`, `architectRef`, `reason` parameters and all associated logic.

#### 4. `cmd/orch/precommit_cmd.go`
Remove `FORCE_ACCRETION=1` bypass check. Gate always passes now.

#### 5. `cmd/orch/spawn_cmd.go`
Remove `--force-hotspot`, `--architect-ref`, `--reason` flag definitions and plumbing.

---

## Consequences

**Positive:**
- Code matches measured reality (gates signal, don't block)
- Removes ~50 lines of bypass logic that was always exercised
- Removes 3 CLI flags that exist only to work around blocking
- Agents stop learning reflexive bypass patterns
- Event-driven daemon extraction (the mechanism that works) is unchanged

**Risks:**
- If blocking had unmeasured deterrent value, removing it could increase hotspot frequency (mitigated: daemon extraction is the actual mechanism)
- Amends a decision that was carefully designed around "Gate Over Remind" (mitigated: measurement shows gates signal effectively without blocking)

## Evidence

### Source Probe
- `.kb/models/harness-engineering/probes/2026-03-17-probe-pre-commit-accretion-gate-2-week-effectiveness.md`
  - "Gate block rate: 3.6% with 100% bypass"
  - "The gate has never successfully prevented a commit that an agent wanted to make"
  - "The gate works, but through extraction pressure, not through blocking"

### Prior Decision (Amended)
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Three-layer design this amends

## Auto-Linked Investigations

- .kb/investigations/2026-03-05-inv-design-orchestrator-coordination-plans-persist.md
