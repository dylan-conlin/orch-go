# Design: Atomic Spawn with Workspace Manifest and orch:agent Tagging

**Date:** 2026-02-19
**Phase:** Complete
**Status:** Complete
**Issue:** orch-go-1083
**ADR:** .kb/decisions/2026-02-18-two-lane-agent-discovery.md

## Design Question

How should orch-go implement atomic spawn (all-or-nothing writes), evolve the workspace manifest to be the canonical binding, apply `orch:agent` tagging at spawn time, and inject session metadata — across all 4 spawn backends (headless, inline, tmux, claude)?

## Problem Framing

### Success Criteria
1. No partial state after spawn failures (the 238-dead-agents bug class is eliminated)
2. Every tracked agent discoverable via `beads.List({Labels: ["orch:agent"], Status: "in_progress"})`
3. Workspace manifest contains the complete binding: beads_id ↔ session_id ↔ project_dir
4. Session metadata written at spawn for API-created sessions
5. All 4 backends (headless, inline, tmux, claude) participate in atomic spawn

### Constraints
- **No Local Agent State** principle: No new caches or projection DBs
- **Graceful Degradation** principle: Claude backend has no OpenCode session — the contract must work without it
- **Escape Hatches** principle: Claude backend is the escape hatch for infrastructure work — it must not be blocked by OpenCode availability
- **Compose Over Monolith**: Atomic spawn should be a clean function, not scattered across backends

### Scope
- **In:** Atomic spawn function, AGENT_MANIFEST.json evolution, orch:agent labeling, session metadata injection, rollback on failure
- **Out:** `orch status` query refactoring (separate issue orch-go-1085), `orch sessions` command (orch-go-1087), dashboard changes

## Exploration (Fork Navigation)

### Fork 1: Where does atomic spawn live?

**Options:**
- A: New `pkg/spawn/atomic.go` — Dedicated file in existing spawn package
- B: New `pkg/atomic/` package — Separate package for atomicity
- C: Inline in `pkg/orch/extraction.go` — Add to existing spawn pipeline

**Substrate says:**
- Principle: Compose Over Monolith — small focused modules
- Model: Spawn Architecture — spawn logic is already split between `pkg/spawn/` (config, context) and `pkg/orch/` (pipeline orchestration)
- Current code: `pkg/orch/extraction.go` already has 1,984 lines

**Recommendation:** Option A — `pkg/spawn/atomic.go`

The spawn package already owns workspace writes (context.go, session.go). Atomicity is a property of the spawn write sequence, not a separate concern. Putting it in `pkg/orch/` would add to an already large file. A separate package adds indirection without value.

**Trade-off:** Tighter coupling between spawn steps than the current scattered approach, but that's the point — the current scattered approach is what causes partial state.

### Fork 2: How does atomic spawn handle the 4 backends?

**Options:**
- A: **Unified atomic function for all backends** — One function handles all cases, backends only differ in session creation
- B: **Per-backend atomic wrappers** — Each backend implements its own atomic sequence
- C: **Two-phase atomic** — Phase 1 (pre-session: beads + workspace) is common; Phase 2 (session-specific) varies per backend

**Substrate says:**
- Model: Spawn Architecture invariant "Session scoping is per-project"
- Probe finding: Claude backend has no OpenCode session, tmux captures session ID via retry-based discovery
- Decision: "No partial state. A half-spawned agent is worse than a failed spawn"

**Recommendation:** Option C — Two-phase atomic

**Rationale:** The 4 backends fundamentally differ in how (and whether) they create OpenCode sessions:

| Backend | Session Creation | Session ID Available? |
|---------|-----------------|----------------------|
| Headless | API call (synchronous) | Immediately |
| Inline | API call (synchronous) | Immediately |
| Tmux | TUI creates it | After retry-based discovery (~1-3s) |
| Claude | Claude CLI binary | Never (no OpenCode session) |

A unified function would need complex branching. Per-backend wrappers would duplicate rollback logic. Two-phase splits cleanly:

- **Phase 1 (common):** Tag beads issue with `orch:agent` + write initial workspace manifest (without session_id)
- **Phase 2 (per-backend):** Create session (if applicable) + update manifest with session_id + write session metadata

Phase 1 rollback is always: remove `orch:agent` label + delete workspace dir.
Phase 2 rollback adds: delete session (if created).

### Fork 3: Should AgentManifest.SessionID be required or optional?

**Options:**
- A: Required — Fail spawn if session ID unavailable
- B: Optional — Allow empty SessionID (claude backend, tmux discovery failure)
- C: Deferred — Write SessionID to manifest asynchronously after session is confirmed

**Substrate says:**
- Constraint: "Claude backend cannot participate in session metadata" (probe finding)
- Model: Spawn Architecture — tmux session ID capture is already best-effort (`client.FindRecentSessionWithRetry()`)
- Principle: Graceful Degradation

**Recommendation:** Option B — Optional SessionID

Claude backend legitimately has no OpenCode session. Tmux session discovery is best-effort. Making SessionID required would block the escape hatch and break tmux spawns on discovery failures. The manifest already works without it — `ReadAgentManifestWithFallback` handles missing fields.

**Trade-off:** `queryTrackedAgents()` (orch-go-1085) must handle agents without session_id, but this is a reasonable degraded state — agent is tracked via beads, just can't query liveness from OpenCode.

### Fork 4: How to evolve AGENT_MANIFEST.json?

**Options:**
- A: Add SessionID field to existing AgentManifest struct
- B: Create new WorkspaceManifest struct (breaking rename)
- C: Add SessionID + rename file to `manifest.json`

**Substrate says:**
- Model: Spawn Architecture — AGENT_MANIFEST.json is already the canonical binding
- ADR: References `WorkspaceManifest` as the concept but doesn't mandate a specific struct name
- Principle: "Evolve by distinction" — but this isn't a conflation problem, it's an extension

**Recommendation:** Option A — Add SessionID to existing struct

The existing `AgentManifest` struct is 90% of what the ADR calls a "workspace manifest." It already has `WorkspaceName`, `Skill`, `BeadsID`, `ProjectDir`, `SpawnTime`, `Tier`, `SpawnMode`, `Model`. Adding `SessionID` completes it. A rename would break all existing workspace readers for no functional gain.

### Fork 5: When in the pipeline does orch:agent tagging happen?

**Options:**
- A: At issue creation time (in `CreateBeadsIssue`)
- B: After beads tracking setup (after `SetupBeadsTracking`)
- C: Inside the atomic spawn function (first step of Phase 1)

**Substrate says:**
- ADR: "Create beads issue OR tag existing issue with orch:agent"
- Current flow: `SetupBeadsTracking` handles both creation and existing issue resolution
- Constraint: Must be first step so rollback can remove it

**Recommendation:** Option C — Inside atomic spawn function

The label is the "I am an agent" marker. It should be applied as the first step of the atomic sequence and rolled back if anything fails. Putting it in `CreateBeadsIssue` would apply it before the spawn decides to proceed. Putting it after `SetupBeadsTracking` but before atomic spawn would leave it applied on spawn failure.

**Trade-off:** `orch:agent` is applied slightly later in the pipeline (after all pre-flight checks, context gathering). This is correct — the label means "an agent is running for this issue," not "someone thought about spawning for this issue."

## Synthesis

### Architecture Recommendation

```
                    SPAWN PIPELINE (existing)
                    ┌─────────────────────────┐
                    │ 1. Pre-flight checks     │
                    │ 2. Resolve project dir   │
                    │ 3. Load skill/workspace  │
                    │ 4. Setup beads tracking  │  ← Issue created here, but NOT tagged yet
                    │ 5. Resolve spawn settings│
                    │ 6-9. Gather context      │
                    │ 10. Build spawn context  │
                    │ 11. Build spawn config   │
                    │ 13. Write SPAWN_CONTEXT  │  ← REMOVE workspace writes from here
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   ATOMIC SPAWN          │  ← NEW: pkg/spawn/atomic.go
                    │                         │
                    │ Phase 1 (common):       │
                    │  1. Add orch:agent label │
                    │  2. Write workspace      │
                    │     manifest (initial)   │
                    │                         │
                    │ Phase 2 (per-backend):  │
                    │  3. Create session       │
                    │  4. Update manifest with │
                    │     session_id           │
                    │  5. Write spawn evidence │
                    │                         │
                    │ On failure: rollback all │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │ 14. Log event, print    │
                    │     summary             │
                    └─────────────────────────┘
```

### Implementation Plan

#### Phase 1: Evolve AgentManifest (small, safe)

**File:** `pkg/spawn/session.go`

1. Add `SessionID` field to `AgentManifest` struct:
   ```go
   type AgentManifest struct {
       // ... existing fields ...
       SessionID string `json:"session_id,omitempty"`
   }
   ```

2. Update `readFromOpenCodeMetadata()` to populate `SessionID` from the session it queries.

3. Add batch lookup function:
   ```go
   // LookupManifestsByBeadsIDs scans workspace directories and returns manifests
   // indexed by beads_id. Used by queryTrackedAgents for batch binding lookup.
   func LookupManifestsByBeadsIDs(projectDir string, beadsIDs []string) (map[string]*AgentManifest, error)
   ```
   Implementation: scan `.orch/workspace/*/AGENT_MANIFEST.json`, filter by beads_id match, return map.

**Tests:** Unit tests for SessionID serialization, batch lookup with multiple workspaces.

#### Phase 2: Create atomic spawn function

**New file:** `pkg/spawn/atomic.go`

```go
// AtomicSpawnOpts holds parameters for atomic spawn.
type AtomicSpawnOpts struct {
    Config      *Config
    BeadsID     string
    NoTrack     bool
    ServerURL   string
}

// AtomicSpawnResult holds the output of a successful atomic spawn.
type AtomicSpawnResult struct {
    SessionID    string
    WorkspacePath string
    ManifestPath  string
}

// AtomicSpawnPhase1 performs the common pre-session writes.
// Returns a rollback function that undoes all Phase 1 writes.
// Must be called before session creation.
func AtomicSpawnPhase1(opts *AtomicSpawnOpts) (rollback func(), err error) {
    var cleanups []func()

    rollback = func() {
        // Execute cleanups in reverse order
        for i := len(cleanups) - 1; i >= 0; i-- {
            cleanups[i]()
        }
    }

    // Step 1: Tag beads issue with orch:agent
    if !opts.NoTrack && opts.BeadsID != "" {
        if err := tagBeadsAgent(opts.BeadsID); err != nil {
            return rollback, fmt.Errorf("beads tag failed: %w", err)
        }
        cleanups = append(cleanups, func() {
            untagBeadsAgent(opts.BeadsID)
        })
    }

    // Step 2: Write workspace (SPAWN_CONTEXT.md + manifest without session_id)
    if err := WriteContext(opts.Config); err != nil {
        rollback()
        return nil, fmt.Errorf("workspace write failed: %w", err)
    }
    cleanups = append(cleanups, func() {
        os.RemoveAll(opts.Config.WorkspacePath())
    })

    return rollback, nil
}

// AtomicSpawnPhase2 performs the session-specific writes.
// Called by each backend after session creation.
// Updates manifest with session_id and writes spawn evidence.
func AtomicSpawnPhase2(opts *AtomicSpawnOpts, sessionID string) error {
    workspacePath := opts.Config.WorkspacePath()

    // Step 3: Write session ID
    if sessionID != "" {
        if err := WriteSessionID(workspacePath, sessionID); err != nil {
            return fmt.Errorf("session ID write failed: %w", err)
        }
    }

    // Step 4: Update manifest with session_id
    manifest, err := ReadAgentManifest(workspacePath)
    if err == nil && manifest != nil {
        manifest.SessionID = sessionID
        if err := WriteAgentManifest(workspacePath, *manifest); err != nil {
            // Non-fatal: manifest still has all other fields
            fmt.Fprintf(os.Stderr, "Warning: failed to update manifest with session ID: %v\n", err)
        }
    }

    // Step 5: Write spawn evidence marker
    writeSpawnEvidence(workspacePath)

    return nil
}
```

**Key design decisions:**
- Phase 1 returns a `rollback` function — callers don't need to know cleanup details
- Phase 2 is non-rolling-back: if session was created, we don't delete it on Phase 2 write failures (the session is the expensive thing, not the metadata)
- `tagBeadsAgent` / `untagBeadsAgent` are thin wrappers around beads `AddLabel` / `RemoveLabel`

**Tests:**
- Phase 1 rollback on beads failure
- Phase 1 rollback on workspace write failure
- Phase 2 with empty session ID (claude backend)
- Phase 2 manifest update

#### Phase 3: Integrate into spawn pipeline

**Files modified:**
- `pkg/spawn/context.go` — Extract workspace writes from `WriteContext()` into separate function (WriteContext still generates content, but writes move to atomic)
- `pkg/orch/extraction.go` — Modify `ValidateAndWriteContext()` and `DispatchSpawn()` to use atomic spawn
- `pkg/spawn/backends/headless.go` — Session creation integrated with Phase 2

**Integration approach:**

```go
// In pkg/orch/extraction.go, replace steps 13-14:

// Step 13: Atomic spawn Phase 1 (beads tag + workspace writes)
atomicOpts := &spawn.AtomicSpawnOpts{
    Config:    cfg,
    BeadsID:   beadsID,
    NoTrack:   cfg.NoTrack,
    ServerURL: serverURL,
}
rollback, err := spawn.AtomicSpawnPhase1(atomicOpts)
if err != nil {
    return err
}

// Step 14: Dispatch spawn (backends call Phase 2 after session creation)
if err := DispatchSpawn(input, cfg, minimalPrompt, beadsID, skillName, task, serverURL, atomicOpts); err != nil {
    rollback() // Undo beads tag + workspace on spawn failure
    return err
}
```

**Backend integration (headless example):**

```go
// In runSpawnHeadless, after startHeadlessSession succeeds:
if err := spawn.AtomicSpawnPhase2(atomicOpts, sessionID); err != nil {
    fmt.Fprintf(os.Stderr, "Warning: Phase 2 writes failed: %v\n", err)
    // Don't rollback — session is running, metadata is best-effort
}
```

#### Phase 4: Session metadata enhancement

**File:** `pkg/spawn/backends/headless.go` (and inline path in extraction.go)

Ensure metadata includes all fields the ADR specifies:
```go
metadata := map[string]string{
    "beads_id":       cfg.BeadsID,
    "workspace_path": cfg.WorkspacePath(),
    "tier":           cfg.Tier,
    "spawn_mode":     "headless",
    "skill":          cfg.SkillName,       // NEW
    "model":          cfg.Model,           // NEW
}
```

**Tmux backend:** After session discovery (`FindRecentSessionWithRetry`), call `client.SetSessionMetadata()` to inject metadata into the TUI-created session. This makes tmux sessions discoverable the same way as headless sessions.

**Claude backend:** No session metadata possible. The manifest is the sole binding.

### File Impact Summary

| File | Change | Lines (est.) |
|------|--------|-------------|
| `pkg/spawn/atomic.go` | **NEW** — Atomic spawn phases + rollback | ~150 |
| `pkg/spawn/session.go` | Add SessionID to AgentManifest + batch lookup | ~50 |
| `pkg/spawn/context.go` | Refactor WriteContext to separate content generation from writes | ~30 |
| `pkg/orch/extraction.go` | Integrate atomic spawn into pipeline | ~40 |
| `pkg/spawn/backends/headless.go` | Add skill/model to metadata | ~5 |
| `pkg/spawn/atomic_test.go` | **NEW** — Tests for atomic spawn | ~200 |
| `pkg/spawn/session_test.go` | Tests for SessionID and batch lookup | ~100 |
| **Total** | | ~575 |

### Blocking Questions

None — all forks navigated with clear substrate reasoning. The ADR provides sufficient design direction.

### Discovered Work

1. **Tmux backend session metadata injection** — After `FindRecentSessionWithRetry`, call `SetSessionMetadata()` to inject beads_id/workspace_path into TUI-created sessions. This makes tmux sessions discoverable via Lane 1 queries. (dependency: `SetSessionMetadata` wrapper in pkg/opencode/client.go — may need to be added if not present)

2. **OpenCode SetSessionMetadata client method** — Verify if `PATCH /session/:id` with metadata body is already wrapped in `pkg/opencode/client.go`. If not, add it as prerequisite for tmux metadata injection.

3. **Spawn evidence marker format** — The ADR mentions "spawn evidence marker" but doesn't specify format. Recommend: `.spawn_evidence` file with JSON `{spawned_at, spawned_by, spawn_mode, backend}`. This provides forensic data for debugging half-spawned states.

## Recommendations

⭐ **RECOMMENDED:** Two-phase atomic spawn in `pkg/spawn/atomic.go`

- **Why:** Cleanly separates the common pre-session writes (beads tag, workspace manifest) from backend-specific session creation. Rollback is simple — Phase 1 returns a cleanup function. Phase 2 is best-effort (session already exists).
- **Trade-off:** Adds a new file and changes the spawn pipeline flow. But the current fire-and-forget approach is the root cause of the 238-dead-agents bug class.
- **Expected outcome:** Zero partial-state agents. Every tracked agent discoverable via `orch:agent` label. Workspace manifest is the canonical binding.

**Alternative: Inline atomicity in each backend**
- **Pros:** No pipeline changes, each backend handles its own rollback
- **Cons:** Duplicated rollback logic across 4 backends, easy to forget rollback in new backends
- **When to choose:** If we were confident no new backends would be added

## Decision Gate Guidance

**Add `blocks:` frontmatter when promoting to decision:**
This decision directly resolves the recurring ghost-agent and partial-state problems (5+ investigations, 238-dead-agents incident). Future agents working on spawn should be blocked by this to prevent re-introducing fire-and-forget writes.

**Suggested blocks keywords:**
- `atomic spawn`
- `spawn pipeline`
- `agent discovery`
- `workspace manifest`
- `partial state`
