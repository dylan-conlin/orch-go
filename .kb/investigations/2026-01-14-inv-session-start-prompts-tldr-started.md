<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented session start prompts for TLDR and "Where We Started" sections in `orch session start`, part of Progressive Session Capture decision.

**Evidence:** Build succeeds, tests pass (`ok cmd/orch 4.076s`), code properly integrated at lines 150-164 of session.go.

**Knowledge:** Progressive capture requires prompting at the point when context is freshest - session start for TLDR and starting state.

**Next:** Close - implementation complete and tested.

**Promote to Decision:** recommend-no - Implementation follows existing decision (.kb/decisions/2026-01-14-progressive-session-capture.md).

---

# Investigation: Session Start Prompts TLDR Started

**Question:** How to implement TLDR and "Where We Started" prompts at session start?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-feat-session-start-prompts-14jan-af53
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing validation infrastructure can be reused

**Evidence:** The `HandoffSection` struct and validation patterns in `handoffSections` array provide a model for start-time prompts.

**Source:** cmd/orch/session.go:237-297 (HandoffSection struct and handoffSections array)

**Significance:** Created parallel `startSections` array using same structure, enabling consistent user experience.

---

### Finding 2: Placeholders in template match expected patterns

**Evidence:** PreFilledSessionHandoffTemplate in pkg/spawn/orchestrator_context.go contains:
- TLDR placeholder: `[Fill within first 5 tool calls: What is this session trying to accomplish?]`
- Where We Started placeholder: `[Fill within first 5 tool calls: What is the current state before you begin working?]`

**Source:** pkg/spawn/orchestrator_context.go:360-467

**Significance:** Placeholders can be directly replaced with user input using strings.ReplaceAll.

---

### Finding 3: runSessionStart flow supports insertion point

**Evidence:** After `createActiveSessionHandoff()` returns the handoff path, we can immediately prompt and update the file before logging events.

**Source:** cmd/orch/session.go:142-164 (handoff creation and prompting integration)

**Significance:** Prompts happen right after handoff creation, capturing context at its freshest.

---

## Synthesis

**Key Insights:**

1. **Parallel structure for start vs end sections** - Created `startSections` separate from `handoffSections` to keep different-time-of-capture concerns separate.

2. **Fail-soft design** - If prompting fails, session still starts successfully; sections can be filled later via session end validation.

3. **Progressive capture principle** - Aligns with decision 2026-01-14: capture context when it's freshest, not just at end.

**Answer to Investigation Question:**

Implemented by:
1. Adding `startSections` array with TLDR and "Where We Started" sections
2. Creating `promptForStartSections()` to interactively collect responses
3. Creating `updateHandoffWithStartResponses()` to update the handoff file
4. Integrating into `runSessionStart()` after handoff creation

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles successfully (`go build ./...`)
- ✅ Session tests pass (`ok cmd/orch 4.076s`)
- ✅ New functions are syntactically correct and callable

**What's untested:**

- ⚠️ Interactive prompting behavior in real terminal (would require manual testing)
- ⚠️ Edge case: empty input handling (logs warning, continues)
- ⚠️ Concurrent agent modifications (observed during development - linter race conditions)

**What would change this:**

- If placeholder strings in template drift, prompts won't work
- If handoff path is empty, prompts are skipped (by design)

---

## References

**Files Modified:**
- cmd/orch/session.go - Added startSections, promptForStartSections(), updateHandoffWithStartResponses()

**Related Artifacts:**
- **Decision:** .kb/decisions/2026-01-14-progressive-session-capture.md - The decision this implements
- **Template:** pkg/spawn/orchestrator_context.go - Contains PreFilledSessionHandoffTemplate

---

## Investigation History

**2026-01-14 21:30:** Investigation started
- Initial question: Implement TLDR and "Where We Started" prompts at session start
- Context: Part of Progressive Session Capture decision

**2026-01-14 22:15:** Implementation complete
- Status: Complete
- Key outcome: Added interactive prompts at session start that update SESSION_HANDOFF.md
