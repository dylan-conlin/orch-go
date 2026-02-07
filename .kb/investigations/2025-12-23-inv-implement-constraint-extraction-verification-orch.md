<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented constraint extraction and verification for orch complete - skills can now declare required outputs that are verified at completion time.

**Evidence:** All 21 verify package tests pass, including 11 new constraint-specific tests.

**Knowledge:** Constraints are embedded as HTML comments in SPAWN_CONTEXT.md via skillc, extracted using regex, and verified as glob patterns against the project directory.

**Next:** Close - implementation complete. Skills can now use `outputs.required` in skill.yaml to enforce deliverable creation.

**Confidence:** High (90%) - Tested extraction, pattern matching, and integration with existing verification flow.

---

# Investigation: Implement Constraint Extraction Verification Orch

**Question:** How should orch complete extract and verify skill constraints from SPAWN_CONTEXT.md?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-feat-implement-constraint-extraction-23dec
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete
**Confidence:** High (90%)

**Extracted-From:** skillc/.kb/investigations/2025-12-23-inv-design-executable-skill-constraints-skillc.md

---

## Findings

### Finding 1: Constraint Block Format

**Evidence:** skillc embeds constraints as HTML comment block in compiled SKILL.md:
```
<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file with findings -->
<!-- optional: .kb/decisions/{date}-*.md | Promoted decision -->
<!-- /SKILL-CONSTRAINTS -->
```

**Source:** skillc/.kb/investigations/2025-12-23-inv-implement-executable-skill-constraints-layer.md

**Significance:** Simple regex extraction is sufficient - no complex parsing needed.

---

### Finding 2: Variable Substitution

**Evidence:** Patterns use `{date}`, `{workspace}`, `{beads}` variables that should match any valid value. For glob matching, these are converted to `*` wildcards.

**Source:** Design investigation specified variable semantics

**Significance:** PatternToGlob function handles all known variables consistently.

---

### Finding 3: Integration with Existing Verification

**Evidence:** `VerifyCompletionWithTier` already provides structured verification results with errors and warnings. Added `VerifyCompletionFull` that extends this with constraint checking.

**Source:** pkg/verify/check.go:335-383

**Significance:** Clean integration - constraint failures appear alongside phase/synthesis failures.

---

## Synthesis

**Key Insights:**

1. **Extraction is straightforward** - HTML comment block is well-defined, regex parsing works reliably.

2. **Glob matching is sufficient** - filepath.Glob handles the patterns after variable substitution.

3. **Required vs optional semantics** - Required failures block completion, optional failures are warnings only.

**Answer to Investigation Question:**

Implemented in pkg/verify/constraint.go with:
- `ExtractConstraints(workspacePath)` - parses SPAWN_CONTEXT.md for constraint block
- `VerifyConstraints(constraints, projectDir)` - checks patterns match files
- `VerifyCompletionFull(beadsID, workspace, projectDir, tier)` - combines with existing verification

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation matches design spec, all tests pass, and clean integration with existing code.

**What's certain:**

- Extraction correctly parses constraint block from SPAWN_CONTEXT.md
- Pattern-to-glob conversion handles known variables ({date}, {workspace}, {beads})
- Integration with VerifyCompletion preserves existing behavior

**What's uncertain:**

- Real-world skill hasn't used constraints yet (skillc implementation is separate)
- Edge cases with malformed constraint patterns not fully explored

**What would increase confidence to Very High (95%+):**

- End-to-end test with a skill that has constraints defined
- Production usage validating constraint messages are helpful

---

## Implementation Details

**Files created:**
- `pkg/verify/constraint.go` - Constraint extraction and verification
- `pkg/verify/constraint_test.go` - 11 tests for constraint functionality

**Files modified:**
- `pkg/verify/check.go` - Added VerifyCompletionFull function
- `cmd/orch/main.go` - Updated complete command to use VerifyCompletionFull
- `cmd/orch/review.go` - Updated review command to use VerifyCompletionFull

**Key functions:**
- `PatternToGlob(pattern)` - Converts skill patterns to glob patterns
- `ExtractConstraints(workspacePath)` - Parses SPAWN_CONTEXT.md
- `VerifyConstraints(constraints, projectDir)` - Checks files exist
- `VerifyCompletionFull(...)` - Combined verification with constraints

---

## References

**Files Examined:**
- skillc/.kb/investigations/2025-12-23-inv-design-executable-skill-constraints-skillc.md
- skillc/.kb/investigations/2025-12-23-inv-implement-executable-skill-constraints-layer.md
- pkg/verify/check.go - Existing verification framework
- cmd/orch/main.go - Complete command implementation

**Related Artifacts:**
- **Design:** skillc/.kb/investigations/2025-12-23-inv-design-executable-skill-constraints-skillc.md
- **Layer 1 (skillc):** skillc/.kb/investigations/2025-12-23-inv-implement-executable-skill-constraints-layer.md

---

## Investigation History

**2025-12-23 ~21:00:** Investigation started
- Initial question: Implement feat-002 from skillc constraint design
- Context: Skills should declare required outputs that orch complete verifies

**2025-12-23 ~21:30:** Implementation complete
- Created constraint.go with extraction and verification
- Added VerifyCompletionFull to integrate with existing verification
- All 21 verify tests passing

**2025-12-23 ~21:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Constraint verification integrated into orch complete
