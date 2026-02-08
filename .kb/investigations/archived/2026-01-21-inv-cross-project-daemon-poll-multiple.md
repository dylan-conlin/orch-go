<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project daemon is straightforward to implement by extending existing patterns - poll `kb projects list`, iterate over projects with `beads.WithCwd()`, spawn with `--workdir`.

**Evidence:** Prior investigation (2026-01-06) validated the approach; `kb projects list --json` returns 17 projects; beads client already supports `WithCwd()` option; `orch spawn` has `--workdir` flag.

**Knowledge:** The infrastructure exists; this is additive work, not architectural change. Global capacity pool (shared across projects) is the right model to prevent runaway spawning.

**Next:** Implement in ~4 phases: project registry integration, cross-project issue listing, cross-project spawning, CLI flags.

**Promote to Decision:** recommend-no (implementation task building on prior decision, not new architecture)

---

# Investigation: Cross Project Daemon Poll Multiple

**Question:** How should the daemon poll multiple beads directories to support spawning agents in different projects?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** Create implementation tasks or proceed with implementation
**Status:** Complete

**Extends:** `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md`

---

## Findings

### Finding 1: Current daemon is single-project by design

**Evidence:** `ListReadyIssues()` in `pkg/daemon/issue_adapter.go:16-36` uses `beads.FindSocketPath("")` which searches from the current working directory. The CLI fallback (`bd ready --json --limit 0`) also runs in the cwd context. `SpawnWork()` calls `orch work <beadsID>` without specifying workdir.

**Source:**
- `pkg/daemon/issue_adapter.go:16-54` - Issue listing functions
- `pkg/daemon/issue_adapter.go:117-135` - SpawnWork function
- `cmd/orch/daemon.go:155-535` - Daemon run loop

**Significance:** All three components (listing, spawning, completion) need cross-project awareness. The fixes are additive - no breaking changes required.

---

### Finding 2: kb projects provides JSON project registry

**Evidence:** Running `kb projects list --json` returns structured data:
```json
[{"name":"kb-cli","path":"/Users/dylanconlin/Documents/personal/kb-cli"},
 {"name":"price-watch","path":"/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch"},
 ...]
```
17 registered projects total.

**Source:** `kb projects list --json` command output

**Significance:** This is the project registry source. No need for a separate `~/.orch/projects.yaml` - reuse existing kb infrastructure. Projects must be kb-registered to be daemon-visible.

---

### Finding 3: beads client already supports directory targeting

**Evidence:** `pkg/beads/client.go` provides:
- `WithCwd(cwd string) Option` - Sets working directory for operations (line 46-50)
- `FindSocketPath(startDir string)` - Finds socket from specified directory
- CLI fallback functions accept directory context

The RPC client can connect to any project's beads daemon by finding its socket path.

**Source:**
- `pkg/beads/client.go:46-50` - WithCwd option
- `pkg/beads/client.go:78-106` - FindSocketPath function

**Significance:** No changes needed to beads client. Just need to call `FindSocketPath(projectPath)` instead of `FindSocketPath("")`.

---

### Finding 4: orch work already supports --workdir

**Evidence:** `spawn_cmd.go:57` defines `spawnWorkdir` flag. The spawn flow resolves the directory and passes it through to agent context.

**Source:** `cmd/orch/spawn_cmd.go:57, 541-578`

**Significance:** `SpawnWork(beadsID)` just needs to become `SpawnWorkForProject(beadsID, projectDir)` and pass `--workdir` when spawning.

---

### Finding 5: Capacity should be global, not per-project

**Evidence:** The daemon guide states: "Daemon is for batch/overnight work." Having per-project pools would:
1. Risk runaway spawning (N projects × M agents each = too many)
2. Require complex configuration (which projects get more slots?)
3. Not match the mental model (orchestrator wants to limit total concurrent work)

**Source:**
- `.kb/guides/daemon.md:127-156` - Capacity Management section
- `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md:96-98`

**Significance:** Keep single `WorkerPool` with shared `MaxAgents` limit. This is simpler and prevents resource exhaustion.

---

## Synthesis

**Key Insights:**

1. **No architectural changes needed** - All required infrastructure exists. Cross-project is an additive feature using existing patterns (kb projects, beads WithCwd, orch spawn --workdir).

2. **Project discovery is kb-based** - Using `kb projects list` avoids a new registry mechanism. Trade-off: projects must be kb-registered. This is acceptable since kb is already part of the orchestration workflow.

3. **Global capacity is correct** - A single pool across projects prevents resource exhaustion and matches the mental model of "limit total concurrent work."

**Answer to Investigation Question:**

The daemon should poll multiple projects by:
1. Getting the project list from `kb projects list --json` at daemon startup (and optionally refreshing periodically)
2. In each poll cycle, iterating over projects and calling a project-aware version of `ListReadyIssues(projectPath)`
3. When spawning, passing `--workdir <projectPath>` to `orch work`
4. Maintaining a single worker pool and rate limiter across all projects

---

## Structured Uncertainty

**What's tested:**

- ✅ `kb projects list --json` returns valid JSON with name/path pairs (verified: ran command, got 17 projects)
- ✅ beads client supports `WithCwd()` option (verified: code inspection of pkg/beads/client.go)
- ✅ `orch spawn` supports `--workdir` flag (verified: code inspection of cmd/orch/spawn_cmd.go:57)
- ✅ Prior investigation validated approach feasibility (verified: read 2026-01-06 investigation)

**What's untested:**

- ⚠️ Performance impact of polling 17+ projects every 60 seconds (not benchmarked)
- ⚠️ Behavior when individual project's beads daemon is unavailable (needs error handling)
- ⚠️ Dashboard cross-project awareness (separate concern, not in scope)

**What would change this:**

- Finding would be wrong if `kb projects list` format changes or becomes unavailable
- Finding would be wrong if some projects need isolated capacity pools (not current requirement)
- Finding would be wrong if polling latency for many projects exceeds poll interval

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Iterative Project Polling with Global Capacity** - Add `--cross-project` flag that enables multi-project polling while maintaining single capacity pool.

**Why this approach:**
- Leverages existing `kb projects list` infrastructure (no new registry)
- Minimal changes to daemon loop (add outer project iteration)
- Single capacity pool prevents runaway spawning
- Opt-in via flag (backward compatible)

**Trade-offs accepted:**
- Projects must be kb-registered to be daemon-visible (acceptable - kb is already in workflow)
- Slightly longer poll cycles with many projects (acceptable - 60s interval is not time-critical)

**Implementation sequence:**

1. **Add project discovery function** - Parse `kb projects list --json`, return `[]Project{Name, Path}`
2. **Add project-aware issue listing** - `ListReadyIssuesForProject(projectPath string)`
3. **Modify SpawnWork to accept project** - `SpawnWorkForProject(beadsID, projectPath string)` passes `--workdir`
4. **Update daemon loop** - When `--cross-project` enabled, iterate over projects
5. **Add CLI flag** - `--cross-project` flag for daemon run command

### Alternative Approaches Considered

**Option B: Per-project daemon instances (current approach)**
- **Pros:** Isolation, simpler per-daemon logic
- **Cons:** Requires running N daemons, no shared capacity management, harder to monitor
- **When to use instead:** If projects need isolated capacity pools or different polling strategies

**Option C: Separate ~/.orch/projects.yaml registry**
- **Pros:** Independent of kb, could have daemon-specific config per project
- **Cons:** Another config file to maintain, duplicates kb functionality
- **When to use instead:** If kb projects ever becomes unavailable or needs different semantics

**Rationale for recommendation:** Option A extends existing patterns without new configuration. Option B is the status quo causing the current friction. Option C adds unnecessary complexity.

---

### Implementation Details

**What to implement first:**
1. `pkg/daemon/projects.go` - Project discovery function (foundation)
2. `ListReadyIssuesForProject()` - Modify issue_adapter.go (low risk, additive)
3. `SpawnWorkForProject()` - Modify issue_adapter.go (low risk, additive)

**Things to watch out for:**
- ⚠️ Error handling: Individual project beads unavailable should not crash daemon or block other projects
- ⚠️ Project ordering: Should be deterministic (sorted by name) for predictable behavior
- ⚠️ Filter capability: Consider `--projects` flag to specify subset of projects to poll
- ⚠️ Logging: Include project name in spawn logs for visibility

**Areas needing further investigation:**
- Whether to cache project list or refresh every cycle (recommend: refresh on startup + every N minutes)
- Whether dashboard needs cross-project awareness (separate feature)
- Completion processing for cross-project agents (may need similar treatment)

**Success criteria:**
- ✅ `orch daemon run --cross-project` spawns work from any registered project
- ✅ `orch daemon preview --cross-project` shows issues from all projects
- ✅ Capacity limit is respected across all projects combined
- ✅ Error in one project doesn't crash daemon or block other projects

---

## Architecture Decision Summary

| Question | Decision | Rationale |
|----------|----------|-----------|
| **Project registry** | Use `kb projects list` | Reuses existing infrastructure, no new config file |
| **Polling strategy** | Sequential iteration over projects | Simple, deterministic, adequate for ~20 projects |
| **Spawn context** | Pass `--workdir <projectPath>` | Existing pattern, already supported |
| **Capacity management** | Global pool (shared across projects) | Prevents runaway spawning, simpler config |
| **Activation** | `--cross-project` flag (opt-in) | Backward compatible, explicit choice |

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Core daemon struct and logic
- `pkg/daemon/issue_adapter.go` - Beads integration (ListReadyIssues, SpawnWork)
- `cmd/orch/daemon.go` - CLI commands and flag definitions
- `.kb/guides/daemon.md` - Daemon guide with capacity management docs
- `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md` - Prior investigation

**Commands Run:**
```bash
# Get registered projects
kb projects list --json

# Check beads ready output
bd ready --json --limit 0
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md` - Prior investigation establishing feasibility

---

## Investigation History

**2026-01-21 ~08:00:** Investigation started
- Initial question: How to implement cross-project daemon polling?
- Context: Task orch-go-8d3dw requested architecture decision for cross-project daemon

**2026-01-21 ~08:15:** Prior investigation reviewed
- Found 2026-01-06 investigation already validated the approach
- Confirmed infrastructure exists (kb projects, beads WithCwd, --workdir)

**2026-01-21 ~08:30:** Investigation completed
- Status: Complete
- Key outcome: Cross-project daemon is straightforward implementation using existing patterns; recommend proceeding with ~4-phase implementation
