# Model: Beads Integration Architecture

**Domain:** Beads Integration / Issue Tracking / RPC Client
**Last Updated:** 2026-03-19
**Synthesized From:** 28 investigations + beads-integration.md guide (synthesized from 17 investigations, Dec 2025 - Jan 2026) on RPC client design, CLI fallback, auto-tracking protocol, performance optimization. Updated 2026-03-06 via 6 probe merges (see References). Updated 2026-03-19 via model-drift audit (file structure, exec.Command inventory, deleted references).

---

## Summary (30 seconds)

Beads integration uses **RPC-first with CLI fallback** pattern: try JSON-over-Unix-socket RPC client (fast, no process spawn), fall back to CLI subprocess if daemon unavailable. The integration operates at **three points in agent lifecycle**: spawn (create issue), work (report phase via comments), complete (close with reason). **Auto-tracking** creates issues automatically unless `--no-track` flag set. The `pkg/beads` package provides a `BeadsClient` interface with three implementations: `Client` (RPC daemon), `CLIClient` (bd CLI subprocess), and `MockClient` (testing). The RPC client provides 10x performance improvement over CLI (single RPC call vs subprocess spawn + JSON parse).

**Important:** Beads is a maintained fork (not upstream) with 43+ local commits. The `pkg/beads` "no direct exec.Command" constraint is aspirational — 12 direct calls exist across 8 files. Beads updates are unconditional (no CAS), creating TOCTOU races in concurrent daemon scenarios.

---

## Core Mechanism

### RPC-First with CLI Fallback

**Architecture:** `pkg/beads` exposes a `BeadsClient` interface (`interface.go`) with three implementations:
- `Client` (`client.go` + `client_operations.go`) — JSON-over-Unix-socket RPC to the beads daemon
- `CLIClient` (`cli_client.go`) — shells out to `bd` CLI commands
- `MockClient` (`mock_client.go`) — configurable mock for testing

**RPC Client (split across 3 files):**

```go
// pkg/beads/client.go (306 lines) — connection, execute, reconnect logic

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
    // If dir empty, uses current working directory
}

func NewClient(socketPath string, opts ...Option) *Client {
    // Options: WithTimeout, WithCwd, WithAutoReconnect
}

// pkg/beads/client_operations.go (327 lines) — RPC operation methods
func (c *Client) Show(id string) (*Issue, error) { ... }
func (c *Client) Ready(args *ReadyArgs) ([]Issue, error) { ... }
func (c *Client) List(args *ListArgs) ([]Issue, error) { ... }
// ... all BeadsClient interface methods

// pkg/beads/client_fallback.go (507 lines) — Fallback* functions + BdPath resolution
var BdPath string  // Resolved at startup via ResolveBdPath()
func ResolveBdPath() (string, error) { ... }
func FallbackReady(...) { ... }
func FallbackShow(...) { ... }
// ... standalone fallback functions for callers not using the interface
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

**Standalone Fallback functions** live in `client_fallback.go` (507 lines) for callers that don't use the interface pattern. This file also contains `BdPath` resolution logic (`ResolveBdPath()`) for environments with minimal PATH (e.g., launchd).

**Performance difference:**

| Method | Time per Call | Why |
|--------|---------------|-----|
| **RPC** | ~2-5ms | In-memory IPC, no process spawn |
| **CLI** | ~50-100ms | Fork/exec overhead, JSON parse, file I/O |

**Dashboard impact:** Before RPC client (Dec 2025), dashboard made 100+ CLI calls per status refresh = 5-10s load time. After RPC (Dec 26), same calls take 200-500ms.

**Source:** `pkg/beads/client.go`, `pkg/beads/client_operations.go`, `pkg/beads/client_fallback.go`, `pkg/beads/cli_client.go`, `pkg/beads/interface.go`, `pkg/beads/mock_client.go`

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
// cmd/orch/spawn_cmd.go calls pkg/orch/spawn_beads.go

// SetupBeadsTracking handles issue creation/reuse based on flags
func SetupBeadsTracking(skillName, task, projectName, beadsIssueFlag string,
    isOrchestrator, isMetaOrchestrator bool, serverURL string,
    noTrack bool, workspaceName string,
    createBeadsFn func(string, string, string, string) (string, error),
    projectDir string) (string, error) {
    // If --issue flag provided, use existing issue (validates not closed/in-progress)
    // If --no-track, creates lightweight issue (deprecation warning shown)
    // Otherwise, auto-create via CreateBeadsIssue()
}

// CreateBeadsIssue creates a new beads issue for spawn tracking
func CreateBeadsIssue(projectName, skillName, task, dir string) (string, error) {
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

**Dedup behavior (fail-closed):** When `synthesisIssueExists()` calls `bd list --json` and receives malformed JSON, it returns `exists=true` (blocking creation) rather than allowing duplicates. Error handling prefers false positives (skip creation) over false negatives (create duplicate). Confirmed via probe 2026-02-08.

**Source:** `pkg/orch/spawn_beads.go` (`SetupBeadsTracking`, `CreateBeadsIssue`), `cmd/orch/spawn_cmd.go` (flag wiring)

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

**Reality check (updated 2026-03-19):** 12 direct `exec.Command("bd")` calls exist outside `pkg/beads` in production code:
- `pkg/focus/guidance.go` (1 call) — `bd ready --json`
- `pkg/daemon/friction_accumulator.go` (2 calls) — `bd list --status=closed --json`, `bd comments list`
- `cmd/orch/init.go` (1 call) — `bd init`
- `cmd/orch/doctor_health.go` (1 call) — health check via bd
- `cmd/orch/reconcile.go` (1 call) — various bd commands
- `cmd/orch/status_infra.go` (1 call) — `bd config get issue_prefix`
- `cmd/orch/debrief_cmd.go` (3 calls) — `bd list --status=in_progress`, `bd ready`
- `cmd/orch/orient_cmd.go` (2 calls) — `bd ready`, `bd list --status=in_progress`

**Note:** Previous violations in `pkg/daemon/issue_adapter.go`, `pkg/daemon/extraction.go`, and `pkg/verify/beads_api.go` have been cleaned up (0 direct calls remaining). New violations were introduced in `friction_accumulator.go`, `debrief_cmd.go`, `orient_cmd.go`, and `doctor_health.go`.

The "no direct exec.Command" constraint is aspirational, not enforced.

**Source:** `pkg/beads/client.go`, `pkg/beads/client_operations.go`, `pkg/beads/cli_client.go`

### Status Update Atomicity (No CAS)

**Current behavior:** All beads status updates use unconditional SQL:

```go
// internal/storage/sqlite/queries.go:892
query := fmt.Sprintf("UPDATE issues SET %s WHERE id = ?", strings.Join(setClauses, ", "))
// No AND status = ? condition anywhere in the chain
```

**TOCTOU gap:** The daemon's fresh-status-check (L5) + UpdateStatus (L6) pattern has a race window:
1. L5 reads issue status
2. [Another daemon process can change status here]
3. L6 writes `in_progress` unconditionally — both processes succeed, both spawn

**CAS is feasible but unimplemented.** The beads SQLite driver (ncruces/go-sqlite3 v0.30.4) supports conditional UPDATE with `WHERE id = ? AND status = ?`. The `ErrConflict` sentinel already exists in `internal/storage/sqlite/errors.go`. Implementing CAS requires ~80-120 LOC across 7 files (additive, backward-compatible via optional `ExpectedStatus *string` in UpdateArgs). See probe 2026-03-01 for full implementation map.

**Source:** Probe 2026-03-01, beads `internal/rpc/server_issues_epics.go:475`, `queries.go:892`

### bd sync Wrapper (REMOVED)

**Status:** `scripts/bd-sync-safe.sh` has been deleted. The hash-mismatch recovery and post-sync readiness validation it provided are no longer available as a standalone wrapper. These issues may have been addressed in the beads fork directly, or the JSONL-only mode may have made them less relevant.

**Historical context:** The wrapper handled hash-mismatch recovery with bounded timeout + automatic import-only retry, and post-sync readiness validation. See probes 2026-02-09 for details.

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

**Mitigation (confirmed):** `synthesisIssueExists()` fails closed on malformed JSON from `bd list --json`, preventing duplicates when the dedup check itself errors. See probe 2026-02-08.

### 4. bd sync Deadlock via Pre-Commit Hook

**What happens:** `bd sync` hangs indefinitely. All subsequent `bd` commands in the same project block. `orch status` times out.

**Root cause:** Two-level deadlock chain:
```
bd sync (PID A)
  → acquires exclusive flock on .beads/jsonl.lock
  → git add .beads/
  → git commit → triggers pre-commit hook
    → pre-commit hook → bd hooks run pre-commit
      → runPreCommitHook() → exec bd sync --flush-only (PID B)
        → bd sync --flush-only
          → FlockExclusiveBlocking(.beads/jsonl.lock) → BLOCKS
    → pre-commit hook blocks waiting for PID B
  → git commit blocks waiting for pre-commit hook
→ bd sync (PID A) blocks waiting for git commit
→ DEADLOCK: A holds lock, waits for B. B needs lock held by A.
```

**Why partial fix is insufficient:** The c2af5a82 deadlock fix only addressed the `importFromJSONL()` subprocess path. The pre-commit hook path remains unpatched. 100% reproducible on every `bd sync` call with uncommitted `.beads/` changes.

**Secondary effects:** Zombie `bd sync` processes hold jsonl.lock indefinitely. Stale `next-index-*.lock` files accumulate from interrupted git operations. Process accumulation compounds over time (2+ zombie processes per hang).

**Scope:** All projects using noDb (JSONL-only) mode with beads git hooks installed.

**Workaround:** Kill zombie `bd sync` processes; use `git commit --no-verify` manually.

**Long-term fix options (in beads repo):**
1. `bd sync` uses `git commit --no-verify` for its internal commits (flush already done)
2. `bd sync` sets `BD_SYNC_IN_PROGRESS=1`; `bd hooks run pre-commit` skips flush when set
3. `bd sync --flush-only` uses non-blocking flock and skips if already held

**Source:** Probe 2026-02-17. Bug is in beads (not orch-go) but blocks all orch-go beads operations.

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

**Constraint:** All beads integration *should* go through `pkg/beads`, never direct `exec.Command("bd")`.

**Reality:** 11 direct `exec.Command("bd")` calls exist outside `pkg/beads`. The constraint is aspirational, not enforced.

**Implication:** Can't use beads CLI shortcuts in code, must use package methods — in principle.

**Workaround:** Add method to pkg/beads if missing.

**This enables:** Centralized RPC/CLI logic, testability, future optimization in one place
**This constrains:** Must use package methods even for simple operations (when followed)

---

## Fork Relationship (Active, Not Upstream)

**The beads fork is a critical dependency, not a third-party library.** The `2025-12-21-beads-oss-relationship-clean-slate.md` decision ("Drop all local features and use upstream beads as-is") was effectively reversed within 9 days. As of 2026-02-20, the fork is 43 commits ahead of upstream, all dated after the clean-slate decision. No superseding decision was recorded.

**Fork features actively used by orch-go:**

| Feature | Fork Commit | orch-go Usage |
|---------|-------------|---------------|
| Question entity type | `2dc8f7dc` (Jan 18) | `serve_beads.go` dashboard API |
| Question gates/deps | `744af9cf` (Jan 18) | `unblocked_collector.go` dependency resolution |
| Title-based dedup | `e19ff3f8` (Feb 16) | `pkg/beads/types.go` CreateArgs.Force field |
| Phase: Complete gate on close | `be871d0c` (Dec 30) | Core verification pipeline |
| Investigation issue type | `d813a87c` (Feb 7) | Tier determination in verify |
| bd close non-zero exit | `a3f8729e` (Feb 5) | Error handling in reconcile.go |

**Infrastructure improvements (implicit dependency):** JSONL-only default mode, sandbox detection (prevents SQLite WAL corruption in Claude Code), pre-flight fingerprint validation, rapid restart loop prevention, cross-process file locking — not called from orch-go code but improve beads reliability orch-go depends on.

**Implication:** When beads has bugs (e.g., the pre-commit hook deadlock), the fix is in the fork, not a report upstream.

**Source:** Probe 2026-02-20.

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

**Key insight:** Scattered `exec.Command("bd")` calls create maintenance burden, prevent optimization, make testing hard. (Note: 12 direct calls remain in production code as of 2026-03-19, though the specific files have shifted — old violations in issue_adapter.go/extraction.go/beads_api.go were cleaned up, new ones appeared in debrief_cmd.go/orient_cmd.go/friction_accumulator.go.)

### Phase 6: BeadsClient Interface (Feb 2026)

**What changed:** Introduced `BeadsClient` interface with `Client` (RPC) and `CLIClient` (CLI) implementations. Added `MockClient` for testing. Socket path changed from global `~/.beads/daemon.sock` to per-project `.beads/bd.sock` with directory walk-up discovery.

**Key insight:** Interface abstraction enables clean dependency injection and testability without sacrificing the RPC-first performance pattern.

### Phase 7: Client File Extraction (Mar 2026)

**What changed:** `client.go` (formerly ~1200+ lines) split into three files:
- `client.go` (306 lines) — connection, execute, reconnect
- `client_operations.go` (327 lines) — RPC operation methods
- `client_fallback.go` (507 lines) — Fallback* functions, BdPath resolution, ClientVersion

Beads tracking logic moved from `pkg/orch/extraction.go` to `pkg/orch/spawn_beads.go`. `scripts/bd-sync-safe.sh` removed.

**Key insight:** Extraction follows the codebase's accretion boundary pattern (files >1500 lines should be split). The split preserves the same public API while making each file focused on a single concern.

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
- `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` — **stale**, effectively reversed by 43 fork commits; needs superseding decision

**Models:**
- `.kb/models/agent-lifecycle-state-model/model.md` - Beads as authoritative source for completion
- `.kb/models/daemon-autonomous-operation/model.md` - How daemon polls beads for ready issues
- `.kb/models/completion-verification/model.md` - How completion closes beads issues

**Source code:**
- `pkg/beads/client.go` - RPC client connection, execute, reconnect logic (306 lines)
- `pkg/beads/client_operations.go` - RPC operation methods (Ready, Show, List, etc.) (327 lines)
- `pkg/beads/client_fallback.go` - Standalone Fallback* functions + BdPath resolution (507 lines)
- `pkg/beads/cli_client.go` - CLIClient implementation (bd CLI subprocess)
- `pkg/beads/interface.go` - BeadsClient interface definition
- `pkg/beads/mock_client.go` - MockClient for testing (434 lines)
- `pkg/beads/types.go` - Issue, Comment, Stats types and RPC protocol types
- `pkg/orch/spawn_beads.go` - SetupBeadsTracking and CreateBeadsIssue
- `cmd/orch/spawn_cmd.go` - Spawn command with --no-track, --issue flags

**Primary Evidence (Verified 2026-03-19):**
- `pkg/beads/client.go` - RPC client with JSON-over-socket protocol (NewClient, FindSocketPath, execute)
- `pkg/beads/client_operations.go` - BeadsClient interface method implementations
- `pkg/beads/client_fallback.go` - Fallback functions + BdPath resolution for launchd environments
- `pkg/beads/cli_client.go` - CLIClient struct implementing BeadsClient via bd CLI
- `pkg/beads/interface.go` - BeadsClient interface (Ready, Show, List, Create, AddComment, CloseIssue, etc.)
- `.beads/bd.sock` - Per-project Unix socket for RPC communication (ephemeral, only exists when daemon running)
- `cmd/orch/spawn_cmd.go` - Auto-tracking implementation with --no-track opt-out
- `.beads/issues.jsonl` - Authoritative issue storage showing beads ID format

**Note on deleted files:** `pkg/beads/fallback.go` (never existed — functions are in `client_fallback.go`), `pkg/beads/id.go` (deleted), `pkg/beads/lifecycle.go` (never existed), `pkg/spawn/tracking.go` (never existed), `cmd/orch/spawn.go` (renamed to `spawn_cmd.go`), `pkg/orch/extraction.go` (replaced by `spawn_beads.go`), `scripts/bd-sync-safe.sh` (deleted).

### Merged Probes

| Probe | Date | Verdict | Summary |
|-------|------|---------|---------|
| `probes/2026-02-08-synthesis-dedup-parse-error-fail-closed.md` | 2026-02-08 | Confirms | Auto-tracking dedup now fails closed (returns exists=true) on malformed `bd list --json`, preventing duplicate issue creation |
| `probes/2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch.md` | 2026-02-09 | Extends | `bd-sync-safe.sh` wrapper handles hash-mismatch path with bounded timeout + automatic import-only retry, eliminating manual kill/retry |
| `probes/2026-02-09-bd-sync-safe-post-sync-readiness-check.md` | 2026-02-09 | Extends | Post-sync wrapper validates read-path freshness; zero stale-read failures across 20 test runs after successful sync |
| `probes/2026-02-17-bd-sync-precommit-hook-deadlock.md` | 2026-02-17 | Contradicts | c2af5a82 deadlock fix is incomplete; second deadlock path via pre-commit hook → `bd sync --flush-only` still exists, 100% reproducible |
| `probes/2026-02-20-beads-fork-integration-audit.md` | 2026-02-20 | Contradicts (multiple) | Wrong socket path (global vs project-local), 5 nonexistent file references, 11 direct exec.Command violations, clean-slate decision reversed with 43 fork commits |
| `probes/2026-03-01-beads-cas-atomic-status-transitions.md` | 2026-03-01 | Confirms + Extends | Confirms unconditional UPDATE (no CAS); extends with feasibility proof that CAS is implementable in ~80-120 LOC using existing ErrConflict infrastructure |

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-26-inv-implement-pkg-beads-rpc-client.md
- .kb/investigations/archived/2025-12-25-inv-implement-pkg-beads-go-rpc.md
- .kb/investigations/archived/2025-12-25-inv-design-beads-integration-strategy-orch.md
- .kb/investigations/archived/2025-12-21-inv-orch-complete-closes-beads-issue.md
- .kb/investigations/archived/2025-12-25-inv-migrate-verify-getcomments-getcommentsbatch-use.md
- .kb/investigations/2026-02-20-inv-audit-beads-fork-integration.md
- .kb/investigations/archived/2025-12-25-inv-migrate-daemon-listreadyissues-use-new.md
