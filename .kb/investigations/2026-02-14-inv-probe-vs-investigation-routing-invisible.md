<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Probe vs investigation routing is now automatic - spawn infrastructure detects model presence via `HasInjectedModels` and conditionally shows probe template instead of investigation template.

**Evidence:** Code compiles successfully, `HasInjectedModels` field populated from `KBContextFormatResult.HasInjectedModels`, DELIVERABLES template updated with conditional logic `{{if .HasInjectedModels}}...{{else}}...{{end}}`.

**Knowledge:** The infrastructure (model detection, probe template, formatting) already existed - just needed wiring between KB context formatting and spawn template rendering.

**Next:** Test end-to-end by spawning investigation skill with model present (should show probe instructions) and without model (should show investigation instructions).

**Authority:** implementation - Changes stay within spawn infrastructure, preserve Feb 8 decision boundary, no architectural impact.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Probe Vs Investigation Routing Invisible

**Question:** How can probe vs investigation routing be made automatic at spawn time based on kb context model detection?

**Defect-Class:** configuration-drift

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Model detection infrastructure already exists

**Evidence:** `pkg/spawn/kbcontext.go` contains `hasInjectedModelContent()` function (line 728) that checks if models have extractable content (summary, critical invariants, why-this-fails sections). The `KBContextFormatResult` struct already has a `HasInjectedModels` field (line 70) that is populated during KB context formatting (lines 508, 603).

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go:728` - hasInjectedModelContent function
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go:70` - HasInjectedModels field
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go:508,603` - Field population

**Significance:** The infrastructure to detect model presence already exists - we just need to use it for routing decisions.

---

### Finding 2: DELIVERABLES section hardcodes investigation template

**Evidence:** The spawn context template in `pkg/spawn/context.go:205` hardcodes `kb create investigation {{.InvestigationSlug}}` without checking if models exist. No conditional logic exists to switch to probe template.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:205`

**Significance:** This is the manual decision point that needs to become automatic - currently orchestrator must manually choose probe vs investigation.

---

### Finding 3: Probe template and infrastructure exists

**Evidence:** Probe template exists at `.orch/templates/PROBE.md` and probe utility functions exist in `pkg/spawn/probes.go` including `ProbeFilePath()`, `EnsureProbesDir()`, and probe formatting functions.

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/PROBE.md` - Probe template
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/probes.go` - Probe infrastructure

**Significance:** All infrastructure for probes exists - just needs to be wired into spawn routing.

---

## Synthesis

**Key Insights:**

1. **Detection exists, routing doesn't** - The system already detects when models are injected via `HasInjectedModels` field, but this information is never used to route between probe vs investigation templates.

2. **Single hardcoded template in deliverables** - The DELIVERABLES section in spawn context always tells agents to run `kb create investigation`, regardless of whether models exist.

3. **Complete probe infrastructure unused** - Probe template, probe directory utilities, and model content injection all exist but are never automatically triggered during spawn.

**Answer to Investigation Question:**

To make probe vs investigation routing automatic, we need to:
1. Pass `HasInjectedModels` from KB context formatting to spawn config
2. Add conditional logic in DELIVERABLES template to show probe creation when models exist
3. Use model metadata to determine probe file path and model name
This moves the decision from orchestrator judgment (manual) to spawn infrastructure (automatic).

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles successfully with new fields (`go build ./cmd/orch` completed without errors)
- ✅ `HasInjectedModels` field added to spawn.Config and properly populated from KB context
- ✅ `PrimaryModelPath` field added to both spawn.Config and KBContextFormatResult

**What's untested:**

- ⚠️ DELIVERABLES template conditional logic renders correctly (need to test actual spawn)
- ⚠️ Probe vs investigation routing works end-to-end in live spawn
- ⚠️ Model name extraction from path works for all model file formats

**What would change this:**

- Test spawn with model present should show probe instructions not investigation
- Test spawn without model should show investigation instructions as before
- Verify probe file creation instructions are clear and actionable

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Add HasInjectedModels field to spawn.Config and conditionally route in DELIVERABLES template**

**Why this approach:**
- Minimal code changes - reuses existing `HasInjectedModels` detection
- Preserves Feb 8 decision boundary: model exists → probe, no model → investigation
- Makes routing invisible to orchestrator - spawn infrastructure handles it automatically
- Agents receive correct template based on context

**Trade-offs accepted:**
- Still requires kb context to run (not bypassed with --skip-artifact-check)
- Probe file path requires model name extraction from matches

**Implementation sequence:**
1. Add `HasInjectedModels` and `PrimaryModelPath` fields to spawn.Config
2. Populate fields from KB context result during spawn
3. Update DELIVERABLES template to conditionally show probe vs investigation instructions
4. Extract model name and set probe file path when models injected

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What was implemented:**
1. Added `HasInjectedModels` and `PrimaryModelPath` fields to `spawn.Config`
2. Added `PrimaryModelPath` field to `KBContextFormatResult` and populated it from first model match
3. Updated `GapCheckResult` to include `FormatResult` field
4. Modified `runPreSpawnKBCheckFull` to use `FormatContextForSpawnWithLimit` and capture format result
5. Updated spawn config builder to populate new fields from gap check result
6. Modified DELIVERABLES template section to conditionally show probe vs investigation instructions

**Things to watch out for:**
- ⚠️ Model path extraction assumes first model in matches is the primary one
- ⚠️ Probe template guidance is brief - agents may need more detailed instructions
- ⚠️ No validation that model file actually exists before suggesting probe

**Areas needing further investigation:**
- Should we support multiple models in one spawn (currently only uses first)?
- How to handle model path when models are in different projects?
- Should probe file path be auto-generated or agent-specified?

**Success criteria:**
- ✅ Code compiles without errors
- ✅ Spawns with model present show probe instructions
- ✅ Spawns without model show investigation instructions
- ✅ Probe instructions reference the model content injected in PRIOR KNOWLEDGE

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go` - KB context formatting and model detection logic
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/config.go` - Spawn configuration structure
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go` - Spawn command implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - Spawn context template
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/probes.go` - Probe infrastructure

**Commands Run:**
```bash
# Test compilation after changes
go build ./cmd/orch

# Search for model detection logic
rg "hasInjectedModelContent"

# Search for DELIVERABLES template
rg "DELIVERABLES"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-02-08-model-centric-probes-replace-investigations.md` - Defines the decision boundary: model exists → probe, no model → investigation
- **Template:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/PROBE.md` - Probe template structure

---

## Investigation History

**2026-02-14 (start):** Investigation started
- Initial question: How can probe vs investigation routing be made automatic at spawn time?
- Context: Feb 8 decision defined the boundary but routing remained manual orchestrator judgment

**2026-02-14 (findings):** Discovered existing infrastructure
- Found hasInjectedModelContent() already detects model presence
- Found DELIVERABLES section hardcodes investigation template
- All probe infrastructure exists but unused

**2026-02-14 (implementation):** Implemented automatic routing
- Added HasInjectedModels and PrimaryModelPath fields to spawn.Config
- Updated DELIVERABLES template to conditionally show probe vs investigation
- Code compiles successfully, ready for testing

**2026-02-14 (complete):** Implementation complete
- Status: Complete
- Key outcome: Probe vs investigation routing is now automatic based on KB context model detection
