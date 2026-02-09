# Investigation: Design Proof Carrying Verification Specs

## Summary (D.E.K.N.)

**Delta:** Proof-carrying verification should be added as a new contract layer that complements existing completion gates, with dual artifact placement and completion-time enforcement.

**Evidence:** Current verification is evidence-heuristic based over beads comments and SYNTHESIS parsing, while spawn/completion flows already support workspace artifacts and completion hooks.

**Knowledge:** The lowest-risk migration is to make a workspace `VERIFICATION_SPEC.yaml` canonical, mirror a compact digest into beads comments, and let `orch complete`/`orch verify --batch` execute contracts.

**Next:** Adopt decision `2026-02-08-proof-carrying-verification-specs.md` and implement in staged rollout (advisory -> required by skill/tier).

**Authority:** architectural - This crosses spawn templates, worker behavior, verification gates, and batch orchestration.

---

**Question:** How should proof-carrying verification specs be designed (placement, schema, and emission path) so overnight verification can run deterministic checks without manual synthesis interpretation?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** OpenCode agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-02-08-proof-carrying-verification-specs.md
**Extracted-From:** N/A

## Findings

### Finding 1: Existing gates infer evidence from text patterns, not executable intent

**Evidence:** Test evidence and visual verification parse comments with regex and heuristics; they do not carry command-level executable contracts.

**Source:** `pkg/verify/test_evidence.go`, `pkg/verify/visual.go`, `.kb/models/completion-verification.md`

**Significance:** Current gates catch missing proof, but cannot directly power batch replay as "run these exact checks".

---

### Finding 2: Completion flow already has natural insertion points for contract execution

**Evidence:** `verifyCompletion()` runs centralized verification, and post-verify gates already process synthesis/probes/discovered work.

**Source:** `cmd/orch/complete_gates.go`

**Significance:** Proof-carrying contracts can be integrated without inventing a separate lifecycle.

---

### Finding 3: Spawn template ownership and tier system can pre-seed valid contracts

**Evidence:** Spawn context template and SYNTHESIS template are centrally controlled by orch-go; full vs light tier behavior is already explicit and enforced.

**Source:** `pkg/spawn/context_template.go`, `.kb/decisions/2025-12-22-template-ownership-model.md`

**Significance:** We can safely introduce pre-populated verification spec skeletons at spawn-time and avoid "blank page" quality drift.

---

## Synthesis

1. **Complement, not replace (initially)** - Proof-carrying verification should augment Phase/Evidence/Approval gates first, then gradually replace heuristic-only checks where contract coverage is high.
2. **Dual placement by temporal model** - Canonical contract belongs in workspace (operational execution), with a compact mirror in beads comments (persistent audit and fallback).
3. **Emission belongs in worker protocol + completion enforcement** - Worker guidance should require authoring/updating the spec; completion must validate/execute it.

**Answer to Investigation Question:** Adopt a dual-artifact proof-carrying design: worker emits `VERIFICATION_SPEC.yaml` in workspace, completion validates and executes it, and completion posts a digest comment in beads. Use a minimal schema based on tiered methods (`cli_smoke`, `integration`, `browser`, `manual`, `static`) with explicit command/expectation tuples so batch verification can execute deterministic closures.

## Structured Uncertainty

**What's tested:**
- ✅ `kb create` contract mismatch was tested; `--defect-class` flag is unsupported in current CLI.
- ✅ Current verification implementation was checked in primary source code (`test_evidence.go`, `visual.go`, `complete_gates.go`).
- ✅ Spawn template ownership and tier behavior were verified in template source.

**What's untested:**
- ⚠️ Runtime performance of executing proof specs in large overnight batches.
- ⚠️ Human ergonomics of authoring specs across all skills.
- ⚠️ False-negative rates for strict expectation schema in real sessions.

**What would change this:**
- If pilot batch runs show unacceptable flake/overhead, schema/execution strictness needs adjustment.
- If worker authoring quality is poor, stronger spawn prefill and linting becomes mandatory.
- If beads mirror becomes noisy, persist only hash + summary and keep full spec workspace-only.

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add canonical workspace `VERIFICATION_SPEC.yaml` with completion execution and beads digest mirror | architectural | Crosses worker prompt, verification engine, and completion UX |
| Keep existing gates as fallback during migration | architectural | Changes quality-control boundary and rollout risk |
| Add spawn-time spec skeleton by skill/tier | implementation | Template-level change within existing spawn ownership |

### Recommended Approach

**Proof-Carrying Verification Contracts (Dual Placement)** - Use a canonical workspace spec executed by orchestrator tooling, plus a compact beads mirror for permanence.

**Why this approach:**
- Converts verification from inferred prose to explicit executable contracts.
- Preserves current safety gates while enabling fast batch replay.
- Aligns with existing artifact ownership and completion lifecycle.

**Implementation sequence:**
1. Define schema and parser with strict validation.
2. Add spawn skeleton + worker-base protocol requirement.
3. Add completion execution + digest emission.
4. Add `orch verify --batch` contract runner and reporting.
5. Enforce required-contract policy by skill/tier after grace period.

## References

**Files Examined:**
- `cmd/orch/complete_gates.go` - Existing completion integration points.
- `pkg/verify/test_evidence.go` - Test-evidence gate behavior and limits.
- `pkg/verify/visual.go` - Visual/approval gate behavior and limits.
- `pkg/spawn/context_template.go` - Spawn-time template ownership and deliverables.
- `.kb/models/completion-verification.md` - Existing architecture model.

**Commands Run:**
```bash
pwd && orch phase orch-go-21492 Planning "Designing proof-carrying verification spec decisions"
kb create investigation design-proof-carrying-verification-specs --defect-class integration-mismatch
kb create investigation design/design-proof-carrying-verification-specs
orch phase orch-go-21492 Implementing "Analyzed verification gates and spawn template constraints; drafting decision record"
```

## Investigation History

**2026-02-08 10:xx:** Investigation started
- Initial question: placement/schema/emission design for proof-carrying verification specs.
- Context: reduce human verification bottleneck for overnight batch closures.

**2026-02-08 10:xx:** Primary-source verification complete
- Confirmed current gates are heuristic evidence checks and identified integration seams.

**2026-02-08 10:xx:** Investigation completed
- Status: Complete
- Key outcome: Architectural decision documented with schema, placement, integration, and migration path.
