# Probe: orch clean --sessions kills daemon-spawned Claude CLI tmux windows

**Status:** Complete

**Date:** 2026-02-24

**Question:** Does `cleanStaleTmuxWindows` correctly protect daemon-spawned Claude CLI tmux windows that have no OpenCode session but an active beads issue?

## What I Tested

1. **Read `cleanStaleTmuxWindows()` in `cmd/orch/clean_cmd.go`**
   - Function now accepts `projectDir` parameter and calls `verify.ListOpenIssuesWithDir(projectDir)` (line 480)
   - Builds `openIssues` map of all open/in_progress/blocked beads issues
   - Passes both `activeBeadsIDs` (from OpenCode) and `openIssues` (from beads) to `classifyTmuxWindows()`

2. **Verified `classifyTmuxWindows()` pure function (lines 413-446)**
   - Three-layer check: (1) active OpenCode session → not stale, (2) open beads issue → protected, (3) neither → stale
   - Returns count of protected windows for visibility in logs

3. **Ran all tests**
   ```bash
   go test ./cmd/orch/ -run TestClassifyTmuxWindows -v
   ```
   All 8 subtests pass, including explicit "daemon-spawned Claude CLI agent scenario" test case

4. **Verified build**
   ```bash
   go build ./cmd/orch/ && go vet ./cmd/orch/
   ```
   Clean build, no vet warnings

## What I Observed

- Fix was committed as `b88cd9db3` ("fix: protect active daemon-spawned tmux windows from orch clean --sessions (orch-go-1221)")
- Extracted `classifyTmuxWindows` pure function for testability
- Used `ListOpenIssuesWithDir(projectDir)` which correctly uses `--limit 0` (avoids 50-issue hidden limit in `ListOpenIssues()`)
- Test coverage includes: OpenCode-only protection, beads-only protection (daemon scenario), stale with neither, mixed, non-agent windows, windows without beads ID

## Model Impact

**Extends** Session Deletion Vectors model:

- **New protection layer confirmed:** `cleanStaleTmuxWindows` now has dual-authority checking (OpenCode sessions + beads issue status). This closes the vector where daemon-spawned Claude CLI agents were killed because they had no OpenCode session.
- **Invariant update:** Critical Invariant 1 ("Sessions exist in SQLite or they don't") is now supplemented: for Claude CLI agents, beads issue status is the authoritative liveness signal, not OpenCode session existence.
- **`ListOpenIssuesWithDir` is the correct API:** `ListOpenIssues()` has a hidden 50-issue FallbackList limit; `ListOpenIssuesWithDir()` correctly uses `--limit 0` for complete results. This distinction matters for projects with many active issues.
