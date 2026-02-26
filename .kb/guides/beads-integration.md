# Beads Integration

**Purpose:** Single authoritative reference for how orch-go integrates with beads for issue tracking. Read this before debugging beads-related issues.

**Last verified:** Feb 26, 2026
**Synthesized from:** 17 investigations (Dec 2025 - Jan 2026) + 16 quick entries consolidated Feb 2026

---

## What Beads Does

Beads (`bd`) is the issue tracking system. In the orchestration flow:

| Stage | Beads Role |
|-------|------------|
| **Spawn** | Create/tag issue, track work |
| **Work** | Agent reports phases via comments |
| **Complete** | Close issue with reason |
| **Query** | `bd ready`, `bd list` for work discovery |
| **Discovery** | `orch:agent` label identifies tracked agents |

---

## The Integration Points

```
orch spawn                           Agent                           orch complete
    │                                  │                                  │
    ▼                                  ▼                                  ▼
bd create "{task}"              bd comments add {id}               bd close {id}
+ tag orch:agent                "Phase: Planning"                   --reason "{summary}"
+ AGENT_MANIFEST.json                  │
    │                                  ▼
    ▼                            Updates issue with
Returns beads ID                progress/status
(orch-go-abc1)
```

---

## Architecture: RPC Client with CLI Fallback

**Always use `pkg/beads`** - never shell out directly with `exec.Command("bd", ...)`.

**Current pattern:**

```go
import "orch-go/pkg/beads"

// RPC-first with automatic CLI fallback
socketPath, err := beads.FindSocketPath("")
client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
issue, err := client.Show("orch-go-abc1")

// Fallback functions for when client unavailable
issues, err := beads.FallbackReady()
```

| Method | When Used | Advantage |
|--------|-----------|-----------|
| **RPC** (default) | Beads daemon running | Faster, no process spawn |
| **CLI fallback** | Daemon unavailable | Always works |

**Client options:**
- `WithAutoReconnect(maxRetries)` — retry on connection errors
- `WithCwd(dir)` — run operations in a specific directory (cross-project)

**Cross-project support:**
- `beads.DefaultDir` package-level var redirects beads lookups to another project
- `FallbackShowWithDir(id, dir)` — show issue in a different directory
- `FallbackCreateInDir(...)` — create issue in a different directory
- `FallbackListWithLabelInDir(label, dir)` — query labels cross-project

**Environment:**
- `BEADS_NO_DAEMON=1` is set on all CLI fallback calls to prevent 5s timeout under launchd
- `beads.BdPath` is resolved at startup via `ResolveBdPath()` to avoid PATH lookup failures

---

## Two-Lane Agent Discovery

Agent discovery uses a strict two-lane architecture (replaced the old registry + multi-source reconciliation in Feb 2026):

**Lane 1: Tracked work** (dashboard, `orch status`)
- Source of truth: beads issues tagged with `orch:agent` label
- Implementation: `queryTrackedAgents()` in `cmd/orch/query_tracked.go`
- Flow:
  1. Query beads: `bd list -l orch:agent` (RPC-first, CLI fallback)
  2. Batch-lookup workspace manifests (`AGENT_MANIFEST.json`) by beads IDs
  3. Extract latest Phase from beads comments (per-issue RPC calls)
  4. Batch-check session liveness from OpenCode
  5. Join with explicit reason codes (`MissingBinding`, `MissingSession`, `MissingPhase`)

**Lane 2: Untracked sessions** (orchestrators, `--no-track`, ad-hoc)
- Source of truth: OpenCode session list
- Orchestrators explicitly skip beads tracking

**Caching:** `globalTrackedAgentsCache` with 3-second TTL wraps `queryTrackedAgents()`.

**Key invariant:** Issues without `orch:agent` label are invisible to the tracked-work lane.

---

## Atomic Spawn Protocol

Spawn uses a two-phase atomic protocol (`pkg/spawn/atomic.go`):

**Phase 1** (`AtomicSpawnPhase1`) — before session creation:
1. Tag beads issue with `orch:agent` label (rollback: remove label)
2. Write workspace: `SPAWN_CONTEXT.md`, dotfiles, `AGENT_MANIFEST.json` (rollback: remove workspace)

**Phase 2** (`AtomicSpawnPhase2`) — after session creation:
1. Write session ID dotfile to workspace
2. Update `AGENT_MANIFEST.json` with `session_id`

If Phase 1 fails, rollback removes the beads tag and workspace. Phase 2 is best-effort (session is already running).

**Critical constraint:** Auto-created beads issues must transition to `in_progress` at spawn time — dashboard discovery depends on status being `open` or `in_progress`.

---

## AGENT_MANIFEST.json

Workspaces contain `AGENT_MANIFEST.json` (written during spawn) that binds beads ID to session ID:

```go
type AgentManifest struct {
    WorkspaceName string `json:"workspace_name"`
    Skill         string `json:"skill"`
    BeadsID       string `json:"beads_id,omitempty"`
    ProjectDir    string `json:"project_dir"`
    SpawnTime     string `json:"spawn_time"`
    Tier          string `json:"tier"`
    SpawnMode     string `json:"spawn_mode,omitempty"`
    Model         string `json:"model,omitempty"`
    SessionID     string `json:"session_id,omitempty"`
}
```

`LookupManifestsByBeadsIDs()` scans `.orch/workspace/` (skipping `archived/`) to batch-match manifests by beads ID. This replaced the old registry.

---

## Beads ID Format

```
{project}-{4-char-hash}
   │          │
   │          └── Unique identifier (e.g., abc1, xyz9)
   └── Project prefix from .beads/ location (e.g., orch-go)
```

**Examples:**
- `orch-go-abc1` - Issue in orch-go project
- `kb-cli-def2` - Issue in kb-cli project
- `orch-go-untracked-1767548133` - Untracked spawn (placeholder, not in DB)

**Short ID Resolution:**
- Short IDs (`abc1`) must be resolved at **spawn time**, not agent time
- `pkg/beads.ResolveID()` converts short → full ID
- SPAWN_CONTEXT.md must contain full ID for agents to use

**ID Consistency Validation:**
- `spawn.ValidateBeadsIDConsistency()` warns when task text mentions a same-project beads ID that differs from the `--issue` flag
- Prevents confusing SPAWN_CONTEXT where task references one issue but `bd comment` instructions use another

---

## Phase Reporting

Agents report progress via beads comments:

```bash
bd comments add {beads-id} "Phase: Planning - analyzing requirements"
bd comments add {beads-id} "Phase: Implementation - writing code"
bd comments add {beads-id} "Phase: Complete - task finished, tests pass"
```

**Phase: Complete is critical.** This is how `orch complete` knows the agent finished successfully.

**ORIENTATION_FRAME belongs in beads comments only**, not in SPAWN_CONTEXT.md. The frame is orchestrator context for completion review reconnection — workers don't use it and it wastes their context window.

---

## Dependency Model

`Issue.GetBlockingDependencies()` implements explicit blocking semantics:

| Dep Type | Blocks? | Rationale |
|----------|---------|-----------|
| `blocks` | Yes (unless closed/answered) | Standard blocking |
| `parent-child` | Never | Children must be spawnable while epic is open |
| `relates_to` | Never | Informational only |
| unknown | Never | Safe default |

The daemon uses `CheckBlockingDependencies()` before spawning any issue. Effective priority computation lives in `orch-go serve` (API enrichment), not in beads core or frontend.

---

## Three-Layer Artifact Architecture

```
BEADS (.beads/)
├── Purpose: Track work in progress (issues, dependencies, status)
├── Data: issues.jsonl with structured JSON per issue
├── Links: Comments contain investigation_path, phase transitions
└── Discovery: bd show, bd ready, bd list

KB (.kb/)
├── Purpose: Persist knowledge artifacts (investigations, decisions)
├── Data: Markdown files with structured frontmatter
├── Links: kb link creates bidirectional issue↔artifact links
└── Discovery: kb context, kb search

WORKSPACE (.orch/workspace/)
├── Purpose: Ephemeral agent execution context
├── Data: SPAWN_CONTEXT.md (input), SYNTHESIS.md (output), AGENT_MANIFEST.json
├── Links: References beads ID, creates kb investigations
└── Discovery: Direct file access, orch review command
```

**Linking mechanisms:**
- Beads → KB: `investigation_path:` comments link to kb files
- KB → Beads: `kb link artifact.md --issue beads-id`
- Workspace → Both: SPAWN_CONTEXT.md contains beads ID, agents create kb investigations

---

## JSON Schema (Important!)

Beads JSON uses snake_case field names:

| Display | JSON Field |
|---------|------------|
| Type | `issue_type` |
| Close Reason | `close_reason` |
| Status | `status` |
| Priority | `priority` |

**Common mistake:** Using `.type` in jq queries returns `null` because the field is actually `issue_type`.

```bash
# Wrong
bd list --json | jq '.[0].type'      # Returns null

# Correct
bd list --json | jq '.[0].issue_type' # Returns "task"
```

**Output format:** `bd list` and `bd ready` use `{beads-id} [{priority}] [{type}] {status} ... - {title}` with ` - ` as the title separator.

---

## Multi-Repo Configuration (Danger!)

**Default to single-repo mode.** Multi-repo hydration imports ALL issues from referenced repos.

```yaml
# DANGEROUS - this imports all issues from beads repo into your database!
repos:
  primary: "."
  additional: ["/path/to/beads"]
```

**Signs of pollution:**
- Issues with foreign prefixes (e.g., `bd-*` in orch-go)
- Nested `.beads/.beads/` directories
- Issue count unexpectedly high

**Cleanup procedure:**
1. Filter issues.jsonl: `jq -c 'select(.id | startswith("orch-go-"))' issues.jsonl > clean.jsonl`
2. Remove nested dirs: `rm -rf .beads/.beads/`
3. Fix config.yaml: Remove `additional` key
4. Reinitialize: `rm .beads/beads.db* && bd init --prefix orch-go`

---

## Deduplication

`BeadsClient.Create()` automatically prevents duplicate issues:

```go
// Returns existing issue if title matches open/in_progress issue
issue, err := client.Create(beads.CreateArgs{
    Title: "My task",
})

// Force creation even if duplicate exists
issue, err := client.Create(beads.CreateArgs{
    Title: "My task",
    Force: true,
})
```

---

## Common Problems

### "bd comment fails with 'issue not found'"

**Possible causes:**

1. **Untracked spawn** - `--no-track` creates placeholder IDs that don't exist in DB
   - Expected behavior, not a bug

2. **Wrong directory** - Running `bd` from different repo than where issue exists
   - Fix: Use `--workdir` or `cd` to correct repo

3. **Short ID not resolved** - Using `abc1` instead of `orch-go-abc1`
   - Fix: Use full ID, or orch-go resolves automatically

### "Cross-project agent can't update beads"

**Cause:** Agent spawned with `--workdir /other/repo` but beads issue is in orchestrator's repo.

**Solutions:**
1. Use `--no-track` for cross-repo work, track manually
2. Create issue in target repo first, use `--issue`

### "Issue shows open but agent is done"

**Cause:** `orch complete` wasn't run.

**Fix:** Run `orch complete <id>`

### "bd ready shows nothing but there's work"

**Possible causes:**

1. **Issues are blocked** - Have unresolved dependencies
   - Check: `bd list --status blocked`

2. **Issues lack triage:ready label** - Daemon only spawns labeled issues
   - Fix: `bd label <id> triage:ready`

3. **Wrong directory** - Looking in wrong repo
   - Check: `pwd` and verify `.beads/` exists

### "Agent invisible on dashboard"

**Possible causes:**

1. **Missing orch:agent label** - Spawn didn't tag the issue
   - Fix: `bd label <id> orch:agent`

2. **Issue still in open status** - Auto-created issues must transition to `in_progress`
   - Fix: `bd update <id> --status in_progress`

---

## Directory Context

Beads operations are directory-sensitive:

```bash
# These use CURRENT directory's .beads/
bd list
bd show abc1
bd comments add abc1 "message"

# To operate on different repo:
cd /path/to/other/repo && bd list
```

**Key insight:** When orchestrator is in orch-go but agent runs in kb-cli, their `bd` commands hit different databases.

**Programmatic cross-project:** Use `beads.DefaultDir` or `WithCwd(dir)` option to redirect operations.

---

## Lifecycle States

| State | Meaning | Transitions To |
|-------|---------|----------------|
| `open` | Work not started | `in_progress` |
| `in_progress` | Agent working on it | `closed` |
| `closed` | Work complete | - |
| `blocked` | Has unresolved dependencies | `open` when unblocked |

**orch spawn** sets issue to `in_progress` and tags with `orch:agent`.
**orch complete** sets issue to `closed`.
**orch abandon** sets issue to `closed` (with abandonment reason).

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| RPC-first with CLI fallback | Performance when daemon running, compatibility when not |
| Two-lane agent discovery | Clean separation: tracked work (beads) vs untracked (OpenCode) |
| `orch:agent` label for discovery | Single query finds all tracked agents |
| AGENT_MANIFEST.json replaces registry | No local state drift; workspace is source of truth |
| No local agent state | Five iterations of caches all drifted; query authoritative sources directly |
| Two-phase atomic spawn | Phase 1 (beads+workspace) is rollback-safe; Phase 2 (session) is best-effort |
| Short ID resolution at spawn time | Agents can't resolve at runtime |
| Single-repo by default | Multi-repo imports all issues (dangerous) |
| Deduplication by default | Prevents duplicate issue accidents |
| `pkg/beads` is canonical interface | Never use raw exec.Command |
| Effective priority in orch-go serve | Policy (urgency weighting) belongs in orchestration, not beads core |
| parent-child deps never block | Children must be independently spawnable while epic is open |

---

## Debugging Checklist

Before spawning an investigation about beads issues:

1. **Check kb:** `kb context "beads"`
2. **Check this guide:** You're reading it
3. **Check issue exists:** `bd show <id>`
4. **Check correct directory:** `pwd` and `ls .beads/`
5. **Check daemon:** `orch doctor` (includes beads daemon check)
6. **Check JSON field names:** `issue_type` not `type`
7. **Check for pollution:** `bd list | wc -l` - unexpectedly high?
8. **Check agent visibility:** `bd list -l orch:agent` - is the issue tagged?
9. **Check manifest:** `cat .orch/workspace/{name}/AGENT_MANIFEST.json` - binding exists?

If those don't answer your question, then investigate. But **update this guide** with what you learn.

---

## Related Investigations

For historical evidence and deep-dives, see:

| Topic | Investigation |
|-------|---------------|
| RPC Client Design | `2025-12-25-inv-design-beads-integration-strategy-orch.md` |
| Multi-Repo Hydration | `2025-12-22-inv-beads-multi-repo-hydration-why.md` |
| Database Pollution | `2025-12-25-inv-beads-database-pollution-orch-go.md` |
| Short ID Resolution | `2026-01-03-inv-fix-short-beads-id-resolution.md` |
| Three-Layer Architecture | `2025-12-21-inv-beads-kb-workspace-relationships-how.md` |
| JSON Field Names | `2026-01-05-inv-fix-beads-type-field-showing.md` |
| Deduplication | `2026-01-03-inv-recover-priority-beads-deduplication-abstraction.md` |
| Two-Lane Discovery | `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` |

---

## Related Decisions

| Decision | Reference |
|----------|-----------|
| Lifecycle ownership boundaries | `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` |
| Two-lane agent discovery | `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` |
