---
title: "Hotspot acceleration: pkg/thread/thread_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-8801f
---

## TLDR

`pkg/thread/thread_test.go` hotspot alert (+570 lines/30d) is a **false positive** — 100% of the growth is from file creation on 2026-03-05 (513 lines initial + 57 lines same-day fix). Zero growth since. File is 570 lines with healthy 1:1 test:production ratio. No extraction needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/thread/thread_test.go`
- **Evidence:** Git history shows exactly 2 commits (both 2026-03-05): initial creation (513 lines) and same-day bugfix (57 lines). No subsequent changes.
- **Knowledge:** Hotspot metric flags file-birth events as "growth" — new test files created alongside new packages will always trigger this alert. The 570-line metric equals the file's entire existence, not incremental bloat.
- **Next:** Close. This is the same false-positive pattern as control_cmd.go (orch-go-ru9ux) — file creation masquerading as dangerous growth.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Same pattern — creation-churn false positive | yes | - |

## Question

Is `pkg/thread/thread_test.go` (+570 lines/30d, now 570 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: Growth is entirely from file creation

Git log shows exactly 2 commits:

```
ca5c6e39e 2026-03-05 feat: implement core thread package and CLI commands (orch-go-8bfwz) [+513 lines]
2406231be 2026-03-05 fix: orch thread append errors on nonexistent slug (orch-go-80q0v) [+57 lines]
```

The +570 lines/30d metric equals the file's total size — it was born at 570 lines and has been unchanged since. This is creation, not accretion.

### Finding 2: Test:production ratio is healthy

| File | Lines |
|---|---|
| `pkg/thread/thread.go` | 566 |
| `pkg/thread/thread_test.go` | 570 |
| **Ratio** | **1.007:1** |

A ~1:1 test:production ratio is normal and healthy for a Go package.

### Finding 3: Test organization is clean

12 test functions, each covering a single exported function:

| Test | Function Tested |
|---|---|
| `TestCreateOrAppend_NewThread` | `CreateOrAppend` (new) |
| `TestCreateOrAppend_AppendNewDate` | `CreateOrAppend` (append, new date) |
| `TestCreateOrAppend_AppendSameDate` | `CreateOrAppend` (append, same date) |
| `TestList` | `List` |
| `TestShow` | `Show` |
| `TestShow_NotFound` | `Show` (error case) |
| `TestResolve` | `Resolve` |
| `TestTodaysEntries` | `TodaysEntries` |
| `TestActiveThreads` | `ActiveThreads` |
| `TestAppend_NonexistentSlug` | `Append` (error case) |
| `TestAppend_ExistingThread` | `Append` |
| `TestSlugify` | `Slugify` |
| `TestParseThread` | `ParseThread` |

No test duplication. No test helpers that should be extracted. Standard Go test patterns.

### Finding 4: File is well under thresholds

- Current: 570 lines
- Advisory threshold (800 lines): 230 lines headroom
- Critical threshold (1,500 lines): 930 lines headroom

Even if the thread package doubles in features, the test file would remain under advisory threshold.

## Test Performed

Verified git history with `git log --diff-filter=AM` and `git diff --stat` per commit to confirm all 570 lines came from creation, not incremental growth.

## Conclusion

**False positive.** The +570 lines/30d hotspot flag is entirely from file creation on 2026-03-05. The file has zero post-creation growth, a healthy 1:1 test:production ratio, clean organization, and substantial headroom below advisory thresholds. No extraction needed. No architect follow-up needed.
