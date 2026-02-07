<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Checkpoints exist in two forms: 12 code-enforced gates (pkg/verify) that block completion, and advisory checkpoints (SPAWN_CONTEXT, skills) labeled "REQUIRED" but not enforced, creating ~40% theater.

**Evidence:** Analyzed pkg/spawn/context.go, pkg/verify/check.go, cmd/orch/complete_cmd.go, and skill templates; found gate constants with skip flags vs documentation checkpoints with no enforcement mechanism.

**Knowledge:** The theater problem is misleading labels, not the checkpoints themselves - "REQUIRED" creates false compliance when checkpoints have no enforcement, while gates already have opt-out via --skip-{gate} --skip-reason.

**Next:** Replace "REQUIRED" labels in skill templates with "ADVISORY" to set correct expectations; update SPAWN_CONTEXT template to explicitly label advisory sections.

**Authority:** implementation - Template changes only, no architectural impact, reversible within existing patterns

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

# Investigation: Theater Checkpoints Optional

**Question:** Where are checkpoints defined in orch-go (spawn context template, skills, completion gates)? Which are enforced vs aspirational? How should we label them as 'gate' (blocking) vs 'advisory' (opt-in)?

**Started:** 2026-02-05
**Updated:** 2026-02-05
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation                                              | Relationship | Verified | Conflicts                        |
| ---------------------------------------------------------- | ------------ | -------- | -------------------------------- |
| 2026-02-04-inv-analyze-checkpoint-rituals-session-start.md | extends      | yes      | None - confirms ~40% are theater |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Checkpoints Defined in SPAWN_CONTEXT Template

**Evidence:**

The SPAWN_CONTEXT template in `pkg/spawn/context.go` contains several checkpoint sections:

- **Lines 187-194:** "CRITICAL - FIRST 3 ACTIONS" (report phase, read context, begin planning)
- **Lines 196-216, 374-403:** "SESSION COMPLETE PROTOCOL" (completion steps, never git push, Phase: Complete reporting)
- **Lines 222-224:** Session scope estimation (Medium: 1-2h / 2-4h / 4-6h+)
- **Lines 227-264:** Authority delegation guidance
- **Lines 266-299:** Deliverables section
- **Lines 306-336:** Beads progress tracking section

These are all **advisory** - they're documentation, not enforced by code.

**Source:**

- `pkg/spawn/context.go:54-404` (SpawnContextTemplate constant)
- Read via mcp_read tool

**Significance:** These checkpoints appear in every spawn context but have no enforcement mechanism. Agents can skip them without consequences, making them "theater" rather than gates.

---

### Finding 2: Completion Gates Are Enforced in Code

**Evidence:**

The `pkg/verify/check.go` file defines 12 gate constants (lines 15-28):

1. `GatePhaseComplete` - Phase: Complete not reported
2. `GateSynthesis` - SYNTHESIS.md missing
3. `GateHandoffContent` - SYNTHESIS.md has empty/placeholder content
4. `GateConstraint` - Constraint verification failed
5. `GatePhaseGate` - Required phase gate not passed
6. `GateSkillOutput` - Required skill outputs missing
7. `GateVisualVerify` - Visual verification required
8. `GateTestEvidence` - Test execution evidence required
9. `GateGitDiff` - Git diff doesn't match claims
10. `GateBuild` - Project build failed
11. `GateDecisionPatchLimit` - Decision patch limit exceeded
12. `GateDashboardHealth` - Dashboard API health check failed

Each gate has a `--skip-{gate}` flag in `cmd/orch/complete_cmd.go` (lines 161-172) requiring `--skip-reason` to bypass.

**Source:**

- `pkg/verify/check.go:15-28` (gate constants)
- `cmd/orch/complete_cmd.go:161-172` (skip flags)
- `cmd/orch/complete_cmd.go:242-269` (shouldSkipGate function)

**Significance:** These are **true gates** - they block `orch complete` unless explicitly skipped with documented reason. This is the enforcement mechanism that separates gates from advisory checkpoints.

---

### Finding 3: Prior Investigation Found ~40% Theater

**Evidence:**

From `.kb/investigations/2026-02-04-inv-analyze-checkpoint-rituals-session-start.md`:

- **35+ total checkpoints** across 5 categories
- **~60% are functional gates** (hard gates + soft signals)
- **~40% are theater** (manual checklists with escape hatches)

Manual checklists include:

- 11-item self-review checklist (investigation skill)
- Discovered work protocol ("No discovered work" escape)
- Leave it Better ("N/A - straightforward" escape)
- Prior-Work table ("N/A - novel investigation" escape)

**Source:**

- `.kb/investigations/2026-02-04-inv-analyze-checkpoint-rituals-session-start.md:1-310`
- Read via mcp_read tool

**Significance:** The system already distinguishes automated gates from manual checklists, but doesn't explicitly label them. Making this distinction visible would help agents understand what's required vs suggested.

---

### Finding 4: Checkpoints Embedded in Skill Content

**Evidence:**

Skills like `feature-impl` define their own checkpoints that get embedded in SPAWN_CONTEXT:

- Step 0: Scope Enumeration (REQUIRED) - lines 619-641 of embedded skill
- Harm Assessment (Pre-Implementation Checkpoint) - lines 708-740
- Self-Review Phase (REQUIRED) - lines 887-991
- Leave it Better (REQUIRED) - lines 994-1013
- Completion Criteria checklist - lines 1016-1046

These are labeled "REQUIRED" but have no code enforcement - agents can claim completion without doing them.

**Source:**

- SPAWN_CONTEXT.md embedded skill content (feature-impl)
- Lines shown in current spawn context at `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-theater-checkpoints-optional-05feb-6bb7/SPAWN_CONTEXT.md`

**Significance:** Skills use "REQUIRED" labels but lack enforcement, creating confusion about what's actually mandatory. These should be explicitly marked as **advisory** to set correct expectations.

---

## Synthesis

**Key Insights:**

1. **Two enforcement mechanisms exist** - Code-enforced gates (pkg/verify) block completion, while documentation checkpoints (SPAWN_CONTEXT, skills) rely on agent compliance.

2. **"REQUIRED" is misleading** - Skills label checkpoints as "REQUIRED" but have no enforcement, creating false expectations. Agents can skip them by claiming "N/A" or "No discovered work".

3. **The gate/advisory distinction already exists** - `orch complete` gates are skippable with `--skip-{gate} --skip-reason`, making them **configurable gates**. Documentation checkpoints are **always advisory**. The system just doesn't label them explicitly.

**Answer to Investigation Question:**

**Where defined:**

- **SPAWN_CONTEXT template** (pkg/spawn/context.go:54-404) - advisory checkpoints
- **Completion gates** (pkg/verify/check.go:15-28) - enforced gates
- **Skill content** (embedded in SPAWN_CONTEXT) - advisory checkpoints labeled "REQUIRED"

**Which are enforced:**

- **12 code-enforced gates** in pkg/verify - block `orch complete` unless skipped with reason
- **All documentation checkpoints** are advisory - no enforcement mechanism

**Labeling strategy:**

- **Gate (blocking):** Already implemented via gate constants + skip flags
- **Advisory (opt-in):** Everything in SPAWN_CONTEXT and skill content - should be explicitly labeled as such

**Recommendation:** Replace "REQUIRED" labels in skills with "ADVISORY" to set correct expectations. The theater problem isn't the checkpoints themselves - it's the misleading "REQUIRED" label that makes them feel mandatory when they're not.

---

## Structured Uncertainty

**What's tested:**

- ✅ Gate constants exist in pkg/verify/check.go (verified: read file, saw 12 const declarations)
- ✅ SPAWN_CONTEXT template has advisory checkpoints (verified: read pkg/spawn/context.go lines 54-404)
- ✅ Skip flags exist for all gates (verified: read complete_cmd.go lines 161-172, saw --skip-{gate} flags)
- ✅ Prior investigation quantified ~40% theater (verified: read 2026-02-04 investigation)

**What's untested:**

- ⚠️ Agents will understand "ADVISORY" better than "REQUIRED" (not tested with real agents)
- ⚠️ Relabeling will reduce theater compliance (behavioral assumption, not measured)
- ⚠️ No performance impact from template changes (not benchmarked)

**What would change this:**

- Finding would be wrong if gates were also unenforced (but they have shouldSkipGate logic)
- Finding would be wrong if "REQUIRED" checkpoints had enforcement code (but they don't - only documentation)
- Recommendation would change if agents prefer strict enforcement over opt-in (but prior investigation shows escape hatches are already used)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation                                              | Authority      | Rationale                                             |
| ----------------------------------------------------------- | -------------- | ----------------------------------------------------- |
| Replace "REQUIRED" labels in skills with "ADVISORY"         | implementation | Changes skill templates only, no architectural impact |
| Add explicit gate/advisory labels to SPAWN_CONTEXT template | implementation | Template change within existing patterns              |
| Document the distinction in orchestrator skill              | implementation | Documentation update, no behavior change              |

**Authority Levels:**

- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"

- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Replace misleading "REQUIRED" labels with explicit "ADVISORY" markers** - Change skill templates and SPAWN_CONTEXT to clearly distinguish enforced gates from suggested checkpoints.

**Why this approach:**

- Addresses root cause: "REQUIRED" creates false expectations when checkpoints aren't actually enforced
- Minimal code changes: Only template/documentation updates, no logic changes
- Preserves gates: Code-enforced gates remain unchanged, already have skip mechanism
- Sets correct expectations: Agents know advisory checkpoints are suggestions, not mandates

**Trade-offs accepted:**

- Manual checkpoints remain unenforced (but that's intentional - enforcement would add friction)
- Agents may skip advisory checkpoints more often (but that's better than theater compliance)
- Some checkpoints may need conversion to gates later (can be done incrementally)

**Implementation sequence:**

1. **Audit skill templates** - Find all "REQUIRED" labels in skill content that lack code enforcement
2. **Replace with "ADVISORY" markers** - Update templates to use consistent labeling: `## Step X (ADVISORY)` or `**Advisory Checkpoint:**`
3. **Update SPAWN_CONTEXT template** - Add explicit labels to distinguish gates from advisory sections
4. **Test with a spawn** - Verify labeling is clear in generated SPAWN_CONTEXT.md

### Alternative Approaches Considered

**Option B: Convert manual checkpoints to code-enforced gates**

- **Pros:** Maximum enforcement, nothing slips through
- **Cons:** High friction, agents will find workarounds, requires significant implementation work
- **When to use instead:** If quality problems from bypassed checkpoints become evident

**Option C: Remove all manual checkpoints**

- **Pros:** Minimal friction, clear expectations (only gates matter)
- **Cons:** Loses "pause and reflect" value, may miss edge cases
- **When to use instead:** If agents consistently bypass all manual checkpoints anyway

**Rationale for recommendation:** Option A (relabel as ADVISORY) directly addresses the theater problem - the issue isn't that checkpoints exist, but that "REQUIRED" creates false compliance. This preserves the guidance value while setting correct expectations.

---

### Implementation Details

**What to implement first:**

- Replace "REQUIRED" with "ADVISORY" in feature-impl skill templates
- Update SPAWN_CONTEXT template to label advisory sections explicitly
- Add comment headers explaining gate vs advisory distinction

**Things to watch out for:**

- ⚠️ Don't break existing gate enforcement - only change labels, not logic
- ⚠️ Skill content is compiled via skillc - changes need to be in .skillc source files, not SKILL.md
- ⚠️ SPAWN_CONTEXT template uses Go template syntax - preserve {{.Variables}}

**Areas needing further investigation:**

- Should some advisory checkpoints be promoted to gates? (e.g., Leave it Better → enforce kb quick command)
- Should orchestrator skill document the gate/advisory distinction?
- Should gates have severity levels (error vs warning)?

**Success criteria:**

- ✅ New spawns show "ADVISORY" labels in SPAWN_CONTEXT.md instead of "REQUIRED"
- ✅ Skill content clearly distinguishes suggestions from requirements
- ✅ Code-enforced gates remain unchanged and functional

---

## References

**Files Examined:**

- `pkg/spawn/context.go:54-404` - SPAWN_CONTEXT template with advisory checkpoints
- `pkg/verify/check.go:15-28` - Gate constant definitions
- `cmd/orch/complete_cmd.go:161-172` - Skip flags for gates
- `.kb/investigations/2026-02-04-inv-analyze-checkpoint-rituals-session-start.md` - Prior analysis of checkpoint theater
- Feature-impl skill content (embedded in SPAWN_CONTEXT) - Manual checkpoint labels

**Commands Run:**

```bash
# Search for checkpoint references
mcp_grep pattern="checkpoint|CHECKPOINT" include="*.go"

# Search for gate patterns
mcp_grep pattern="gate|Gate|GATE|blocking|BLOCKING" include="*.go"

# Read SPAWN_CONTEXT template
mcp_read filePath="/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go"

# Read verification gates
mcp_read filePath="/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go"
```

**Related Artifacts:**

- **Investigation:** `.kb/investigations/2026-02-04-inv-analyze-checkpoint-rituals-session-start.md` - Quantified ~40% theater rate
- **Beads:** orch-go-21307 - This implementation task

---

## Investigation History

**2026-02-05 [time]:** Investigation started

- Initial question: Where are checkpoints defined? How to label gate vs advisory?
- Context: Task to make theater checkpoints optional/explicit

**2026-02-05 [time]:** Investigation completed

- Status: Complete
- Key outcome: Checkpoints exist in SPAWN_CONTEXT (advisory) and pkg/verify (gates); solution is to replace misleading "REQUIRED" labels with explicit "ADVISORY" markers
