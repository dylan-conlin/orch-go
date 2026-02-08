## Summary (D.E.K.N.)

**Delta:** Added explicit markdown-only and outside-project exemptions to test_evidence gate so synthesis/doc work completes without test evidence demand.

**Evidence:** All 53 tests pass in pkg/verify/; new helper functions (isMarkdownFile, areAllFilesMarkdown, isFileOutsideProject, areAllFilesOutsideProject) work correctly.

**Knowledge:** The skill-based exemption alone was insufficient because feature-impl can be used for both code AND non-code work; exemptions must be based on WHAT was changed, not just skill type.

**Next:** Close - implementation complete with tests.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Exempt Non Code Work Test

**Question:** How do we exempt non-code work (markdown-only or files-outside-project) from the test_evidence gate?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-feat-exempt-non-code-14jan-3239
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Skill-based exemption was insufficient

**Evidence:** The existing `skillsRequiringTestEvidence` map includes `feature-impl`, but feature-impl can be used for non-code work (synthesis, documentation, kb work). The skill type doesn't determine whether test evidence is needed - the actual changes do.

**Source:** `pkg/verify/test_evidence.go:28-33`

**Significance:** Exemptions needed to be based on WHAT files were changed, not just which skill was used.

---

### Finding 2: Two exemption categories identified

**Evidence:** Task specified two exemptions:
1. Markdown-only changes: All modified files are `.md`
2. Files outside project dir: No test harness available (e.g., `~/.claude/skills/...`)

**Source:** Task description in SPAWN_CONTEXT.md

**Significance:** Both represent valid cases where running tests is impossible or meaningless.

---

### Finding 3: Implementation approach - early exemption checks

**Evidence:** Added exemption checks in `VerifyTestEvidenceWithComments` after skill check but before code change detection:
1. Get all changed files since spawn
2. Check if all files are markdown → exempt with warning
3. Check if all files are outside project → exempt with warning
4. Fall back to existing code change detection

**Source:** `pkg/verify/test_evidence.go:502-522`

**Significance:** This approach provides explicit logging/warnings for each exemption type and short-circuits before expensive operations.

---

## Synthesis

**Key Insights:**

1. **Content-based exemptions are more accurate than skill-based** - The same skill can produce code or non-code artifacts depending on the task.

2. **Exemptions need visibility** - Added `MarkdownOnlyExempt` and `OutsideProjectExempt` fields to `TestEvidenceResult` for tracking and debugging.

3. **Order matters** - Markdown-only check comes before outside-project check because markdown exemption is more common and faster to compute.

**Answer to Investigation Question:**

Added two explicit exemptions to `VerifyTestEvidenceWithComments`:
1. If ALL changed files are `.md`, exempt with warning "markdown-only changes (N .md files) - test evidence not required"
2. If ALL changed files are outside projectDir, exempt with warning "all changes outside project dir (N files) - no test harness available"

Both exemptions are tracked in the result struct and propagate correctly to `VerifyTestEvidenceForCompletionWithComments`.

---

## Structured Uncertainty

**What's tested:**

- ✅ `isMarkdownFile` correctly identifies .md files (verified: 11 test cases pass)
- ✅ `isFileOutsideProject` correctly identifies outside files (verified: 9 test cases pass)
- ✅ `areAllFilesMarkdown` returns correct (bool, count) (verified: 6 test cases pass)
- ✅ `areAllFilesOutsideProject` returns correct (bool, count) (verified: 6 test cases pass)
- ✅ All 53 verify package tests pass

**What's untested:**

- ⚠️ End-to-end with real markdown-only agent completion (not practical to spawn agent in test)
- ⚠️ Performance impact of additional git operations (likely negligible)

**What would change this:**

- Finding would be wrong if getChangedFilesSinceSpawn returns incorrect files
- Finding would be wrong if workspace-filtered commits don't match agent's actual commits

---

## References

**Files Examined:**
- `pkg/verify/test_evidence.go` - Main implementation file
- `pkg/verify/test_evidence_test.go` - Test file
- `pkg/verify/check.go` - Integration point for test evidence checks

**Commands Run:**
```bash
# Build verification
go build ./...

# Run all verify tests
go test ./pkg/verify/...
# Result: ok github.com/dylan-conlin/orch-go/pkg/verify 4.928s
```

---

## Investigation History

**2026-01-14 13:30:** Investigation started
- Initial question: How to exempt non-code work from test_evidence gate
- Context: Synthesis/doc work was being blocked by test evidence requirements

**2026-01-14 14:00:** Implementation completed
- Added helper functions: isMarkdownFile, isFileOutsideProject, areAllFilesMarkdown, areAllFilesOutsideProject, parseFileList, getChangedFilesSinceSpawn, getChangedFilesInWorkspaceCommits
- Integrated exemption checks into VerifyTestEvidenceWithComments
- Added new fields to TestEvidenceResult struct

**2026-01-14 14:15:** Investigation completed
- Status: Complete
- Key outcome: Markdown-only and outside-project exemptions added to test_evidence gate
