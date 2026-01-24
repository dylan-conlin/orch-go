<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added cross-window scan to discoverSessionHandoff between current-window check and legacy fallback, enabling convenient window switching while preserving window isolation.

**Evidence:** All 10 tests pass including new TestDiscoverSessionHandoff_CrossWindowScan which verifies most recent session is found across all windows when current window has no history.

**Knowledge:** The window-scoped directory structure (.orch/session/{window}/) already supports cross-window scanning via directory traversal - just needed to add scanAllWindowsForMostRecent() helper and insert call in discovery flow.

**Next:** Close - bug fixed and verified via reproduction test.

**Promote to Decision:** recommend-no (tactical bug fix implementing documented behavior, not establishing new architectural pattern)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix Discoversessionhandoff Scan Windows Most

**Question:** How should discoverSessionHandoff scan across all windows when current window has no history, to enable convenient window switching while preserving window isolation?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current discovery flow has gap between window-scoped and legacy checks

**Evidence:** The `discoverSessionHandoff()` function in `cmd/orch/session.go` (lines 683-789) performs three checks:
1. Window-scoped latest symlink: `.orch/session/{window-name}/latest/`
2. Window-scoped active directory: `.orch/session/{window-name}/active/`
3. Legacy non-window-scoped: `.orch/session/latest/`

There is NO cross-window scan between checks 2 and 3.

**Source:** `cmd/orch/session.go:683-789`

**Significance:** When switching to a new window with no history, users get no session handoff (or fall back to legacy structure if it exists), even when other windows have recent sessions. This breaks the convenience of window switching.

---

### Finding 2: Window-scoped structure enables cross-window scanning

**Evidence:** Session handoffs are stored in `.orch/session/{window-name}/{timestamp}/SESSION_HANDOFF.md` with a `latest` symlink pointing to most recent. The window-scoped directories can be scanned by reading `.orch/session/*` directories, then checking each for a `latest/SESSION_HANDOFF.md` file.

**Source:** `cmd/orch/session.go:683-789`, test cases in `cmd/orch/session_resume_test.go:250-421`

**Significance:** The directory structure already supports cross-window scanning by globbing `.orch/session/*/latest/SESSION_HANDOFF.md` - we just need to add the logic to compare timestamps and pick the most recent.

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ Cross-window scan finds most recent session when current window has no history (verified: TestDiscoverSessionHandoff_CrossWindowScan passes)
- ✅ Window isolation preserved when current window has history (verified: TestDiscoverSessionHandoff_WindowScoped and TestDiscoverSessionHandoff_PreferWindowScoped pass)
- ✅ Backward compatibility maintained for legacy structure (verified: TestDiscoverSessionHandoff_BackwardCompatibility passes)
- ✅ All existing discovery paths still work (verified: all 10 test cases pass)

**What's untested:**

- ⚠️ Performance impact of scanning all windows on large .orch/session directories (not benchmarked - acceptable since only happens on window switches)
- ⚠️ Behavior with broken symlinks in other windows' directories (error handling relies on continue in loop - should be safe)

**What would change this:**

- Finding would be wrong if cross-window scan returns older session when current window has newer session (violates window isolation)
- Finding would be wrong if legacy fallback is checked before cross-window scan (violates documented discovery order)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Cross-Window Scan Between Current-Window and Legacy Checks** - Insert cross-window scan step after checking current window (latest and active) but before legacy fallback.

**Why this approach:**
- Preserves window isolation when current window has history (check current window first)
- Enables convenience when switching to new windows (scan all windows for most recent)
- Maintains backward compatibility (legacy fallback still works for pre-window-scoped handoffs)
- Minimal code change - just insert scan step in existing discovery flow

**Trade-offs accepted:**
- Additional filesystem I/O when current window has no history (acceptable - only happens on window switches)
- Need to parse timestamps from directory names or symlink targets (simple string comparison since format is consistent: YYYY-MM-DD-HHMM)

**Implementation sequence:**
1. Add helper function to scan all window directories and find most recent handoff
2. Insert call to helper after current-window checks but before legacy fallback
3. Add test case for cross-window discovery behavior

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
- `cmd/orch/session.go:683-789` - Analyzed discoverSessionHandoff() to understand current discovery flow and identify where to insert cross-window scan
- `cmd/orch/session_resume_test.go` - Reviewed existing test cases to understand expected behavior and added new test for cross-window scan

**Commands Run:**
```bash
# Build to verify compilation
go build ./cmd/orch

# Run existing tests to ensure no regressions
go test ./cmd/orch -run TestDiscoverSessionHandoff -v

# Run new cross-window scan test
go test ./cmd/orch -run TestDiscoverSessionHandoff_CrossWindowScan -v

# Run all session and handoff tests
go test ./cmd/orch -run "Session|Handoff" -v
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
