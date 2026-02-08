<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch session end` only archives SESSION_HANDOFF.md without populating placeholders because `archiveActiveSessionHandoff()` moves the file without updating content.

**Evidence:** Code analysis shows template created at session start (cmd/orch/session.go:202) with placeholders, but archiving (line 238) only renames directory; existing bug confirmed via grep showing {end-time} and {success | partial | blocked | failed} in .orch/session/orch-go-5/2026-01-14-0805/SESSION_HANDOFF.md; unit tests pass for template replacement logic.

**Knowledge:** Session handoff template designed for progressive documentation but lacks finalization step; data needed for population (duration, spawns, outcome) already collected in runSessionEnd() but only used for console output; fix requires interactive prompting before archiving to collect outcome and summary, then replace placeholders.

**Next:** Implementation complete and tested - commit changes, mark Phase: Complete, verify bug no longer reproduces.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural pattern)

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

# Investigation: Orch Session End Creates Handoff

**Question:** Why does `orch session end` create SESSION_HANDOFF.md with unfilled placeholders instead of prompting for session summary?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-debug-orch-session-end-14jan-a227
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Template created at session start, not session end

**Evidence:** SESSION_HANDOFF.md is created by `createActiveSessionHandoff()` at session start (cmd/orch/session.go:175-214), which calls `spawn.GeneratePreFilledSessionHandoff()` to generate content with placeholders like `{end-time}`, `{success | partial | blocked | failed}`, etc.

**Source:**
- cmd/orch/session.go:127 (createActiveSessionHandoff call in runSessionStart)
- cmd/orch/session.go:202 (spawn.GeneratePreFilledSessionHandoff call)
- pkg/spawn/orchestrator_context.go:532-550 (GeneratePreFilledSessionHandoff implementation)
- pkg/spawn/orchestrator_context.go:358-522 (PreFilledSessionHandoffTemplate constant)

**Significance:** The template is designed for progressive documentation during the session, but it's never populated/finalized at session end.

---

### Finding 2: Session end only archives, doesn't populate

**Evidence:** `runSessionEnd()` calls `archiveActiveSessionHandoff()` which only renames the `active/` directory to a timestamped directory (e.g., `2026-01-14-0805/`) and updates the `latest` symlink. No prompting or template population occurs.

**Source:**
- cmd/orch/session.go:565 (archiveActiveSessionHandoff call in runSessionEnd)
- cmd/orch/session.go:219-257 (archiveActiveSessionHandoff implementation)
- Line 238: `os.Rename(activeDir, timestampedDir)` - just moves the directory

**Significance:** This is the root cause of the bug - the handoff file is moved/archived but never filled in with actual session data.

---

### Finding 3: Session end has session data available

**Evidence:** `runSessionEnd()` already collects session data before ending:
- duration (line 547)
- spawnCount (line 548)
- statuses with active count (lines 551-557)
- ended session info (line 572)

This data is used for the console output and event logging but not for populating the handoff template.

**Source:**
- cmd/orch/session.go:547-557 (data collection)
- cmd/orch/session.go:594-596 (console output using this data)
- cmd/orch/session.go:578-592 (event logging using this data)

**Significance:** The data needed to populate the template is already available in `runSessionEnd()`, we just need to use it to fill the template before archiving.

---

## Synthesis

**Key Insights:**

1. **Template created at wrong time** - SESSION_HANDOFF.md is created with placeholders at session start (for progressive documentation), but never finalized at session end.

2. **Archive without populate** - `archiveActiveSessionHandoff()` only moves the directory but doesn't update the file content, leaving placeholders unfilled.

3. **Data available but unused** - `runSessionEnd()` already collects duration, spawn count, and statuses but only uses them for console output and event logging, not for populating the handoff template.

**Answer to Investigation Question:**

`orch session end` creates SESSION_HANDOFF.md with unfilled placeholders because the archiving logic only moves the directory without updating the template content. The fix is to add interactive prompting before archiving to collect session summary (outcome and optional description) and replace placeholders in the template file before it's archived.

---

## Structured Uncertainty

**What's tested:**

- ✅ Template replacement logic works correctly (verified: TestUpdateHandoffTemplate passes)
- ✅ End time and outcome placeholders are replaced (verified: test assertions confirm replacement)
- ✅ TLDR placeholder replaced when summary provided (verified: TestUpdateHandoffTemplate)
- ✅ TLDR placeholder preserved when no summary (verified: TestUpdateHandoffTemplateNoSummary)
- ✅ Original bug confirmed: existing handoff has unfilled placeholders (verified: grep on .orch/session/orch-go-5/2026-01-14-0805/SESSION_HANDOFF.md)

**What's untested:**

- ⚠️ User experience of interactive prompting (not tested in CI - requires manual stdin)
- ⚠️ Behavior when user provides invalid outcome (error handling tested via unit test validation logic, but not end-to-end)
- ⚠️ Edge cases: very long summaries, special characters in summary

**What would change this:**

- Finding would be wrong if TestUpdateHandoffTemplate failed (placeholders not replaced)
- Finding would be wrong if manual testing showed prompts not appearing during `orch session end`
- Finding would be wrong if archived handoff still contained placeholders after the fix

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Interactive Prompting Before Archive** - Add user prompts for outcome and summary, then update template before archiving.

**Why this approach:**
- Minimal code change - adds prompting and string replacement before existing archive logic
- Uses data already available in runSessionEnd() (duration, spawn counts)
- Preserves existing template design (progressive documentation pattern)
- Prompts only for required fields (outcome) and optional fields (summary)

**Trade-offs accepted:**
- Requires user interaction during `orch session end` (cannot be fully automated)
- Only populates outcome and TLDR; other template sections still require manual filling
- Empty summary leaves TLDR placeholder unchanged (acceptable - user can skip)

**Implementation sequence:**
1. Add promptForSessionSummary() function - collects outcome (required) and summary (optional) via stdin
2. Add updateHandoffTemplate() function - reads handoff, replaces placeholders, writes back
3. Modify runSessionEnd() - call prompt and update before archiveActiveSessionHandoff()
4. Add unit tests - verify template replacement logic works correctly

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
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
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
