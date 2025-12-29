<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Implemented PostToolUse hook extension that logs Read/Glob/Grep outcomes to ~/.orch/action-log.jsonl for behavioral pattern detection.

**Evidence:** Tests pass (9 test cases), hook correctly logs error/empty/success outcomes, orch action command works for CLI logging.

**Knowledge:** Hook-based action logging enables detection of futile action patterns (e.g., repeatedly reading SYNTHESIS.md on light-tier agents) that were previously ephemeral.

**Next:** Complete - hook is active, patterns will be surfaced via `orch patterns`.

---

# Investigation: PostToolUse Hook Extension for Logging

**Question:** How should we extend PostToolUse hooks to log action outcomes (success/empty/error) for behavioral pattern detection?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent (orch-go-zjed)
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

**Supersedes:** None (follow-up implementation from orch-go-eq8k investigation)

---

## Findings

### Finding 1: Hook Input Format

**Evidence:** PostToolUse hooks receive JSON with:
- `tool`: Tool name (Read, Glob, Grep, Bash, etc.)
- `tool_input`: Tool parameters (filePath, pattern, etc.)
- `tool_result`: Result with `content` and `is_error` fields
- `session_id`: OpenCode session identifier
- `cwd`: Current working directory

**Source:** ~/.claude/hooks/post-tool-use.sh, ~/.orch/hooks/check-workspace-complete.py

**Significance:** Hooks can determine outcome by examining tool_result.is_error and tool_result.content.

---

### Finding 2: Existing Action Package

**Evidence:** pkg/action/ already implements:
- ActionEvent struct with Tool, Target, Outcome, ErrorMessage fields
- Logger for appending to ~/.orch/action-log.jsonl
- Tracker for loading and analyzing patterns
- FindPatterns() to detect recurring futile actions (3+ occurrences)

**Source:** pkg/action/action.go:41-69 (ActionEvent), pkg/action/action.go:163-194 (Logger.Log)

**Significance:** No new package needed - just add CLI command and hook to use existing infrastructure.

---

### Finding 3: Hook Configuration

**Evidence:** ~/.claude/settings.json uses PostToolUse hooks with matcher patterns:
- "Bash" -> post-tool-use.sh
- "Edit|Write" -> check-workspace-complete.py
- Added: "Read|Glob|Grep" -> log-tool-outcomes.py

**Source:** ~/.claude/settings.json:114-145

**Significance:** Adding new hook is just a configuration change - no code changes to Claude/OpenCode needed.

---

## Synthesis

**Key Insights:**

1. **Minimal implementation** - All infrastructure existed, just needed CLI entry point and hook script.

2. **Silent logging** - Hook runs in background, doesn't produce output (keeps tool execution fast).

3. **Outcome detection** - Hook examines tool_result.is_error and content to classify as success/empty/error.

**Answer to Investigation Question:**

Extended PostToolUse hooks by:
1. Creating `orch action log` command for hook-based logging
2. Creating ~/.orch/hooks/log-tool-outcomes.py that detects outcomes and calls orch action log
3. Adding hook configuration for Read|Glob|Grep tools in ~/.claude/settings.json

Patterns will now be detected and surfaced via `orch patterns`.

---

## Structured Uncertainty

**What's tested:**

- ✅ orch action log command logs to ~/.orch/action-log.jsonl (9 test cases pass)
- ✅ Hook correctly classifies error outcome (tested with JSON input)
- ✅ Hook correctly classifies empty outcome (tested with empty array input)
- ✅ Settings.json is valid JSON after modification (jq parse succeeds)

**What's untested:**

- ⚠️ Hook performance under high tool volume (not benchmarked)
- ⚠️ Real-world pattern detection accuracy (requires live usage)
- ⚠️ Whether orchestrators will act on surfaced patterns

**What would change this:**

- Finding would be wrong if hook causes noticeable latency (would need optimization)
- Finding would be wrong if pattern detection produces too many false positives

---

## Implementation Details

**Files created:**
- cmd/orch/action.go - CLI command for action logging
- cmd/orch/action_test.go - Tests for action command
- ~/.orch/hooks/log-tool-outcomes.py - PostToolUse hook for Read/Glob/Grep

**Files modified:**
- cmd/orch/main.go - Added rootCmd.AddCommand(actionCmd)
- ~/.claude/settings.json - Added Read|Glob|Grep hook configuration

**Success criteria:**
- ✅ orch action log --tool Read --target "/path" --outcome empty works
- ✅ orch action summary shows event count
- ✅ orch patterns includes futile action patterns

---

## References

**Files Examined:**
- /Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go - Existing action package
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go - Patterns command
- ~/.claude/settings.json - Hook configuration
- ~/.claude/hooks/post-tool-use.sh - Existing PostToolUse hook

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md - Root cause analysis that led to this work

---

## Investigation History

**2025-12-28 21:37:** Investigation started
- Initial question: How to extend PostToolUse hooks for action outcome logging?
- Context: Follow-up from orch-go-eq8k investigation on self-correction mechanisms

**2025-12-28 21:45:** Implementation complete
- Created orch action command, hook script, updated configuration
- All tests passing
