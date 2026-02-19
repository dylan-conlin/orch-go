# Probe: Daemon cleanup after pkg/cleanup deletion

**Status:** Complete

## Question

Does daemon periodic cleanup still execute after pkg/cleanup removal, and does it close stale tmux windows when due?

## What I Tested

- `go test ./pkg/daemon -run TestRunPeriodicCleanupRunsWhenDue`
- `go test ./pkg/daemon -run TestRunPeriodicCleanup`

## What I Observed

- Before fix: `TestRunPeriodicCleanupRunsWhenDue` failed with "RunPeriodicCleanup should call cleanup func once, got 0" (cleanup no-op).
- After fix: `TestRunPeriodicCleanup` passed.

## Model Impact

Extends daemon-autonomous-operation: daemon cleanup must still run to close stale tmux windows even when OpenCode handles session TTL, and RunPeriodicCleanup should execute when due and update lastCleanup.
