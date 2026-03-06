## Summary (D.E.K.N.)

**Delta:** Cross-project identity failures stem from 4 independent project-resolution implementations that each reinvent the same fallback cascade; consolidating into a single `pkg/identity/` package with shared `ResolveProject()` eliminates Class 4 defects structurally.

**Evidence:** 4 cross-project failures in one session (7zg08, wrq9j, akiyk, dnfpw); 7 archived investigations from Dec 2025–Jan 2026 showing the same pattern discovered and patch-fixed repeatedly; 20+ Class 4 instances in defect taxonomy; 4 distinct resolution implementations across abandon, complete, thread, and daemon code.

**Knowledge:** The beads ID prefix already encodes project identity, the `AgentManifest.ProjectDir` is already ground truth, and the daemon's `ProjectRegistry` already does prefix→directory mapping — the missing piece is promoting these from siloed implementations to a shared identity layer.

**Next:** Implement `pkg/identity/` in 4 phases (~450 lines total). Phase 1: extract ProjectRegistry + ResolveProject. Phase 2: wire into commands. Phase 3: liveness guard. Phase 4: spawn-time env vars.

**Authority:** architectural - Cross-component pattern affecting 8+ commands, requires shared package creation and interface changes across module boundaries.

---

# Investigation: Design Cross-Project Identity Layer

**Question:** How should orch represent and enforce cross-project agent identity so that every command correctly handles agents spanning project boundaries?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-jpsun)
**Phase:** Complete
**Next Step:** Implementation via feature-impl in 4 phases
**Status:** Complete

**Patches-Decision:** N/A (new design; may promote to decision when accepted)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2025-12-25 Cross-Project Agent Visibility Fetch | extends | Yes — extractProjectDirFromWorkspace() still exists | None |
| 2025-12-26 Design Proper Cross-Project Agent Visibility | extends | Yes — kb projects list is the registry, not OpenCode session dirs | OpenCode session.Directory assumption was wrong (Jan 7 investigation confirmed) |
| 2025-12-26 Improve Orch Abandon Cross-Project | extends | Yes — --workdir pattern replicated in 5 commands | None |
| 2025-12-27 Design Cross-Project Completion UX | extends | Yes — 3-layer fallback in complete_pipeline.go | None |
| 2026-01-06 Cross-Project Daemon Single Daemon | extends | Yes — ProjectRegistry in pkg/daemon/ confirmed working | None |
| 2026-01-07 Cross-Project Agents Show Wrong Project Dir | confirms | Yes — all OpenCode sessions show server CWD, not agent CWD | None |
| 2026-02-18 Two-Lane Agent Discovery ADR | extends | Yes — QueryTrackedAgents already cross-project | None |

---

## Findings

### Finding 1: Four independent project-resolution implementations with divergent behavior

**Evidence:** Each command independently implements cross-project resolution with different fallback logic:

| Command | Implementation | Auto-resolves? | Fallback |
|---------|---------------|----------------|----------|
| `abandon` | `runAbandon()` at abandon_cmd.go:73 | Yes (via `resolveProjectDirForBeadsID`) | Iterate all KB projects |
| `complete` | `resolveCompletionTarget()` at complete_pipeline.go:55 | Yes (3-layer: workspace scan, beads prefix parse, --workdir) | `findProjectDirByName()` with hardcoded path candidates |
| `thread` | `threadsDir()` at thread_cmd.go:16 | No (CWD only, fixed with --workdir in wrq9j) | None — fails silently |
| `daemon` | `ProjectRegistry.Resolve()` at pkg/daemon/project_resolution.go | Yes (prefix→directory mapping from KB projects + beads config) | Falls back to current dir |

**Source:** cmd/orch/abandon_cmd.go:73-114, cmd/orch/complete_pipeline.go:55-120, cmd/orch/thread_cmd.go:16-24, pkg/daemon/project_resolution.go

**Significance:** The same conceptual operation (beads ID → project directory) is implemented 4 different ways. When a new command is added, the developer must rediscover the pattern. The `thread` command didn't have auto-resolve at all until today's fix. This is the structural root cause of recurring Class 4 defects — there's no canonical resolution function.

---

### Finding 2: The daemon's ProjectRegistry already solves the prefix→directory mapping

**Evidence:** `ProjectRegistry` in `pkg/daemon/project_resolution.go` builds a `map[string]string` (prefix→directory) from `kb projects list --json` + reading each project's `.beads/config.yaml` for `issue-prefix`. It resolves beads IDs like `pw-ed7h` to `/Users/dylanconlin/Documents/personal/price-watch` in O(1) without iterating all projects.

By contrast, `resolveProjectDirForBeadsID()` in `cmd/orch/shared.go:106` iterates all KB projects and probes beads for each one — O(projects) with shell-outs per project.

**Source:** pkg/daemon/project_resolution.go (full file), cmd/orch/shared.go:106-114

**Significance:** The fast, correct implementation exists but is locked inside the daemon package. Promoting it to a shared utility gives all commands O(1) project resolution. The daemon already validates this approach works for 17+ registered projects.

---

### Finding 3: Destructive commands lack cross-project liveness verification

**Evidence:** The `orch abandon` phantom incident (orch-go-dnfpw) killed 3 live cross-project agents because they appeared as phantoms from the wrong project's tmux session perspective. The fix added a 30-minute activity check via phase comment recency. However:

- `orch complete` has no equivalent guard — a cross-project agent in-progress could be completed from the wrong project
- The activity check only examines beads comments — it doesn't verify whether the agent's tmux window actually exists in the agent's *own* project's tmux session (it checks the caller's project)
- The `CheckTmuxWindowAlive` function in discovery.go:63 correctly uses `projectDir` to find the right tmux session, but abandon doesn't call it

**Source:** cmd/orch/abandon_cmd.go:282-333 (checkRecentActivity), pkg/discovery/discovery.go:63-71 (CheckTmuxWindowAlive)

**Significance:** The liveness check exists in two places — phase recency (abandon) and tmux window existence (discovery) — but they're not combined. Defect Class 7 prevention requires "2+ independent signals before killing." The infrastructure exists; it just needs to be composed into a shared guard.

---

### Finding 4: AgentManifest.ProjectDir is ground truth but underutilized

**Evidence:** Every spawned agent has `ProjectDir` in `AGENT_MANIFEST.json` (written at spawn via `pkg/spawn/context.go:816`). The discovery engine reads it (`LookupManifestsAcrossProjects`). But most commands don't consult the manifest for project resolution — they re-derive it from beads ID prefix parsing, KB project iteration, or `SPAWN_CONTEXT.md` heuristic parsing.

`extractProjectDirFromWorkspace()` reads `SPAWN_CONTEXT.md` to find `PROJECT_DIR:` — a heuristic parse of a markdown file — when the same data is available in structured JSON in `AGENT_MANIFEST.json`.

**Source:** pkg/spawn/session.go:170-212 (manifest struct), cmd/orch/complete_pipeline.go (extractProjectDirFromWorkspace)

**Significance:** The manifest is the canonical identity document per Session Amnesia ("self-describing artifacts"). Resolution functions should consult it first, falling back to prefix parsing only when workspace isn't found.

---

### Finding 5: Beads ID prefix reliably encodes project identity

**Evidence:** `extractProjectFromBeadsID()` in `cmd/orch/shared.go:118` parses the project name from beads IDs. Beads IDs follow the pattern `{issue-prefix}-{4char}` where issue-prefix is configured in `.beads/config.yaml`. This is deterministic and immutable after creation.

The daemon's `ProjectRegistry` maps prefixes to directories using `kb projects list` + beads config. This mapping is stable across sessions — projects don't change their issue prefix.

However, `findProjectDirByName()` in status_cmd.go has hardcoded fallback paths (`~/Documents/personal/<name>`, `~/projects/<name>`) — fragile and non-generalizable.

**Source:** cmd/orch/shared.go:118-130, pkg/daemon/project_resolution.go, cmd/orch/status_cmd.go (findProjectDirByName)

**Significance:** The beads ID prefix is a reliable, zero-cost identity signal that every command already has access to. Combined with `ProjectRegistry`, it provides O(1) resolution without iterating projects or parsing workspace files.

---

## Synthesis

**Key Insights:**

1. **The identity layer already exists — it's just fragmented.** AgentManifest.ProjectDir is ground truth. Beads ID prefix encodes project. ProjectRegistry maps prefix→directory. CheckTmuxWindowAlive verifies liveness per-project. Phase comments provide recency. All 5 components exist. They're just scattered across daemon, discovery, shared, abandon, and complete — never composed into a single coherent layer.

2. **The recurring Class 4 pattern is a missing canonical function.** Principles "Evolve by distinction" and "Coherence over patches" both point to the same conclusion: 4+ independent implementations of the same operation means consolidation. Every new orch command will need project resolution. Without a canonical function, each will reinvent it with new bugs.

3. **Destructive protection requires composing existing signals.** The abandon fix (phase recency) and the discovery engine (tmux window check) each provide one signal. Defect Class 7 prevention requires 2+. The liveness guard should compose both, checking the agent's *actual* project tmux session, not the caller's.

**Answer to Investigation Question:**

The cross-project identity model should be a shared `pkg/identity/` package that consolidates the 4 existing resolution implementations into a single canonical function. It promotes the daemon's `ProjectRegistry` (prefix→directory mapping) to a shared utility, composes the existing liveness signals (phase recency + tmux window check) into a destructive command guard, and enriches spawn-time environment with explicit identity variables. The design adds no new state (respecting "No Local Agent State") and no central registry (respecting "inline lineage metadata"). It simply makes the fragmented identity infrastructure available to all commands through a shared interface.

---

## Structured Uncertainty

**What's tested:**

- ✅ `ProjectRegistry` correctly maps 17+ projects via prefix→directory (daemon uses it daily for cross-project spawns)
- ✅ `AgentManifest.ProjectDir` is written for every spawn (verified in pkg/spawn/context.go:816-829)
- ✅ Phase recency check prevents accidental abandon of active agents (abandon_cmd.go, tested in abandon_cmd_test.go)
- ✅ `CheckTmuxWindowAlive` correctly checks per-project tmux sessions (discovery.go:63-71)
- ✅ Beads ID prefix parsing works for all project naming conventions (extractProjectFromBeadsID in shared.go)

**What's untested:**

- ⚠️ Composing phase recency + tmux liveness into a single guard (not yet built)
- ⚠️ Performance of ProjectRegistry construction in command-line context (works in daemon's long-running process; untested for per-invocation construction)
- ⚠️ Edge case: project with beads prefix different from directory name AND no KB registration (would fail resolution)
- ⚠️ Race condition: agent closes between resolution and command execution (theoretical, mitigated by idempotent operations)

**What would change this:**

- If `kb projects list` becomes unreliable or slow (>500ms), the per-invocation ProjectRegistry construction would need caching
- If beads ID prefixes are no longer unique across projects, prefix→directory mapping breaks (currently guaranteed by beads config)
- If the system expands to 50+ projects, iterating all for fallback resolution could become noticeable (~50 shell-outs)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create `pkg/identity/` with shared ResolveProject | architectural | New shared package affecting 8+ commands, cross-component |
| Promote ProjectRegistry from daemon-only | architectural | Changes daemon's internal to shared interface |
| Add VerifyLiveness guard | implementation | Composing existing functions, no new interface |
| Spawn-time env vars | implementation | Additive change within existing spawn flow |

### Recommended Approach ⭐

**Shared Identity Layer (`pkg/identity/`)** - Consolidate 4 independent project-resolution implementations into a single package with `ResolveProject()`, promote `ProjectRegistry` from daemon-only to shared, and compose existing liveness signals into a destructive command guard.

**Why this approach:**
- Directly addresses the structural root cause (Finding 1): fragmented resolution logic
- Leverages existing, proven components (Finding 2): ProjectRegistry already works for 17+ projects
- Composes existing signals (Finding 3): phase recency + tmux liveness = Class 7 prevention
- Follows substrate: "Coherence over patches" principle, "No Local Agent State" constraint, "inline lineage metadata" decision
- Prevents future Class 4 instances: new commands call `identity.ResolveProject()` instead of reinventing

**Trade-offs accepted:**
- Per-invocation ProjectRegistry construction adds ~50-100ms to each command (shell-out to `kb projects list` + read N beads configs)
- Commands must import `pkg/identity/` — adds a dependency, but a correct one

**Defect class exposure:**
- Class 0 (Scope Expansion): Low — `ResolveProject` has a defined output contract
- Class 1 (Filter Amnesia): Eliminated — single filter function shared by all commands
- Class 4 (Boundary Bleed): Eliminated — canonical resolution replaces CWD assumptions
- Class 5 (Contradictory Authority): Eliminated — manifest is ground truth, prefix is fast index
- Class 7 (Premature Destruction): Mitigated — `VerifyLiveness()` composes 2+ signals

**Implementation sequence:**

#### Phase 1: Extract and Promote (~200 lines)

Create `pkg/identity/` with two files:

**`pkg/identity/registry.go`** — Promoted from `pkg/daemon/project_resolution.go`:
```go
type ProjectRegistry struct {
    prefixToDir map[string]string
    currentDir  string
}

func NewProjectRegistry() (*ProjectRegistry, error)  // kb projects list + beads configs
func (r *ProjectRegistry) Resolve(beadsID string) string  // prefix → directory, O(1)
func (r *ProjectRegistry) AllProjects() []string  // for fallback iteration
```

**`pkg/identity/resolve.go`** — Canonical resolution function:
```go
// ResolveProject determines which project directory owns a beads ID.
// Three-layer fallback:
//   1. Explicit --workdir override (highest priority)
//   2. ProjectRegistry prefix → directory mapping (O(1), fast)
//   3. Iterate all projects, probe beads (O(n), reliable fallback)
// Returns the absolute project directory path.
func ResolveProject(beadsID, workdirOverride string) (string, error)

// ResolveProjectWithManifest tries the workspace manifest first, then falls back
// to ResolveProject. Use when you have a workspace path.
func ResolveProjectWithManifest(workspacePath, beadsID, workdirOverride string) (string, error)
```

Update `pkg/daemon/` to import from `pkg/identity/` instead of its own `project_resolution.go`.

#### Phase 2: Wire into Commands (~100 lines)

Replace ad-hoc resolution in:
- `abandon_cmd.go`: Replace inline auto-resolve (lines 73-114) with `identity.ResolveProject()`
- `complete_pipeline.go`: Replace 3-layer fallback with `identity.ResolveProject()`
- `thread_cmd.go`: Replace CWD + --workdir with `identity.ResolveProject()`
- `send_cmd.go`: Add project resolution (currently CWD-only)
- `tail_cmd.go`: Add project resolution (currently CWD-only)

Each command's `--workdir` flag is preserved as the explicit override passed to `ResolveProject()`.

#### Phase 3: Destructive Command Guard (~100 lines)

**`pkg/identity/liveness.go`**:
```go
// VerifyLiveness checks multiple signals before allowing destructive actions.
// Returns nil if safe to proceed, error with explanation if agent appears active.
// Checks: (1) phase comment recency, (2) tmux window in agent's actual project.
func VerifyLiveness(beadsID, projectDir string) error
```

Wire into:
- `abandon_cmd.go`: Replace `checkRecentActivity()` with `identity.VerifyLiveness()`
- `complete_cmd.go`: Add guard before force-completion of non-Phase:Complete agents

#### Phase 4: Spawn-Time Identity Envelope (~50 lines)

In `pkg/spawn/context.go` `WriteContext()`, add environment variables:
- `ORCH_PROJECT_DIR` — absolute path to agent's owning project
- `ORCH_SOURCE_PROJECT` — source project name if cross-repo (empty otherwise)
- `BEADS_DIR` — already implemented (keep as-is)

These env vars make identity available to the agent's session without reading files.

### Alternative Approaches Considered

**Option B: Cobra Middleware**
- **Pros:** Automatic — commands don't need to call resolution explicitly
- **Cons:** Invisible coupling; harder to test; not all commands need resolution (e.g., `orch version`); middleware pattern doesn't compose well with Cobra's command tree
- **When to use instead:** If orch had 50+ commands that all need resolution (currently ~15, only 8 need it)

**Option C: Central Agent Registry**
- **Pros:** Single query for all agent info; fast
- **Cons:** Violates "No Local Agent State" architectural constraint; violates "inline lineage metadata" decision; creates synchronization/staleness problems; adds state that can drift
- **When to use instead:** Never — explicitly rejected by two constraints

**Option D: Beads-only identity (add `project:<dir>` label)**
- **Pros:** All identity in beads; discoverable via `bd list -l project:orch-go`
- **Cons:** Beads labels are strings, not structured; duplicates info already in manifest; labels can drift if project moves; doesn't eliminate the resolution problem (still need label→directory mapping)
- **When to use instead:** If workspace manifests didn't exist

**Rationale for recommendation:** Option A (shared identity layer) consolidates existing proven components without adding new state, respects both architectural constraints, and provides the simplest path to eliminating Class 4 defects. Every alternative either adds state (violating constraints) or adds indirection without solving the core problem.

---

### Implementation Details

**What to implement first:**
- `pkg/identity/registry.go` (Phase 1) — this is the foundation everything else uses
- The daemon import update should happen in the same PR to avoid divergence

**Things to watch out for:**
- ⚠️ `findProjectDirByName()` in status_cmd.go has hardcoded path candidates — should be replaced with `ProjectRegistry.Resolve()` but may break if any project isn't KB-registered
- ⚠️ `beads.DefaultDir` global state — `ResolveProject()` must NOT set this; callers should pass the resolved dir explicitly to beads functions
- ⚠️ Testing: Mock `kb projects list` output, not the shell command itself — use the existing `getKBProjectsFn` injection pattern
- ⚠️ Race between resolution and action (agent closes between resolve and abandon) — mitigated by idempotent beads operations, but worth noting in tests

**Areas needing further investigation:**
- `send_cmd.go` and `tail_cmd.go` cross-project behavior (not examined in detail — likely CWD-only)
- Whether `orch work` needs identity resolution (it resolves via daemon's ProjectRegistry already, but manual `orch work <beads-id>` from wrong CWD might not)
- Impact on web UI dashboard — `serve_agents.go` handlers may have their own resolution logic

**Success criteria:**
- ✅ `orch abandon <cross-project-beads-id>` auto-resolves without --workdir and checks liveness
- ✅ `orch complete <cross-project-beads-id>` auto-resolves without --workdir
- ✅ `orch thread new <cross-project-beads-id> "content"` auto-resolves without --workdir
- ✅ No orch command uses `os.Getwd()` for project resolution (grep test)
- ✅ `identity.ResolveProject()` returns same result as --workdir explicit path for all known projects
- ✅ Destructive commands (abandon, complete) refuse to act on agents with recent activity + live tmux window unless --force

---

## References

**Files Examined:**
- `cmd/orch/abandon_cmd.go` — Abandon command with inline cross-project resolution
- `cmd/orch/complete_pipeline.go` — Complete with 3-layer fallback
- `cmd/orch/thread_cmd.go` — Thread with CWD assumption (fixed in wrq9j)
- `cmd/orch/shared.go` — `resolveProjectDirForBeadsID()`, `extractProjectFromBeadsID()`, `findWorkspaceByBeadsID()`
- `cmd/orch/status_cmd.go` — Status with cross-project via discovery engine
- `pkg/daemon/project_resolution.go` — `ProjectRegistry` with prefix→directory mapping
- `pkg/discovery/discovery.go` — `QueryTrackedAgents()`, `CheckTmuxWindowAlive()`
- `pkg/spawn/session.go` — `AgentManifest` struct with `ProjectDir`
- `pkg/spawn/context.go` — `WriteContext()` manifest population
- `pkg/spawn/atomic.go` — `AtomicSpawnPhase1/2` identity writing
- `pkg/orch/spawn_beads.go` — Cross-repo detection, BEADS_DIR resolution
- `.kb/models/defect-class-taxonomy/model.md` — Class 4 (Boundary Bleed) and Class 7 (Premature Destruction)
- `~/.kb/principles.md` — "Evolve by distinction", "Coherence over patches"

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-05-does-detection-become-prevention.md` — Origin of this design question
- **Decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Discovery engine architecture
- **Investigations:** `.kb/investigations/archived/cross-project-operations/` — 7 prior cross-project investigations (Dec 2025 – Jan 2026)
- **Defect Model:** `.kb/models/defect-class-taxonomy/model.md` — Class 4 and Class 7 definitions

---

## Investigation History

**2026-03-05 14:00:** Investigation started
- Initial question: How should orch represent and enforce cross-project agent identity?
- Context: 4 cross-project failures in one session triggered design review

**2026-03-05 14:30:** Exploration complete
- Found 4 independent resolution implementations with divergent behavior
- Identified ProjectRegistry in daemon as the correct solution already existing
- Confirmed liveness signals exist but aren't composed

**2026-03-05 15:00:** Synthesis complete
- 5 forks navigated with substrate consultation
- Recommendation: shared pkg/identity/ consolidating existing components
- 4-phase implementation plan (~450 lines)

**2026-03-05 15:30:** Investigation completed
- Status: Complete
- Key outcome: Cross-project identity layer design as shared `pkg/identity/` package consolidating 4 fragmented implementations
