# Session Synthesis

**Agent:** og-debug-orchestrator-agents-task-27feb-4bdf
**Issue:** orch-go-2crn
**Outcome:** success

---

## Plain-Language Summary

Orchestrator agents were using Claude Code's Task tool to spawn workers instead of `orch spawn` or `bd create`, bypassing beads tracking, dashboard visibility, and completion verification. The root cause is a 17:1 signal imbalance: Claude Code's system prompt aggressively promotes Task tool (~500 words) while the orchestrator skill has ~30 words prohibiting it. The fix adds a PreToolUse hook (`gate-orchestrator-task-tool.py`) that blocks Task tool at runtime when `CLAUDE_CONTEXT=orchestrator`, providing defense-in-depth alongside the existing `--disallowedTools` CLI flag already in `pkg/spawn/claude.go`.

## TLDR

Added a PreToolUse hook that blocks Task tool for orchestrator sessions as defense-in-depth. The `--disallowedTools` CLI flag was already implemented in `pkg/spawn/claude.go` (Layer 1, spawn-time). The new hook (Layer 2, runtime) catches edge cases where the CLI flag isn't set. Both layers verified working.

---

## Delta (What Changed)

### Files Created
- `~/.orch/hooks/gate-orchestrator-task-tool.py` - PreToolUse hook that blocks Task tool when CLAUDE_CONTEXT is orchestrator or meta-orchestrator

### Files Modified
- `~/.claude/settings.json` - Added PreToolUse matcher for "Task" tool pointing to the new hook

### Commits
- (pending - will commit before Phase: Complete)

---

## Evidence (What Was Observed)

- `--disallowedTools` flag already exists in `pkg/spawn/claude.go:92-96` and is confirmed working via `--dangerously-skip-permissions` (verified via GitHub issue #12232 and Claude Code docs)
- The CLI flag format `--disallowedTools 'Task,Edit,Write,NotebookEdit'` is correct — Claude CLI accepts comma-separated tool names
- No PreToolUse hook existed for the Task tool prior to this fix
- Existing hooks (`gate-orchestrator-code-access.py`, `gate-bd-close.py`) demonstrate the pattern works

### Tests Run
```bash
# Layer 1: Go tests pass for --disallowedTools
go test -run TestBuildClaudeLaunchCommand ./pkg/spawn/ -count=1 -v
# PASS: 14/14 tests including orchestrator/meta-orchestrator/worker cases

# Layer 2: Hook tests pass
# orchestrator + Task → DENIED ✅
# worker + Task → ALLOWED ✅
# meta-orchestrator + Task → DENIED ✅
# orchestrator + Bash → ALLOWED ✅
# no context + Task → ALLOWED ✅

# Build/vet
go build ./cmd/orch/  # PASS
go vet ./cmd/orch/    # PASS
```

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Hook correctly denies Task for orchestrator/meta-orchestrator contexts
- Hook correctly allows Task for worker/unset contexts
- Hook correctly ignores non-Task tools
- settings.json is valid JSON with new PreToolUse matcher

---

## Architectural Choices

### Defense-in-depth: hook alongside --disallowedTools
- **What I chose:** Add PreToolUse hook as Layer 2 defense, keeping existing `--disallowedTools` as Layer 1
- **What I rejected:** Relying solely on `--disallowedTools` CLI flag
- **Why:** The CLI flag only applies to orch-spawned sessions. Manually started orchestrator sessions (where CLAUDE_CONTEXT might be set via hooks but `--disallowedTools` isn't) would still have Task available. The hook catches both paths.
- **Risk accepted:** Minor runtime overhead (hook runs on every Task tool call, but exits fast for non-orchestrator contexts)

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `--disallowedTools` DOES work correctly with `--dangerously-skip-permissions` (blacklist approach respected; whitelist `--allowedTools` is ignored)
- Sub-agents spawned via Task tool do NOT inherit deny rules from parent settings.json
- Hooks run BEFORE the permission system, making them the most reliable enforcement layer

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (Go tests + hook tests)
- [x] Ready for `orch complete orch-go-2crn`

---

## Unexplored Questions

- Whether orchestrator sessions spawned via the OpenCode API (headless) path could bypass both layers — currently moot since orchestrators use Claude CLI (tmux), but worth noting if headless orchestators are ever supported
- Whether the hook should also gate `TaskCreate`/`TaskUpdate`/`TaskList` tools (not just `Task`) — currently only `Task` is gated since it's the tool used for spawning subagents

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-orchestrator-agents-task-27feb-4bdf/`
**Beads:** `bd show orch-go-2crn`
