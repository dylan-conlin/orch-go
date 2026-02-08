## Summary (D.E.K.N.)

**Delta:** Orchestrator workspaces now use "orch" prefix instead of "work" (e.g., og-orch-* vs og-work-*) and write .tier file with "orchestrator" value for programmatic detection.

**Evidence:** Tests pass - TestGenerateWorkspaceName_MetaOrchestrator confirms og-orch-* naming for orchestrators and og-work-* for workers.

**Knowledge:** Visual distinction requires both naming convention (quick scan) and marker file (programmatic detection) - Option C from the spawn context.

**Next:** Close - implementation complete, tests passing.

---

# Investigation: Orchestrator Workspaces Clear Visual Distinction

**Question:** How can orchestrator workspaces be visually distinguished from worker workspaces when browsing .orch/workspace/ directories?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Feature-impl agent (orch-go-snk4)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current naming uses generic "work" prefix for orchestrators

**Evidence:** GenerateWorkspaceName in config.go uses `prefix := "work"` for any skill without a specific mapping (investigation=inv, feature-impl=feat, etc.). Orchestrator skills fall through to this default.

**Source:** pkg/spawn/config.go:196

**Significance:** The "work" prefix is ambiguous - it's used for unknown worker skills AND orchestrators. No visual distinction at a glance.

### Finding 2: Orchestrator context already writes marker files

**Evidence:** WriteOrchestratorContext writes `.orchestrator` marker file and WriteMetaOrchestratorContext writes `.meta-orchestrator` marker file. Both have spawn time tracking.

**Source:** pkg/spawn/orchestrator_context.go:201-205, pkg/spawn/meta_orchestrator_context.go:235-239

**Significance:** Programmatic detection mechanism already exists, but .tier file wasn't being written for orchestrators.

### Finding 3: Tier file used for completion verification

**Evidence:** VerifyCompletionWithTier in pkg/verify/check.go reads .tier file from workspace to determine verification rules. TierOrchestrator = "orchestrator" skips beads checks and verifies SESSION_HANDOFF.md instead.

**Source:** pkg/verify/check.go:21-24, 240-243

**Significance:** Writing "orchestrator" to .tier file enables proper completion verification AND serves as programmatic detection mechanism.

---

## Synthesis

**Key Insights:**

1. **Dual detection needed** - Visual scan (directory listing) and programmatic detection (orch status, orch complete) both need to work. Solution must address both.

2. **Minimal changes required** - WorkspaceNameOptions already exists with IsMetaOrchestrator. Adding IsOrchestrator follows same pattern.

3. **Tier file is the right mechanism** - Already used for verification logic, now also serves detection purpose.

**Answer to Investigation Question:**

Orchestrator workspaces are now visually distinct through:
1. **Naming:** og-orch-* instead of og-work-* (e.g., og-orch-test-session-05jan)
2. **Tier file:** .tier contains "orchestrator" for programmatic detection

---

## Structured Uncertainty

**What's tested:**

- TestGenerateWorkspaceName_MetaOrchestrator confirms og-orch-* naming for orchestrators
- TestGenerateWorkspaceName_MetaOrchestrator confirms og-work-* preserved for workers
- All spawn package tests pass

**What's untested:**

- End-to-end spawn of an orchestrator skill (would require actual skill to be spawned)
- Dashboard visual display of workspace names

**What would change this:**

- Finding would be incomplete if orchestrator skills don't pass IsOrchestrator=true to workspace generation (verified: spawn_cmd.go does this)

---

## Implementation Summary

**Changes made:**

1. **pkg/spawn/config.go** - Added IsOrchestrator field to WorkspaceNameOptions, updated GenerateWorkspaceName to use "orch" prefix when IsOrchestrator or IsMetaOrchestrator is true

2. **cmd/orch/spawn_cmd.go** - Pass IsOrchestrator to WorkspaceNameOptions when generating workspace name

3. **pkg/spawn/orchestrator_context.go** - Write .tier file with "orchestrator" value

4. **pkg/spawn/meta_orchestrator_context.go** - Write .tier file with "orchestrator" value

5. **pkg/spawn/context_test.go** - Updated tests for new naming convention

---

## References

**Files Modified:**
- pkg/spawn/config.go - Added IsOrchestrator to WorkspaceNameOptions
- cmd/orch/spawn_cmd.go - Pass IsOrchestrator option
- pkg/spawn/orchestrator_context.go - Write .tier file
- pkg/spawn/meta_orchestrator_context.go - Write .tier file
- pkg/spawn/context_test.go - Updated test expectations

**Commands Run:**
```bash
# Verify tests pass
go test ./pkg/spawn/... -v

# Verify build works
go build ./...
```

---

## Investigation History

**2026-01-05:** Investigation started
- Initial question: How to visually distinguish orchestrator from worker workspaces
- Context: Example workspace pw-work-test-workdir-fix-05jan is orchestrator but looks like worker

**2026-01-05:** Implementation complete
- Status: Complete
- Key outcome: Orchestrators use og-orch-* naming and .tier=orchestrator for clear distinction
