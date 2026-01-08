<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Added human approval requirement to UI completion gate - agents can no longer self-certify visual correctness.

**Evidence:** Tests pass for new approval patterns and verification logic. Build succeeds.

**Knowledge:** Visual verification alone is insufficient - agents can claim "screenshot captured" without actually doing it. Explicit human approval (via `--approve` flag or beads comment) provides the necessary gate.

**Next:** Close - implementation complete with tests passing.

---

# Investigation: UI Completion Gate - Require Screenshot + Human Approval

**Question:** How to prevent agents from self-certifying UI correctness when they can't verify it themselves?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing visual verification checks only for evidence patterns

**Evidence:** `pkg/verify/visual.go` contains `visualEvidencePatterns` that match phrases like "screenshot", "visual verification", "playwright" in beads comments. Agents can write these phrases without actually taking a screenshot.

**Source:** `pkg/verify/visual.go:77-102`

**Significance:** Pattern matching alone cannot distinguish between agent self-certification and actual human verification.

---

### Finding 2: Solution requires two-layer verification

**Evidence:** Implemented `humanApprovalPatterns` that match explicit approval markers like "✅ APPROVED", "UI APPROVED", "human_approved: true". These patterns are designed to be unlikely to be accidentally used by agents.

**Source:** `pkg/verify/visual.go:103-117`

**Significance:** Separating "evidence" from "approval" allows agents to report they captured screenshots while requiring human confirmation.

---

### Finding 3: CLI integration via --approve flag

**Evidence:** Added `--approve` flag to `orch complete` command that adds approval comment before verification runs. This allows single-command approval workflow.

**Source:** `cmd/orch/main.go:352-381, 2958-2968`

**Significance:** Orchestrator can review UI changes and approve in one step without manually adding beads comments.

---

## Synthesis

**Key Insights:**

1. **Two-layer verification** - Separate "evidence exists" from "human approved" to prevent agent self-certification.

2. **CLI-first workflow** - The `--approve` flag enables `orch complete <id> --approve` as the primary approval path.

3. **Fallback patterns** - Alternative approval patterns (LGTM, "I approve") support manual beads comment workflow.

**Answer to Investigation Question:**

The UI completion gate now requires BOTH visual verification evidence AND explicit human approval. Agents can report screenshots and browser verification, but completion is blocked until a human (via `--approve` flag or explicit approval comment) confirms the visual changes are correct. This prevents the "agent renders → thinks done → human discovers wrong" problem.

---

## Structured Uncertainty

**What's tested:**

- ✅ Human approval patterns match expected inputs (verified: unit tests pass)
- ✅ Verification fails without approval when evidence exists (verified: TestSkillAwareVisualVerification)
- ✅ Build compiles successfully (verified: go build ./...)
- ✅ All visual verification tests pass (verified: go test ./pkg/verify/...)

**What's untested:**

- ⚠️ Integration with actual orch complete workflow (not tested in real scenario)
- ⚠️ beads RPC client AddComment integration (mocked in tests)

**What would change this:**

- False positives if agents learn to output approval patterns
- Need for more sophisticated approval tracking if current patterns prove insufficient

---

## Implementation Details

**Files modified:**

1. `pkg/verify/visual.go` - Added `humanApprovalPatterns`, `HasHumanApproval()`, `NeedsApproval` field, updated `VerifyVisualVerification()`
2. `pkg/verify/visual_test.go` - Added tests for human approval patterns and logic
3. `cmd/orch/main.go` - Added `--approve` flag and `addApprovalComment()` function

**Approval workflow:**

1. Agent modifies web/ files and reports "Visual verification: screenshot captured"
2. Agent reports "Phase: Complete"
3. Orchestrator runs `orch complete <id>` - FAILS with "requires human approval"
4. Orchestrator reviews UI visually
5. Orchestrator runs `orch complete <id> --approve` - adds approval comment and completes

**Alternative workflow:**

1. Orchestrator reviews UI
2. Orchestrator adds `bd comment <id> "✅ APPROVED"` 
3. Orchestrator runs `orch complete <id>` - passes

---

## References

**Files Examined:**
- `pkg/verify/visual.go` - Core visual verification logic
- `pkg/verify/check.go` - Integration point for VerifyCompletionFull
- `cmd/orch/main.go` - CLI command implementation
- `pkg/beads/client.go` - Beads RPC client for comment adding

**Commands Run:**
```bash
# Build verification
go build ./...

# Test execution
go test -v ./pkg/verify/... -run "Visual|Approval"
```

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: How to prevent agents from self-certifying UI correctness?
- Context: Agents claim visual verification without actually doing it or getting human confirmation

**2025-12-26:** Solution implemented
- Added human approval patterns to visual verification
- Added --approve flag to orch complete
- All tests pass

**2025-12-26:** Investigation completed
- Status: Complete
- Key outcome: UI completion now requires explicit human approval via --approve flag or approval comment
