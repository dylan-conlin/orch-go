<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Yes, orch-go HAS built follow-up issue extraction from SYNTHESIS.md - implemented in `orch complete` (interactive prompting) and dashboard (POST /api/issues endpoint with "Create Issue" buttons).

**Evidence:** Found implementation in `cmd/orch/complete_cmd.go:281-356` (follow-up prompting), `pkg/verify/check.go:189-312` (SYNTHESIS.md parsing), and prior investigations documenting the feature (2025-12-25-inv-orch-complete-prompt-follow-up.md, 2025-12-26-inv-synthesis-review-view-parse-synthesis.md).

**Knowledge:** The feature extracts NextActions, AreasToExplore, Uncertainties, and Recommendation fields from SYNTHESIS.md using D.E.K.N. structure parsing, then prompts orchestrator with [y/N/q] for each item to create beads issues.

**Next:** Close - feature exists and is documented. No `orch extract` standalone command exists, but extraction is integrated into `orch complete` workflow.

---

# Investigation: Follow-up Issue Extraction from SYNTHESIS.md

**Question:** Has orch-go built or designed a feature to automatically extract follow-up issues from SYNTHESIS.md recommendations?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Starting approach

**Evidence:** Will search for: orch extract, orch review creating issues, synthesis parsing, follow-up extraction, recommendation-to-issue automation in code, .kb/investigations, .kb/decisions, and features.json.

**Source:** SPAWN_CONTEXT.md task description

**Significance:** This is the initial checkpoint documenting the investigation approach before exploration begins.

---

### Finding 2: SYNTHESIS.md parsing infrastructure exists in pkg/verify/check.go

**Evidence:** The `verify.ParseSynthesis()` function (lines 189-237) parses SYNTHESIS.md files and extracts:
- Header fields: Agent, Issue, Duration, Outcome
- D.E.K.N. sections: TLDR, Delta, Evidence, Knowledge, Next
- Unexplored Questions: AreasToExplore, Uncertainties
- Parsed fields: Recommendation (close/spawn-follow-up/escalate/resume), NextActions (list of follow-up items)

Helper functions include:
- `extractNextActions()` (lines 282-312) - parses "## Next Actions" and follow-up subsections
- `parseActionItems()` (lines 318-340) - extracts bullet points and numbered lists
- `extractBoldSubsection()` (lines 342-368) - extracts items from **bold header:** sections

**Source:** `pkg/verify/check.go:163-368`

**Significance:** Full parsing infrastructure exists to extract actionable follow-up items from SYNTHESIS.md files.

---

### Finding 3: Follow-up prompting implemented in `orch complete`

**Evidence:** In `cmd/orch/complete_cmd.go:281-356`, after verification passes:
1. Parses SYNTHESIS.md via `verify.ParseSynthesis(workspacePath)`
2. Checks for non-close recommendations or next actions
3. Collects all actionable items: NextActions, AreasToExplore, Uncertainties
4. Displays count and list of items
5. For each item, prompts with `[y/N/q to quit]`
6. If yes, creates beads issue via `beads.FallbackCreate()` with P2 priority and `triage:review` label
7. Reports count of created issues

**Source:** `cmd/orch/complete_cmd.go:281-356`

**Significance:** Interactive follow-up issue creation is fully integrated into the completion workflow.

---

### Finding 4: Dashboard also supports follow-up issue creation

**Evidence:** From prior investigation `2025-12-26-inv-synthesis-review-view-parse-synthesis.md`:
- POST /api/issues endpoint added to serve.go for issue creation
- Frontend has "Create Issue" buttons for each next_action item
- Uses beads RPC client (or CLI fallback) for issue creation

**Source:** `.kb/investigations/2025-12-26-inv-synthesis-review-view-parse-synthesis.md`

**Significance:** Two entry points exist for follow-up issue creation: CLI (`orch complete`) and dashboard UI.

---

### Finding 5: No standalone `orch extract` command exists

**Evidence:** Searched for `extractCmd` and `orch extract` patterns in `cmd/orch/*.go` - no matches found. The command file list shows no `extract_cmd.go` or similar file.

**Source:** `grep "orch extract|extractCmd" cmd/orch/*.go` - no results

**Significance:** Follow-up extraction is integrated into `orch complete` workflow, not a standalone command.

---

## Synthesis

**Key Insights:**

1. **Feature is fully built and operational** - SYNTHESIS.md parsing extracts NextActions, AreasToExplore, Uncertainties, and Recommendation fields. The infrastructure was built in Dec 2025.

2. **Two entry points for issue creation** - CLI via `orch complete` (interactive prompts) and dashboard via POST /api/issues (button clicks). Both use the same underlying beads issue creation.

3. **Integration-first design** - Rather than a standalone `orch extract` command, extraction is integrated into the completion workflow where it naturally fits (after agent work is done, before closing the issue).

**Answer to Investigation Question:**

YES, orch-go has built this feature. The implementation includes:
- `verify.ParseSynthesis()` for extracting recommendations from SYNTHESIS.md
- `extractNextActions()` for parsing follow-up items
- Integration in `orch complete` that prompts for each actionable item
- Dashboard endpoint POST /api/issues for UI-based creation

There is no standalone `orch extract` command - the functionality is embedded in the completion workflow.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code exists in `cmd/orch/complete_cmd.go:281-356` (verified by reading file)
- ✅ Parsing functions exist in `pkg/verify/check.go:189-312` (verified by reading file)
- ✅ Prior investigations document the feature (verified: 2025-12-25-inv-orch-complete-prompt-follow-up.md)

**What's untested:**

- ⚠️ Did not run `orch complete` to observe interactive prompts (no Go runtime available in this session)
- ⚠️ Did not verify dashboard UI buttons work (would need running server)

**What would change this:**

- Finding would be wrong if the code was removed or broken since the investigations were written
- Finding would be incomplete if there's a separate automated (non-interactive) extraction feature not discovered

---

## Implementation Recommendations

N/A - Feature already exists.

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` - Main completion command with follow-up prompting
- `pkg/verify/check.go` - SYNTHESIS.md parsing and extraction functions
- `.kb/investigations/2025-12-25-inv-orch-complete-prompt-follow-up.md` - Prior investigation documenting implementation
- `.kb/investigations/2025-12-26-inv-synthesis-review-view-parse-synthesis.md` - Dashboard integration investigation

**Commands Run:**
```bash
# Search for follow-up extraction code
grep -r "follow.?up|synthesis.*extract" --include="*.go"

# Search for orch extract command
grep "orch extract|extractCmd" cmd/orch/*.go

# List command files
ls -la cmd/orch/*.go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-inv-orch-complete-prompt-follow-up.md` - Original implementation investigation
- **Investigation:** `.kb/investigations/2025-12-26-inv-synthesis-review-view-parse-synthesis.md` - Dashboard integration
- **Investigation:** `.kb/investigations/2025-12-27-inv-completion-escalation-model-completed-agents.md` - Escalation model design

---

## Investigation History

**2026-01-04:** Investigation started
- Initial question: Has orch-go built or designed a feature to automatically extract follow-up issues from SYNTHESIS.md recommendations?
- Context: Orchestrator wanted to understand if this feature existed

**2026-01-04:** Found comprehensive implementation
- SYNTHESIS.md parsing in verify package (lines 189-368)
- Follow-up prompting in complete_cmd.go (lines 281-356)
- Dashboard integration via POST /api/issues

**2026-01-04:** Investigation completed
- Status: Complete
- Key outcome: Feature exists and is fully implemented in orch complete workflow and dashboard

## Self-Review

- [x] Real test performed (searched actual codebase, read source files)
- [x] Conclusion from evidence (based on code review and prior investigations)
- [x] Question answered (YES, feature exists)
- [x] File complete

**Self-Review Status:** PASSED
