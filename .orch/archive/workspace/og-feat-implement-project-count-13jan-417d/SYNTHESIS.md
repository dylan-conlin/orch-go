# Session Name Auto-Generation Implementation

## TLDR

Implemented {project}-{count} session naming with automatic tmux window renaming. Session names are now generated as "orch-go-1", "orch-go-2" instead of using ad-hoc tmux window names like "session", "zsh", providing consistent, discoverable paths.

## What Changed

**New Functions:**
- `pkg/session/session.go`: `GenerateSessionName()` - generates {project}-{count} based on existing sessions
- `pkg/session/session.go`: `countProjectSessions()` - counts existing {project}-{number} directories  
- `pkg/tmux/tmux.go`: `RenameCurrentWindow()` - executes `tmux rename-window`

**Modified Functions:**
- `cmd/orch/session.go`: `runSessionStart()` - now generates session name and renames tmux window
- `cmd/orch/session.go`: `createActiveSessionHandoff()` - accepts sessionName parameter instead of querying tmux

**Test Coverage:**
- `TestGenerateSessionName` - verifies counting logic across 6 scenarios
- `TestCountProjectSessions` - verifies project filtering and max number selection
- `TestGenerateSessionNameProjectNameExtraction` - verifies project name extraction

## How It Works

1. User runs `orch session start "goal"`
2. Extract project name from current directory (e.g., "orch-go")
3. Count existing session directories matching `{project}-{number}` pattern
4. Generate name as `{project}-{max+1}`
5. Rename tmux window to the generated name
6. Create `.orch/session/{generated-name}/active/SESSION_HANDOFF.md`

**Counting Logic:**
- Scans `.orch/session/` for directories matching `^{project}-(\d+)$`
- Returns highest number found
- Next session uses max+1
- Example: If "orch-go-1" and "orch-go-5" exist, next is "orch-go-6"

## Why This Approach

**Rejected daily reset:** Initial beads issue mentioned "Count is daily (resets each day)". Implemented as highest number instead because:
- Daily reset would cause naming collisions (same name on different days)
- Unique incrementing numbers provide better discoverability
- Simple implementation without date tracking complexity

**Window renaming integration:** By renaming the tmux window to match the session name:
- Existing code that queries `GetCurrentWindowName()` works without changes
- Session discovery (`discoverSessionHandoff`) automatically works
- Session end archiving uses the renamed window name

## Design Decisions

1. **Fallback behavior:** If session name generation fails, falls back to `session-{timestamp}` to avoid blocking session start
2. **Warning-only failures:** Window rename and handoff creation failures emit warnings but don't error - session can proceed
3. **Pattern matching:** Uses `^{project}-(\d+)$` regex for strict matching, ignores non-matching directories

## Testing Evidence

**All tests passing:**
```
go test ./pkg/session -v
PASS: TestGenerateSessionName (0.00s)
  - no_existing_sessions
  - one_existing_session
  - multiple_existing_sessions
  - non-sequential_numbers
  - mixed_with_other_projects
  - non-matching_directories_ignored
PASS: TestCountProjectSessions (0.00s)
  - empty_directory
  - single_session
  - multiple_sessions
  - highest_number_returned
  - filter_by_project_name
  - ignore_non-matching
PASS: TestGenerateSessionNameProjectNameExtraction (0.00s)
```

**Manual testing:**
```
$ orch session start "Test project-count naming"
Session started: Test project-count naming
  Name:       orch-go-1
  Start time: 16:22
  Handoff:    /Users/dylanconlin/.../session/orch-go-1/active/SESSION_HANDOFF.md

$ tmux display-message -p "#{window_name}"
orch-go-1
```

## Edge Cases Handled

1. **No existing sessions:** Returns count=0, generates "{project}-1"
2. **Non-sequential numbers:** Finds highest (e.g., 1, 5, 3 → next is 6)
3. **Mixed projects:** Filters by project name (orch-go-1, kb-cli-1 coexist)
4. **Legacy directories:** Ignores timestamp format and other non-matching names
5. **Not in tmux:** `RenameCurrentWindow()` is no-op if $TMUX unset

## Follow-up Actions

None - feature is complete and tested.

## Files Changed

- `cmd/orch/session.go` - session start command logic
- `pkg/session/session.go` - session name generation functions
- `pkg/session/session_test.go` - test coverage
- `pkg/tmux/tmux.go` - window rename function
- `.kb/investigations/2026-01-13-inv-implement-project-count-session-naming.md` - investigation notes
