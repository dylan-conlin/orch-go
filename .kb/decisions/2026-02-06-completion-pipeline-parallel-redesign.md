# Decision: Redesign Completion Pipeline for Multi-Agent Parallel Workflows

**Date:** 2026-02-06
**Status:** Active
**Context:** Completion gate ceremony is primary orchestrator bottleneck in parallel workflows (orch-go-21351, orch-go-21345, orch-go-21356)
**Resolves:** orch-go-21357

---

## Decision

Restructure the completion pipeline around three changes: a **two-tier gate model** (careful vs batch), **session-level skip reasons** that persist across completions, and **automatic blame attribution** for the build gate. Together these reduce 26-completion ceremony from 5+ minutes to under 60 seconds.

---

## 1. Gate Tier Model: Load-Bearing vs Theater

### Analysis of Current 12 Gates

| Gate                   | Load-Bearing?   | Catches Real Problems?                                   | Evidence                                                 |
|------------------------|:---------------:|----------------------------------------------------------|----------------------------------------------------------|
| `phase_complete`       | **Yes**         | Prevents closing work agent didn't finish                | Core lifecycle signal                                    |
| `build`                | **Conditional** | Only when THIS agent broke it; theater when pre-existing | 4 of 5 build failures in Feb 6 session were pre-existing |
| `test_evidence`        | **Yes**         | Catches "tests pass" claims without proof                | Designed after agents shipped broken code                |
| `visual_verification`  | **Yes**         | Catches broken UI that tests can't cover                 | Requires subjective human judgment                       |
| `synthesis`            | **Conditional** | Valuable for full-tier; already skipped for light-tier   | Light tier spawns skip it correctly                      |
| `constraint`           | **Low**         | Catches missing artifacts, but rarely fires              | Most agents produce required artifacts                   |
| `phase_gate`           | **Low**         | Checks intermediate phases were reported                 | Process verification, not quality                        |
| `skill_output`         | **Low**         | Checks skill.yaml output patterns                        | Rarely configured, rarely fails                          |
| `git_diff`             | **Low**         | Catches false claims in SYNTHESIS.md delta               | Only matters for full-tier with SYNTHESIS.md             |
| `decision_patch_limit` | **Low**         | Prevents excessive patching of decisions                 | Niche guard, almost never triggers                       |
| `handoff_content`      | **Conditional** | Only for orchestrator sessions                           | Correct scope already                                    |
| `dashboard_health`     | **Low**         | HTTP health check for web/ changes                       | Frequently fails due to server state, not agent work     |

### Two-Tier Gate Model

**Tier 1: Core Gates (always run)**

- `phase_complete` - Agent says it's done
- `build` - But only if blame attribution says THIS agent broke it (see section 4)
- `test_evidence` - For implementation skills with code changes
- `visual_verification` - For web/ changes requiring human approval

**Tier 2: Quality Gates (run in careful mode, skip in batch mode)**

- `synthesis` - SYNTHESIS.md exists and has content
- `constraint` - Skill constraints satisfied
- `phase_gate` - Required phases reported
- `skill_output` - Required skill outputs exist
- `git_diff` - Git changes match SYNTHESIS claims
- `decision_patch_limit` - Decision patch count
- `dashboard_health` - Dashboard API health
- `handoff_content` - Orchestrator session content

### Mode Selection

```
orch complete <id>                    # Careful mode (default, all gates)
orch complete <id> --batch            # Batch mode (core gates only)
orch batch-complete <id1> <id2> ...   # Bulk batch complete (core gates only)
```

**Rationale:** Core gates catch real problems (unfinished work, broken builds, untested code, broken UI). Quality gates verify process compliance. In batch scenarios where the orchestrator has already reviewed the work, process compliance is redundant.

---

## 2. Batch Completion Design

### `orch batch-complete`

New command for bulk-closing already-reviewed completions:

```bash
# Complete all agents that reported Phase: Complete
orch batch-complete --all

# Complete specific agents
orch batch-complete orch-go-abc1 orch-go-def2 orch-go-ghi3

# Complete all agents in a project
orch batch-complete --project orch-go

# Dry run to see what would be completed
orch batch-complete --all --dry-run
```

**Behavior:**

1. Discovers all agents with `Phase: Complete` in beads comments
2. Runs Tier 1 (core) gates only on each
3. Collects results: passed, failed, skipped
4. Closes passed agents in bulk
5. Reports failures with specific gate details
6. Produces audit trail (one `agent.completed` event per agent)

**Audit trail format:**

```json
{
  "event": "agent.completed",
  "beads_id": "orch-go-abc1",
  "mode": "batch",
  "gates_run": ["phase_complete", "build"],
  "gates_skipped": ["synthesis", "constraint", "phase_gate", ...],
  "skip_reason": "batch mode - core gates only",
  "completed_by": "orchestrator",
  "timestamp": "2026-02-06T10:30:00Z"
}
```

**Why not `orch review done`?** The existing `orch review done` marks work as reviewed but doesn't close beads issues. `batch-complete` performs the full completion chain: verify → close beads → delete session → archive workspace → close tmux.

---

## 3. Skip Reason Simplification

### Problem

Current: Each `--skip-X` flag requires `--skip-reason` with min 10 chars. For 3-5 skips, that's one long reason string shared across all gates, which is clumsy but technically works. The real friction is **per-completion repetition**: the same reason applies to every completion in the session.

### Session-Level Skip Reasons (Skip Memory)

Generalize the existing `build-skip.json` pattern to all gates:

```bash
# Set a session-level skip reason (persists for duration)
orch skip-set build "Build broken by concurrent extraction agents"
orch skip-set dashboard-health "Go refactor, not dashboard work"

# View active skips
orch skip-list
# Output:
#   build: "Build broken by concurrent extraction agents" (expires in 1h45m, set by orchestrator)
#   dashboard-health: "Go refactor, not dashboard work" (expires in 1h50m, set by orchestrator)

# Clear a skip
orch skip-clear build

# Clear all skips
orch skip-clear --all
```

**Mechanism:**

- Stored in `.orch/gate-skips.json` (project-level, like `build-skip.json`)
- Each entry: `{ gate, reason, set_at, set_by, expires_at }`
- Default TTL: 2 hours (same as current `BuildSkipDuration`)
- Checked by `VerifyCompletionFull` before running each gate
- Auto-expire after TTL (prevents stale skips from haunting future sessions)

**Integration with `orch complete`:**

```go
// In VerifyCompletionFull, before running each gate:
if skip := ReadGateSkipMemory(projectDir, gateName); skip != nil {
    result.GateResults = append(result.GateResults, GateResult{
        Gate: gateName, Passed: true, Skipped: true,
        SkipReason: skip.Reason,
    })
    continue // skip this gate
}
```

**CLI flags still work:** `--skip-build --skip-reason "..."` still works for one-off skips. Session-level skips are for repeated patterns.

**Audit:** Every gate skip (whether from CLI flag or session memory) emits a `verification.bypassed` event. This is already implemented for CLI skips.

---

## 4. Blame Attribution for Build Gate

### Current State

`build_blame.go` already implements `AttributeBuildFailure()`:

1. Reads agent's spawn time
2. Finds commits since spawn time
3. If no commits → pre-existing failure
4. Creates temp git worktree at parent commit, tries building
5. If build fails at parent → pre-existing; if passes → agent caused it

**Problem:** This logic exists but isn't integrated into the main verification flow. `VerifyBuildForCompletion` runs `go build ./...` and fails immediately on error, without checking blame.

### Integration Design

Modify `VerifyBuildForCompletion` in `build_verification.go`:

```go
func VerifyBuildForCompletion(workspacePath, projectDir string) *BuildVerificationResult {
    // 1. Check gate skip memory first (session-level skip)
    if skip := ReadGateSkipMemory(projectDir, GateBuild); skip != nil {
        return &BuildVerificationResult{Passed: true, Skipped: true}
    }

    // 2. Run build
    output, err := RunGoTestCompile(projectDir)
    if err == nil {
        return &BuildVerificationResult{Passed: true}
    }

    // 3. Build failed - run blame attribution
    blame := AttributeBuildFailure(workspacePath, projectDir)
    if blame.PreExisting {
        // Build was broken before this agent - auto-skip with warning
        return &BuildVerificationResult{
            Passed: true,  // Don't block this agent
            Warnings: []string{
                fmt.Sprintf("Build broken (pre-existing): %s", blame.BlameDetail),
            },
        }
    }

    // 4. This agent broke it - block completion
    return &BuildVerificationResult{
        Passed: false,
        Errors: []string{
            fmt.Sprintf("Build failed (agent caused): %s\n%s", blame.BlameDetail, output),
        },
    }
}
```

**Key change:** Build gate only blocks when the agent caused the failure. Pre-existing failures produce a warning, not an error. This eliminates the need for `--skip-build` in the common case where concurrent agents broke the build.

---

## 5. Gate Memory Design

### Generalized Gate Skip Memory

Replace the build-specific `build-skip.json` with a general `gate-skips.json`:

```json
{
  "skips": [
    {
      "gate": "build",
      "reason": "Build broken by concurrent extraction agents",
      "set_at": "2026-02-06T10:15:00Z",
      "set_by": "orchestrator",
      "expires_at": "2026-02-06T12:15:00Z"
    },
    {
      "gate": "dashboard_health",
      "reason": "Go refactor session, no dashboard changes",
      "set_at": "2026-02-06T10:15:00Z",
      "set_by": "orchestrator",
      "expires_at": "2026-02-06T12:15:00Z"
    }
  ]
}
```

**File location:** `.orch/gate-skips.json` (project-level)

**Operations:**

- `ReadGateSkipMemory(projectDir, gate) *GateSkip` - Read skip for specific gate
- `WriteGateSkipMemory(projectDir, gate, reason, setBy) error` - Write skip
- `ClearGateSkipMemory(projectDir, gate) error` - Clear specific skip
- `ClearAllGateSkipMemory(projectDir) error` - Clear all skips
- `ListGateSkipMemory(projectDir) []GateSkip` - List all active skips

**Expiry:** Skips auto-expire after 2 hours. Expired entries are cleaned up on read. This prevents stale skips from persisting across sessions.

**Migration:** Remove `build-skip.json` in favor of `gate-skips.json`. One-time migration reads existing build-skip and writes to new format.

---

## Implementation Priority

| Change                                         | Effort | Impact                                                    | Priority |
|------------------------------------------------|--------|-----------------------------------------------------------|----------|
| Build blame integration                        | Small  | High - eliminates most `--skip-build` usage               | P0       |
| Generalized gate skip memory                   | Medium | High - eliminates repeated `--skip-reason` per completion | P0       |
| `orch skip-set/list/clear` commands            | Small  | Medium - CLI ergonomics for session-level skips           | P1       |
| `--batch` flag on `orch complete`              | Small  | High - skips quality gates for reviewed work              | P1       |
| `orch batch-complete` command                  | Medium | High - bulk close for 26-agent scenarios                  | P1       |
| Migrate `build-skip.json` to `gate-skips.json` | Small  | Low - cleanup                                             | P2       |

---

## Constraints Respected

1. **Gates remain available for high-risk work** - Careful mode (default) runs all gates. Batch mode is opt-in.
2. **Bulk mode produces audit trail** - Every batch-completed agent emits `agent.completed` event with mode, gates run, gates skipped.
3. **Works for tracked and untracked agents** - `batch-complete --all` discovers from both beads and workspace scan.
4. **Session idle ≠ agent complete** - Phase: Complete remains the authoritative signal, even in batch mode.
5. **orch complete must verify SYNTHESIS.md for full tier** - SYNTHESIS.md moves to Tier 2 (quality gates), so it's checked in careful mode but not batch mode. This is acceptable because batch mode is for already-reviewed work.

## Consequences

### Accepted trade-offs

- **Batch mode reduces verification depth** - Tier 2 gates are skipped. This is acceptable because the orchestrator has already reviewed the work before using batch mode.
- **Blame attribution adds build time** - Creating a temporary worktree and building at parent commit adds ~5-15 seconds. This is offset by eliminating manual `--skip-build` for every completion.
- **Gate skip memory can mask real problems** - If a skip is set and the underlying problem changes, the skip still applies until expiry. The 2-hour TTL limits this risk.

### What this enables

- **26-agent completion in ~60s** - `orch batch-complete --all` replaces 26 individual `orch complete` commands with skip flags
- **Zero manual build gate skips** - Blame attribution auto-skips pre-existing build failures
- **Session-level skip reasons** - Set once, apply to all completions in the session
- **Clean audit trail** - Every bypass is logged with reason, gate, and mode

### What this doesn't solve

- **Daemon auto-completion** - The escalation model (None/Info/Review/Block/Failed) is unaffected. Batch mode is for orchestrator-driven completion.
- **Gate false positives** - Individual gates still have false positive rates. This decision addresses the ceremony, not the gate accuracy.
- **Cross-project completions** - `--workdir` is still needed for cross-project agents. Batch mode respects this.

---

## References

- `.kb/guides/completion-gates.md` - Gate reference
- `.kb/guides/completion.md` - Completion workflow
- `.kb/models/completion-verification.md` - Architecture model
- `pkg/verify/build_blame.go` - Existing blame attribution (to integrate)
- `cmd/orch/complete_verify.go` - SkipConfig (to generalize)
- `cmd/orch/complete_cmd.go` - Main completion flow (to add batch flag)
- orch-go-21351 - Gate ceremony bottleneck
- orch-go-21356 - System-wide build failure suppression
