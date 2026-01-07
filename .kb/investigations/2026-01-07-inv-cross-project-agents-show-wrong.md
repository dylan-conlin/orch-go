<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project agents show wrong project_dir because OpenCode sessions inherit the server's directory (orch-go), not the --workdir target, causing workspace scanning to miss cross-project workspaces entirely.

**Evidence:** All 248 OpenCode sessions have directory="/Users/dylanconlin/Documents/personal/orch-go" despite some being spawned with --workdir to other projects; `kb projects list` shows 17 registered projects that could be used as alternative source.

**Knowledge:** OpenCode `run --attach` mode connects to a running server which determines session directory from its own cwd, not the CLI's cwd; setting cmd.Dir doesn't help; solution requires sourcing project directories from alternative sources (kb projects).

**Next:** Implement Option C (kb projects + fallback) - augment extractUniqueProjectDirs to include registered kb projects, ensuring all known project workspaces are scanned regardless of OpenCode session state.

**Promote to Decision:** recommend-yes (architectural choice affecting cross-project visibility)

---

# Investigation: Cross Project Agents Show Wrong Project Dir

**Question:** Why do cross-project agents spawned with --workdir show wrong project_dir in the dashboard, and how should this be fixed?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: OpenCode session directory is server-determined, not client-determined

**Evidence:** 
- All 248 OpenCode sessions have `directory="/Users/dylanconlin/Documents/personal/orch-go"`
- `opencode run --attach http://localhost:4096 ...` connects to the running OpenCode server
- The server creates sessions with directory = server's cwd, not the CLI's cwd
- Setting `cmd.Dir = cfg.ProjectDir` in spawn_cmd.go:1433 has no effect on session directory

**Source:** 
- `curl -s http://localhost:4096/session | jq '[.[] | .directory] | unique'` returns only orch-go
- cmd/orch/spawn_cmd.go:1431-1433 (startHeadlessSession sets cmd.Dir but server ignores it)
- `~/.bun/bin/opencode run --help` shows no --directory flag for run command

**Significance:** The root cause is architectural - OpenCode's attach mode doesn't support per-session directory override. Fixing this in spawn would require changes to OpenCode itself.

---

### Finding 2: Workspace cache relies on session directories for project discovery

**Evidence:**
- `extractUniqueProjectDirs(sessions, projectDir)` collects directories from OpenCode sessions
- `buildMultiProjectWorkspaceCache(projectDirs)` only scans directories in that list
- If price-watch isn't in the session directory list, its `.orch/workspace/` is never scanned
- `lookupProjectDir(beadsID)` returns empty because the workspace was never indexed

**Source:**
- cmd/orch/serve_agents_cache.go:239-269 (extractUniqueProjectDirs)
- cmd/orch/serve_agents.go:361-362 (cache building)
- cmd/orch/serve_agents_cache.go:428-431 (lookupProjectDir)

**Significance:** This creates a chicken-and-egg problem: can't find project dirs without scanning workspaces, can't scan workspaces without knowing project dirs.

---

### Finding 3: kb projects provides a reliable alternative source of project directories

**Evidence:**
- `kb projects list` returns 17 registered projects with full paths
- Projects include: orch-go, price-watch, kb-cli, beads, snap, opencode, and 11 others
- This is a stable, user-maintained registry that already exists
- Using kb projects would capture all known orchestration-capable projects

**Source:**
- `kb projects list` command output
- Projects are registered via `kb project register` when users set up new repos

**Significance:** kb projects solves the chicken-and-egg problem by providing a source of project directories independent of OpenCode session state.

---

## Synthesis

**Key Insights:**

1. **OpenCode architecture prevents fix at spawn time** - The `--attach` mode sends commands to an existing server which determines session directory. There's no CLI flag to override this. Fixing in OpenCode would require architectural changes.

2. **Current design assumes session.Directory is authoritative** - The multi-project cache was designed assuming OpenCode would report correct directories. This assumption fails for `--attach` mode.

3. **kb projects is a better source of truth** - It's explicitly user-managed, captures intent (which projects should be orchestrated), and already exists.

**Answer to Investigation Question:**

Cross-project agents show wrong project_dir because:
1. `orch spawn --workdir ~/price-watch` runs `opencode run --attach http://localhost:4096`
2. The OpenCode server (running in orch-go) creates a session with directory = orch-go
3. `extractUniqueProjectDirs` only gets orch-go from session directories
4. price-watch's `.orch/workspace/` is never scanned
5. `lookupProjectDir(beadsID)` returns empty for the price-watch agent

The fix is to augment `extractUniqueProjectDirs` to include kb-registered projects, ensuring all known project workspaces are scanned.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 248 sessions have directory=orch-go (verified: `curl | jq` query)
- ✅ kb projects list returns 17 projects (verified: ran command)
- ✅ SPAWN_CONTEXT.md contains correct PROJECT_DIR (known from prior knowledge)
- ✅ lookupProjectDir extracts PROJECT_DIR correctly when workspace is indexed (code review)

**What's untested:**

- ⚠️ Performance impact of scanning all 17 project workspaces (may be significant if some have many workspaces)
- ⚠️ Handling of kb projects with no .orch/ directory (may cause errors)
- ⚠️ Race conditions between kb projects changes and cache invalidation

**What would change this:**

- Finding would be wrong if OpenCode has a hidden directory override mechanism we didn't find
- Finding would be wrong if kb projects isn't reliably available (e.g., kb not installed)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Use kb projects as additional source of project directories** - Augment extractUniqueProjectDirs to include registered kb projects alongside OpenCode session directories.

**Why this approach:**
- kb projects is explicitly user-managed and captures intent
- Already exists and is maintained by users when setting up new repos
- Doesn't require changes to OpenCode
- Solves the problem completely for registered projects

**Trade-offs accepted:**
- Unregistered projects won't benefit (acceptable: users should register projects for orchestration)
- Adds dependency on kb CLI being available (acceptable: already required for orchestration)
- May scan more directories than strictly necessary (acceptable: workspaces are empty for inactive projects)

**Implementation sequence:**
1. Add `getKBProjects()` function to fetch registered project paths via kb CLI
2. Modify `extractUniqueProjectDirs` to merge kb projects with session directories
3. Add graceful fallback if kb CLI fails (log warning, continue with session dirs only)
4. Test with cross-project spawn from orch-go to price-watch

### Alternative Approaches Considered

**Option A: Fix spawn to pass directory to OpenCode session**
- **Pros:** Would be the "correct" fix at the source
- **Cons:** Requires OpenCode changes; `--attach` mode is architecturally server-controlled
- **When to use instead:** If OpenCode adds a `--directory` flag for `run --attach` mode

**Option B: Scan all known parent directories (hardcoded or config-based)**
- **Pros:** Simple, no dependency on kb
- **Cons:** Requires maintenance, doesn't auto-discover new projects
- **When to use instead:** If kb projects is unreliable or unavailable

**Rationale for recommendation:** Option C (kb projects) provides a reliable, user-maintained source of project directories without requiring changes to OpenCode or manual configuration maintenance.

---

### Implementation Details

**What to implement first:**
- `getKBProjects()` function using `kb projects list --format json` (or parse text output)
- Integration into `extractUniqueProjectDirs` with graceful fallback
- Test case with actual cross-project spawn

**Things to watch out for:**
- ⚠️ kb CLI might not be in PATH when running from dashboard server context
- ⚠️ Some kb projects may not have .orch/ directories (need to check before scanning)
- ⚠️ Performance: 17 projects × N workspaces = potentially slow first load

**Areas needing further investigation:**
- Whether kb has a Go library or needs CLI invocation
- Optimal caching strategy (current 30s TTL may need adjustment)

**Success criteria:**
- ✅ `orch spawn --workdir ~/price-watch investigation "test"` creates agent visible in dashboard
- ✅ Agent shows correct project = "price-watch" (not "orch-go")
- ✅ Dashboard works correctly even if kb CLI fails (graceful degradation)

---

## References

**Files Examined:**
- cmd/orch/spawn_cmd.go:1306-1466 - headless spawn implementation, cmd.Dir setting
- cmd/orch/serve_agents_cache.go:239-445 - workspace cache building and lookups
- cmd/orch/serve_agents.go:354-540 - agent API endpoint, cache usage
- pkg/opencode/client.go:196-212 - BuildSpawnCommand (no directory parameter)

**Commands Run:**
```bash
# Check unique session directories
curl -s http://localhost:4096/session | jq '[.[] | .directory] | unique'
# Result: ["/Users/dylanconlin/Documents/personal/orch-go"]

# List kb projects
kb projects list
# Result: 17 registered projects with paths

# Check opencode run help for directory flag
~/.bun/bin/opencode run --help
# Result: No --directory flag available
```

**Related Artifacts:**
- **Prior Knowledge:** "Cross-project beads queries require PROJECT_DIR from workspace SPAWN_CONTEXT.md"
- **Prior Knowledge:** "Cross-project agent visibility requires extracting PROJECT_DIR from SPAWN_CONTEXT.md"

---

## Investigation History

**2026-01-07 18:44:** Investigation started
- Initial question: Why do cross-project agents spawned with --workdir show wrong project_dir?
- Context: Dashboard shows orch-go for all agents despite spawning to other projects

**2026-01-07 19:15:** Root cause identified
- OpenCode --attach mode uses server's cwd, not CLI's cwd
- All 248 sessions point to orch-go regardless of spawn --workdir

**2026-01-07 19:30:** Solution identified
- kb projects provides reliable alternative source
- Can merge with session directories for comprehensive coverage

**2026-01-07 19:45:** Investigation completed
- Status: Complete
- Key outcome: Recommend using kb projects as additional source for project directory discovery
