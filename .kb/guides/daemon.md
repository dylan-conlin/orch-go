# Daemon Guide

**Purpose:** Single authoritative reference for the orch daemon's autonomous agent spawning system. This guide synthesizes learnings from 33 investigations conducted between Dec 2025 - Jan 2026.

**Last verified:** Jan 7, 2026

---

## Executive Summary

The daemon is an autonomous agent spawner that:
1. Polls beads for issues labeled `triage:ready`
2. Infers skill from issue type
3. Spawns agents within capacity limits
4. Monitors for completion via Phase: Complete comments
5. Supports cross-project operation (Jan 2026)

**Key insight:** Daemon is for batch/overnight work. Orchestrator labels issues, daemon spawns them. Orchestrator stays available for triage and synthesis.

---

## Architecture Overview

### Core Package Structure (pkg/daemon/)

| File | Lines | Purpose |
|------|-------|---------|
| `daemon.go` | ~700 | Main daemon struct, poll loop, Next/Once methods |
| `pool.go` | ~250 | WorkerPool for capacity management |
| `completion.go` | ~310 | SSE-based completion tracking (legacy) |
| `completion_processing.go` | ~325 | Beads-polling completion detection |
| `reflect.go` | ~270 | kb reflect integration for synthesis surfacing |
| `status.go` | ~130 | Status file management |
| `hotspot.go` | ~100 | Hotspot detection interface |
| `skill_inference.go` | ~120 | Issue type → skill mapping |
| `rate_limiter.go` | ~110 | Spawn rate limiting |
| `issue_adapter.go` | ~90 | Beads integration |
| `issue_queue.go` | ~60 | Issue filtering logic |
| `active_count.go` | ~160 | OpenCode session counting |
| `spawn_tracker.go` | ~150 | Spawn tracking for dedup |

### Poll Loop Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│  orch daemon run                                                         │
│                                                                          │
│  Startup:                                                                │
│    - Load config from ~/.orch/config.yaml                                │
│    - Initialize WorkerPool with MaxAgents                                │
│    - Start completion polling (separate interval)                        │
│                                                                          │
│  Poll Loop (every 60s default):                                          │
│    1. Reconcile with OpenCode (free stale slots)                        │
│    2. If periodic reflect due → run kb reflect                          │
│    3. Poll beads: bd ready --limit 0                                    │
│    4. Filter for triage:ready label                                     │
│    5. For each ready issue (within capacity):                           │
│       - Check rejection reasons (type, status, deps)                     │
│       - Infer skill from issue type                                      │
│       - Acquire slot from WorkerPool                                     │
│       - Spawn agent: orch work <id>                                      │
│    6. Sleep for poll interval                                            │
│    7. Repeat                                                             │
│                                                                          │
│  Completion Loop (every 60s):                                            │
│    - Poll for Phase: Complete comments                                   │
│    - Verify completion (check artifacts)                                 │
│    - Close beads issues                                                  │
│    - Release pool slots                                                  │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Skill Inference

Daemon infers skill from issue type (NOT labels):

| Issue Type | Skill | Use Case |
|------------|-------|----------|
| `bug` | `systematic-debugging` | Fix broken behavior |
| `investigation` | `investigation` | Understand how something works |
| `feature` | `feature-impl` | Build new capability |
| `task` | `feature-impl` | Generic implementation work |
| `question` | `investigation` | Answer a specific question |
| `epic` | (not spawnable) | Container for child issues |
| `chore` | (not spawnable) | Non-agent maintenance work |

**To control skill selection:** Set the correct issue type when creating:
```bash
bd create "fix login bug" --type bug          # → systematic-debugging
bd create "add dark mode" --type feature      # → feature-impl
bd create "how does auth work" --type investigation  # → investigation
bd create "what is the db schema" --type question    # → investigation
```

**Common mistake:** Missing or null type causes spawn failure. Always specify `--type`.

---

## Triage Labels and Workflow

| Label | Meaning | Daemon Action |
|-------|---------|---------------|
| `triage:ready` | High confidence, daemon can spawn | Auto-spawn |
| `triage:review` | Needs orchestrator review | Skip |
| (no triage label) | Not triaged yet | Skip |

**Workflow:**
1. Create issue with correct type
2. If confident: `bd label <id> triage:ready`
3. If unsure: Leave as `triage:review`, review later
4. Daemon picks up `triage:ready` issues

**Batch labeling:**
```bash
# Release multiple issues to daemon
bd label <id1> triage:ready
bd label <id2> triage:ready
bd label <id3> triage:ready
```

---

## Capacity Management

### WorkerPool

The daemon uses a semaphore-based worker pool:
- Tracks active slots with beads IDs
- Prevents over-spawning
- Reconciles with actual OpenCode sessions

**Key insight (from 2025-12-26 investigation):** Pool tracks spawns internally but must reconcile with OpenCode to avoid stale capacity. The daemon calls `ReconcileWithOpenCode()` at the start of each poll cycle.

### SpawnedIssueTracker (Duplicate Prevention)

**From 2026-01-06 investigation:** The daemon can spawn duplicate agents for the same issue due to a race condition:
1. Daemon polls beads, finds issue with `triage:ready`
2. Spawns agent via `orch work <id>`
3. Status update to `in_progress` happens AFTER spawn initialization
4. Before status updates, next poll sees issue still as "open" → spawns again

**Fix:** `SpawnedIssueTracker` in `pkg/daemon/spawn_tracker.go` tracks issue IDs immediately before calling `spawnFunc`:
- 5-minute TTL allows entries to expire naturally
- `CleanStale()` called during `ReconcileWithOpenCode()`
- On spawn failure, issue is unmarked to allow retry

**Behavior:**
```go
// Before spawn
daemon.SpawnedIssues.Mark(issueID)
err := daemon.spawnFunc(issueID)
if err != nil {
    daemon.SpawnedIssues.Unmark(issueID)  // Allow retry
}
```

### Configuration

```yaml
# ~/.orch/config.yaml
max_agents: 5  # Default concurrent agents
```

```bash
# CLI override
orch daemon run --max-agents 3
```

### Checking Capacity

```bash
orch status  # Shows "Active: X/Y" where Y is max capacity
```

---

## Running the Daemon

### Foreground (Interactive)

```bash
orch daemon run
orch daemon run --verbose  # Show debug output
orch daemon run --poll-interval 30  # Override poll interval (seconds)
```

### Preview Mode

```bash
orch daemon preview    # Show what would spawn with rejection reasons
orch daemon run --dry-run  # Same as preview
```

**Preview output (from 2026-01-04 fix):**
```
Rejected issues:
  orch-go-78jw: status is in_progress (already being worked on)
  orch-go-eysk: type 'epic' not spawnable (must be bug/feature/task/investigation)
  orch-go-eysk.4: missing label 'triage:ready'

Would spawn:
  orch-go-abc1: bug → systematic-debugging
```

### Background (launchd)

For persistent overnight operation:

**Plist location:** `~/Library/LaunchAgents/com.orch.daemon.plist`

**Configuration options:**
```xml
<key>ProgramArguments</key>
<array>
    <string>/Users/dylanconlin/bin/orch</string>
    <string>daemon</string>
    <string>run</string>
    <string>--poll-interval</string>
    <string>60</string>
    <string>--max-agents</string>
    <string>3</string>
    <string>--label</string>
    <string>triage:ready</string>
    <string>--verbose</string>
</array>
<key>WorkingDirectory</key>
<string>/Users/dylanconlin/Documents/personal/orch-go</string>
<key>EnvironmentVariables</key>
<dict>
    <key>BEADS_NO_DAEMON</key>
    <string>1</string>
</dict>
```

**Control commands:**
```bash
# Check status
launchctl list | grep orch

# Restart (after make install)
launchctl kickstart -k gui/$(id -u)/com.orch.daemon

# View logs
tail -f ~/.orch/daemon.log
```

**After rebuilding:**
```bash
make install-restart  # Builds, installs, restarts daemon
# OR
make install && launchctl kickstart -k gui/$(id -u)/com.orch.daemon
```

---

## Completion Detection

### Why Beads Polling (Not SSE)

**From 2025-12-25 investigation:** SSE-based idle detection has false positives:
- Agents go idle during tool loading
- Agents go idle during thinking/planning
- Only `Phase: Complete` comment is reliable

The daemon polls beads for `Phase: Complete` comments instead of relying on session state.

### Auto-Completion Integration

**From 2026-01-06 investigation:** The daemon calls `CompletionOnce()` during each poll cycle to auto-complete agents that report `Phase: Complete`.

**Poll loop flow:**
```
1. ReconcileWithOpenCode (free stale slots)
2. CompletionOnce (auto-complete finished agents)  ← NEW
3. Run periodic reflection
4. Write daemon status
5. Check capacity
6. Spawn new agents
```

**Escalation model:** Auto-completion respects the 5-tier escalation model:
- `None`/`Info`/`Review` → Auto-complete (routine work)
- `Block` → Requires human visual approval
- `Failed` → Requires human review of verification errors

**Status tracking:**
- `LastCompletion` field in daemon status shows last auto-completion timestamp
- Completion events logged as `daemon.complete` for monitoring

### Completion Flow

1. Daemon calls `CompletionOnce()` each poll cycle
2. `ListCompletedAgents()` finds issues with `Phase: Complete` comment
3. `VerifyCompletionFull()` checks workspace artifacts
4. `DetermineEscalationFromCompletion()` decides if auto-close is safe
5. If `ShouldAutoComplete()` → closes beads issue with reason
6. Releases pool slot
7. Logs auto-completion event

---

## Dependency Handling

### Parent-Child Dependencies

**From 2026-01-06 investigation:** Parent-child dependencies have different semantics than "blocks":

| Parent Status | Child Blocked? |
|---------------|----------------|
| `open` | Yes (epic not started) |
| `in_progress` | No (epic active, children should run) |
| `closed` | No |

**Root cause:** `GetBlockingDependencies()` in `pkg/beads/types.go` treated all dependency types the same, blocking when `status != "closed"`. For parent-child, children should only be blocked when parent is `open`, not when `in_progress`.

**Fix:** Switch on `DependencyType` field:
- `"parent-child"` → Only blocks when parent status is `"open"`
- `"blocks"` (default) → Blocks when status is not `"closed"`

**Example:** Epic `pw-u8th` has children. When epic moves to `in_progress`, children should become spawnable. Before fix, they were blocked until epic was `closed`.

### Blocked Issue Behavior

The daemon skips issues with:
- Status `blocked`
- Status `in_progress` (already being worked)
- Unresolved blocking dependencies

---

## Issue Fetching

### bd ready vs bd list

**From 2025-12-24 investigation:** The daemon must use `bd ready --limit 0` because:
- `bd ready` returns both `open` and `in_progress` issues without blockers
- `bd list --status open` misses `in_progress` issues
- Default limit is 10, must pass `--limit 0` for all issues

### Beads Integration

The daemon tries RPC client first, falls back to CLI:
1. Try `beads.Client.Ready()` (faster, more reliable)
2. If fails, fall back to `bd ready --json --limit 0`

---

## Reflection Integration

### Periodic kb reflect

**From 2026-01-06 investigation:** The daemon can run `kb reflect` periodically to surface synthesis opportunities:

```yaml
# ~/.orch/config.yaml
reflect:
  enabled: true
  interval_minutes: 60  # Run every hour
  create_issues: true   # Auto-create beads issues for topics with 10+ investigations
```

**CLI flags:**
```bash
orch daemon run --reflect-interval 120 --reflect-issues
```

### Two-Tier Reflection Automation

**From 2026-01-06 investigation:** Not all kb reflect types should auto-create issues. Signal quality determines automation suitability:

**High signal (auto-create issues):**

| Type | Threshold | Triage Label | Reason |
|------|-----------|--------------|--------|
| `synthesis` | 10+ investigations | `triage:review` | Clear consolidation need |
| `open` | Any item >3 days | `triage:review` | Explicit Next: actions (self-declared) |

**Surface-only (no auto-issues):**

| Type | Why No Auto-Create |
|------|--------------------|
| `promote` | Requires human judgment on kb vs principles |
| `stale` | Weak signal - decisions may be valid but rarely cited |
| `drift` | High false positive rate (~30-50%) from heuristic detection |
| `skill-candidate` | Noisy clustering (72 entries for "spawn" alone) |
| `refine` | Requires human evaluation of principle refinement |

**Current implementation:** Only `synthesis` type auto-creates issues. `open` type issue creation not yet implemented in kb-cli.

### On-Exit Reflection

```bash
orch daemon run --reflect  # Run kb reflect when daemon exits (default: true)
```

---

## Cross-Project Daemon

**From 2026-01-06 investigation + 2026-01-21 implementation:** A single daemon can poll all registered projects.

### How It Works

1. Daemon calls `kb projects list --json` to get registered projects (sorted alphabetically)
2. For each project: calls `ListReadyIssuesForProject(projectPath)`
3. Collects all triage:ready issues across projects, sorted by priority
4. Spawns highest priority issue with `orch work <id> --workdir <projectPath>`
5. Maintains single global capacity pool across all projects

### Implementation (pkg/daemon/)

| File | Function | Purpose |
|------|----------|---------|
| `projects.go` | `ListProjects()` | Parse `kb projects list --json` |
| `issue_adapter.go` | `ListReadyIssuesForProject()` | Get issues from specific project |
| `issue_adapter.go` | `SpawnWorkForProject()` | Spawn with `--workdir` |
| `daemon.go` | `CrossProjectOnce()` | Process one issue from any project |
| `daemon.go` | `CrossProjectPreview()` | Preview issues across all projects |

### Enabling Cross-Project

```bash
orch daemon run --cross-project       # Poll all kb-registered projects
orch daemon preview --cross-project   # Preview what would spawn across projects
orch daemon once --cross-project      # Process one issue from any project
```

### launchd Configuration

```xml
<key>ProgramArguments</key>
<array>
    <string>/path/to/orch</string>
    <string>daemon</string>
    <string>run</string>
    <string>--verbose</string>
    <string>--cross-project</string>
</array>
```

### Error Handling

**Design principle:** One project failing should not crash daemon or block other projects.

- If `kb projects list` fails: Returns empty list (graceful degradation)
- If a project's beads fails: Logs warning, continues to next project
- If spawn fails: Issue marked "failed to spawn this cycle", skipped until next poll

### Constraints

- Projects must be registered with `kb projects add`
- Issues in unregistered projects won't be seen
- Capacity is shared globally across all projects (prevents runaway spawning)
- Flag defaults to false (backward compatible with single-project mode)
- Projects without beads initialized will log warnings but not crash

### Common Issues

**"warning: failed to get ready issues for project X"**
- Project may not have beads initialized (`bd init`)
- Project's `.beads/` directory may be missing or corrupt
- Recovery: `cd <project> && rm -f .beads/beads.db* && bd init`

**Issues not spawning cross-project**
- Verify issue has `triage:ready` label in correct project
- Check daemon log for rejection reasons: `[project-name] Skipping <id> (reason)`

---

## Common Problems and Solutions

### "Daemon not spawning my issue"

**Checklist:**
1. Issue has `triage:ready` label? `bd show <id>`
2. Issue type is set? (not null/empty)
3. Issue type is spawnable? (bug/feature/investigation/task)
4. Daemon is running? `launchctl list | grep orch`
5. At capacity? `orch status`
6. Issue has blocking dependencies? `bd show <id> --deps`
7. Issue already recently spawned? (SpawnedIssueTracker TTL)

**Use preview to diagnose:**
```bash
orch daemon preview  # Shows rejection reasons per-issue
```

### "Daemon sees only 10 issues"

**Cause:** Missing `--limit 0` flag. `bd ready` defaults to limit 10.

**From 2026-01-06 investigation:** Both RPC path and CLI fallback need `--limit 0`:
- RPC: `client.Ready(&beads.ReadyArgs{Limit: 0})`
- CLI: `bd ready --json --limit 0`

**Fix:** Already fixed in daemon code (Jan 2026). If on old binary:
```bash
make install-restart
```

### "Daemon spawned duplicate agents for same issue"

**Cause (from 2026-01-06):** Race condition between spawn initiation and beads status update. Daemon polls, spawns, but status isn't `in_progress` until after spawn initialization completes.

**Fix:** `SpawnedIssueTracker` with 5-minute TTL tracks issues immediately before spawn. See "Capacity Management" section.

**If still happening:**
1. Check daemon binary is recent: `make install-restart`
2. Verify SpawnedIssueTracker in logs: look for "already recently spawned"
3. TTL may need adjustment if spawns take >5 minutes

### "Daemon capacity stuck at max"

**Cause:** Pool not reconciling with actual OpenCode sessions.

**Fix (from 2025-12-26):** 
- Daemon now calls `ReconcileWithOpenCode()` each poll cycle
- If still stuck: `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`

### "Child issues blocked when parent is in_progress"

**Cause:** Old code treated all dependency types the same.

**Fix (from 2026-01-06):** Parent-child dependencies now only block when parent is `open`, not `in_progress`.

### "Daemon not picking up newly labeled issues"

**Possible causes:**
1. **Wrong directory** - Daemon runs from fixed WorkingDirectory in plist
2. **Beads daemon not running** - RPC client fails silently
3. **Label added after daemon cache** - Wait for next poll cycle

### "Multiple daemons spawning"

**Cause (from 2025-12-24):** Race condition between startlock release and flock acquisition.

**Fix:** Use single launchd-managed daemon. If manual spawns needed:
```bash
orch daemon run --once  # Process single issue and exit
```

### "Beads daemon not running, slow API"

**From 2026-01-07 investigation:** Dashboard API can be slow (6.5s) on first request if beads daemon not running.

**Why NOT to auto-start:**
- Beads daemons are per-project (one per `.beads/` directory)
- 5+ bd daemons already run across projects
- `BEADS_NO_DAEMON=1` in orch daemon plist is intentional

**Actual fix:** TTL-based caching in `orch serve`:
- 30s TTL for stats
- 15s TTL for ready issues
- First request may be slow, subsequent requests hit cache (~15ms)

---

## Daemon vs Manual Spawn

| Scenario | Approach | Why |
|----------|----------|-----|
| Batch of 3+ issues | Daemon (label `triage:ready`) | Autonomous processing |
| Overnight processing | Daemon | No human presence needed |
| Single urgent item | Manual `orch spawn` | Immediate attention |
| Complex/ambiguous | Manual `orch spawn` | Orchestrator judgment needed |
| Needs custom context | Manual `orch spawn` | Daemon uses issue description only |

**Daemon is preferred for batch work** because:
- Orchestrator stays available for triage and synthesis
- Automatic capacity management
- Overnight processing without human presence
- Issues have full context vs ephemeral spawn prompts

---

## Key Decisions (Historical)

From investigations, these design decisions were made:

| Decision | Reason | Date |
|----------|--------|------|
| Skill from issue type, not labels | Type is required; labels can be added/removed | Dec 2025 |
| Beads polling over SSE | SSE idle has false positives | Dec 2025 |
| WorkerPool with reconciliation | Prevents stale capacity | Dec 2025 |
| RPC-first with CLI fallback | Performance + reliability | Dec 2025 |
| --limit 0 for bd ready | Default 10 misses issues | Jan 2026 |
| Parent-child unblocked when in_progress | Epics should start children | Jan 2026 |
| Periodic kb reflect | Auto-surface synthesis opportunities | Jan 2026 |
| SpawnedIssueTracker (5-min TTL) | Prevent duplicate spawns from race condition | Jan 2026 |
| Auto-completion via CompletionOnce | Free capacity slots without orchestrator | Jan 2026 |
| Two-tier reflection (synthesis+open) | Only high-signal types auto-create issues | Jan 2026 |
| No beads daemon auto-start | Caching solves API latency; daemons are per-project | Jan 2026 |
| Cross-project uses global capacity | Prevents runaway spawning (N projects × M agents) | Jan 2026 |
| Cross-project uses kb projects registry | Reuses existing infrastructure, no new config file | Jan 2026 |

---

## Debugging Checklist

Before spawning an investigation about daemon issues:

1. **Check kb:** `kb context "daemon"`
2. **Read this guide:** You're reading it
3. **Check daemon running:** `launchctl list | grep orch`
4. **Check services:** `orch doctor`
5. **Check preview:** `orch daemon preview`
6. **Check capacity:** `orch status`
7. **Check logs:** `tail -50 ~/.orch/daemon.log`

If those don't answer your question, then investigate. But update this guide with what you learn.

---

## Related Resources

- **Orchestrator skill:** Reference for when to use daemon vs manual spawn
- **pkg/daemon/ source:** Implementation details
- **~/.orch/config.yaml:** User configuration
- **launchd plist:** `~/Library/LaunchAgents/com.orch.daemon.plist`

---

## Synthesized From

This guide consolidates learnings from 33 investigations:
- 2025-12-20: Initial daemon command implementation
- 2025-12-21: Hook integration for kb reflect
- 2025-12-22: Concurrency control (WorkerPool)
- 2025-12-24: Race condition analysis, bd ready vs bd list
- 2025-12-25: Completion polling, beads RPC migration
- 2025-12-26: Capacity reconciliation, launchd documentation
- 2026-01-03: Skip functionality verification
- 2026-01-04: Structure analysis, rejection reason visibility
- 2026-01-06: Parent-child deps, periodic reflect, cross-project design, auto-completion integration, duplicate spawn prevention, --limit 0 fix, two-tier reflection
- 2026-01-07: Beads daemon auto-start analysis, synthesis update
- 2026-01-21: Cross-project daemon implementation, workdir fix for resolveShortBeadsID
