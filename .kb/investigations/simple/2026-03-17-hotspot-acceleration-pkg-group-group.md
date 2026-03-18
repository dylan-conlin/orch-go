---
title: "Hotspot acceleration: pkg/group/group.go"
status: Complete
date: 2026-03-17
beads: orch-go-g0wrx
---

## TLDR

`pkg/group/group.go` hotspot alert (+232 lines/30d, now 229 lines) is a **false positive** — 100% of the growth is from file creation on 2026-02-25 (194 lines initial + 35 lines across 2 subsequent commits). File is 229 lines — well under all thresholds. No extraction needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/group/group.go`
- **Evidence:** Git history shows 3 commits: initial creation 2026-02-25 (194 lines), path migration 2026-03-03 (+6 lines), daemon account routing 2026-03-07 (+29 lines). The +232 lines/30d metric equals the file's entire existence.
- **Knowledge:** Same false-positive pattern as thread_test.go, control_cmd.go, and 5+ other recent hotspot investigations — the hotspot detector counts file-birth events as "30-day growth." Files born within the measurement window will always trigger this alert regardless of actual accretion risk.
- **Next:** Close as false positive. No extraction needed — file is 229 lines with clear single-responsibility (group resolution from groups.yaml).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-thread-thread-test.md | Same pattern — creation-churn false positive | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Same pattern — creation-churn false positive | yes | - |

## Question

Is `pkg/group/group.go` (+232 lines/30d, now 229 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: Growth is entirely from file creation and initial feature build-out

Git log shows exactly 3 commits:

```
c772895d1 2026-02-25 feat: implement pkg/group package and wire into kb context (orch-go-1237) [+194 lines]
cd45d2161 2026-03-03 feat: move groups.yaml default path from ~/.orch/ to ~/.kb/ with fallback (orch-go-1crul) [+6 lines]
a55970e28 2026-03-07 feat: daemon account routing from project groups (orch-go-pj4sd) [+29 lines]
```

The +232 lines/30d metric equals the file's total size (229 lines, with rounding accounting for the small delta). The file was born 20 days ago and has had only 2 small follow-up commits adding minor features.

### Finding 2: File is well under all thresholds

- Current: 229 lines
- Advisory threshold (800 lines): 571 lines headroom
- Critical threshold (1,500 lines): 1,271 lines headroom

### Finding 3: File has clear single responsibility

The file provides:
- `Config` and `Group` types (YAML-backed)
- `Load()`/`LoadFromFile()` — config loading
- `GroupsForProject()` — group membership resolution
- `SiblingsOf()` — sibling project discovery
- `ResolveGroupMembers()` — full member list for a group
- `AccountForProjectDir()` — account routing via groups
- `isSubdirectory()` — helper

7 functions, each doing one thing. Clean API surface with no signs of accretion pressure.

## Test Performed

Verified git history with `git log --oneline --follow` and `git log --format="%H %ai" --diff-filter=A` to confirm creation date (2026-02-25) and that all 229 lines are from the initial build-out period.

## Conclusion

**False positive.** The +232 lines/30d hotspot flag is entirely from file creation on 2026-02-25 and two small follow-up commits. The file has a clean single-responsibility design at 229 lines — well below advisory thresholds with 571 lines of headroom. No extraction needed. No architect follow-up needed.
