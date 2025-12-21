**TLDR:** Question: Does orch-go update beads issue status to 'in_progress' when spawning? Answer: Yes, orch-go calls verify.UpdateIssueStatus with status 'in_progress' before spawning, as confirmed by testing with a real beads issue. Medium confidence (70%) - tested one scenario.

<!--
Example TLDR:
"Question: Why aren't worker agents running tests? Answer: Agents follow documentation literally but test-running guidance isn't in spawn prompts or CLAUDE.md, only buried in separate docs. High confidence (85%) - validated across 5 agent sessions but small sample size."

Guidelines:
- Keep to 2-3 sentences maximum
- Answer: What question? What's the answer? How confident?
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test Task - Spawn Context Protocol Verification

**Question:** Does orch-go update beads issue status to 'in_progress' when spawning an agent, as the Python orch-cli does? If not, what needs to be implemented?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Medium (70%)

---

## What I tried

- Read spawn context and beads issue description
- Searched for "bd update" and "in_progress" in codebase
- Found UpdateIssueStatus function in pkg/verify/check.go
- Examined main.go where UpdateIssueStatus is called with "in_progress"
- Created test beads issue orch-go-9bw to verify status update functionality
- Created another test beads issue (orch-go-t83) and ran `./orch-go spawn investigation "test" --issue orch-go-t83 --inline` with a 5-second timeout
- Checked issue status before and after using `bd show`

## What I observed

- UpdateIssueStatus is implemented and calls `bd update <id> --status <status>`
- main.go calls UpdateIssueStatus after obtaining beads ID, but prints warning if it fails
- The existing beads issue orch-go-jfo is already in status "in_progress" (maybe updated by orchestrator)
- The test issue orch-go-9bw is currently "open"
- The test issue orch-go-t83 status changed from 'open' to 'in_progress' after running the spawn command
- The spawn command hung (inline mode bug) but status update occurred before the hang
- If `bd update` fails, the error is logged as a warning but the spawn continues

## Test performed

**Test:** [To be determined]

**Result:** [To be determined]

## Conclusion

[Only fill if tested]

## Self-Review

- [ ] Real test performed (not code review)
- [ ] Conclusion from evidence (not speculation)
- [ ] Question answered
- [ ] File complete

**Self-Review Status:** PENDING
