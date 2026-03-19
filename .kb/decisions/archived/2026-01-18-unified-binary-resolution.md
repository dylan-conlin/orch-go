# Decision: Unified Binary Resolution Pattern

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** 
**Supersedes:** ~/Documents/personal/orch-go/.kb/investigations/2026-01-18-design-opencode-binary-resolution.md
**Superseded-By:** 


**Date:** 2026-01-18
**Status:** Accepted
**Context:** Synthesized from investigation on inconsistent opencode binary resolution patterns

## Summary

orch-go must use a unified binary resolution utility (`pkg/binutil` or equivalent) that follows env var → PATH → known locations order. Never rely on PATH alone in orchestration contexts where minimal environments (launchd, daemon) are expected.

## The Problem

The codebase uses three inconsistent patterns for finding the opencode binary:

1. **OPENCODE_BIN env var** — Used in `pkg/opencode/client.go`, `pkg/tmux/tmux.go`, `cmd/orch/attach.go`
2. **Hardcoded in shell commands** — Inline "opencode" strings in shell exec
3. **PATH-only lookup** — `exec.Command("opencode", ...)` with no fallback

This inconsistency causes spawn failures in minimal PATH environments (launchd, daemon context) where `~/.bun/bin` is not inherited.

## The Decision

### Resolution Order

All binary resolution must follow this order:
1. **Environment variable** (e.g., `OPENCODE_BIN`) — User's explicit override
2. **PATH lookup** via `exec.LookPath` — Fast path for normal environments
3. **Known locations** — `$HOME/.bun/bin`, `$HOME/bin`, `$HOME/go/bin`, `$HOME/.local/bin`, `/usr/local/bin`, `/opt/homebrew/bin`

### Implementation Pattern

Create a common utility (or generalize `ResolveBdPath()` from `pkg/beads/client.go`) that:
- Takes binary name, env var name, and search paths
- Returns resolved absolute path or error listing all searched locations
- Is used by ALL binary resolution in the codebase (opencode, bd, etc.)

### Constraint

**Never use hardcoded binary names in shell commands.** Always interpolate the resolved path. Shell commands like `sh -c "opencode serve ..."` must become `sh -c "/resolved/path/to/opencode serve ..."`.

## Why This Design

### Proven Precedent

`pkg/beads/client.go:42-81` implements `ResolveBdPath()` using PATH → known locations. This pattern has worked reliably for the `bd` binary. Generalizing it provides consistency.

### Minimal PATH Is Expected

Orchestration contexts (launchd daemons, headless spawns, cron jobs) inherit minimal PATH. The resolution pattern must handle this as normal operation, not an edge case.

### Trade-offs Accepted

1. **Small refactor** — Migrating existing code to use unified utility (one-time cost)
2. **Extra os.Stat calls** — Checking 6 known locations adds ~microseconds (negligible)
3. **New package** — ~100 lines of code for `pkg/binutil` or extension to existing utilities

## Evidence

- **Investigation:** `.kb/investigations/2026-01-18-design-opencode-binary-resolution.md`
- **Existing pattern:** `pkg/beads/client.go:42-81` (`ResolveBdPath()`)
- **CLAUDE.md:** Documents PATH fix via `~/.bun/bin` symlinks as a known constraint
- **Failure mode:** "exec: 'opencode': executable file not found in $PATH" during headless spawns

## Implementation Status

**Not yet implemented.** The investigation's recommendation for `pkg/binutil` has not been created. Current workaround: `~/.bun/bin` symlinks + OPENCODE_BIN env var cover most cases, but inconsistent patterns remain.

## Related Decisions

- CLAUDE.md "CLI PATH Fix" section — Documents the symlink workaround
- `pkg/beads/client.go` `ResolveBdPath()` — Proven implementation of the pattern
