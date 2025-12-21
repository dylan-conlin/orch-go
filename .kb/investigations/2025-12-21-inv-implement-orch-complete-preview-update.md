## Summary (D.E.K.N.)

**Delta:** Implemented `orch review <id>` command for single-agent review before completion.

**Evidence:** All tests pass (go test ./... ok), build succeeds, help output shows new functionality.

**Knowledge:** `orch review <id>` is cleaner than `--preview` flag - matches mental model (review and complete are distinct actions).

**Next:** Close - implementation complete, tests passing.

**Confidence:** High (90%) - Well-tested, follows existing patterns.

---

# Investigation: Implement orch complete --preview and orchestrator skill update

**Question:** How should we implement single-agent review functionality?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent og-feat-implement-orch-complete-21dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Pivot to `orch review <id>` over `--preview` flag

**Evidence:** Mid-implementation pivot based on orchestrator feedback. Option A (--preview on complete) was initially planned but Option B (separate review command) was chosen.

**Source:** Beads comment: "PIVOT: Use 'orch review <id>' instead of '--preview' flag."

**Significance:** Cleaner separation of concerns - reviewing doesn't imply completing. Extends existing `orch review` command naturally (batch mode exists, added single-agent mode).

---

### Finding 2: AgentReview struct provides comprehensive review data

**Evidence:** Created `pkg/verify/review.go` with AgentReview struct containing:
- Synthesis data (TLDR, outcome, recommendation)
- Git delta (files created/modified, commits)
- Beads comments history
- Artifact detection (SYNTHESIS.md, investigation files)

**Source:** `pkg/verify/review.go:16-48`

**Significance:** Provides all the data an orchestrator needs to make a complete/abandon/feedback decision.

---

## Synthesis

**Key Insights:**

1. **Review and complete are distinct actions** - Separating them into different commands matches the orchestrator's mental model and enables better workflow (review first, then decide).

2. **Extending existing commands is cleaner** - Adding single-agent mode to `orch review` was simpler than adding flags to `orch complete`.

**Answer to Investigation Question:**

Implement single-agent review as `orch review <id>` rather than `orch complete --preview`. This extends the existing review command to handle both batch (no args) and single-agent (with beads ID) modes.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
Well-tested implementation following existing patterns. All tests pass, build succeeds.

**What's certain:**
- ✅ Implementation compiles and tests pass
- ✅ Command shows expected help output
- ✅ Integrates cleanly with existing review batch mode

**What's uncertain:**
- ⚠️ Real-world testing with actual agent completions not done yet

---

## References

**Files Created:**
- `pkg/verify/review.go` - AgentReview struct and GetAgentReview function
- `pkg/verify/review_test.go` - Unit tests

**Files Modified:**
- `cmd/orch/review.go` - Extended to accept optional beads ID
- `~/.claude/skills/policy/orchestrator/SKILL.md` - Updated documentation

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-single-agent-review-command.md` - Original design decision
- **Workspace:** `.orch/workspace/og-feat-implement-orch-complete-21dec/`
- **Beads:** `bd show orch-go-3anf`

---

## Investigation History

**2025-12-21:** Investigation started
- Initial question: How to implement orch complete --preview?
- Context: Orchestrator needs to review agent work before completing

**2025-12-21:** Pivot decision
- Changed from --preview flag to `orch review <id>` command
- Rationale: Better separation of concerns, cleaner mental model

**2025-12-21:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Implemented `orch review <id>` for single-agent review
