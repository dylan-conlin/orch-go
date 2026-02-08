# Process Census False Positives Investigation

**Date:** 2026-02-07
**Status:** Complete
**Issue:** orch-go-21475

## Problem Statement

The process census in operator health (`/api/operator-health`) is flagging legitimate launchd-managed processes as orphans because they have PPID=1. Specifically:
- overmind (process manager)
- tmux (terminal multiplexer)
- /System/Library/ processes (macOS system processes)

These are intentionally managed by launchd (PID 1 on macOS), not orch orphans.

## Findings

### Finding 1: isOrchRelatedProcess is too broad
**Source:** `cmd/orch/serve_operator_health.go:553-573`
**Evidence:**
```go
func isOrchRelatedProcess(command, args string) bool {
	text := strings.ToLower(command + " " + args)

	keywords := []string{
		"opencode",
		"orch",
		".orch",
		"run --attach",
		"overmind",  // ← FALSE POSITIVE: overmind is launchd-managed
		"vite",
		"beads",
	}
	// ...
}
```

**Significance:** The function includes "overmind" as a keyword, which matches overmind processes that are legitimately PPID=1 (managed by launchd for orch-dashboard). This causes false positives in the orphan detection.

### Finding 2: Process census checks all PPID=1 processes
**Source:** `cmd/orch/serve_operator_health.go:494-551`
**Evidence:**
```go
func listOrphanedOrchProcesses(limit int) ([]orphanProcessEntry, error) {
	// ... ps query ...
	for scanner.Scan() {
		// ... parse fields ...
		if ppid != 1 {
			continue  // Only processes with PPID=1
		}

		command := fields[2]
		args := strings.Join(fields[3:], " ")
		if !isOrchRelatedProcess(command, args) {
			continue  // ← Filters using isOrchRelatedProcess
		}
		// ... add to orphans list ...
	}
}
```

**Significance:** The logic is: "find all PPID=1 processes, filter through isOrchRelatedProcess, report as orphans". This is correct in principle but fails because isOrchRelatedProcess has false positives.

### Finding 3: clean_processes.go uses a more precise approach
**Source:** `cmd/orch/clean_processes.go:44-46`
**Evidence:**
```go
orphans, err := process.FindOrphanProcesses(activeTitles)
```

**Significance:** The `orch clean` command uses `process.FindOrphanProcesses` which checks against active session titles - more precise than keyword matching. However, the operator health endpoint doesn't have access to this context.

### Finding 4: Legitimate PPID=1 patterns
**Source:** System observation (documented in issue)
**Evidence:**
- overmind: Launched by orch-dashboard script, managed by launchd
- tmux: Terminal multiplexer, managed by launchd
- /System/Library/: macOS system processes

**Significance:** Not all PPID=1 processes are orphans. We need to distinguish between:
1. **Legitimate PPID=1**: launchd-managed services (overmind, tmux, system processes)
2. **Orch orphans**: bun/node/orch processes that should have a parent but don't

## Synthesis

The false positive issue stems from conflating two categories:
1. **Infrastructure processes** (overmind, tmux) - intentionally PPID=1
2. **Orch agent processes** (bun running agents) - PPID=1 means orphaned

The fix should:
1. Remove broad infrastructure keywords ("overmind", "vite") from `isOrchRelatedProcess`
2. Focus on actual agent-related processes: bun, node, opencode binary
3. Add whitelist logic for known legitimate PPID=1 process names
4. Consider adding a check for /System/Library/ paths (macOS system processes)

## Recommendation

**Approach: Whitelist legitimate PPID=1 processes**

Modify `isOrchRelatedProcess` to:
1. First check if process is a known legitimate PPID=1 process (return false)
2. Then check if it's an orch-related process (bun, node, opencode, orch binary)

Example whitelist:
- overmind
- tmux
- Commands starting with /System/Library/

This preserves the simple keyword approach while eliminating false positives.

## Testing Strategy

Write tests for `isOrchRelatedProcess` that verify:
1. Returns `false` for overmind
2. Returns `false` for tmux
3. Returns `false` for /System/Library/ processes
4. Returns `true` for bun processes with .orch in path
5. Returns `true` for opencode processes
6. Returns `true` for orch binary

Use TDD: write failing tests first, then implement the fix.
