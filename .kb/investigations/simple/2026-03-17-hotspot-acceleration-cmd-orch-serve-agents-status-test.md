---
title: "Hotspot acceleration: cmd/orch/serve_agents_status_test.go false positive"
status: Complete
created: 2026-03-17
beads_id: orch-go-phnzp
---

**TLDR:** `cmd/orch/serve_agents_status_test.go` hotspot is a false positive — file was born 2026-02-18 via a deliberate test split (orch-go-1065), and its entire 243-line existence is counted as 30-day growth. At 243 lines, no extraction is warranted.

## D.E.K.N. Summary

- **Delta:** Confirmed false positive. File born 27 days ago as part of a deliberate test file split.
- **Evidence:** `git log --follow` shows single creation commit `c3b2595d4` on 2026-02-18 ("feat: split serve_agents tests by area"). File has not grown since creation — entire 243 lines are from the initial split.
- **Knowledge:** Same false positive pattern as 6+ prior investigations (ooda.go, publish.go, serve_daemon_actions.go, kb_ask_test.go, etc.). Hotspot detector counts file-birth additions as "growth" when file was created within the 30-day window.
- **Next:** No action needed. File is 243 lines — well below 1,500-line accretion threshold. Contains well-structured table-driven tests for `checkWorkspaceSynthesis` and `determineAgentStatus`.

## What I Tested

```bash
# Check git history — file was created in a single commit
git log --oneline --follow cmd/orch/serve_agents_status_test.go
# Output: c3b2595d4 feat: split serve_agents tests by area (orch-go-1065)

# Check file size
wc -l cmd/orch/serve_agents_status_test.go
# Output: 243

# Check source file size (the code being tested)
wc -l cmd/orch/serve_agents_status.go
# Output: 297
```

## What I Observed

1. File was created on 2026-02-18 in commit `c3b2595d4` as part of a deliberate refactoring to split `serve_agents` tests by area (issue orch-go-1065)
2. No subsequent commits — the entire 243-line file is from the initial creation
3. The hotspot detector counted all 243 lines as "30-day growth" because the file was born within the 30-day window
4. Source file `serve_agents_status.go` is only 297 lines — both files are healthy size
5. Tests are well-structured: table-driven `TestDetermineAgentStatus` with 15 cases, plus focused tests for `checkWorkspaceSynthesis`

## Conclusion

**False positive.** The file was created as part of a healthy test-splitting refactor. It has not grown since creation. Both the test file (243 lines) and its source file (297 lines) are well below any hotspot threshold. No extraction needed.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-serve-daemon-actions.md | Same pattern | yes | - |
