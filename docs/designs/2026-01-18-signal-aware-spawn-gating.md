# Signal-Aware Spawn Gating

**Date:** 2026-01-18  
**Status:** Approved for Implementation  
**Phase:** Design

## Problem

The daemon spawns duplicate agents for issues that have already completed but are awaiting orchestrator review. This happens because:

1. SpawnedIssueTracker TTL expires (5min for tests, 6h for prod) but agent is still running
2. Status update to "in_progress" fails silently
3. Agent completes and reports "Phase: Complete" but beads status remains "open"

This creates unnecessary duplicate work and wastes agent slots.

## Solution Overview

Add **completion signal detection** before spawning. Check for three signals that indicate work is complete:

1. **Recent "Phase: Complete" comment** in beads
2. **SYNTHESIS.md file** in workspace
3. **Commits with beads ID** in git history

If any signal found AND session <6h old, skip spawn (agent is completed, awaiting review).

## Architecture

### New Function: CheckCompletionSignals

```go
// CheckCompletionSignals checks if an issue has completion signals that indicate
// the work is already done and awaiting review.
//
// Returns true if ANY of these signals are found within the last 6 hours:
// 1. Recent "Phase: Complete" comment in beads
// 2. SYNTHESIS.md file in workspace
// 3. Commits containing the beads ID
//
// This prevents duplicate spawns when:
// - SpawnedIssueTracker TTL expires but agent is done
// - Status update failed but work is complete
// - Daemon restarts with completed-awaiting-review work in queue
func CheckCompletionSignals(issueID string, projectDir string) (bool, string, error)
```

**Return values:**
- `bool`: true if completion signal found
- `string`: reason message for logging/debugging
- `error`: error if checks failed (treat as "no signal" and continue)

### Integration Point

Call `CheckCompletionSignals` in `OnceExcluding()` after session-level dedup check (line 779), before acquiring slot:

```go
// Session-level dedup: Check if there's an existing OpenCode session
if HasExistingSessionForBeadsID(issue.ID) {
    // ... existing code ...
}

// NEW: Check for completion signals before acquiring slot
if hasSignal, reason, err := CheckCompletionSignals(issue.ID, getProjectDir()); err == nil && hasSignal {
    if d.Config.Verbose {
        fmt.Printf("  DEBUG: Skipping %s (completion signal detected: %s)\n", issue.ID, reason)
    }
    return &OnceResult{
        Processed: false,
        Issue:     issue,
        Skill:     skill,
        Message:   fmt.Sprintf("Completion signal detected: %s", reason),
    }, nil
}

// If pool is configured, acquire a slot first
var slot *Slot
// ... existing code ...
```

### Signal Checks

#### 1. Phase: Complete Comment

```go
// Use beads.DefaultClient() to get comments
comments, err := beads.DefaultClient().Comments(issueID)
if err != nil {
    return false, "", fmt.Errorf("failed to get comments: %w", err)
}

for _, comment := range comments {
    if strings.Contains(strings.ToLower(comment.Text), "phase: complete") {
        // Parse comment.CreatedAt and check if < 6h old
        commentTime, err := time.Parse(time.RFC3339, comment.CreatedAt)
        if err == nil && time.Since(commentTime) < 6*time.Hour {
            return true, fmt.Sprintf("Phase: Complete comment at %s", comment.CreatedAt), nil
        }
    }
}
```

#### 2. SYNTHESIS.md in Workspace

```go
// Check all workspaces matching the beads ID
pattern := filepath.Join(projectDir, ".orch/workspace/*-"+issueID+"*/SYNTHESIS.md")
matches, err := filepath.Glob(pattern)
if err != nil {
    return false, "", fmt.Errorf("failed to glob workspace: %w", err)
}

if len(matches) > 0 {
    // Check file modification time < 6h
    for _, match := range matches {
        stat, err := os.Stat(match)
        if err == nil && time.Since(stat.ModTime()) < 6*time.Hour {
            return true, fmt.Sprintf("SYNTHESIS.md found at %s", match), nil
        }
    }
}
```

#### 3. Commits with Beads ID

```go
// Check git log for recent commits containing the beads ID
since := time.Now().Add(-6 * time.Hour).Format("2006-01-02 15:04:05")
cmd := exec.Command("git", "log", "--since="+since, "--grep="+issueID, "--oneline")
cmd.Dir = projectDir
output, err := cmd.Output()
if err != nil {
    return false, "", fmt.Errorf("git log failed: %w", err)
}

if len(output) > 0 {
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    return true, fmt.Sprintf("%d commits found with beads ID", len(lines)), nil
}
```

## Edge Cases

### No Workspace Found
- Return `false, "", nil` (no signal, continue spawn)
- This is expected for issues that haven't been worked on yet

### Multiple Workspaces
- Check all matching workspaces
- Return true if ANY workspace has SYNTHESIS.md < 6h old
- Pattern: `.orch/workspace/*-{beadsID}*/SYNTHESIS.md`

### Failed Beads Comment Fetch
- Log error, return `false, "", err`
- Caller treats error as "no signal" and continues spawn
- Better to risk duplicate spawn than block all spawns

### Failed Git Log
- Log error, return `false, "", err`
- Caller treats error as "no signal" and continues spawn

### Session Age > 6h
- All checks include 6h time window
- Old completions don't trigger signal (assumed reviewed by now)
- If work completed >6h ago and still "open", orchestrator should have reviewed it

## Testing Strategy

### Unit Tests

1. **TestCheckCompletionSignals_PhaseComplete**
   - Mock beads client with Phase: Complete comment < 6h
   - Assert returns true with reason

2. **TestCheckCompletionSignals_SynthesisFile**
   - Create SYNTHESIS.md in test workspace < 6h old
   - Assert returns true with reason

3. **TestCheckCompletionSignals_GitCommit**
   - Mock git command with recent commits containing beads ID
   - Assert returns true with reason

4. **TestCheckCompletionSignals_NoSignals**
   - No comments, no SYNTHESIS.md, no commits
   - Assert returns false

5. **TestCheckCompletionSignals_OldSignals**
   - Phase: Complete comment > 6h ago
   - SYNTHESIS.md > 6h old
   - Assert returns false (expired signals)

6. **TestCheckCompletionSignals_MultipleWorkspaces**
   - Create multiple workspaces with beads ID
   - One has SYNTHESIS.md < 6h, one doesn't
   - Assert returns true (found in at least one)

7. **TestCheckCompletionSignals_BeadsError**
   - Mock beads client to return error
   - Assert returns false, logs error

### Integration Tests

Not required for Phase 1 - unit tests cover the functionality sufficiently.

## Success Criteria

1. No duplicate spawns for completed-awaiting-review work
2. All three signal types detected within 6h window
3. Edge cases handled gracefully (no workspace, multiple workspaces, errors)
4. Existing spawn flow unchanged when no signals present
5. Unit tests pass with >80% coverage

## Future Enhancements (Out of Scope)

- **Phase 2:** Add signal to beads status (e.g., "completed-awaiting-review" status)
- **Phase 3:** Dashboard UI showing completion signals
- **Phase 4:** Automatic orchestrator notification when signal detected

## Trade-offs

### False Negatives
Risk: Miss completion signal due to error in check → spawn duplicate
Mitigation: Errors are logged but don't block spawn (better to spawn than hang)

### False Positives
Risk: Detect old signal, block legitimate spawn
Mitigation: 6h time window ensures old signals expire

### Performance
Impact: 3 additional checks before spawn (beads API, filesystem, git command)
Mitigation: Only runs when passing all other filters (rate limit, capacity, etc.)

## Dependencies

- beads package: `Comments(id string) ([]Comment, error)`
- stdlib: `filepath.Glob`, `os.Stat`, `exec.Command`
- daemon package: Access to projectDir (via config or parameter)

## Rollout Plan

1. Implement CheckCompletionSignals in pkg/daemon/completion_signals.go
2. Add unit tests in pkg/daemon/completion_signals_test.go
3. Integrate into OnceExcluding() with verbose logging
4. Monitor daemon logs for signal detection messages
5. Verify no duplicate spawns in production over 48h period
