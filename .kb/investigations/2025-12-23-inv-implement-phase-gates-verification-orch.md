<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented Phase Gates verification in `orch complete` - extracts `<!-- SKILL-PHASES -->` block from SPAWN_CONTEXT.md and verifies required phases were reported via beads comments.

**Evidence:** All 35 tests pass, including 6 new phase gates tests covering extraction, comment parsing, and verification.

**Knowledge:** Phase gates follow the same pattern as constraints (HTML comment blocks in SPAWN_CONTEXT.md), making implementation consistent and extensible.

**Next:** Skillc needs to embed `<!-- SKILL-PHASES -->` blocks in compiled skill output for this to be enforced.

**Confidence:** High (90%) - Tests comprehensive, but needs integration testing with skillc.

---

# Investigation: Implement Phase Gates Verification Orch

**Question:** How should `orch complete` verify phase progression from SPAWN_CONTEXT.md?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-feat-implement-phase-gates-23dec
**Phase:** Complete
**Next Step:** None - ready for skillc integration
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Phase Gates Follow Constraint Pattern

**Evidence:** The existing constraint system uses `<!-- SKILL-CONSTRAINTS -->` blocks embedded in SPAWN_CONTEXT.md. Phase gates use the same pattern with `<!-- SKILL-PHASES -->` blocks.

**Source:** 
- `/pkg/verify/constraint.go:47-116` - Constraint extraction and verification
- `/pkg/verify/phase_gates.go:32-85` - New phase extraction (same pattern)

**Significance:** Consistent architecture makes the system easier to extend and maintain. Skills can declare both constraints (file patterns) and phases (workflow stages).

---

### Finding 2: Phase Comments Already Parsed in check.go

**Evidence:** `ParsePhaseFromComments` in check.go already extracts the latest phase from beads comments using regex pattern `Phase:\s*(\w+)`.

**Source:** `/pkg/verify/check.go:60-82`

**Significance:** The new `ExtractReportedPhases` function reuses this regex pattern and extends it to track all reported phases (not just the latest), enabling sequence verification.

---

### Finding 3: Phase Gates Verification Integrated into VerifyCompletionFull

**Evidence:** `VerifyCompletionFull` now runs three verification checks in order:
1. Standard completion (Phase: Complete + SYNTHESIS.md)
2. Constraint verification (file patterns)
3. Phase gate verification (required phases in comments)

**Source:** `/pkg/verify/check.go:330-382`

**Significance:** Agents cannot complete if they skipped required phases, even if they reported Phase: Complete.

---

## Synthesis

**Key Insights:**

1. **Consistent Extension Pattern** - Phase gates follow the same HTML comment block pattern as constraints, making the system predictable for skill authors.

2. **Non-Breaking Change** - Skills without `<!-- SKILL-PHASES -->` blocks pass verification automatically (backward compatible).

3. **Declarative Enforcement** - Required phases are declared in skill definition, not procedural code. Enforcement happens at completion time.

**Answer to Investigation Question:**

`orch complete` now:
1. Extracts phases from `<!-- SKILL-PHASES -->` block in workspace's SPAWN_CONTEXT.md
2. Gets all beads comments for the issue
3. Parses reported phases from comments (Phase: X pattern)
4. Verifies all required phases were reported
5. Fails completion if required phases are missing

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Comprehensive unit tests pass. The implementation follows proven patterns from constraint verification.

**What's certain:**

- ✅ Phase extraction from SPAWN_CONTEXT.md works correctly
- ✅ Comment parsing handles various Phase: formats (dashes, case, etc.)
- ✅ Required vs optional phases respected in verification
- ✅ Integration with VerifyCompletionFull complete

**What's uncertain:**

- ⚠️ Skillc integration not tested (needs skillc to embed phase blocks)
- ⚠️ Real-world workflow with actual skills not tested

**What would increase confidence to Very High (95%+):**

- End-to-end test with skillc embedding phase declarations
- Production usage with feature-impl skill

---

## Implementation Recommendations

### Recommended Approach ⭐

**Layer 2 of Executable Skill Constraints** - Phase gates extend the constraint system to enforce workflow stages.

**Why this approach:**
- Reuses proven patterns from Layer 1 (structural constraints)
- Minimal new concepts for skill authors
- Declarative rather than procedural

**Trade-offs accepted:**
- Verification at completion time (not runtime)
- Requires skillc changes to embed phase blocks

**Implementation sequence:**
1. ✅ Phase extraction in pkg/verify/phase_gates.go
2. ✅ Integration in VerifyCompletionFull
3. ⏳ Skillc embedding (next step)

---

### Implementation Details

**What was implemented:**

1. `pkg/verify/phase_gates.go` - Phase extraction and verification logic
2. `pkg/verify/phase_gates_test.go` - Comprehensive tests
3. `pkg/verify/check.go` - Integration in VerifyCompletionFull

**SKILL-PHASES block format:**
```markdown
<!-- SKILL-PHASES -->
<!-- phase: investigation | required: false -->
<!-- phase: design | required: false -->
<!-- phase: implementation | required: true -->
<!-- phase: validation | required: true -->
<!-- phase: complete | required: true -->
<!-- /SKILL-PHASES -->
```

**Areas needing further investigation:**
- Skillc skill.yaml schema extension for phases
- skillc embedding of phase blocks in compiled output

**Success criteria:**
- ✅ `orch complete` fails if required phases missing (tested)
- ⏳ Skills can declare phases in skill.yaml (needs skillc)
- ⏳ End-to-end workflow verified (needs integration test)

---

## References

**Files Created/Modified:**
- `pkg/verify/phase_gates.go` - New file: Phase extraction and verification
- `pkg/verify/phase_gates_test.go` - New file: Comprehensive tests
- `pkg/verify/check.go` - Modified: Added phase gate verification to VerifyCompletionFull

**Related Artifacts:**
- **Design:** `~/Documents/personal/skillc/.kb/investigations/2025-12-23-inv-design-phase-gates-skillc-layer.md`

---

## Investigation History

**2025-12-23 ~21:00:** Investigation started
- Initial question: How to implement phase gates in orch complete
- Context: Layer 2 of executable skill constraints

**2025-12-23 ~21:30:** Implementation complete
- Created phase_gates.go and phase_gates_test.go
- Integrated into VerifyCompletionFull
- All tests passing

**2025-12-23 ~21:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Phase gates verification implemented, ready for skillc integration
