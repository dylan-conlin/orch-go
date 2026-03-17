---
title: "Hotspot acceleration: pkg/daemon/issue_adapter_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-k9vr4
---

## TLDR

`pkg/daemon/issue_adapter_test.go` hotspot alert (+373 lines/30d, now 330 lines) is a **false positive** — 66% of lines (246) came from file creation on 2026-02-24 as an extraction from `daemon_test.go`. One feature addition (+127 lines for workspace selection tests) and one cleanup (-43 lines). No extraction needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/daemon/issue_adapter_test.go`
- **Evidence:** Git history shows 3 commits: extraction from daemon_test.go (246 lines, Feb 24), workspace selection tests (127 lines, Feb 28), untracked ID cleanup (-43 lines, Mar 3). 66% is birth churn from refactoring, not organic growth.
- **Knowledge:** File was created as part of a deliberate extraction (`refactor: extract daemon_test.go into focused test files`). The hotspot detector counts extraction births as growth — same false-positive pattern as thread_test.go and control_cmd.go.
- **Next:** Close. False positive — birth churn from extraction plus one legitimate feature addition.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-thread-thread-test.md | Same pattern — creation-churn false positive | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Same pattern — creation-churn false positive | yes | - |

## Question

Is `pkg/daemon/issue_adapter_test.go` (+373 lines/30d, now 330 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: Growth is primarily from extraction-birth

Git log shows exactly 3 commits:

```
32c739ec5 2026-02-24 refactor: extract daemon_test.go into focused test files (orch-go-1195) [+246 lines]
334b16caa 2026-02-28 fix: prefer SYNTHESIS.md and newest spawn time when resolving duplicate workspaces (orch-go-qbfv) [+127 lines]
22cacd80e 2026-03-03 fix: remove isUntrackedBeadsID guards and untracked exclusions (orch-go-u8bv) [-43 lines]
```

The +373 reported by the hotspot detector is 246 (extraction birth) + 127 (feature). The file was born from an intentional refactoring of `daemon_test.go` into focused test files — the opposite of accretion.

### Finding 2: Post-birth growth is modest and net-shrinking

After the initial extraction:
- Feb 28: +127 lines (workspace selection disambiguation tests — genuine feature tests)
- Mar 3: -43 lines (removal of untracked beads ID logic)

Net post-birth change: +84 lines. At this rate, the file would take many months to approach even the advisory threshold.

### Finding 3: Test organization is clean

7 test functions organized by the functions they test:

| Test | Function Tested |
|---|---|
| `TestIssue_HasLabel` | `Issue.HasLabel` |
| `TestConvertBeadsIssues_Empty` | `convertBeadsIssues` (empty) |
| `TestConvertBeadsIssues_ConvertsAllFields` | `convertBeadsIssues` (full) |
| `TestConvertBeadsIssues_MultipleIssues` | `convertBeadsIssues` (multi) |
| `TestExtractBeadsIDFromSessionTitle` | `extractBeadsIDFromSessionTitle` |
| `TestGetClosedIssuesBatch_EmptyInput` | `GetClosedIssuesBatch` (empty) |
| `TestGetClosedIssuesBatch_Integration` | `GetClosedIssuesBatch` (integration) |
| `TestFindWorkspaceForIssue_NoWorkspaceDir` | `findWorkspaceForIssue` (missing dir) |
| `TestFindWorkspaceForIssue_PrefersSynthesis` | `findWorkspaceForIssue` (SYNTHESIS pref) |
| `TestFindWorkspaceForIssue_PrefersNewestWhenNoSynthesis` | `findWorkspaceForIssue` (newest) |
| `TestFindWorkspaceForIssue_SynthesisBeatsNewer` | `findWorkspaceForIssue` (SYNTHESIS wins) |
| `TestPickBestWorkspacePath` | `pickBestWorkspacePath` |
| `TestReadSpawnTime` | `readSpawnTime` |

No duplication. No shared test helpers that warrant extraction. Standard Go table-driven test patterns.

### Finding 4: File is well under thresholds

- Current: 330 lines
- Advisory threshold (800 lines): 470 lines headroom
- Critical threshold (1,500 lines): 1,170 lines headroom

### Finding 5: File was itself the result of extraction

The commit message (`refactor: extract daemon_test.go into focused test files`) shows this file was created as part of an extraction effort — the exact remedy the hotspot system recommends. Flagging the output of extraction as a hotspot is ironic — the system is alarming on its own cure.

## Test Performed

Verified git history with `git log --numstat --follow` to confirm per-commit additions/deletions. Confirmed 246 of 373 lines (66%) came from the extraction commit, and the remaining 127 lines were a single feature addition.

## Conclusion

**False positive.** The +373 lines/30d hotspot flag is primarily birth churn from an intentional extraction of `daemon_test.go` into focused test files (Feb 24). Post-birth growth is a modest net +84 lines across two commits. The file is well-organized at 330 lines with 1,170 lines of headroom before the critical threshold. No extraction needed. No architect follow-up needed.
