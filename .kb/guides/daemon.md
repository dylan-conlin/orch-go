# Daemon Guide

**Purpose:** Single authoritative reference for the orch daemon's autonomous agent spawning system. This guide synthesizes learnings from 33 investigations conducted between Dec 2025 - Jan 2026.

**Last verified:** Mar 1, 2026

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
| `spawn_tracker.go` | ~360 | Spawn tracking: ID dedup (L1), title dedup (L3), disk persistence, thrash detection |
| `session_dedup.go` | ~140 | Session/tmux existence checking (L2) |

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
| `epic` | (not spawnable) | Container for child issues |
| `chore` | (not spawnable) | Non-agent maintenance work |

### Model Inference by Skill Type

The daemon infers model from skill type for optimal cost/quality tradeoffs:

| Skill Category | Model | Rationale |
|---------------|-------|-----------|
| Deep reasoning (investigation, architect, debugging, audit, research) | Opus | Requires thorough analysis |
| Implementation (feature-impl, issue-creation) | Sonnet | Execution-focused work |
| Default (unmapped skills) | Sonnet | Conservative default |

### Triage Routing Success Rate

Spawn prompt audit found daemon-routed (`triage:ready`) agents succeed **9.4x** more often than direct spawns. This strongly supports the daemon-first workflow.

### Model Compatibility Constraints

Not all models can follow the orch worker agent protocol:

| Model | Result | Issue |
|-------|--------|-------|
| Opus / Sonnet | Works reliably | Primary models |
| GPT-5.2-codex | **Failed** (3/3 agents stalled) | Hallucinated constraints, excessive token consumption, failed session close |
| gpt-4o | **Failed** | Spawns but never starts working — can't handle agentic workflows |

**To control skill selection:** Set the correct issue type when creating:
```bash
bd create "fix login bug" --type bug          # → systematic-debugging
bd create "add dark mode" --type feature      # → feature-impl
bd create "how does auth work" --type investigation  # → investigation
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

### Spawn Dedup Architecture (6-Layer)

The daemon prevents duplicate spawns via 6 sequential dedup layers in `spawnIssue()` (pkg/daemon/daemon.go). These layers accumulated over 9 tactical fixes (Jan-Mar 2026), each patching a gap in the previous layer.

#### Dedup Pipeline (Execution Order)

| Layer | Check | Fail Mode | Nature |
|-------|-------|-----------|--------|
| L1 | SpawnedIssueTracker (ID-based, 6h TTL) | Blocks spawn | Heuristic |
| L2 | Session/Tmux existence check | Blocks spawn (fail-open if API down) | Heuristic |
| L3 | Title dedup (in-memory, TTL-coupled) | Blocks spawn | Heuristic |
| L4 | Title dedup (beads DB query) | Blocks spawn (fail-open) | Structural-ish |
| L5 | Fresh beads status re-check | Blocks spawn (fail-open) | Structural |
| L6 | UpdateStatus("in_progress") | **Fail-fast** | Structural (PRIMARY) |

**Key properties:**
- L6 is the only fail-fast layer — if it fails, spawn is aborted
- L2, L4, L5 are fail-open — they allow spawn if their backing service is unavailable
- L1-L3 are in-memory heuristics that survive daemon restarts via disk persistence (`~/.orch/spawn_cache.json`)

#### SpawnedIssueTracker (L1)

`pkg/daemon/spawn_tracker.go` tracks issue IDs and titles immediately before spawn:
- **6-hour TTL** matches typical agent work duration (backup protection when session-level dedup fails)
- Disk-backed via `~/.orch/spawn_cache.json` (survives daemon restarts)
- `CleanStale()` called during capacity reconciliation
- Includes spawn count tracking for thrash detection (warns at 3+ spawns)

#### Session/Tmux Dedup (L2)

`pkg/daemon/session_dedup.go` checks for existing OpenCode sessions AND tmux windows for the issue:
- Queries OpenCode API for active sessions matching the beads ID
- Falls back to tmux window name matching if API unavailable
- Fail-open: if both checks fail, spawn proceeds (relies on downstream layers)

#### Title Dedup (L3/L4)

Catches content duplicates where different beads issues have identical titles:
- L3: In-memory normalized title map (fast, but lost on restart without disk persistence)
- L4: Beads DB query via `FindInProgressByTitle()` (persistent, but fail-open)

#### Fresh Status + Atomic Update (L5/L6)

- L5 re-queries beads for current status (guards against stale data from poll)
- L6 sets status to `in_progress` — fail-fast if update fails

#### Orphan Detector Interaction

The orphan detector (`pkg/daemon/orphan_detector.go`) resets dead agents' issues from `in_progress` → `open` but **intentionally does NOT clear spawn cache entries**. This prevents thrash loops where an issue is repeatedly spawned and fails. The trade-off: legitimate retries are blocked for the remainder of the 6h TTL.

#### Known Limitations

- **Correlated failures:** When beads is unavailable, L4, L5, and L6 all degrade simultaneously
- **No atomic CAS:** L5 and L6 are separate operations with a TOCTOU race window
- **Unbounded spawnCounts:** The spawn count map grows indefinitely (minor leak)
- **ReconcileWithIssues() is dead code:** Exists in spawn_tracker.go but never called from production

**Structural redesign recommended:** See `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md` for architect analysis recommending CAS-based primary gate with advisory-only heuristic layers.

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
orch daemon run --replace         # Stop existing daemon first (graceful takeover)
orch daemon run --verbose         # Show debug output
orch daemon run --poll-interval 30  # Override poll interval (seconds)
```

**Graceful takeover with `--replace`:** When you want to restart the daemon with new flags or after rebuilding, `--replace` stops the existing instance before acquiring the PID lock. This avoids the "cannot start daemon: already running" error without needing a separate `daemon stop` first. Equivalent to `daemon stop && daemon run` but atomic.

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
make install-restart  # Builds, installs, restarts daemon (launchd)
# OR
make install && launchctl kickstart -k gui/$(id -u)/com.orch.daemon
# OR (foreground daemon)
make install && orch daemon run --replace  # Graceful takeover with new binary
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

**From 2026-01-06 investigation:** A single daemon can poll all registered projects:

### How It Works

1. Daemon calls `kb projects list` to get registered projects
2. Iterates over each project's beads issues
3. Spawns with `--workdir` to target correct project
4. Maintains single capacity pool across all projects

### Enabling Cross-Project

```bash
orch daemon run --cross-project  # Poll all kb-registered projects
```

**Constraints:**
- Projects must be registered with `kb projects add`
- Issues in unregistered projects won't be seen
- Capacity is shared across all projects

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
7. Issue already recently spawned? (6h TTL in spawn cache, or active session/tmux window)

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

**Cause:** Race condition between spawn initiation and beads status update, or content duplicates (different IDs, same title).

**Fix:** 6-layer dedup pipeline in `spawnIssue()`. See "Spawn Dedup Architecture" section.

**If still happening:**
1. Check daemon binary is recent: `make install-restart`
2. Verify dedup in logs: look for "already recently spawned" (L1), "session exists" (L2), "title already spawned" (L3/L4)
3. Check if beads was unavailable (degrades L4, L5, L6 simultaneously)
4. Check `~/.orch/spawn_cache.json` for current tracked entries

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

**Fix:** Use single launchd-managed daemon. For foreground use, `--replace` handles graceful takeover:
```bash
orch daemon run --replace  # Stop existing, start new (graceful takeover)
orch daemon run --once     # Process single issue and exit
```

### "Daemon shows as running but isn't"

**Cause:** `handleDaemon`/`readDaemonStatus` check `daemon-status.json` file existence but don't verify PID liveness. After an unclean shutdown (crash, SIGKILL), the status file persists with stale data.

**Fix:** Status readers must check PID liveness (e.g., `kill -0 <pid>`) in addition to file existence. If PID is dead, treat daemon as stopped regardless of status file content.

**Related:** The extraction convergence constraint also applies to daemon — never create an extraction for a file if extraction was already attempted and the file is still above threshold.

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
| SpawnedIssueTracker (6h TTL, disk-backed) | Prevent duplicate spawns; 6h matches agent work duration | Jan 2026 (TTL updated Feb 2026) |
| Auto-completion via CompletionOnce | Free capacity slots without orchestrator | Jan 2026 |
| Two-tier reflection (synthesis+open) | Only high-signal types auto-create issues | Jan 2026 |
| No beads daemon auto-start | Caching solves API latency; daemons are per-project | Jan 2026 |
| Session/tmux dedup (L2) | Check active sessions before spawn; fail-open | Jan 2026 |
| Content-aware title dedup (L3/L4) | Catch duplicate issues with different IDs, same title | Feb 2026 |
| Fresh status re-check + atomic update (L5/L6) | Structural gate as final dedup authority | Feb 2026 |
| Orphan detector retains spawn cache | Prevents thrash loops after agent death | Mar 2026 |
| `--replace` flag for graceful takeover | Avoids manual stop/start; atomic daemon restart | Feb 2026 |

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
- 2026-03-01: Structural review of daemon dedup — 6 layers mapped, architect redesign recommended
