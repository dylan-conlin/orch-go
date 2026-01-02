<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added VERIFICATION REQUIREMENTS section to SPAWN_CONTEXT.md template documenting test evidence patterns, visual verification, and git commit requirements that `orch complete` checks.

**Evidence:** Tested: go test ./pkg/spawn/... ./pkg/verify/... - all tests pass; verified section appears in generated context and is correctly omitted for NoTrack spawns.

**Knowledge:** Agents need explicit documentation of what evidence to capture because `orch complete` has verification gates (test_evidence.go, visual.go, git_commits.go) that check specific patterns in beads comments.

**Next:** Close - implementation complete, tests pass, verification requirements now documented in agent context.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Add Verification Requirements Skill Templates

**Question:** How can we document verification requirements in SPAWN_CONTEXT.md so agents know what evidence to capture for `orch complete`?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** Agent og-feat-add-verification-requirements-02jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Test evidence verification uses regex pattern matching

**Evidence:** `pkg/verify/test_evidence.go` defines `testEvidencePatterns` (lines 74-112) that match actual test output like "go test ./... - PASS", "15 passing, 0 failing", etc. Also defines `falsePositivePatterns` (lines 117-131) to reject vague claims like "tests pass".

**Source:** `pkg/verify/test_evidence.go:74-131`

**Significance:** Agents must include quantifiable test output in beads comments (pass counts, timing) or completion will be blocked. Vague claims are explicitly rejected.

---

### Finding 2: Visual verification required for web/ file changes

**Evidence:** `pkg/verify/visual.go` defines `visualEvidencePatterns` (lines 81-107) looking for screenshot/browser verification mentions. Only applies to skills in `skillsRequiringVisualVerification` map (currently just feature-impl).

**Source:** `pkg/verify/visual.go:14-18, 81-107`

**Significance:** Agents modifying web/ files must capture screenshots or mention browser verification. Uses Playwright MCP or Glass tools for evidence capture.

---

### Finding 3: Git commits required for code-producing skills

**Evidence:** `pkg/verify/git_commits.go` defines `codeProducingSkills` (lines 27-31) including feature-impl, systematic-debugging, reliability-testing. `CountCommitsSinceTime` (line 74) checks if commits exist since spawn time.

**Source:** `pkg/verify/git_commits.go:27-31, 74-100`

**Significance:** If a code-producing skill reports Phase: Complete with no commits since spawn time, it's flagged as a false positive and completion is blocked.

---

## Synthesis

**Key Insights:**

1. **Evidence must be machine-verifiable** - The verification gates use regex patterns to detect actual test output, not natural language claims. This means agents must include specific patterns like "PASS (12 tests in 0.8s)" rather than "tests pass".

2. **Skill-aware verification** - Different skills have different requirements. Only `feature-impl` requires visual verification for web/ changes; only code-producing skills require git commits. This prevents false positives from investigation/research skills.

3. **Timing matters for scoping** - Git commit and constraint verification use spawn time to scope checks to THIS agent's work, preventing false positives from prior agents' commits.

**Answer to Investigation Question:**

Added a VERIFICATION REQUIREMENTS section to `SpawnContextTemplate` in `pkg/spawn/context.go` that explicitly documents:
- Test evidence patterns that `orch complete` accepts (with examples of good vs bad)
- Visual verification requirements for web/ file changes
- Git commit requirements for code-producing skills
- Evidence capture timing guidance

The section is conditionally included only when `NoTrack=false` (tracked spawns) since verification gates depend on beads comments.

---

## Structured Uncertainty

**What's tested:**

- ✅ Template generates verification section correctly (verified: ran spawn.GenerateContext with test config)
- ✅ NoTrack spawns omit verification section (verified: ran with NoTrack=true, section absent)
- ✅ All spawn and verify tests pass (verified: go test ./pkg/spawn/... ./pkg/verify/... - all pass)

**What's untested:**

- ⚠️ Agents actually read and follow the verification guidance (behavioral - would need observation)
- ⚠️ The guidance reduces failed completions (metric - would need historical comparison)

**What would change this:**

- If agents still fail to capture evidence after this change, may need skill-specific embedding of requirements (not just spawn context)
- If test patterns are too strict or lenient, may need to adjust regex patterns in test_evidence.go

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Add VERIFICATION REQUIREMENTS section to SpawnContextTemplate** - Document verification gates in agent context.

**Why this approach:**
- Agents follow documentation literally; guidance must be in loaded context
- Centralizes verification docs in one place agents always see
- Uses examples to show exact patterns expected

**Trade-offs accepted:**
- Adds ~50 lines to SPAWN_CONTEXT.md (acceptable for clarity)
- Only for tracked spawns (NoTrack spawns have no beads comments to verify)

**Implementation sequence:**
1. Add verification section after SERVER CONTEXT, before FINAL STEP
2. Condition on NoTrack to avoid showing beads commands in ad-hoc spawns
3. Include concrete examples of good vs bad evidence

### Implementation Details

**What was implemented:**
- Added VERIFICATION REQUIREMENTS section with three subsections
- Documented test evidence patterns with ✅ good / ❌ bad examples
- Documented visual verification for web/ files
- Documented git commit requirements
- Added evidence capture timing guidance

**Success criteria:**
- ✅ Tests pass (verified)
- ✅ Section appears in tracked spawns (verified)
- ✅ Section omitted in NoTrack spawns (verified)

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Main verification orchestration, VerifyCompletionFull function
- `pkg/verify/test_evidence.go` - Test evidence patterns and false positive detection
- `pkg/verify/visual.go` - Visual verification for web/ changes
- `pkg/verify/git_commits.go` - Git commit requirements for code-producing skills
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template (modified)

**Commands Run:**
```bash
# Test spawn context generation
go test ./pkg/spawn/... -v -run TestGenerateContext

# Test all spawn and verify packages
go test ./pkg/spawn/... ./pkg/verify/...

# Verify output manually
go run /tmp/test_context.go
```

**Related Artifacts:**
- **Decision:** Verification gates documented in pkg/verify/ - Existing constraint verification system

---

## Investigation History

**2026-01-02 09:00:** Investigation started
- Initial question: How to document verification requirements in SPAWN_CONTEXT.md
- Context: Agents don't know what evidence to capture for orch complete

**2026-01-02 09:30:** Findings documented
- Discovered test_evidence.go, visual.go, git_commits.go verification patterns

**2026-01-02 10:00:** Implementation complete
- Added VERIFICATION REQUIREMENTS section to SpawnContextTemplate
- All tests pass, section correctly conditional on NoTrack

**2026-01-02 10:15:** Investigation completed
- Status: Complete
- Key outcome: Verification requirements now documented in agent context
