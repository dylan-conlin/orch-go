<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode sessions are NOT project-scoped at the server level - all servers share session storage via ~/.opencode/. The issue is workspace directories being set incorrectly during cross-project spawns.

**Evidence:** Two OpenCode servers (ports 4096 and 55450) both return identical session lists (167 sessions). Sessions spawned for price-watch show directory as orch-go instead of price-watch path.

**Knowledge:** The problem isn't session scoping but spawn configuration. Cross-project spawns via `--workdir` may not be setting session directory correctly. The `x-opencode-directory` header determines which project's sessions are shown.

**Next:** Investigate why cross-project spawns have wrong directory; fix spawn_cmd.go to set correct directory; consider adding `orch sessions --project` for filtering.

---

# Investigation: Cannot Query OpenCode Sessions from Other Projects

**Question:** Why can't meta-orchestrator query OpenCode sessions from other projects, and what's the correct architecture for cross-project session visibility?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent (spawned from orch-go-6g2mf)
**Phase:** Complete
**Next Step:** None - findings documented, follow-up issues recommended
**Status:** Complete

---

## Findings

### Finding 1: OpenCode servers share session storage

**Evidence:** 
```bash
# Two OpenCode servers running
lsof -iTCP -sTCP:LISTEN | grep opencode
# opencode  58199 ... TCP localhost:4096 (LISTEN)
# opencode  64494 ... TCP localhost:55450 (LISTEN)

# Both return identical session counts
curl -s http://localhost:4096/session | jq length    # 167
curl -s http://localhost:55450/session | jq length   # 167

# Both show same sessions
curl -s http://localhost:4096/session | jq '.[0].id'
curl -s http://localhost:55450/session | jq '.[0].id'
# Both return: "ses_469ea7e32ffeAr8PvfeyhH0zx3"
```

**Source:** Direct testing via curl commands

**Significance:** This disproves the hypothesis that servers are project-scoped. Sessions are stored centrally (likely ~/.opencode/) and all servers can access them. The `x-opencode-directory` header filters results but doesn't isolate storage.

---

### Finding 2: Cross-project spawns have incorrect directory metadata

**Evidence:**
```bash
# Price-watch sessions (spawned via cross-project meta-orchestration)
curl -s http://localhost:4096/session | jq '.[] | select(.title | contains("pw-"))'
# Returns sessions with:
# "directory": "/Users/dylanconlin/Documents/personal/orch-go"
# NOT the price-watch project directory

# Session example:
# "title": "pw-debug-fix-re-scrape-06jan-f07f [pw-u7ht]"
# "directory": "/Users/dylanconlin/Documents/personal/orch-go"
```

**Source:** Session query results showing `pw-` prefixed sessions with orch-go directory

**Significance:** Sessions created for price-watch work are being registered with orch-go as their directory, not the actual price-watch project path. This explains why querying by price-watch directory returns no results.

---

### Finding 3: OpenCode client uses x-opencode-directory header for filtering

**Evidence:**
```go
// From pkg/opencode/client.go:286-293
func (c *Client) ListSessions(directory string) ([]Session, error) {
    req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
    ...
    if directory != "" {
        req.Header.Set("x-opencode-directory", directory)
    }
```

The `FindRecentSession` method filters by `projectDir`:
```go
// From pkg/opencode/client.go:540
if s.Directory != projectDir {
    continue
}
```

**Source:** pkg/opencode/client.go:286-293, 540

**Significance:** The filtering is working correctly - the problem is that sessions are being created with the wrong directory in the first place.

---

### Finding 4: Session directory is set during CreateSession

**Evidence:**
```go
// From pkg/opencode/client.go:432-453
func (c *Client) CreateSession(title, directory, model string) (*CreateSessionResponse, error) {
    payload := CreateSessionRequest{
        Title:     title,
        Directory: directory,
        Model:     model,
    }
    ...
    if directory != "" {
        req.Header.Set("x-opencode-directory", directory)
    }
```

The spawn command sets directory from `cfg.ProjectDir`:
```go
// From cmd/orch/spawn_cmd.go:1225-1227
func startHeadlessSession(...) (*headlessSpawnResult, error) {
    cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model)
    cmd.Dir = cfg.ProjectDir  // This sets working directory for opencode process
```

**Source:** pkg/opencode/client.go:432-453, cmd/orch/spawn_cmd.go:1225-1227

**Significance:** The `cmd.Dir` is set but when `BuildSpawnCommand` runs `opencode run --attach`, the session gets registered with whatever directory OpenCode detects, which may default to the client's cwd or the server's project context.

---

### Finding 5: The real price-watch project has no .orch directory

**Evidence:**
```bash
# Price-watch project in work directory
ls -la ~/Documents/work/SendCutSend/scs-special-projects/price-watch/.orch
# Shows only .claude directory, no .orch

# Price-watch spawns are creating workspaces in orch-go:
ls .orch/workspace | grep pw-
# pw-debug-fix-re-scrape-06jan-f07f
# (etc.)
```

**Source:** Filesystem check showing workspaces in orch-go despite being for price-watch work

**Significance:** When meta-orchestrator spawns cross-project work, the workspaces are created in the orchestrator's project (orch-go), not the target project. This is by design for `--workdir` spawns but the session directory metadata isn't being set correctly.

---

## Synthesis

**Key Insights:**

1. **Session storage is centralized** - OpenCode stores sessions in a central location, not per-server or per-project. All servers see the same sessions. The `x-opencode-directory` header is for filtering, not isolation.

2. **Cross-project spawns don't set directory correctly** - When `orch spawn --workdir ~/other-project` runs, the OpenCode session is created with the orchestrator's directory (orch-go) instead of the target directory (other-project). This breaks directory-based session discovery.

3. **The fix is in spawn configuration, not architecture** - The original hypothesis (separate servers per project) was incorrect. The issue is simply that session directory isn't being set correctly during cross-project spawns.

**Answer to Investigation Question:**

Meta-orchestrator CAN query sessions from other projects - all sessions are visible from any server. The problem is that cross-project sessions have incorrect directory metadata, making them unfindable by directory-based queries.

The fix is to ensure `orch spawn --workdir` passes the correct directory to OpenCode when creating sessions. This likely requires updating the `--attach` URL or adding explicit directory parameters to the session creation.

---

## Structured Uncertainty

**What's tested:**

- ✅ Multiple OpenCode servers share session storage (verified: both servers return 167 identical sessions)
- ✅ Sessions can be filtered by directory header (verified: code review of client.go)
- ✅ pw-* sessions have orch-go directory (verified: curl query showing wrong directory)

**What's untested:**

- ⚠️ Whether `opencode run --attach --dir` explicitly would fix the directory
- ⚠️ Whether headless HTTP spawn (CreateSession) correctly sets directory
- ⚠️ Whether this affects only meta-orchestrator or all cross-project spawns

**What would change this:**

- Finding that directory IS set correctly but overwritten somewhere
- Finding that OpenCode intentionally ignores directory for attached sessions
- Finding a different root cause in the TUI/CLI spawn path

---

## Implementation Recommendations

### Recommended Approach ⭐

**Fix directory setting in spawn command** - Update `orch spawn --workdir` to pass explicit directory to OpenCode session creation.

**Why this approach:**
- Directly addresses the root cause (wrong directory metadata)
- Doesn't require architectural changes to OpenCode
- Existing sessions remain queryable once directory is correct

**Trade-offs accepted:**
- Existing sessions with wrong directory won't be retroactively fixed
- May require testing both headless and tmux spawn paths

**Implementation sequence:**
1. Verify headless spawn correctly sets directory in CreateSession
2. Test tmux spawn path to ensure directory is passed via `opencode attach --dir`
3. Add integration test for cross-project session discovery

### Alternative Approaches Considered

**Option B: Central session registry in orch-go**
- **Pros:** Doesn't depend on OpenCode behavior
- **Cons:** Duplicates session tracking, adds maintenance burden
- **When to use instead:** If OpenCode can't be fixed or if more metadata needed

**Option C: Query all sessions and filter client-side**
- **Pros:** Works around directory issue immediately
- **Cons:** Inefficient, exposes all sessions unnecessarily
- **When to use instead:** Quick fix while proper solution is developed

**Rationale for recommendation:** Option A addresses the root cause. The fix is localized to spawn_cmd.go and doesn't require changes to OpenCode or new infrastructure.

---

### Implementation Details

**What to implement first:**
- Audit `cmd/orch/spawn_cmd.go` runSpawnHeadless() to trace directory propagation
- Check if `opencode run --attach <url> --dir <path>` is supported
- Test explicit directory in HTTP CreateSession call

**Things to watch out for:**
- ⚠️ The CLI and HTTP paths may handle directory differently
- ⚠️ Workspace path vs project path distinction (workspaces go to orchestrator, sessions should go to target)
- ⚠️ Existing sessions won't migrate - may need cleanup/re-spawn

**Areas needing further investigation:**
- How does OpenCode determine session directory when not explicitly provided?
- Does the TUI attach session show different behavior than CLI?
- Is there a session migration/update API?

**Success criteria:**
- ✅ `orch spawn --workdir ~/other-project` creates session with correct directory
- ✅ Sessions queryable by `curl -H "x-opencode-directory: ~/other-project" /session`
- ✅ Meta-orchestrator can find and message cross-project sessions

---

## References

**Files Examined:**
- `pkg/opencode/client.go` - Session API client, directory header handling
- `cmd/orch/spawn_cmd.go` - Spawn command, directory configuration
- `pkg/spawn/config.go` - Spawn configuration struct
- `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Related session architecture investigation

**Commands Run:**
```bash
# Count sessions on both servers
curl -s http://localhost:4096/session | jq length   # 167
curl -s http://localhost:55450/session | jq length  # 167

# Check session directories
curl -s http://localhost:4096/session | jq '.[] | .directory' | sort -u

# Find price-watch sessions
curl -s http://localhost:4096/session | jq '.[] | select(.title | contains("pw-"))'

# Check running OpenCode servers
lsof -iTCP -sTCP:LISTEN | grep opencode
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Session/workspace architecture
- **Investigation:** `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md` - Cross-project patterns

---

## Investigation History

**2026-01-06 17:32:** Investigation started
- Initial question: Why can't meta-orchestrator query sessions from other projects?
- Context: price-watch session not found in orch-go OpenCode server

**2026-01-06 17:45:** Key finding - servers share storage
- Discovered both OpenCode servers (4096, 55450) return identical sessions
- Disproved hypothesis that servers are project-scoped

**2026-01-06 17:55:** Root cause identified
- pw-* sessions have orch-go directory instead of price-watch
- Problem is spawn configuration, not architecture

**2026-01-06 18:05:** Investigation completed
- Status: Complete
- Key outcome: Cross-project spawns don't set session directory correctly; fix is in spawn command configuration

---

## Discovered Work

1. **Bug: Cross-project spawn sets wrong session directory** - Sessions created via `orch spawn --workdir` have orchestrator's directory, not target project directory
   - Impact: Sessions unfindable via directory-based queries
   - Recommend: Create issue for spawn_cmd.go fix

2. **Enhancement: `orch sessions` command** - Add command to list/filter sessions across projects
   - Would help diagnose cross-project issues
   - Recommend: Low priority, fix spawn first
