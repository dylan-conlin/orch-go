## Summary (D.E.K.N.)

**Delta:** Implemented session start focus guidance that groups `bd ready` issues into thematic threads using keyword detection.

**Evidence:** Manual testing shows 10 ready issues correctly grouped into 6 threads (Session tooling, Knowledge base, Model system, Cleanup, Dashboard, Spawn system); all 20 unit tests pass.

**Knowledge:** Keyword-based thread grouping provides useful orientation for orchestrators at session start; heuristics for common prefixes (session, model, kb, orch, dashboard, spawn) cover most issue types.

**Next:** Close this issue; feature is complete and integrated into `orch session start`.

**Promote to Decision:** recommend-no - Implementation feature, not architectural pattern.

---

# Investigation: Implement Session Start Focus Guidance

**Question:** How should `orch session start` group ready issues into thematic threads for orchestrator orientation?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: bd ready JSON format provides structured issue data

**Evidence:** `bd ready --json` returns JSON array with fields: id, title, description, status, priority, issue_type, labels.

**Source:** `bd ready --json | head -100` command output

**Significance:** JSON format enables programmatic grouping; title field is primary source for keyword detection.

---

### Finding 2: Keyword-based thread detection is sufficient for initial implementation

**Evidence:** Testing with 10 real issues showed keywords (session, model, kb, orch, dashboard, spawn) correctly grouped all themed issues; only truly unrelated issues fell to Misc.

**Source:** Manual testing of `orch session start "Test focus guidance feature"`

**Significance:** Simple substring matching on lowercase titles provides good accuracy without needing complex NLP.

---

### Finding 3: Thread grouping integrates cleanly after existing session start output

**Evidence:** Focus guidance appears after session info and synthesis warnings, before returning control to user. Output format matches design specification.

**Source:** cmd/orch/session.go:180 - surfaceFocusGuidance() call location

**Significance:** Natural placement in session start flow; doesn't disrupt existing functionality.

---

## Synthesis

**Key Insights:**

1. **Keyword precedence matters** - Earlier keywords in the list take precedence when multiple match (e.g., "orch doctor" matches "orch" before "doctor").

2. **MaxThreads cap prevents noise** - Limiting to 7 threads keeps output scannable; overflow issues merge into Misc thread.

3. **Notes from first issue provide context** - Truncated title of first issue in thread gives quick context without overwhelming display.

**Answer to Investigation Question:**

Focus guidance groups issues by detecting keywords in titles (session, model, kb, orch, dashboard, spawn, etc.), sorts threads by issue count descending, caps at 7 threads, and displays with issue IDs and notes. Implementation in pkg/focus/guidance.go provides reusable grouping logic callable from session start.

---

## Structured Uncertainty

**What's tested:**

- ✅ Thread grouping logic correctly categorizes issues (20 unit tests pass)
- ✅ Manual end-to-end test shows expected output format
- ✅ Empty issue list handled gracefully with "No ready issues found" message

**What's untested:**

- ⚠️ Performance with large backlogs (>100 issues) not tested
- ⚠️ Keyword list completeness - may miss some thematic patterns

**What would change this:**

- Finding would be wrong if keyword detection produces too many false positives
- Finding would be wrong if users report threads are not helpful for orientation

---

## References

**Files Modified:**
- pkg/focus/guidance.go - New file with thread grouping logic (185 lines)
- pkg/focus/guidance_test.go - New file with 10 test functions
- cmd/orch/session.go:15 - Added focus package import
- cmd/orch/session.go:178-180 - Added surfaceFocusGuidance() call
- cmd/orch/session.go:514-535 - Added surfaceFocusGuidance() function

**Commands Run:**
```bash
# Build and test
go build ./...
go test ./pkg/focus/... -v

# Manual testing
make install && orch session start "Test focus guidance feature"
```

---

## Investigation History

**2026-01-14 20:53:** Investigation started
- Initial question: How to implement focus guidance feature per issue orch-go-u7jws
- Context: Part of Capture at Context principle - surface context when it matters

**2026-01-14 21:00:** Implementation completed
- Status: Complete
- Key outcome: Focus guidance feature implemented and integrated into orch session start
