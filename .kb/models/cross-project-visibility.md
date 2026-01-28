# Model: Cross-Project Agent Visibility

**Domain:** Dashboard / Agent Discovery / Multi-Project Orchestration
**Last Updated:** 2026-01-27
**Synthesized From:** 9 cross-project investigations, 6 untracked agent investigations, follow-orchestrator-mechanism model, dashboard-agent-status model

---

## Summary (30 seconds)

Cross-project visibility is how agents spawned in different repositories appear in the dashboard and CLI. The system uses **project name extraction from beads IDs** to identify agents, **MultiProjectConfig** to define which projects are visible together, and **follow-orchestrator filtering** to show only relevant agents in production views. Untracked spawns (`--no-track`) are intentionally filtered from dashboard views because their project name format (`{project}-untracked`) doesn't match `included_projects`.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PROJECT NAME FLOW                                  │
│                                                                             │
│  SPAWN                    BEADS ID                    DASHBOARD FILTER      │
│  ─────                    ────────                    ────────────────      │
│                                                                             │
│  orch spawn               orch-go-abc12               included_projects:    │
│  --workdir ~/orch-go      ┌──────┐ ┌────┐             [orch-go, beads,      │
│                           │project│ │hash│              kb-cli, ...]        │
│                           └──────┘ └────┘                                   │
│                                                                             │
│  orch spawn               pw-xyz98                    ✓ pw in list          │
│  --workdir ~/price-watch  ┌──┐ ┌────┐                 ✓ visible             │
│                           │pw│ │hash│                                       │
│                           └──┘ └────┘                                       │
│                                                                             │
│  orch spawn --no-track    orch-go-untracked-123       ✗ NOT in list         │
│                           ┌─────────────────┐ ┌────┐  ✗ filtered out        │
│                           │orch-go-untracked│ │hash│                        │
│                           └─────────────────┘ └────┘                        │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Mechanism

### 1. Project Name Extraction from Beads ID

Every beads issue ID follows the format `{project}-{short-id}` (e.g., `orch-go-abc12`, `pw-xyz98`).

**Extraction logic** (`cmd/orch/shared.go:extractProjectFromBeadsID`):
```go
// orch-go-abc12 → orch-go
// pw-xyz98 → pw
// orch-go-untracked-123 → orch-go-untracked
```

This is the **source of truth** for which project an agent belongs to. The beads ID is self-describing.

### 2. MultiProjectConfig (Included Projects)

Some projects should see agents from related repositories. This is configured in `pkg/tmux/follower.go`:

```go
type MultiProjectConfig struct {
    Project         string   // e.g., "orch-go"
    IncludeProjects []string // e.g., ["orch-cli", "beads", "kb-cli", "pw"]
}
```

**Default configuration:**
```go
{
    Project: "orch-go",
    IncludeProjects: ["orch-cli", "beads", "kb-cli", "orch-knowledge", "opencode", "price-watch", "pw"],
}
```

When the dashboard follows orchestrator context, it queries `/api/context` which returns:
```json
{
  "project": "orch-go",
  "included_projects": ["orch-go", "orch-cli", "beads", "kb-cli", "orch-knowledge", "opencode", "price-watch", "pw"]
}
```

### 3. Dashboard Project Filtering

The dashboard uses a **filter query string** built from `included_projects`:

```
GET /api/agents?project=orch-go,orch-cli,beads,kb-cli,...
```

**Filter flow:**
1. Dashboard polls `/api/context` (2s interval)
2. Extracts `included_projects` from response
3. Builds query string: `?project=proj1,proj2,...`
4. API filters agents where `agent.project IN (included_projects)`

**Implementation:** `web/src/lib/stores/context.ts:buildFilterQueryString()`

### 4. Untracked Spawn Filtering (Intentional)

Spawns with `--no-track` get special beads IDs:
```
orch-go-untracked-1769534403
```

This means their project name is `orch-go-untracked`, which is **not** in `included_projects`. Therefore:

| View | Shows Untracked? | Why |
|------|------------------|-----|
| Dashboard (follow mode ON) | No | `orch-go-untracked != orch-go` |
| Dashboard (follow mode OFF) | Yes | No project filter applied |
| `orch frontier` CLI | Yes | No project filtering |
| `orch status --json` | Yes | Shows all agents |

**This is intentional design:** Untracked spawns are test/ad-hoc work that should be excluded from production views.

---

## Cross-Project Agent Discovery

### How the Dashboard Finds Agents

```
┌─────────────────────────────────────────────────────────────────┐
│                    AGENT DISCOVERY FLOW                          │
│                                                                  │
│  1. List OpenCode sessions (global, port 4096)                  │
│     └── Sessions contain directory field                        │
│                                                                  │
│  2. Extract unique project directories                          │
│     └── From session.Directory + kb projects list               │
│                                                                  │
│  3. For each project directory:                                 │
│     └── Scan {project}/.orch/workspace/                        │
│     └── Extract beadsID and PROJECT_DIR from SPAWN_CONTEXT.md  │
│                                                                  │
│  4. Merge workspace metadata into unified cache                 │
│     └── beadsToWorkspace map                                    │
│     └── beadsToProjectDir map                                   │
│                                                                  │
│  5. Query beads comments per-project                            │
│     └── Use beadsToProjectDir to route queries                  │
│     └── GetCommentsBatchWithProjectDirs()                       │
└─────────────────────────────────────────────────────────────────┘
```

### Key Insight: OpenCode Session Directories Are Server-Determined

**Problem discovered:** OpenCode `--attach` mode connects to a running server. The server determines session directory from its own CWD, not the CLI's CWD.

**Evidence:** All sessions show `directory="/Users/dylan/Documents/personal/orch-go"` even when spawned with `--workdir ~/price-watch`.

**Solution:** Use `kb projects list` as additional source of project directories:
- Parse registered projects from `kb projects list --json`
- Merge with session directories
- Scan all known project workspaces

**Implementation:** `cmd/orch/serve_agents_cache.go:extractUniqueProjectDirs()`

---

## Cross-Project Completion

### How `orch complete` Works Across Projects

**Problem:** `orch complete pw-abc12` fails when run from orch-go because beads ID resolution looks in orch-go's `.beads/` database.

**Solution:** Auto-detect project from beads ID prefix:

```go
// In complete_cmd.go:359-374
projectName := extractProjectFromBeadsID(beadsID)  // "pw"
projectDir := findProjectDirByName(projectName)    // ~/Documents/.../price-watch
beads.DefaultDir = projectDir                       // Set before resolution
// Now resolution looks in price-watch's .beads/
```

**Search locations for project by name:**
```go
~/Documents/personal/{name}
~/{name}
~/projects/{name}
~/src/{name}
~/Documents/work/**/{name}  // Recursive search
```

---

## What Dashboard Shows vs What CLI Shows

| Command | Project Filter | Shows Untracked? | Cross-Project? |
|---------|---------------|------------------|----------------|
| Dashboard (follow ON) | `included_projects` | No | Yes (via filter) |
| Dashboard (follow OFF) | None | Yes | Yes (all) |
| `orch frontier` | None | Yes | Yes (all active) |
| `orch status` | Current project | No by default | Yes with `--all` |
| `orch complete` | Auto-detected | N/A | Yes (auto-detect) |

**Why the difference?**

- **Dashboard** is a production monitoring view - filters noise
- **`orch frontier`** is an operational tool - shows everything active
- **`orch status`** is project-scoped by default for focus

---

## Constraints

### 1. Beads Issues Are Per-Project

**Constraint:** Each project has its own `.beads/` directory with its own issue database.

**Implication:** Cross-project queries require knowing which project to query. The `beadsToProjectDir` map enables routing.

**This enables:** Project isolation, no cross-contamination of issues
**This constrains:** Must track project directories for cross-project visibility

### 2. Project Name Is Extracted From Beads ID

**Constraint:** The beads ID format `{project}-{hash}` is the source of truth for project membership.

**Implication:** Changing how beads IDs are generated would break project extraction.

**This enables:** Self-describing agents without external registry
**This constrains:** Project names must be valid beads ID prefixes

### 3. Untracked Spawns Use Distinct Project Names

**Constraint:** `--no-track` spawns get beads IDs like `{project}-untracked-{timestamp}`.

**Implication:** `extractProjectFromBeadsID("orch-go-untracked-123") → "orch-go-untracked"`

**This enables:** Intentional filtering of test/ad-hoc spawns from production views
**This constrains:** Untracked agents are invisible in follow-mode dashboard

---

## Failure Modes

### Failure 1: Cross-Project Agent Shows Wrong Project

**Symptom:** Agent spawned to price-watch shows as orch-go in dashboard

**Root cause:** OpenCode session directory is server-determined, not from --workdir

**Fix:** `kb projects list` provides alternative project directory source

**Verify:** Check if agent's project matches spawn --workdir target

### Failure 2: Untracked Agents Not Visible in Dashboard

**Symptom:** Spawned with `--no-track`, visible in `orch frontier` but not dashboard

**Root cause:** Working as designed - project name is `{project}-untracked` which isn't in `included_projects`

**Fix:** Not a bug. Use `orch frontier` or disable "Follow Orchestrator" to see untracked

### Failure 3: Cross-Project Complete Fails

**Symptom:** `orch complete pw-abc12` says "beads issue not found"

**Root cause:** Beads resolution happens in wrong project directory

**Fix:** Auto-detection from beads ID prefix (implemented in complete_cmd.go:359-374)

### Failure 4: Agent Not Visible Because Project Not in MultiProjectConfig

**Symptom:** Agent from new project doesn't appear when following orch-go

**Root cause:** New project not added to `DefaultMultiProjectConfigs()`

**Fix:** Add project to `IncludeProjects` in `pkg/tmux/follower.go`

---

## Configuration Reference

### MultiProjectConfig Location

**File:** `pkg/tmux/follower.go:362-384`

**To add a new project to orch-go's view:**
```go
{
    Project: "orch-go",
    IncludeProjects: []string{
        "orch-cli", "beads", "kb-cli", "orch-knowledge", "opencode",
        "price-watch", "pw",
        "new-project",  // Add here
    },
},
```

### Project Directory Search Locations

**File:** `cmd/orch/status_cmd.go:findProjectDirByName()`

```go
~/Documents/personal/{name}
~/{name}
~/projects/{name}
~/src/{name}
~/Documents/work/**/{name}
```

### Follow Orchestrator Toggle

**Dashboard:** Settings → "Follow Orchestrator" checkbox

**Default:** Enabled (`followOrchestrator: true`)

**When disabled:** Shows all agents without project filtering

---

## Related Artifacts

**Investigations Synthesized:**
- `2026-01-27-inv-investigate-untracked-agents-no-track.md` - Why untracked agents not visible
- `2026-01-21-inv-cross-project-daemon-poll-multiple.md` - Daemon cross-project polling
- `2026-01-15-inv-support-cross-project-agent-completion.md` - Cross-project completion
- `2026-01-07-inv-cross-project-agents-show-wrong.md` - Wrong project directory
- `2026-01-06-inv-cross-project-daemon-single-daemon.md` - Single daemon design
- `2025-12-26-inv-design-proper-cross-project-agent.md` - Multi-project aggregation
- `2025-12-25-inv-cross-project-agent-visibility-fetch.md` - Beads comment fetching
- And 4 more cross-project/untracked investigations

**Related Models:**
- `follow-orchestrator-mechanism.md` - Dashboard context following
- `dashboard-agent-status.md` - Status calculation priority cascade

**Key Implementation Files:**
- `pkg/tmux/follower.go` - MultiProjectConfig definition
- `cmd/orch/serve_agents_cache.go` - Workspace cache and project discovery
- `cmd/orch/serve_context.go` - `/api/context` endpoint
- `cmd/orch/shared.go` - `extractProjectFromBeadsID()`
- `cmd/orch/complete_cmd.go:359-374` - Cross-project auto-detection

---

## Evolution

### 2025-12-21: Initial Cross-Project Epic
- First recognition that multi-repo orchestration needs cross-project visibility
- Design investigations for completion UX

### 2025-12-25: Cross-Project Beads Fetching
- Added `extractProjectDirFromWorkspace`
- Implemented `GetCommentsBatchWithProjectDirs`

### 2025-12-26: Multi-Project Workspace Aggregation
- Designed dynamic project discovery from OpenCode sessions
- Identified `kb projects` as reliable project registry

### 2026-01-06: Single Daemon Design
- Established cross-project daemon polling pattern
- Used `kb projects list` for project registry

### 2026-01-07: Wrong Project Directory Fix
- Discovered OpenCode session directory is server-determined
- Added `kb projects` as alternative source

### 2026-01-15: Cross-Project Completion
- Implemented auto-detection from beads ID prefix
- `orch complete` now works across projects without flags

### 2026-01-27: Untracked Agent Visibility Documented
- Clarified intentional filtering of `{project}-untracked`
- Documented as expected behavior, not bug
