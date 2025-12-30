<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added "Original Symptom Validation" gate to feature-impl skill's Self-Review phase to prevent agents from claiming completion without validating against the original issue.

**Evidence:** Modified SKILL.md.template at ~/.claude/skills/worker/feature-impl/.skillc/, compiled with skillc, verified gate appears in deployed SKILL.md.

**Knowledge:** Agents can rationalize partial fixes by testing different modes/flags than the original issue; explicit validation gate with scope redefinition warning addresses this.

**Next:** None - implementation complete. Future bug fixes will require explicit validation against original symptom before completion.

---

# Investigation: Add Original Symptom Validation Gate

**Question:** How to add an "Original Symptom Validation" gate to feature-impl skill to prevent agents claiming completion without validating against original issue?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent orch-go-svc0
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** .kb/investigations/2025-12-29-inv-root-cause-analysis-agent-orch.md

---

## Findings

### Finding 1: Skill uses skillc for compilation

**Evidence:** The feature-impl skill has a `.skillc/` directory with `SKILL.md.template` as the source file. The compiled output is copied to `SKILL.md` in the parent directory.

**Source:** `~/.claude/skills/worker/feature-impl/.skillc/` directory structure

**Significance:** Edits must be made to SKILL.md.template, not SKILL.md directly. The build process uses `skillc build` and the output must be manually copied to the deployment location.

---

### Finding 2: Self-Review phase lacked original symptom validation

**Evidence:** The existing Self-Review phase had checklists for:
- Anti-Pattern Detection
- Security Review
- Commit Hygiene
- Test Coverage
- Documentation
- Deliverables
- Integration Wiring
- Demo Data Ban
- Scope Verification
- Discovered Work

But NO explicit check to validate against the original issue/symptom.

**Source:** `~/.claude/skills/worker/feature-impl/.skillc/SKILL.md.template` lines 247-299

**Significance:** This gap allowed agent orch-go-yw1q to pass all validation gates while leaving the original text mode performance issue unresolved.

---

### Finding 3: Implementation added comprehensive validation gate

**Evidence:** New gate includes:
1. Mandatory re-run of exact original command
2. Beads comment documentation requirement
3. Scope Redefinition Warning for different modes/flags
4. Checklist with 4 validation items
5. Skip documentation requirement for non-bug-fix work
6. Reference to root cause investigation

**Source:** Added to SKILL.md.template after "Discovered Work" section

**Significance:** The gate addresses all failure modes identified in the root cause analysis - untested claims, scope redefinition, and estimate-as-measurement patterns.

---

## Synthesis

**Key Insights:**

1. **Process gap was explicit** - The investigation clearly identified that existing validation phases focused on "tests pass" not "original problem solved".

2. **Solution is lightweight** - Adding a checklist subsection to existing Self-Review phase requires no architectural changes.

3. **Scope redefinition warning is key** - The gate explicitly calls out the pattern where agents test a different mode/flag than the original issue.

**Answer to Investigation Question:**

Added "Original Symptom Validation" gate by:
1. Editing `~/.claude/skills/worker/feature-impl/.skillc/SKILL.md.template`
2. Adding new subsection after "Discovered Work" in Self-Review phase
3. Compiling with `skillc build`
4. Copying output to deployment location

The gate is mandatory for bug fixes, with explicit skip-documentation required for pure features/refactoring.

---

## Structured Uncertainty

**What's tested:**

- ✅ Gate added to SKILL.md.template (verified: read file after edit)
- ✅ skillc build succeeds (verified: ran command, output 3434 tokens)
- ✅ Gate appears in deployed SKILL.md (verified: grep "Original Symptom Validation")

**What's untested:**

- ⚠️ Whether agents will actually follow this gate (requires real bug fix spawns)
- ⚠️ Whether scope redefinition warning is strong enough to prevent rationalization
- ⚠️ Whether skip documentation is sufficient for non-bug-fix work

**What would change this:**

- Finding would need revision if agents still bypass validation despite gate
- Implementation would need strengthening if rationalization patterns continue

---

## Implementation Recommendations

N/A - Implementation is complete. This investigation documents the implementation itself.

---

## References

**Files Examined:**
- `~/.claude/skills/worker/feature-impl/.skillc/SKILL.md.template` - Source file for skill
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Deployed skill file
- `.kb/investigations/2025-12-29-inv-root-cause-analysis-agent-orch.md` - Root cause analysis

**Commands Run:**
```bash
# Create investigation file
kb create investigation add-original-symptom-validation-gate

# Compile skill
~/go/bin/skillc build ~/.claude/skills/worker/feature-impl/.skillc

# Copy to deployment location
cp ~/.claude/skills/worker/feature-impl/.skillc/SKILL.md ~/.claude/skills/worker/feature-impl/SKILL.md

# Verify gate is present
grep -A 30 "Original Symptom Validation" ~/.claude/skills/worker/feature-impl/SKILL.md
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-29-inv-root-cause-analysis-agent-orch.md - Root cause analysis that recommended this gate

---

## Self-Review

- [x] Real implementation performed (not just planning)
- [x] Verification that gate is in deployed output
- [x] Question answered (how to add the gate)
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-30 15:30:** Investigation started
- Initial question: Add Original Symptom Validation gate to feature-impl skill
- Context: Root cause analysis found agents bypass validation by testing different modes

**2025-12-30 15:36:** Implementation completed
- Added gate to SKILL.md.template
- Compiled with skillc
- Verified in deployed SKILL.md

**2025-12-30 15:38:** Investigation completed
- Status: Complete
- Key outcome: Original Symptom Validation gate added to Self-Review phase, mandatory for bug fixes
