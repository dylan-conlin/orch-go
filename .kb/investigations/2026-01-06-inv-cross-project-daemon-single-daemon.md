<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** A cross-project daemon is achievable by polling `kb projects list` to get registered projects and iterating over each project's beads issues.

**Evidence:** Current `ListReadyIssues()` uses `beads.DefaultDir` and `WithCwd()` to target specific project directories; `kb projects list` returns 17 registered projects with paths.

**Knowledge:** The key change is moving from single-project polling (current daemon runs from one cwd) to multi-project polling (daemon iterates over registered projects).

**Next:** Create epic with implementation tasks: add project registry integration, modify `ListReadyIssues` to accept project path, update daemon loop for cross-project iteration.

---

# Investigation: Cross Project Daemon Single Daemon

**Question:** How can we implement a single daemon that polls all registered projects and spawns workers wherever work is ready?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** Create epic with implementation tasks
**Status:** Complete

---

## Findings

### Finding 1: Current daemon is implicitly project-scoped

**Evidence:** The daemon's `ListReadyIssues()` function in `pkg/daemon/issue_adapter.go:13-36` relies on `beads.FindSocketPath("")` which searches for `.beads/bd.sock` starting from the current working directory. This means:
- Daemon polls only the project where it was started
- Cross-project work requires multiple daemon instances (one per project)

**Source:** 
- `pkg/daemon/issue_adapter.go:16-32` - Socket discovery starts from cwd
- `pkg/beads/client.go:78-106` - `FindSocketPath` walks up from given dir

**Significance:** The current design doesn't inherently prevent cross-project polling - it's just that no mechanism exists to tell the daemon about other projects.

---

### Finding 2: kb projects list provides cross-project registry

**Evidence:** Running `kb projects list` returns 17 registered projects with their paths:
```
kb-cli: /Users/dylanconlin/Documents/personal/kb-cli
orch-knowledge: /Users/dylanconlin/orch-knowledge
beads: /Users/dylanconlin/Documents/personal/beads
...
```

**Source:** `kb projects list` command output

**Significance:** This is the existing mechanism to discover what projects exist and their paths. A cross-project daemon can iterate over this list rather than needing a separate project registry.

---

### Finding 3: Beads client already supports targeted directory operations

**Evidence:** The beads RPC client (`pkg/beads/client.go`) has:
- `WithCwd(cwd string) Option` - Line 46-50: Sets working directory for operations
- `DefaultDir` package variable - Line 22-23: Can be set to override cwd-based discovery
- All `Fallback*` functions check `DefaultDir` and use it if set (e.g., lines 648-665)

**Source:** `pkg/beads/client.go:22-23, 46-50, 648-726`

**Significance:** The infrastructure to query beads from arbitrary project directories already exists. The daemon just needs to set `beads.DefaultDir` or use `WithCwd()` before each project's poll cycle.

---

### Finding 4: orch spawn already supports --workdir for cross-project spawns

**Evidence:** `spawn_cmd.go:57` defines `spawnWorkdir` flag, and lines 541-578 handle resolving and validating the target directory. The spawn config includes `ProjectDir` which is passed through to agent context.

**Source:** `cmd/orch/spawn_cmd.go:57, 541-578, 778-801`

**Significance:** The spawning infrastructure already handles cross-project operations. The daemon just needs to pass the correct `--workdir` when spawning.

---

### Finding 5: Daemon status and capacity management is single-daemon ready

**Evidence:** 
- `pkg/daemon/status.go` writes to `~/.orch/daemon.status.json` (user-level, not project-level)
- `pkg/daemon/pool.go` manages a single `WorkerPool` with configurable `MaxAgents`
- Rate limiting in `pkg/daemon/rate_limiter.go` is per-daemon-instance

**Source:** 
- `pkg/daemon/status.go` - Status file path
- `pkg/daemon/pool.go` - Worker pool management
- `pkg/daemon/rate_limiter.go` - Spawn rate limiting

**Significance:** A single daemon can already manage cross-project capacity through a shared worker pool. No architectural changes needed for capacity management.

---

## Synthesis

**Key Insights:**

1. **No architectural blockers** - The current daemon architecture can be extended to cross-project polling without major refactoring. The beads client, spawn system, and capacity management all already support the necessary patterns.

2. **kb projects as the source of truth** - Using `kb projects list` as the project registry avoids creating a new configuration mechanism and integrates with existing tooling. Projects must be registered with kb to be daemon-visible.

3. **Polling strategy matters** - Need to decide whether to:
   - Poll all projects every cycle (simple, may be slow with many projects)
   - Round-robin through projects (fair, prevents one project from starving others)
   - Priority-based polling (poll projects with triage:ready labels first)

**Answer to Investigation Question:**

A single cross-project daemon is achievable by:
1. Getting registered projects from `kb projects list` at startup (and periodically refreshing)
2. In each poll cycle, iterating over projects and calling `ListReadyIssues()` with each project's path
3. When spawning, using `--workdir <project-path>` to target the correct project
4. Maintaining a single worker pool and rate limiter across all projects

---

## Structured Uncertainty

**What's tested:**

- ✅ `kb projects list` returns project paths (verified: ran command, got 17 projects)
- ✅ `beads.DefaultDir` controls where beads commands run (verified: code inspection of client.go fallback functions)
- ✅ `--workdir` flag works for cross-project spawns (per prior kb constraint: "orch abandon uses same --workdir pattern as spawn")

**What's untested:**

- ⚠️ Performance impact of polling 17+ projects every 60 seconds (not benchmarked)
- ⚠️ Behavior when one project's beads daemon is down (need error handling)
- ⚠️ Interaction with launchd-managed daemon (single instance constraint)

**What would change this:**

- Finding would be wrong if `kb projects list` format changes or becomes unavailable
- Finding would be wrong if beads operations require per-project daemon connections (currently supports CLI fallback)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach: Iterative Project Polling

**Why this approach:**
- Leverages existing `kb projects list` infrastructure
- Minimal changes to daemon loop (add outer project iteration)
- Single capacity pool prevents runaway spawning across projects

**Trade-offs accepted:**
- Projects must be kb-registered to be daemon-visible (acceptable - kb is already in orchestration workflow)
- Slightly longer poll cycles with many projects (acceptable - 60s interval is not time-critical)

**Implementation sequence:**

1. **Add project discovery function** - Parse `kb projects list --json` (or plain text with regex)
2. **Modify ListReadyIssues to accept project path** - Set `beads.DefaultDir` or create client with `WithCwd()`
3. **Update daemon loop** - Iterate over projects, aggregate issues, apply single capacity limit
4. **Add cross-project spawn** - Ensure `orch work` passes `--workdir` when spawning
5. **Add project-aware logging** - Include project name in spawn logs for visibility

### Alternative Approaches Considered

**Option B: Per-project daemon instances (current approach)**
- **Pros:** Isolation, simpler per-daemon logic
- **Cons:** Requires running N daemons, no shared capacity management, harder to monitor
- **When to use instead:** If projects need isolated capacity pools or different polling strategies

**Option C: Aggregated beads database**
- **Pros:** Single source of truth, simplest polling
- **Cons:** Requires beads architectural change, cross-repo data mixing risks (per kb constraint: "Beads cross-repo contamination can create orphaned FK references")
- **When to use instead:** Never - violates existing constraint

**Rationale for recommendation:** Option A maintains existing patterns while solving the core friction. Option B is the status quo that created this investigation. Option C is explicitly ruled out by prior decisions.

---

### Implementation Details

**What to implement first:**
- Project discovery function (foundation for everything else)
- Modify `ListReadyIssues` to accept project path (low risk, additive)

**Things to watch out for:**
- Project ordering in poll cycle - should be deterministic (sorted) for predictability
- Error handling when individual project beads is unavailable - should continue to other projects
- Spawn failure in one project shouldn't block spawns in other projects

**Areas needing further investigation:**
- Whether to cache project list or refresh every cycle
- Whether to add `--projects` flag to filter which projects daemon monitors
- Whether dashboard needs cross-project awareness

**Success criteria:**
- ✅ Single daemon instance can spawn work from any registered project
- ✅ `orch status` shows agents across all projects
- ✅ Capacity limit is respected across all projects combined
- ✅ Error in one project doesn't crash daemon or block other projects

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Core daemon logic (775 lines)
- `pkg/daemon/issue_adapter.go` - Beads integration (89 lines)
- `pkg/beads/client.go` - RPC client with cross-project support (957 lines)
- `cmd/orch/daemon.go` - CLI commands (613 lines)
- `cmd/orch/spawn_cmd.go` - Spawn with --workdir support (1690 lines)

**Commands Run:**
```bash
# List registered projects
kb projects list

# Check beads ready output
bd ready --json
```

**Related Artifacts:**
- **Constraint:** "cross project agent visibility requires fetching beads comments from agent's project directory" - Supports per-project query approach
- **Decision:** "Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track" - Informs tracking strategy

---

## Investigation History

**2026-01-06 14:00:** Investigation started
- Initial question: How to implement single daemon for cross-project work?
- Context: Current friction requires running multiple daemon instances

**2026-01-06 14:30:** Core architecture analyzed
- Found: beads client already supports targeted directories
- Found: spawn already supports --workdir
- Found: kb projects provides project registry

**2026-01-06 15:00:** Investigation completed
- Status: Complete
- Key outcome: Cross-project daemon is achievable by iterating over kb projects list and using existing beads/spawn patterns
