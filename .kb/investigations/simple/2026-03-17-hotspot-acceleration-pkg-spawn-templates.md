---
title: "Hotspot acceleration: pkg/spawn/templates.go"
status: Complete
date: 2026-03-17
beads: orch-go-r905c
---

## TLDR

`pkg/spawn/templates.go` hotspot alert (+367 lines/30d, now 367 lines) is a **false positive** — 100% of the growth is from a single extraction commit on 2026-03-09 that moved embedded templates from `context.go` into their own file. Zero post-birth modifications. The file is ~90% string constants (template content) with only ~40 lines of actual Go logic. No extraction needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/spawn/templates.go`
- **Evidence:** Git history shows exactly 1 commit (763d1f62b, 2026-03-09): "refactor: extract embedded templates from context.go to templates.go (orch-go-jnio7)". 367 lines added, 0 deleted, 0 subsequent changes.
- **Knowledge:** This file was itself the *product* of a prior extraction — templates were moved out of `context.go` to reduce that file's hotspot pressure. Flagging the extraction target as a new hotspot is a known false-positive pattern (creation churn). The file composition is ~90% string constants (DefaultSynthesisTemplate, DefaultFailureReportTemplate, generateFailureReport template), making line count a poor proxy for complexity.
- **Next:** Close. Same false-positive pattern as control_cmd.go (orch-go-ru9ux) and thread_test.go (orch-go-8801f) — file creation masquerading as dangerous growth.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Same pattern — creation-churn false positive | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-thread-thread-test.md | Same pattern — creation-churn false positive | yes | - |

## Question

Is `pkg/spawn/templates.go` (+367 lines/30d, now 367 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: 100% birth churn

**What I tested:** `git log --format="%h %as %s" --numstat --follow -- pkg/spawn/templates.go`

**What I observed:** Exactly one commit:
```
763d1f62b 2026-03-09 refactor: extract embedded templates from context.go to templates.go (orch-go-jnio7)
367	0	pkg/spawn/templates.go
```

All 367 lines were added in a single extraction commit. Zero subsequent modifications.

### Finding 2: File composition is ~90% template strings

**What I tested:** Manual code review of file structure.

**What I observed:**
| Section | Lines | Type |
|---|---|---|
| `EnsureSynthesisTemplate()` | 13-33 (20 lines) | Go logic |
| `EnsureFailureReportTemplate()` | 38-58 (20 lines) | Go logic |
| `WriteFailureReport()` | 62-71 (10 lines) | Go logic |
| `generateFailureReport()` | 74-156 (82 lines) | Template string |
| `DefaultSynthesisTemplate` const | 160-281 (121 lines) | Template string |
| `DefaultFailureReportTemplate` const | 285-367 (82 lines) | Template string |

~40 lines of actual Go logic, ~327 lines of string constants. Line count is a poor complexity proxy for template files.

### Finding 3: File was product of prior extraction

The commit message explicitly says "extract embedded templates from context.go to templates.go" — this file exists *because* an extraction already happened. The hotspot detector is flagging the extraction destination, not identifying new accretion risk.

## Test Performed

```bash
git log --format="%h %as %s" --numstat --follow -- pkg/spawn/templates.go
# Result: 1 commit, 367 additions, 0 deletions, 0 post-birth changes
```

## Conclusion

**False positive.** The 367-line growth metric equals the file's entire existence from a single extraction commit. The file is well-structured, almost entirely template strings, and shows zero organic growth. No extraction is needed now and the file is unlikely to become a critical hotspot since template content rarely grows incrementally.
