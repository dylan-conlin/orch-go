## Summary (D.E.K.N.)

**Delta:** Added `--backend claude --inline` support for interactive orchestrator sessions with Claude Code CLI running directly in current terminal.

**Evidence:** Successfully implemented and tested inline mode - claude CLI runs blocking in current terminal with SPAWN_CONTEXT.md piped in.

**Knowledge:** Inline mode requires reordering spawn flow to check backend first, then apply mode within backend-specific spawn functions.

**Next:** Implementation complete.

**Promote to Decision:** recommend-no (tactical enhancement, not architectural)

---

# Investigation: Add Backend Claude Inline Support

**Question:** How to add `--backend claude --inline` support for interactive orchestrator sessions?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current spawn flow checks inline before backend

**Evidence:** In `runSpawnWithSkillInternal` (spawn_cmd.go:1311-1319):
```go
if inline {
    return runSpawnInline(...)  // Always uses opencode
}
if cfg.SpawnMode == "claude" {
    return runSpawnClaude(...)  // Only tmux mode
}
```

**Source:** cmd/orch/spawn_cmd.go:1311-1324

**Significance:** `--inline` is checked FIRST before backend mode, so `--backend claude --inline` would use opencode, not claude.

---

### Finding 2: SpawnClaude in pkg/spawn/claude.go only supports tmux

**Evidence:** Function creates tmux window and sends claude command to it:
```go
func SpawnClaude(cfg *Config) (*tmux.SpawnResult, error) {
    // Creates tmux session and window
    // Sends: cat SPAWN_CONTEXT.md | claude --dangerously-skip-permissions
}
```

**Source:** pkg/spawn/claude.go:12-70

**Significance:** Need separate inline function that runs claude directly in current terminal without tmux.

---

### Finding 3: Claude CLI uses piped input, not --file flag

**Evidence:** Existing code uses `cat SPAWN_CONTEXT.md | claude` because claude has no --file flag.

**Source:** pkg/spawn/claude.go:55

**Significance:** Inline mode must also pipe context file content to claude's stdin.

---

## Implementation Approach

1. Reorder spawn flow: check backend mode first, then handle inline within each backend
2. Add `SpawnClaudeInline` function in pkg/spawn/claude.go
3. Update `runSpawnClaude` to accept inline parameter or create separate inline handler

---

## References

**Files Examined:**
- cmd/orch/spawn_cmd.go - Main spawn command logic
- pkg/spawn/claude.go - Claude backend implementation
