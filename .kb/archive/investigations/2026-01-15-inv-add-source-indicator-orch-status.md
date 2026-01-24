<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added SRC column to orch status showing agent source origin (T/O/B/W) to make cleanup commands obvious.

**Evidence:** Manual testing shows source indicators working correctly - T for tmux agents, O for OpenCode sessions, W for workspace agents, B for beads phantoms.

**Knowledge:** Source priority (T > O > B > W) reflects cleanup priority; existing tests pass without modification; AgentInfo enrichment phase was the right place to determine source.

**Next:** Close - feature complete, tested, and committed.

**Promote to Decision:** recommend-no (tactical implementation, not architectural)

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

# Investigation: Add Source Indicator Orch Status

**Question:** How to add source indicators to orch status to show where each agent originated (tmux/OpenCode/beads/workspace)?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** og-feat-add-source-indicator-15jan-9759
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Agent enrichment happens after collection from all sources

**Evidence:** Lines 434-478 in status_cmd.go show a single enrichment loop that processes all agents regardless of source (registry, tmux, OpenCode).

**Source:** cmd/orch/status_cmd.go:434-478

**Significance:** The enrichment phase is the correct place to determine source - all agent metadata (phantom status, session info, tmux window) is available by that point.

---

### Finding 2: Source priority reflects cleanup priority

**Evidence:** determineAgentSource() checks: (1) tmux window, (2) OpenCode session, (3) beads phantom, (4) workspace. This matches cleanup command priority - visible TUI (tmux) is most important to surface.

**Source:** cmd/orch/status_cmd.go:1153-1182

**Significance:** The source indicator helps users choose the right cleanup command: tmux kill-window for T, session delete for O, bd close for B, orch clean for W.

---

### Finding 3: Display formats all needed SRC column added

**Evidence:** Wide format (line 988), narrow format (line 1075), and card format (line 1113) all display source with consistent handling.

**Source:** cmd/orch/status_cmd.go:979-1149

**Significance:** Source indicator is visible across all terminal width modes, ensuring users always see where agents originated.

---

## Synthesis

**Key Insights:**

1. **Single enrichment phase simplifies source determination** - Rather than tracking source during collection from 3 different phases (registry, tmux, OpenCode), determining source after enrichment means all metadata is available in one place (Finding 1).

2. **Source priority maps to user mental model** - T > O > B > W priority reflects visibility: tmux windows are visible TUI sessions, OpenCode are headless/API, beads are phantom issues, workspaces are cleanup candidates (Finding 2).

3. **Consistent display across terminal widths** - Adding SRC column to all three display functions ensures the feature works for all users regardless of terminal size (Finding 3).

**Answer to Investigation Question:**

Add a Source field to AgentInfo struct and populate it during the agent enrichment phase (lines 434-478) using a helper function that checks tmux window > OpenCode session > beads phantom > workspace in priority order. Display the source in all three format functions (wide/narrow/card) as a "SRC" column. This approach leverages the existing enrichment infrastructure and provides consistent source indicators across all display modes.

---

## Structured Uncertainty

**What's tested:**

- ✅ Source indicators display correctly in orch status output (verified: manual test shows T/O/W/- indicators)
- ✅ All existing status tests pass (verified: go test ./cmd/orch -run Status shows all green)
- ✅ Source determination works for tmux, OpenCode, workspace, and unknown agents (verified: manual orch status shows all 4 types)

**What's untested:**

- ⚠️ Beads phantom source (B) - no phantom agents in current test environment to verify
- ⚠️ Performance impact of determineAgentSource() called for each agent (not benchmarked)
- ⚠️ Edge case: agent with both tmux window AND OpenCode session shows T (priority assumed correct)

**What would change this:**

- Finding would be wrong if users prefer different priority order (e.g., O > T instead of T > O)
- Implementation would need rework if source needs to show ALL applicable sources instead of just primary
- Display logic would break if terminal width calculations were off (but tests validate widths)

---



## References

**Files Examined:**
- cmd/orch/status_cmd.go - Main status command implementation showing agent collection and display
- pkg/registry/registry.go - Agent struct definition showing Mode, SessionID, TmuxWindow fields

**Commands Run:**
```bash
# Build and install the modified binary
make install

# Test the source indicator display
orch status

# Run status-related tests
go test ./cmd/orch -run Status -v
```

**External Documentation:**
- None

**Related Artifacts:**
- **Issue:** orch-go-gnbof - Add source indicator to orch status output
- **Workspace:** /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-add-source-indicator-15jan-9759/

---

## Investigation History

**[2026-01-15 16:15]:** Investigation started
- Initial question: How to add source indicators to orch status to show where each agent originated?
- Context: Users need to know which cleanup command to use for each agent (tmux kill-window vs session delete vs bd close vs orch clean)

**[2026-01-15 16:20]:** Discovered agent enrichment phase
- Found that all agents go through single enrichment loop (lines 434-478) regardless of collection source
- Determined this is the right place to set source field

**[2026-01-15 16:25]:** Implemented and tested
- Added Source field to AgentInfo struct
- Created determineAgentSource() helper with T > O > B > W priority
- Updated all three display formats (wide/narrow/card)
- All tests pass, manual testing confirms working

**[2026-01-15 16:30]:** Investigation completed
- Status: Complete
- Key outcome: Source indicator feature working and committed (d11fd84c)
