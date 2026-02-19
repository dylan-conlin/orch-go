# Decision: Two-Lane Agent Discovery Architecture

**Date:** 2026-02-18
**Status:** Accepted
**Deciders:** Dylan, Architect Agent (og-arch-write-adr-two-18feb-8fab)
**Context Issue:** orch-go-1081
**Supersedes:** 
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` (registry now eliminated)
- `.kb/decisions/2026-01-14-registry-contract-spawn-cache-only.md` (registry contract obsolete)

## Context

Dylan spent 6 weeks (Dec 21 - Feb 18) fighting ghost agents, phantom status, and silent enrichment failures caused by multi-source reconciliation. Five local state layers (registry.json, sessions.json, state.db, workspace cache, multi-source reconciliation) were each built to solve slow/incomplete queries and each drifted from reality.

The 5-iteration cache/remove cycle:
1. **Dec 21:** Registry drift identified, Phase 4 removal proposed
2. **Jan 5:** First attempt at beads-first — killed by semantic mismatch ("orchestrators are not issues")
3. **Jan 12:** Registry defended as "spawn-time metadata cache"
4. **Feb 13:** Registry eliminated (529 lines), state.db removed
5. **Feb 18:** 84/85 agents showing empty metadata, cross-repo visibility broken

The core problem: **Reconciliation logic is where every bug lives.** Each source has different lifecycle semantics (OpenCode sessions persist indefinitely, tmux windows are transient, workspace files are permanent, beads is canonical), and no single source has complete truth.

The Feb 18 beads-first solution correctly inverts the query direction but has two unresolved pressure points:
- `--no-track` agents are invisible
- Orchestrator sessions don't fit the beads work-item model

These are the exact gaps that triggered cache-building in January. Without addressing them, the cycle restarts.

## Decision

### Domain Boundaries (Enforceable, Not Just Preferred)

| Domain | Owns | Does NOT Own |
|--------|------|--------------|
| **Beads** | Lifecycle for tracked work (exists/done) | Session liveness, infrastructure state |
| **Workspace manifest** | Binding (beads_id ↔ session_id ↔ project_dir) | Work state, completion status |
| **OpenCode** | Session liveness only (busy/idle/dead) | Work completion, agent identity |
| **Tmux** | Presentation layer | Any state whatsoever |

**No other persisted lifecycle state allowed.** Any new `pkg/state/`, `pkg/registry/`, `pkg/cache/`, or `sessions.json` triggers CI lint failure (see Regression Guardrails).

### Two-Lane Split

Stop trying to merge tracked work and untracked sessions into one truth view.

| Lane | Query Path | What's Visible | Source of Truth |
|------|------------|----------------|-----------------|
| **Tracked work** | `orch status`, dashboard `/api/agents` | Agents with beads_id | Beads issues with `orch:agent` tag |
| **Untracked sessions** | `orch sessions`, `/api/sessions` | Orchestrator sessions, ad-hoc, `--no-track` | OpenCode session list |

**Why two lanes:** The Jan 5 beads-first attempt failed because orchestrators are not work items. Forcing them into beads semantics ("in_progress", "closed") created constant edge cases. The two-lane split resolves this by:
- Tracked work uses beads lifecycle (exists = in_progress, closed = done)
- Untracked sessions use OpenCode lifecycle (exists = running, gone = ended)

### Atomic Spawn (All-or-Nothing)

On spawn, require all 3 writes to succeed or spawn fails entirely:

```go
func atomicSpawn(ctx context.Context, opts SpawnOpts) (Agent, error) {
    // 1. Create beads issue OR tag existing issue with orch:agent
    issue, err := beads.CreateOrTag(opts.BeadsID, "orch:agent")
    if err != nil {
        return Agent{}, fmt.Errorf("beads write failed: %w", err)
    }
    
    // 2. Write workspace manifest with binding
    manifest := WorkspaceManifest{
        BeadsID:    issue.ID,
        SessionID:  "", // Filled after spawn
        ProjectDir: opts.WorkDir,
        SpawnTime:  time.Now(),
    }
    if err := writeManifest(opts.WorkspacePath, manifest); err != nil {
        beads.RemoveTag(issue.ID, "orch:agent") // Rollback
        return Agent{}, fmt.Errorf("manifest write failed: %w", err)
    }
    
    // 3. Spawn session and write session_id to manifest
    session, err := opencode.CreateSession(opts)
    if err != nil {
        beads.RemoveTag(issue.ID, "orch:agent") // Rollback
        removeWorkspace(opts.WorkspacePath)     // Rollback
        return Agent{}, fmt.Errorf("session spawn failed: %w", err)
    }
    
    manifest.SessionID = session.ID
    writeManifest(opts.WorkspacePath, manifest) // Update with session_id
    
    // 4. Write spawn evidence marker
    writeSpawnEvidence(opts.WorkspacePath, SpawnEvidence{
        At: time.Now(),
        By: os.Getenv("USER"),
    })
    
    return Agent{BeadsID: issue.ID, SessionID: session.ID}, nil
}
```

**No partial state.** A half-spawned agent is worse than a failed spawn.

**Tradeoff:** Beads availability becomes a hard dependency. If beads is down, spawn fails. This is intentional — partial state caused the 238-dead-agents bug (orch-go-1074).

### Single-Pass Query with Reason Codes

Never return silent empty metadata. Every missing field must have an explicit reason code.

```go
type AgentStatus struct {
    BeadsID     string
    SessionID   string
    Status      string
    Phase       string
    ProjectDir  string
    
    // Reason codes for missing/partial data
    MissingBinding  bool   // Workspace manifest not found
    MissingSession  bool   // OpenCode session not found
    SessionDead     bool   // Session exists but idle/errored
    MissingPhase    bool   // No Phase comment in beads
    
    Reason string // Human-readable explanation
}

func queryTrackedAgents(projectDirs []string) ([]AgentStatus, error) {
    // 1. Start from beads (source of truth for what work exists)
    issues, err := beads.ListByTag("orch:agent", beads.WithStatus("in_progress"))
    if err != nil {
        return nil, fmt.Errorf("beads query failed: %w", err)
    }
    
    // 2. Batch lookup workspace bindings
    bindings, err := workspace.LookupByBeadsIDs(issueIDs(issues))
    if err != nil {
        return nil, fmt.Errorf("workspace lookup failed: %w", err)
    }
    
    // 3. Batch check session liveness
    sessionIDs := extractSessionIDs(bindings)
    liveness, err := opencode.BatchLiveness(sessionIDs)
    if err != nil {
        // OpenCode down: agents shown with status=unknown
        liveness = unknownLiveness(sessionIDs)
    }
    
    // 4. Join with explicit reason codes
    return joinWithReasonCodes(issues, bindings, liveness), nil
}
```

**Why reason codes:** Every previous debugging session was caused by silent enrichment failures. When the dashboard showed empty metadata for 84/85 agents, there was no indication of *why* — no error, no warning, just missing data. Reason codes make failure modes visible.

### Performance Without Drift

| Cache Type | Allowed | Location | TTL |
|------------|---------|----------|-----|
| In-memory, process-local | Yes | Dashboard server | 1-5 seconds |
| In-memory, process-local | Yes | CLI commands | None (short-lived) |
| Disk-backed, persistent | **NO** | N/A | N/A |

**Rationale:** Disk-backed projection caches for lifecycle data were the root cause of drift. The registry drifted from beads. state.db drifted from OpenCode. Each cache required reconciliation, and reconciliation is where every bug lives.

**Performance targets:**
- Dashboard: <500ms (in-memory cache acceptable for long-lived process)
- CLI: <2s (currently 4s, no cache needed for short-lived process)

**What this rejects:**
- `~/.orch/registry.json` — Deleted
- `~/.orch/sessions.json` — Deleted  
- `~/.orch/state.db` — Deleted
- `pkg/session/` — Deleted
- Any new persistent lifecycle state package

### Regression Guardrails

#### 1. Contract Tests

| Scenario | Expected | Verification |
|----------|----------|--------------|
| Tracked agent spawned | Visible in `orch status` with full metadata | Check beads issue exists, workspace manifest exists, session exists |
| Tracked agent completed | Gone from `orch status` | Beads issue closed |
| `--no-track` agent spawned | Visible in `orch sessions`, NOT in `orch status` | Two-lane split respected |
| Orchestrator session | Visible in `orch sessions` | OpenCode session list |
| Beads down during spawn | Spawn fails with clear error | No partial state left behind |
| OpenCode down | Agents shown with `status=unknown` | Degradation visible, not silent |
| Workspace missing | Agent shown with `reason=missing_binding` | Partial data with reason code |
| Cross-project `--workdir` | Correct `project_dir` in metadata | Workspace manifest has correct path |
| Concurrent spawns (5x) | All 5 visible, no duplicates | Race condition prevention |
| Server restart | Agents survive, no ghosts | Beads is source of truth, not in-memory state |

#### 2. Architecture Lint Rule

**CI gate blocks PRs that add:**
- Any file under `pkg/state/`
- Any file under `pkg/registry/`
- Any file under `pkg/cache/` with lifecycle semantics
- Any `.json` or `.db` file in `~/.orch/` for session/agent state

**Implementation:** Add to `.golangci.yml` or custom lint script:
```yaml
# .github/workflows/lint.yml
- name: Block new lifecycle state packages
  run: |
    if git diff --name-only origin/main | grep -E 'pkg/(state|registry|cache)/'; then
      echo "ERROR: New lifecycle state package detected."
      echo "This architecture uses beads + workspace manifests only."
      echo "See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md"
      exit 1
    fi
```

**Why structural gates:** Agents ignore reminders under pressure. The 5-iteration cycle happened because each iteration seemed locally reasonable ("just a small cache to speed things up"). Structural gates prevent the first step of the accretion pattern.

## Acceptance Test Matrix

| Scenario | Lane | Expected | Reason Code if Missing |
|----------|------|----------|------------------------|
| Tracked agent spawned | `orch status` | Visible with full metadata | — |
| Tracked agent completed | `orch status` | Gone (issue closed) | — |
| `--no-track` agent spawned | `orch sessions` | Visible | — |
| `--no-track` agent spawned | `orch status` | NOT visible (by design) | — |
| Orchestrator session | `orch sessions` | Visible | — |
| Beads down during spawn | — | Spawn fails with error | — |
| OpenCode down | `orch status` | Agents shown, `status=unknown` | `missing_session` |
| Workspace missing | `orch status` | Agent shown, metadata partial | `missing_binding` |
| Cross-project `--workdir` | `orch status` | Correct `project_dir` | — |
| Concurrent spawns (5x) | `orch status` | All 5 visible, no duplicates | — |
| Server restart | `orch status` | Agents survive, no ghosts | — |
| New `pkg/state/` file added | CI | Lint failure, blocked | — |

## Consequences

### Positive
- **Single source of truth per domain:** Beads for work lifecycle, OpenCode for liveness, workspace for binding
- **Silent failures become visible:** Reason codes expose every failure mode
- **Structural prevention of drift:** Lint rules block the first step of cache accretion
- **Semantic clarity:** Tracked work and untracked sessions have different lifecycles; stop pretending they're the same
- **Atomic spawn guarantees:** No partial state means no 238-dead-agents scenarios

### Negative
- **Beads availability becomes hard dependency:** If beads is down, spawn fails (intentional — partial state is worse)
- **CLI performance hit:** No persistent cache means every `orch status` queries beads + workspace + OpenCode (~2s target)
- **Two lanes to maintain:** `orch status` and `orch sessions` have different code paths

### Risks
- **Beads reliability:** Any beads outage blocks all spawning. Mitigation: beads daemon has 99.9%+ uptime in production, and failure mode is clear (spawn fails) not silent (partial state)
- **Two-lane maintenance burden:** Code duplication between tracked and untracked paths. Mitigation: Clear abstraction boundary; each lane is simple on its own
- **Lint rule circumvention:** Developers could add lifecycle state with different package names. Mitigation: Code review culture + principle documentation

## Complementary Work (Separate Track)

**orch-go-1078/1079/1080:** OpenCode fork session metadata API. Makes the authoritative source richer so the workspace cross-reference step eventually becomes unnecessary.

The ADR should note this as a future simplification but not depend on it. This decision works with current OpenCode capabilities.

## References

- `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` — Full cycle history
- `.kb/investigations/2026-02-18-design-agent-observability-rethink.md` — Beads-first design
- `.kb/decisions/2026-02-14-lifecycle-ownership-own-accept-build.md` — Own/Accept/Build
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` — The decision that defended the registry (now superseded)
- orch-go-1074 — 238 dead agents from partial state
- orch-go-1058 — Strategic question that led to beads-first
