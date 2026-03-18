# Probe: Architectural Defect Intelligence — Decay Verification

**Date:** 2026-03-18
**Model:** architectural-defect-intelligence
**Purpose:** Verify model claims against current codebase after 15 days without probing
**Method:** Code exploration + cross-reference with decisions and implementations

---

## Claims Verified (Still Accurate)

### 1. Four hotspot detection types exist
**Claim:** `orch hotspot` has 4 types: fix-density, bloat-size, investigation-cluster, coupling-cluster.
**Status:** CONFIRMED
**Evidence:** All 4 implemented in `cmd/orch/hotspot.go`, `cmd/orch/hotspot_analysis.go`, `cmd/orch/hotspot_coupling.go` with test coverage.

### 2. Defect class taxonomy has 7 named classes
**Claim:** 7 classes (0-7) documented in `.kb/models/defect-class-taxonomy/model.md`
**Status:** CONFIRMED
**Evidence:** Model file exists, all 7 classes present and unchanged.

### 3. Daemon architect escalation is implemented
**Claim:** Daemon routes feature-impl/systematic-debugging to architect when targeting hotspot files.
**Status:** CONFIRMED
**Evidence:** `pkg/daemon/architect_escalation.go` — `CheckArchitectEscalation()` at line 78, integrated in `pkg/daemon/coordination.go:85`.

### 4. Spawn context hotspot injection works
**Claim:** Hotspot info injected into SPAWN_CONTEXT.md.
**Status:** CONFIRMED
**Evidence:** `cmd/orch/spawn_cmd.go:476-478` populates `HotspotArea`, `HotspotFiles`, `HotspotDefectClasses` fields. Template renders them in `pkg/spawn/context.go:156-158`.

### 5. Coupling scores are computed
**Claim:** Git co-change frequency produces coupling scores for cross-layer concepts.
**Status:** CONFIRMED
**Evidence:** `cmd/orch/hotspot_coupling.go:354` — formula: `layerCount x fileCount x avgCoChangeFrequency`. Scores >=15 become hotspots.

### 6. All referenced artifacts exist
**Claim:** Referenced investigations, decisions, and models are present.
**Status:** CONFIRMED
**Evidence:** All 5 referenced files verified present in `.kb/`.

---

## Claims Stale or Incorrect

### 7. Spawn gate BLOCKS spawns to hotspot files
**Claim (model line 117):** "Block spawn to files >1500 lines unless architect-reviewed"
**Actual:** Gates are advisory-only. They warn and emit events but NEVER block.
**Evidence:** `pkg/spawn/gates/hotspot.go:39-69` — comment: "Advisory only — emits warnings and events but never blocks." Decision: `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` — 2-week probe showed 100% bypass rate on blocks (55 fires, 2 blocks, both bypassed). Hotspot reduction (12->3 files, 75%) driven by daemon extraction cascades triggered by gate events, not by blocking.
**Severity:** HIGH — model describes enforcement mechanism that no longer exists.

### 8. `orch doctor --defect-scan` is "designed but not implemented"
**Claim (model line 126):** Listed under "Designed but Not Implemented"
**Actual:** Fully implemented.
**Evidence:** `cmd/orch/doctor_defect_scan.go` — 650+ line scanner detecting Class 2 (multi-backend blindness) and Class 5 (contradictory authority signals). Wired into `cmd/orch/doctor.go:144-146`.
**Severity:** MEDIUM — model understates current capabilities.

### 9. Predictive layer "doesn't exist as tooling yet"
**Claim (model line 85):** "This layer doesn't exist as tooling yet."
**Actual:** Partially implemented via defect class injection into spawn warnings.
**Evidence:** `cmd/orch/hotspot_spawn.go:269` — `DefectClassesForHotspots()` maps hotspot file paths to likely defect classes using keyword matching. Classes are embedded in spawn warning box (lines 226-241). This IS automated spatial x typological intersection, contradicting the claim that "no tooling computes spatial x typological automatically" (line 132).
**Severity:** MEDIUM — model's core "gap" claim is partially outdated.

### 10. "The predictive intersection exists only as this model"
**Claim (model line 132):** "no tooling computes spatial x typological automatically"
**Actual:** Spawn context now injects `HotspotDefectClasses` automatically, which IS spatial x typological intersection computed at spawn time.
**Severity:** MEDIUM — the gap is smaller than described.

---

## Open Questions Updated

| Original Question | Current Status |
|---|---|
| Does fixing upstream class reduce downstream instances? | Still open — no intervention experiment conducted |
| False positive rate for coupling hotspot detection? | Still open — no calibration study |
| Can predictive intersection be automated in `orch spawn`? | PARTIALLY ANSWERED — defect class injection exists but is keyword-based, not evidence-driven |
| Is there a fifth layer (temporal)? | Still open |

---

## Verdict

**Model accuracy: ~70%** — Core framework (4 layers, 7 classes) is solid. But the enforcement and gap sections are stale: gates changed from blocking to advisory (2026-03-17), defect-scan was implemented, and partial predictive tooling exists. The model overstates the gap between current state and predictive capability.

**Recommended model updates:**
1. Correct enforcement table: spawn gate is advisory, not blocking
2. Move `orch doctor --defect-scan` from "designed" to "implemented"
3. Update Layer 4 description to acknowledge partial implementation via `DefectClassesForHotspots()`
4. Update "The Gap" section to reflect smaller delta
