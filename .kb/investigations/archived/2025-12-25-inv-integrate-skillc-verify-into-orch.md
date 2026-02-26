## Summary (D.E.K.N.)

**Delta:** Implemented skill output verification in pkg/verify that checks outputs.required from skill.yaml during orch complete.

**Evidence:** All 16 new tests pass; build succeeds; investigation skill has outputs.required defined with pattern `.kb/investigations/{date}-inv-*.md`.

**Knowledge:** The skillc verify command doesn't exist as a CLI command, but skill.yaml files have outputs.required sections that can be parsed and verified. Integration into VerifyCompletionFull provides automated verification.

**Next:** Close - implementation complete with tests.

**Confidence:** High (90%) - Full test coverage for new functionality; graceful skip for skills without outputs.required.

---

# Investigation: Integrate Skillc Verify Into Orch

**Question:** How can we integrate skill output verification into orch complete so agents can't complete without producing required outputs?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent og-feat-integrate-skillc-verify-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: skillc verify CLI command doesn't exist

**Evidence:** Running `skillc verify --help` returns error; `skillc --help` shows available commands which don't include verify.

**Source:** `skillc --help` output, `skillc verify --help` returns "no check help"

**Significance:** We can't shell out to `skillc verify`. Instead, we need to implement the verification logic in Go by parsing skill.yaml files directly.

---

### Finding 2: Skill manifests have outputs.required section

**Evidence:** investigation skill.yaml has:
```yaml
outputs:
  required:
    - pattern: ".kb/investigations/{date}-inv-*.md"
      description: "Investigation file with findings"
```

**Source:** `~/orch-knowledge/skills/src/worker/investigation/.skillc/skill.yaml`

**Significance:** This is the data we need to parse. The pattern format uses `{date}` variable that needs to be converted to a glob wildcard `*`.

---

### Finding 3: Existing constraint system provides pattern matching

**Evidence:** `pkg/verify/constraint.go` already has `PatternToGlob()` function that converts `{date}`, `{workspace}`, `{beads}` patterns to glob wildcards.

**Source:** `pkg/verify/constraint.go:207-230`

**Significance:** We can reuse this pattern conversion for skill output verification.

---

## Synthesis

**Key Insights:**

1. **No skillc CLI dependency needed** - The skill.yaml files can be parsed directly using Go's yaml package.

2. **Graceful skip pattern works well** - Skills without outputs.required defined should pass verification silently, which allows incremental adoption.

3. **Spawn time filtering prevents false positives** - Only files created after the spawn time are counted as matches, preventing pre-existing files from satisfying constraints.

**Answer to Investigation Question:**

Skill output verification was implemented by:
1. Creating `pkg/verify/skill_outputs.go` with functions to parse skill.yaml and verify outputs
2. Integrating into `VerifyCompletionFull` to check skill outputs alongside existing constraint verification
3. Using the existing `PatternToGlob()` function for pattern matching
4. Adding spawn time filtering to only match files created during this spawn

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
- All 16 new tests pass
- Build succeeds
- Integration verified via code review
- Graceful handling of edge cases (missing skill, no outputs, etc.)

**What's certain:**

- ✅ Skill output parsing works for investigation skill format
- ✅ Pattern matching converts `{date}` to `*` correctly
- ✅ Spawn time filtering works correctly
- ✅ Graceful skip when skill has no outputs.required

**What's uncertain:**

- ⚠️ Haven't tested with a live orch complete run
- ⚠️ Error messages may need tuning based on user feedback

**What would increase confidence to Very High (95%+):**

- Run `orch complete` on a real investigation workspace that has/hasn't produced output
- Verify error messages are clear and actionable

---

## Implementation Recommendations

**Purpose:** Implementation is complete.

### Implemented Approach ⭐

**Skill Output Verification** - Added new verification step in VerifyCompletionFull that parses skill.yaml and checks outputs.required patterns.

**Files created/modified:**
- `pkg/verify/skill_outputs.go` - New file with SkillManifest parsing, output verification
- `pkg/verify/skill_outputs_test.go` - 16 tests covering all scenarios
- `pkg/verify/check.go` - Added call to VerifySkillOutputsForCompletion in VerifyCompletionFull

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Existing verification logic
- `pkg/verify/constraint.go` - Pattern matching utilities
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/skill.yaml` - Skill manifest format

**Commands Run:**
```bash
# Check skillc CLI
skillc --help
skillc verify --help

# Build and test
go build ./pkg/verify/...
go test ./pkg/verify/... -v
make build
```

---

## Investigation History

**2025-12-25 10:20:** Investigation started
- Initial question: How to integrate skillc verify into orch complete
- Context: Issue orch-go-loh8 describes the gap

**2025-12-25 10:25:** Discovery - skillc verify doesn't exist
- Need to implement verification in Go

**2025-12-25 10:35:** Implementation complete
- Created skill_outputs.go and tests
- Integrated into VerifyCompletionFull

**2025-12-25 10:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Skill output verification integrated into orch complete
