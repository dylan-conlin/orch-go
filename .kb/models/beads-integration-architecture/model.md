# Model: Beads Integration Architecture

**Domain:** Beads Integration / Issue Tracking / RPC Client
**Last Updated:** 2026-02-26
**Synthesized From:** 28 investigations + beads-integration.md guide (synthesized from 17 investigations, Dec 2025 - Jan 2026) on RPC client design, CLI fallback, auto-tracking protocol, performance optimization

---

## Summary (30 seconds)

Beads integration uses **RPC-first with CLI fallback** pattern: try JSON-over-Unix-socket RPC client (fast, no process spawn), fall back to CLI subprocess if daemon unavailable. The integration operates at **three points in agent lifecycle**: spawn (create issue), work (report phase via comments), complete (close with reason). **Auto-tracking** creates issues automatically unless `--no-track` flag set. The `pkg/beads` package provides a `BeadsClient` interface with two implementations: `Client` (RPC daemon) and `CLIClient` (bd CLI subprocess). The RPC client provides 10x performance improvement over CLI (single RPC call vs subprocess spawn + JSON parse).

---

## Core Mechanism

### RPC-First with CLI Fallback

**Architecture:** `pkg/beads` exposes a `BeadsClient` interface (`interface.go`) with two implementations:
- `Client` (`client.go`) — JSON-over-Unix-socket RPC to the beads daemon
- `CLIClient` (`cli_client.go`) — shells out to `bd` CLI commands

**RPC Client:**

```go
// pkg/beads/client.go

type Client struct {
    mu            sync.Mutex
    conn          net.Conn
    socketPath    string
    timeout       time.Duration
    cwd           string // Working directory for operations
    autoReconnect bool
    maxRetries    int
}

// Socket is per-project at .beads/bd.sock (NOT global ~/.beads/daemon.sock)
func FindSocketPath(dir string) (string, error) {
    // Walks up directory tree looking for .beads/bd.sock
    // If dir empty, uses DefaultDir or current working directory
}

func NewClient(socketPath string, opts ...Option) *Client {
    // Options: WithTimeout, WithCwd, WithAutoReconnect
}

func (c *Client) Show(id string) (*Issue, error) {
    // JSON request/response protocol over Unix socket
    resp, err := c.execute(OpShow, ShowArgs{ID: id})
    // Handles both array and single-object response formats
}
```

**CLI Client:**

```go
// pkg/beads/cli_client.go

type CLIClient struct {
    WorkDir string // Working directory for bd commands
    BdPath  string // Path to bd executable
    Env     []string
}

func NewCLIClient(opts ...CLIOption) *CLIClient {
    // Options: WithWorkDir, WithBdPath, WithEnv
}
```

**Standalone Fallback functions** also exist in `client.go` (e.g., `FallbackReady()`, `FallbackShow()`, `FallbackClose()`) for callers that don't use the interface pattern.

**Performance difference:**

| Method | Time per Call | Why |
|--------|---------------|-----|
| **RPC** | ~2-5ms | In-memory IPC, no process spawn |
| **CLI** | ~50-100ms | Fork/exec overhead, JSON parse, file I/O |

**Dashboard impact:** Before RPC client (Dec 2025), dashboard made 100+ CLI calls per status refresh = 5-10s load time. After RPC (Dec 26), same calls take 200-500ms.

**Source:** `pkg/beads/client.go`, `pkg/beads/cli_client.go`, `pkg/beads/interface.go`

### Three Integration Points

Beads is accessed at three lifecycle stages:

**1. Spawn (Create Issue)**

```
orch spawn SKILL "task"
    ↓
If --no-track not set:
    bd create --title "task" --type task
    Returns: orch-go-abc1
    ↓
Issue ID embedded in SPAWN_CONTEXT.md
Agent references in beads comments
```

**2. Work (Phase Reporting)**

```
Agent reports progress:
    bd comment orch-go-abc1 "Phase: Planning"
    bd comment orch-go-abc1 "Phase: Implementing"
    bd comment orch-go-abc1 "Phase: Complete"
    ↓
Dashboard polls comments
Status updates based on phase
```

**3. Complete (Close Issue)**

```
orch complete orch-go-abc1
    ↓
Verification passes
    ↓
bd close orch-go-abc1 --reason "Completed: <summary>"
    ↓
Issue status: closed
Dashboard shows blue "completed" badge
```

**Key insight:** Beads is the **authoritative source for completion**. OpenCode sessions persist indefinitely. Session existence != agent done. Only beads matters.

**Source:** `pkg/beads/client.go` (RPC methods: `Create`, `AddComment`, `CloseIssue`), `pkg/beads/cli_client.go` (CLI equivalents)

### Auto-Tracking Protocol

**Default behavior:** `orch spawn` creates beads issue automatically.

**Opt-out:** `--no-track` flag skips issue creation.

```go
// cmd/orch/spawn_cmd.go calls pkg/orch/extraction.go

// SetupBeadsTracking handles issue creation/reuse based on flags
func SetupBeadsTracking(skillName, task, projectName, beadsIssueFlag string,
    isOrchestrator, isMetaOrchestrator bool, serverURL string,
    noTrack bool, workspaceName string,
    createBeadsFn func(string, string, string) (string, error)) (string, error) {
    // If --issue flag provided, use existing issue
    // If --no-track, return empty beadsID
    // Otherwise, auto-create via CreateBeadsIssue()
}

// CreateBeadsIssue creates a new beads issue for spawn tracking
func CreateBeadsIssue(projectName, skillName, task string) (string, error) {
    // Uses beads RPC client or CLI fallback to create issue
    // Returns issue ID like "orch-go-abc1"
}
```

**Why auto-tracking:**
- Prevents "spawned agents without issues" (loses work visibility)
- Ensures completion verification has beads ID to close
- Connects spawns to backlog (can query `bd list` to see all active work)

**When to opt-out:**
- Ad-hoc exploration (not tracked work)
- Cross-project spawns where issue exists in target project
- Temporary debugging agents

**Source:** `pkg/orch/extraction.go` (`SetupBeadsTracking`, `CreateBeadsIssue`), `cmd/orch/spawn_cmd.go` (flag wiring)

### Beads ID Format

**Format:** `{project}-{4-char-hash}`

**Examples:**
- `orch-go-abc1` - Issue in orch-go project
- `kb-cli-xyz9` - Issue in kb-cli project
- `snap-def2` - Issue in snap project

**How project prefix determined:** Configured in `.beads/config.json` per project. The `bd create` command generates the ID using the project prefix plus a 4-character hash.

**Why project prefix matters:**
- Enables cross-project operation (orch-go orchestrator managing kb-cli work)
- Prevents ID collisions (abc1 in orch-go != abc1 in kb-cli)
- Provides context at a glance (see issue ID, know which project)

**Source:** `pkg/beads/types.go` (Issue struct with ID field)

### RPC vs CLI Decision Tree

**When to use RPC (`Client`):**
- High-frequency calls (dashboard polling, status checks)
- Multiple calls in sequence (list + show + comments)
- Performance-sensitive paths (daemon poll loop)

**When CLI fallback acceptable (`CLIClient` or `Fallback*` functions):**
- One-time operations (manual spawn, complete)
- User-initiated commands (not automated loops)
- When RPC client unavailable (daemon not running)

**Implementation:**

```go
// Use BeadsClient interface for code that needs either backend
import "orch-go/pkg/beads"

// RPC client (preferred for performance-sensitive paths)
socketPath, err := beads.FindSocketPath("")  // Walks up to find .beads/bd.sock
client := beads.NewClient(socketPath, beads.WithAutoReconnect(2))
if err := client.Connect(); err == nil {
    defer client.Close()
    issues, err := client.Ready(&beads.ReadyArgs{})
}

// CLI client (when daemon unavailable or for simple operations)
cliClient := beads.NewCLIClient(beads.WithWorkDir("/path/to/project"))
issues, err := cliClient.Ready(&beads.ReadyArgs{})
```

**Anti-pattern:**

```go
// WRONG - don't shell out directly
cmd := exec.Command("bd", "ready", "--limit", "10")
output, _ := cmd.Output()
issues := parseJSON(output)
```

**Source:** `pkg/beads/client.go`, `pkg/beads/cli_client.go`

---

## Why This Fails

### 1. RPC Client Unavailable

**What happens:** RPC calls fail, caller must fall back to CLI, performance degrades.

**Root cause:** Beads daemon not running. Per-project socket `.beads/bd.sock` doesn't exist.

**Why detection is hard:** Code using `Fallback*` functions degrades silently. Callers using `Client` directly get explicit connection errors.

**Fix:** Start beads daemon: `bd daemon start` or ensure launchd starts it on boot.

**Detection:** Log RPC failures, surface in `orch doctor` health check.

### 2. Beads ID Not Found

**What happens:** `orch complete orch-go-abc1` fails with "issue not found".

**Root cause:** Cross-project spawn. Issue created in orch-knowledge, but trying to complete from orch-go. Beads scoped to current directory's `.beads/`.

**Why detection is hard:** Beads ID looks valid (correct format), but doesn't exist in current project's `.beads/issues.jsonl`.

**Fix:** `cd` into correct project before completion, or use `--workdir` flag.

**Prevention:** `orch complete` should detect project from workspace, auto-cd.

### 3. Auto-Tracking Creates Duplicates

**What happens:** `orch spawn` creates issue, but issue already exists for same work.

**Root cause:** User creates issue manually, then spawns with auto-tracking. Both create issue.

**Why detection is hard:** No deduplication. Beads doesn't check if similar issue exists.

**Fix:** Use `--issue <id>` flag to reference existing issue instead of auto-creating.

**Prevention:** Better UX: `orch spawn` could check for related issues and prompt user.

---

## Constraints

### Why RPC-First, Not RPC-Only?

**Constraint:** Client falls back to CLI if RPC unavailable.

**Implication:** Performance varies based on daemon state. RPC = fast, CLI = slow.

**Workaround:** Ensure daemon running for consistent performance.

**This enables:** Commands always work regardless of daemon state (reliability over performance)
**This constrains:** Cannot guarantee consistent performance without daemon running

---

### Why Beads Scoped to Project Directory?

**Constraint:** Beads issues stored in `.beads/issues.jsonl` per project. Can't query issues from other projects without changing directory.

**Implication:** Cross-project work requires directory switching or separate tracking.

**Workaround:** Use `--no-track` for cross-project spawns, manage issues manually in target project.

**This enables:** Project-specific workflows, prevents pollution between projects
**This constrains:** Cannot query or manage issues across projects from single location

---

### Why Auto-Tracking Default?

**Constraint:** `orch spawn` creates beads issue unless `--no-track` set.

**Implication:** Every spawn creates issue, even for ad-hoc debugging.

**Workaround:** Use `--no-track` for temporary work.

**This enables:** Full work visibility, completion verification has issue to close
**This constrains:** Cannot spawn without creating issues unless explicitly opted out

---

### Why pkg/beads Package?

**Constraint:** All beads integration goes through `pkg/beads`, never direct `exec.Command("bd")`.

**Implication:** Can't use beads CLI shortcuts in code, must use package methods.

**Workaround:** Add method to pkg/beads if missing.

**This enables:** Centralized RPC/CLI logic, testability, future optimization in one place
**This constrains:** Must use package methods even for simple operations

---

## Evolution

### Phase 1: CLI-Only (Dec 2025)

**What existed:** Simple subprocess calls to `bd` CLI.

**Implementation:**
```go
cmd := exec.Command("bd", "create", "--title", task)
output, _ := cmd.Output()
```

**Gap:** Performance issues at scale (dashboard polling 100+ issues).

**Trigger:** Dashboard load time 5-10s, unacceptable UX.

### Phase 2: RPC Client (Dec 25-26, 2025)

**What changed:** JSON-over-Unix-socket RPC client with CLI fallback. 10x performance improvement.

**Investigations:** 8 investigations on RPC protocol, socket connection, error handling, fallback logic.

**Key insight:** Dashboard polling is high-frequency operation. Subprocess overhead (50-100ms per call) compounds to seconds. RPC (2-5ms) makes it instant.

### Phase 3: Auto-Tracking (Dec 27-29, 2025)

**What changed:** `orch spawn` creates issues automatically, `--no-track` to opt-out.

**Investigations:** 5 investigations on lost work, tracking gaps, duplicate prevention.

**Key insight:** Manual tracking fails under cognitive load. Auto-tracking with opt-out ensures work doesn't fall through cracks.

### Phase 4: Cross-Project Support (Jan 2-6, 2026)

**What changed:** `--workdir` flag, project detection from workspace, auto-cd for completion.

**Investigations:** 7 investigations on cross-project completion failures, ID scoping, directory detection.

**Key insight:** Beads is project-scoped, but orchestration is cross-project. Integration must handle project boundaries.

### Phase 5: pkg/beads Package (Jan 2026)

**What changed:** Consolidated all beads integration into single package, banned direct CLI usage.

**Investigations:** 3 investigations on code duplication, performance regressions, testing difficulties.

**Key insight:** Scattered `exec.Command("bd")` calls create maintenance burden, prevent optimization, make testing hard.

### Phase 6: BeadsClient Interface (Feb 2026)

**What changed:** Introduced `BeadsClient` interface with `Client` (RPC) and `CLIClient` (CLI) implementations. Added `MockClient` for testing. Socket path changed from global `~/.beads/daemon.sock` to per-project `.beads/bd.sock` with directory walk-up discovery.

**Key insight:** Interface abstraction enables clean dependency injection and testability without sacrificing the RPC-first performance pattern.

---

## References

**Guide:**
- `.kb/guides/beads-integration.md` - Procedural guide (commands, workflows, troubleshooting)

**Investigations:**
- Beads-integration.md references 17 investigations from Dec 2025 - Jan 2026
- Additional 11+ investigations on RPC client, auto-tracking, cross-project support

**Decisions:**
- `.kb/decisions/2025-12-25-beads-rpc-integration.md` (if exists)
- `.kb/decisions/2025-12-27-auto-tracking-default.md` (if exists)

**Models:**
- `.kb/models/agent-lifecycle-state-model/model.md` - Beads as authoritative source for completion
- `.kb/models/daemon-autonomous-operation/model.md` - How daemon polls beads for ready issues
- `.kb/models/completion-verification/model.md` - How completion closes beads issues

**Source code:**
- `pkg/beads/client.go` - RPC client + standalone Fallback* functions
- `pkg/beads/cli_client.go` - CLIClient implementation (bd CLI subprocess)
- `pkg/beads/interface.go` - BeadsClient interface definition
- `pkg/beads/types.go` - Issue, Comment, Stats types and RPC protocol types
- `pkg/orch/extraction.go` - SetupBeadsTracking and CreateBeadsIssue
- `cmd/orch/spawn_cmd.go` - Spawn command with --no-track, --issue flags

**Primary Evidence (Verify These):**
- `pkg/beads/client.go` - RPC client with JSON-over-socket protocol (NewClient, FindSocketPath, execute)
- `pkg/beads/cli_client.go` - CLIClient struct implementing BeadsClient via bd CLI
- `pkg/beads/interface.go` - BeadsClient interface (Ready, Show, List, Create, AddComment, CloseIssue, etc.)
- `.beads/bd.sock` - Per-project Unix socket for RPC communication (when daemon running)
- `cmd/orch/spawn_cmd.go` - Auto-tracking implementation with --no-track opt-out
- `.beads/issues.jsonl` - Authoritative issue storage showing beads ID format
