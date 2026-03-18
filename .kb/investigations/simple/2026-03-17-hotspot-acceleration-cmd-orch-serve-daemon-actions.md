---
title: "Hotspot acceleration: cmd/orch/serve_daemon_actions.go false positive"
status: Complete
created: 2026-03-17
beads_id: orch-go-gu0wg
---

**TLDR:** `cmd/orch/serve_daemon_actions.go` hotspot is a false positive — file was born 2026-02-27 as new handler code, and its entire 195-line existence is counted as 30-day growth. At 195 lines, no extraction is warranted.

## D.E.K.N. Summary

- **Delta:** Confirmed false positive. File born 18 days ago with new daemon action HTTP handlers.
- **Evidence:** `git log --diff-filter=A` shows creation commit `19226232f` on 2026-02-27 ("session cleanup — add new serve files"). File started at 106 lines, grew to 195 via batch close endpoint (Feb 27) and verification tracker revert (Mar 1).
- **Knowledge:** Same false positive pattern as 5+ prior investigations (ooda.go, publish.go, synthesis_auto_create_test.go, kb_ask_test.go). Hotspot detector counts file-birth additions as "growth" when file was created within the 30-day window.
- **Next:** No action needed. File is 195 lines — well below 1,500-line accretion threshold. Contains 3 clean HTTP handlers with clear separation of concerns.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| commit 3353f82c5 (ooda.go false positive) | same pattern | yes | - |
| commit 1e2955d9e (publish.go false positive) | same pattern | yes | - |
| commit 484d2b369 (synthesis_auto_create_test.go false positive) | same pattern | yes | - |

## Question

Is `cmd/orch/serve_daemon_actions.go` (+245 lines/30d, now 195 lines) a genuine hotspot requiring extraction?

## Findings

### Finding 1: File birth date

File was created on 2026-02-27 in commit `19226232f` ("session cleanup — archive workspaces, sync beads/kb, add new serve files"). Initial size: 106 lines.

### Finding 2: Growth is entirely initial creation + one feature

- Feb 27: Created at 106 lines (handleDaemonResume, handleCloseIssue)
- Feb 27: Added batch close endpoint (+89 lines → 195 lines)
- Mar 1: Verification tracker removed (-46), then reverted (+46) — net zero
- Mar 4: Minor refactor (2 lines changed) — beads.DefaultDir elimination

The "245 lines of growth" is an artifact of the detector summing all insertions (106 + 89 + 46 + 2 = 243 ≈ 245), even though some were reverted. The file's actual size has been stable at 195 lines since Feb 27.

### Finding 3: File structure is clean

The file contains 3 HTTP handlers with clear responsibilities:
1. `handleDaemonResume` (lines 21-41) — POST /api/daemon/resume
2. `handleCloseIssue` (lines 58-106) — POST /api/issues/close
3. `handleCloseIssueBatch` (lines 130-195) — POST /api/issues/close-batch

Each handler follows the same pattern: method check → decode → validate → call business logic → respond. No shared state, no complex logic.

## Test Performed

```bash
# Verified file birth date
git log --format="%H %ad %s" --date=short --follow -- cmd/orch/serve_daemon_actions.go
# → 19226232f 2026-02-27 (creation date within 30-day window)

# Verified line counts per commit
git log --follow --numstat -- cmd/orch/serve_daemon_actions.go
# → 106+0 (creation), then 46+2 (revert), 2+2 (refactor)

# Verified current size
wc -l cmd/orch/serve_daemon_actions.go
# → 195 lines

# Verified handler functions didn't exist before file creation
git grep -n "func handleDaemonResume\|func handleCloseIssue" 19226232f~1 -- cmd/orch/
# → no results (new code, not extraction)
```

## Conclusion

**False positive.** The hotspot detector counts the file's entire 195-line existence as "30-day growth" because it was born within the measurement window. The "+245 lines" metric is inflated by summing all insertions including a reverted change. The file is a clean 195-line module with 3 well-structured HTTP handlers — no extraction warranted.
