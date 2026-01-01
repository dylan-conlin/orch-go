# Strengthen VerifySynthesis to Validate Content Against Primary Sources

**Status:** Complete
**Created:** 2026-01-01
**Agent:** og-feat-strengthen-verifysynthesis-validate-01jan
**Issue:** orch-go-54y7

## TLDR

VerifySynthesis only checks file existence and non-zero size, but should validate that SYNTHESIS.md claims are consistent with primary sources (git commits, test output in beads comments). This inverts the Evidence Hierarchy principle by treating artifact existence as truth.

## What I Tried

### 1. Analyzed Current Implementation

Current `VerifySynthesis` (check.go:363-377):
- Only checks file exists via `os.Stat`
- Only checks `info.Size() > 0`
- No content validation whatsoever

Related verifications already exist:
- `VerifyGitDiff` (git_diff.go) - validates claimed files in Delta section exist in git diff
- `VerifyTestEvidence` (test_evidence.go) - validates test execution evidence in beads comments
- `VerifyBuildForCompletion` - validates Go builds succeed
- `VerifyGitCommitsForCompletion` - validates commits exist since spawn

### 2. Identified What's Missing

The gap: `VerifySynthesis` is only called for existence check, while `VerifyGitDiff` and others are called separately. The issue description asks for:

1. Claimed file changes exist in git - **Already exists via `VerifyGitDiff`**
2. Claimed commits exist - **Already exists via `VerifyGitCommitsForCompletion`** (checks for any commits)
3. Test output section matches reality - **Partially covered by `VerifyTestEvidence`** (checks beads comments, not SYNTHESIS)

### 3. Design: VerifySynthesisContent

The real gap is: SYNTHESIS.md's Evidence section may claim test results that don't match actual test output in beads comments.

Validation to add:
1. **Evidence section claims test results** → cross-validate with beads comment test evidence patterns
2. **Duration claims** → verify against spawn time and Phase: Complete time
3. **Outcome claims** → verify against actual phase status

## What I Observed

Looking at the codebase:
- `ParseSynthesis` already extracts all D.E.K.N. sections (Delta, Evidence, Knowledge, Next)
- The `Evidence` field contains claims like "15 tests passed in 0.8s"
- `HasTestExecutionEvidence` has patterns to detect valid test output

Key insight: The validation should focus on **cross-referencing** SYNTHESIS claims against beads comments, not just checking for existence.

## Conclusion

**Implemented `VerifySynthesisContent` with the following capabilities:**

1. **Evidence section cross-validation** - Checks if SYNTHESIS.md claims tests passed, then verifies beads comments have actual test evidence (using existing `HasTestExecutionEvidence` patterns)
2. **Duration validation** - Parses claimed duration and compares against actual spawn-to-now time with 50% variance tolerance
3. **Warnings, not errors** - Following "trust but verify" principle, uncorroborated claims produce warnings not blocking errors

**Files created:**
- `pkg/verify/synthesis_content.go` - Main implementation
- `pkg/verify/synthesis_content_test.go` - Comprehensive tests

**Integration:**
- Added as step #9 in `VerifyCompletionFull` workflow in `check.go`
- Returns warnings to alert orchestrator about uncorroborated claims

This respects Evidence Hierarchy by treating primary sources (git, beads) as authoritative while not blocking on unverifiable claims.
