---
title: "Hotspot acceleration: cmd/orch/status_infra.go"
status: Complete
date: 2026-03-17
---

## TLDR

status_infra.go hotspot is a false positive — the file was born 2026-03-10 (281 lines) as an extraction from status_cmd.go, with one 27-line bug fix added Mar 13. The entire "+308 lines/30d" metric is the file's 7-day existence counted as growth.

## D.E.K.N. Summary

- **Delta:** status_infra.go growth is entirely from file creation via extraction. Not accretion.
- **Evidence:** git log shows 281-line birth on Mar 10 (commit 0f887fd56, extraction from status_cmd.go which shrank by 772 lines), then +27 lines on Mar 13 (commit 56313aa0f, cross-project beads prefix fix). Raw additions total 308 = current file size.
- **Knowledge:** Birth-from-extraction files produce inflated hotspot metrics. The file contains two cohesive groups: infrastructure health checking (lines 19-169) and beads project discovery utilities (lines 171-308). At 308 lines with clear responsibility boundaries, no extraction is needed.
- **Next:** No action needed. File is well below any threshold (308 vs 800 advisory, 1500 critical).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-daemon-pidlock.md | same pattern | yes | - |
| 2026-03-17-hotspot-acceleration-pkg-daemon-mock-test.md | same pattern | yes | - |

## Question

Is cmd/orch/status_infra.go a genuine accretion hotspot requiring extraction, or a false positive from recent file creation?

## Findings

### Finding 1: File was born within the 30-day window

The file was created on 2026-03-10 (7 days ago) as part of commit 0f887fd56:

```
refactor: extract review_done.go and review_helpers.go from review.go (orch-go-zwp48)
```

In that commit, status_cmd.go shrank by 772 lines and status_infra.go was created with 281 lines. This was a deliberate extraction to reduce status_cmd.go size.

### Finding 2: Only one subsequent modification

One commit on 2026-03-13 added `findIssueInAlternateProjects` (27 lines) — a targeted bug fix for cross-project beads prefix resolution. This is not accretion; it's a necessary feature addition.

### Finding 3: File has clear responsibility boundaries

The file contains two cohesive groups:
1. **Infrastructure health checking** (lines 19-169): types (`InfraServiceStatus`, `DaemonStatus`, `InfrastructureHealth`) and functions (`checkInfrastructureHealth`, `checkTCPPort`, `readDaemonStatus`, `printInfrastructureHealth`)
2. **Beads project discovery** (lines 171-308): functions for finding projects by beads prefix or name (`getBeadsIssuePrefix`, `getKBProjectsWithNames`, `findProjectByBeadsPrefix`, `findProjectDirByName`, `findIssueInAlternateProjects`)

These two groups are related (both used by status/completion commands) but could be separated if either grows significantly.

## Test Performed

```bash
git log --diff-filter=A --format="%h %ad %s" --date=short -- cmd/orch/status_infra.go
# 0f887fd56 2026-03-10 refactor: extract review_done.go and review_helpers.go from review.go (orch-go-zwp48)

git log --format="%h %ad %s" --date=short -- cmd/orch/status_infra.go
# 56313aa0f 2026-03-13 fix: cross-project orch complete handles shared beads prefix via fallback search
# 0f887fd56 2026-03-10 refactor: extract review_done.go and review_helpers.go from review.go (orch-go-zwp48)

wc -l cmd/orch/status_infra.go
# 308

git show 0f887fd56 --stat -- cmd/orch/status_infra.go cmd/orch/status_cmd.go
# status_cmd.go     | 772 +---------------------------------------------
# status_infra.go   | 281 +++++++++++++++++
```

## Conclusion

**False positive.** The file was born from extraction 7 days ago. Its entire 308-line count is being measured as "30-day growth." At 308 lines with two cohesive responsibility areas, the file is healthy and well below any extraction threshold.
