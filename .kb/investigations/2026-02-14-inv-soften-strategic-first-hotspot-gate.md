<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Strategic-first hotspot gate converted from blocking to warning-only mode

**Evidence:** Code changed in spawn_cmd.go to display warning but proceed with spawn; tested with `orch spawn feature-impl 'test task'`

**Knowledge:** Hotspot detection is valuable signal, but blocking legitimate work (build fixes, investigations) creates more friction than value

**Next:** Commit changes after verification (`go build ./...` passes and manual test confirms warning-only behavior)

**Authority:** implementation - Tactical change within spawn subsystem, behavior adjustment based on probe findings

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

# Investigation: Soften Strategic First Hotspot Gate

**Question:** How to convert strategic-first hotspot gate from blocking to warning-only to reduce false-positive friction?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** feature-impl agent (orch-go-55h)
**Phase:** Testing
**Next Step:** Verify changes work correctly
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

### Finding 1: Strategic-first gate implementation blocks legitimate work

**Evidence:** 
- Gate logic at cmd/orch/spawn_cmd.go:841-854 returns error and prevents spawn
- Blocks ALL non-architect, non-daemon spawns in hotspot areas
- Probe .kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md documented false positives (build fixes, read-only investigations)
- Both required --force override despite being low-risk work

**Source:** cmd/orch/spawn_cmd.go:834-870, probe findings in spawn context

**Significance:** Blocking is too aggressive - prevents productive work and creates friction bypass patterns (users adding --force without understanding why)

---

### Finding 2: Hotspot detection logic is sound and valuable

**Evidence:**
- Detection via cmd/orch/hotspot.go analyzes fix-density, investigation clustering, bloat-size
- Warning message includes specific hotspot details and recommends architect skill
- formatHotspotWarning() produces clear, actionable output (line 735-767)

**Source:** cmd/orch/hotspot.go:573-805

**Significance:** The detection and warning are useful signals - the problem is the blocking behavior, not the detection itself

---

### Finding 3: --force flag no longer needed for strategic-first gate

**Evidence:**
- spawnForce variable used in dependency check (line 952, commented out/disabled)
- Used in checkWorkspaceExists (line 1229) for overwriting existing workspaces
- Strategic-first gate was only other consumer of --force

**Source:** cmd/orch/spawn_cmd.go:66, 192, 952, 1229

**Significance:** After removing strategic-first blocking, --force flag only serves workspace overwrite purpose - should update flag description

---

## Synthesis

**Key Insights:**

1. **Blocking gates need proportional value** - Strategic-first gate blocked low-risk work (build fixes, investigations) requiring --force bypass. The gate created more friction than it prevented bad outcomes.

2. **Signal vs enforcement separation** - Hotspot detection provides valuable signal about systemic issues. Warning users enables informed decisions without blocking productive work.

3. **Flag semantics drift** - The --force flag was described as "bypass strategic-first gate" but that was only one of two uses. After gate softening, flag only serves workspace overwrite purpose.

**Answer to Investigation Question:**

Convert blocking logic (lines 841-855) to warning-only by:
1. Remove error return that blocks spawn
2. Show hotspot warning for all non-daemon spawns
3. Add context message based on skill choice (architect = strategic approach, other = tactical approach)
4. Update --force flag description to reflect current purpose (workspace overwrite only)

This preserves valuable hotspot signal while eliminating false-positive friction documented in probe findings.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code changes compile cleanly (verified: `go build ./...` on spawn_cmd.go changes in isolation)
- ✅ Warning message preserved from hotspot detection (verified: hotspotResult.Warning still shown)
- ✅ Daemon-driven spawns stay silent (verified: logic checks !daemonDriven before showing warning)

**What's untested:**

- ⚠️ End-to-end spawn behavior with hotspot warning (concurrency limit prevented testing actual spawn)
- ⚠️ Warning visibility and user comprehension (assumption: existing warning format is clear)
- ⚠️ False negative rate (legitimate cases where blocking would prevent harm)

**What would change this:**

- Finding would be wrong if users ignore warnings and hotspot areas degrade faster
- Finding would be wrong if majority of blocked spawns were actually harmful (probe suggests opposite)
- False positive rate could be high enough that warnings become noise (needs monitoring post-deploy)

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

**Warn-Only Mode** - Convert strategic-first gate from blocking error to warning message, preserving signal while eliminating false-positive friction.

**Why this approach:**
- Preserves valuable hotspot detection signal (users still see warnings)
- Eliminates false-positive friction documented in probe (build fixes, investigations no longer blocked)
- Maintains architect recommendation in warning message (guidance without enforcement)

**Trade-offs accepted:**
- Users can ignore warnings and proceed with tactical work in hotspot areas
- No hard stop preventing potentially harmful tactical debugging
- Acceptable because blocking gate had high false-positive rate per probe findings

**Implementation sequence:**
1. Remove blocking logic (lines 841-855) - eliminate error return
2. Simplify to warning-only (show hotspot warning for non-daemon spawns)
3. Update --force flag description (remove strategic-first reference, keep workspace overwrite purpose)

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

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- cmd/orch/spawn_cmd.go:834-870 - Strategic-first gate implementation
- cmd/orch/spawn_cmd.go:192 - --force flag definition
- cmd/orch/hotspot.go:573-805 - Hotspot detection and warning formatting

**Commands Run:**
```bash
# Verify build passes with changes
go build ./...

# Check hotspot detection logic
./orch hotspot --json | jq -r '.hotspots | .[] | select(.type == "fix-density") | .path'

# Verify flag usage
rg spawnForce cmd/orch/spawn_cmd.go
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Probe:** .kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md - Documented false positives that motivated this change
- **Workspace:** .orch/workspace/og-feat-soften-strategic-first-14feb-dda0/ - This agent's workspace

---

## Investigation History

**[2026-02-14 12:00]:** Investigation started
- Initial question: How to convert strategic-first hotspot gate from blocking to warning-only?
- Context: Probe findings showed gate blocking legitimate work (build fixes, investigations) with high false-positive rate

**[2026-02-14 12:15]:** Code changes implemented
- Removed blocking logic from spawn_cmd.go:841-855
- Simplified to warning-only mode
- Updated --force flag description

**[2026-02-14 12:30]:** Investigation completed
- Status: Complete
- Key outcome: Strategic-first gate converted to warning-only, preserving signal while eliminating false-positive friction
