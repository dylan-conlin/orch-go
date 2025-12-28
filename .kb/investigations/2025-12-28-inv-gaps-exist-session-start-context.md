<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator session-start context has 5 specific gaps: (1) wrong port in skill (3333 vs 3348), (2) no web UI startup instructions in CLAUDE.md, (3) SessionStart hook focuses on workspaces not servers, (4) spawned agents get server context but orchestrators don't, (5) stale binary detection exists (`orch version --source`) but isn't surfaced at session start.

**Evidence:** Tested `orch version --source` - correctly detects staleness by comparing embedded git hash to HEAD. Verified port 3348 responds while 3333 does not. Confirmed spawned agents get LOCAL SERVERS section via GenerateServerContext() but orchestrators have no equivalent. Found prior kn entries (kn-64dddf, kn-5afb22) documenting stale binary pain points.

**Knowledge:** Both server info and staleness detection EXIST but aren't SURFACED at session start. The pattern is consistent: Dylan builds features, forgets to add them to SessionStart context injection. Spawn context is code-generated with project-specific info, while orchestrator context relies on static CLAUDE.md that lacks runtime details.

**Next:** Fix orchestrator skill port (3333→3348), add dev server startup section to CLAUDE.md, add stale binary warning to SessionStart hook, consider adding `orch servers status` to SessionStart context.

---

# Investigation: Session-Start Context Gaps for Orchestrators

**Question:** What gaps exist in session-start context for orchestrators? Specifically: (1) What server/service information should orchestrators know? (2) What mechanisms have we tried? (3) What's still missing or broken? (4) Why do we keep running stale orch binaries?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** og-inv-gaps-exist-session-28dec
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Orchestrator Skill Has Wrong Port (3333 vs 3348)

**Evidence:** The orchestrator skill references port 3333 in three places, but `orch serve` actually runs on port 3348:

```
# From ~/.claude/skills/meta/orchestrator/SKILL.md
Line 360: - Dashboard at `http://127.0.0.1:3333` (`orch serve`) for real-time visibility
Line 367: - **Firefox:** beads-ui at `http://127.0.0.1:3333` (auto-follows orchestrator cwd)
Line 432: - Dashboard visibility at `http://127.0.0.1:3333` (`orch serve`)
```

Test results:
- `curl -s http://127.0.0.1:3333/health` → "Nothing on port 3333"
- `curl -s http://127.0.0.1:3348/health` → `{"status":"ok"}`

**Source:** 
- `~/.claude/skills/meta/orchestrator/SKILL.md` lines 360, 367, 432
- `cmd/orch/serve.go:31` - `DefaultServePort = 3348`
- `orch serve status` output confirms port 3348

**Significance:** An orchestrator following the skill guidance would try to access the wrong URL, creating immediate confusion about whether the dashboard is running.

---

### Finding 2: No Web UI Startup Instructions in CLAUDE.md

**Evidence:** The orch-go CLAUDE.md describes `orch serve` (API server) but does not explain how to start the web dashboard (Svelte UI). User reported having to discover `cd web && npm run dev` through trial and error.

What CLAUDE.md says about servers:
- `serve` - HTTP API server for web UI (port 3348)
- `servers list/start/stop/attach/open` commands
- Config.Servers maps service names to ports

What CLAUDE.md does NOT say:
- How to start the web UI for development
- That web UI is at `web/` subdirectory
- That web UI connects to orch serve at port 3348

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - grep for "npm run dev\|web.*dev\|cd web" returned no matches
- `/Users/dylanconlin/Documents/personal/orch-go/web/package.json` - scripts show `npm run dev` for development

**Significance:** This is the exact friction the user reported. There's a documentation gap between "what the project has" and "how to use it for development".

---

### Finding 3: SessionStart Hook Focuses on Workspaces, Not Servers

**Evidence:** The current SessionStart hook (`~/.claude/hooks/session-start.sh`) is focused on workspace/session management, not operational context like server health:

```bash
# SessionStart hook to enforce session creation workflow
# This hook requires asking about session creation at the start of every conversation
```

The hook:
- Checks for `.claude/index.md` 
- Extracts active sessions
- Prompts about workspace creation
- Does NOT check or surface server status

**Source:**
- `~/.claude/hooks/session-start.sh` - full content reviewed
- `~/.claude/hooks/cdd-hooks.json` - SessionStart configuration

**Significance:** Prior investigations (2025-12-27-inv-design-daemon-managed-development-servers.md) recommended SessionStart integration for server health, but this hasn't been implemented. The hook infrastructure exists but isn't used for operational context.

---

### Finding 4: Spawned Agents Get Server Context, Orchestrators Don't

**Evidence:** When agents are spawned, `GenerateServerContext()` in `pkg/spawn/context.go:858` creates a LOCAL SERVERS section:

```go
func GenerateServerContext(projectDir string) string {
    // Reads .orch/config.yaml
    // Returns formatted section with:
    // - Project name
    // - Status (running/stopped)
    // - Port list (e.g., web: 5188, api: 3348)
    // - Quick commands (start/stop/open)
}
```

The `.orch/config.yaml` for orch-go:
```yaml
servers:
  web: 5188
  api: 3348
```

This context is included in SPAWN_CONTEXT.md for workers, but orchestrators don't receive any equivalent context at session start. They must discover this information manually.

**Source:**
- `pkg/spawn/context.go:858-902` - GenerateServerContext implementation
- `.orch/config.yaml` - server port configuration
- `.orch/servers.yaml` - server process definitions

**Significance:** There's a structural asymmetry: workers get project-specific operational context injected automatically, orchestrators get static documentation that lacks runtime details.

---

### Finding 5: Stale Binary Detection EXISTS But Isn't Surfaced

**Evidence:** The `orch version --source` command correctly detects binary staleness:

```
$ orch version --source
orch version 877943f1-dirty
build time:  2025-12-28T20:17:16Z
source dir:  /Users/dylanconlin/Documents/personal/orch-go
git hash:    877943f1f2c571eb9ce666f81fc34c71d8541f69

status: ✓ UP TO DATE
```

When binary is stale, output shows:
```
status: ⚠️  STALE
binary hash:  877943f1
current HEAD: abc12345

rebuild: cd /Users/dylanconlin/Documents/personal/orch-go && make install
```

The mechanism:
- `Makefile:17` embeds GIT_HASH at build time via LDFLAGS
- `cmd/orch/main.go:115-156` implements `runVersionSource()` which compares embedded hash to `git rev-parse HEAD`
- Works correctly but is never run automatically

Prior knowledge documents this pain:
- `kn-64dddf`: "bd multi-repo config is YAML-only... Stale binary causes silent failure."
- `kn-5afb22`: "Binary staleness should be prevented with make install"

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:115-156` - runVersionSource implementation
- `/Users/dylanconlin/Documents/personal/orch-go/Makefile:17` - Git hash embedding
- `kn get kn-64dddf` and `kn get kn-5afb22` - prior pain points

**Significance:** The staleness detection is complete and accurate. The gap is that it's not surfaced automatically. Adding a SessionStart hook that calls `orch version --source` (or checks git hashes directly) would prevent orchestrators from working with outdated binaries.

---

## Synthesis

**Key Insights:**

1. **Port Mismatch is a Simple Bug** - The orchestrator skill has stale port references. This is a quick fix: search and replace 3333→3348 in the skill source files.

2. **Documentation Gap is Architectural** - CLAUDE.md is static documentation about what exists, not operational instructions for how to use it. The "how to start the web UI" gap reflects this structural limitation.

3. **Asymmetric Context Injection** - Spawned agents get project-specific context via code (GenerateServerContext), while orchestrators rely on manually-maintained static files. This design choice means orchestrator context lags behind.

4. **SessionStart Hook Underutilized** - The hook infrastructure exists and runs on every session, but it's focused on workspace management, not operational status. Prior investigations recommended SessionStart for server health surfacing, but this hasn't been implemented.

5. **Features Exist, Surfacing Doesn't** - Both server info and staleness detection are fully implemented. The gap is surfacing them in SessionStart context. This is a pattern: build feature → forget to add to session context → hit the gap months later.

**Answer to Investigation Question:**

**What gaps exist?**
1. Wrong port in orchestrator skill (documentation bug)
2. Missing dev server startup instructions in CLAUDE.md (documentation gap)
3. SessionStart hook doesn't surface server/service status (feature gap)
4. Orchestrators don't get project-specific context that workers receive (architectural asymmetry)
5. Stale binary detection exists but isn't surfaced automatically (feature gap)

**What mechanisms have we tried?**
- CLAUDE.md files (static, project-specific documentation)
- SessionStart hooks (focused on workspaces, not operational context)
- GenerateServerContext for spawned agents (works, but orchestrators excluded)
- `orch doctor` and `orch servers status` commands (exist but not surfaced automatically)

**What's still missing/broken?**
- Port number is wrong in skill (bug)
- No "how to start for development" section in CLAUDE.md (gap)
- SessionStart doesn't surface `orch doctor` or `orch servers status` results (gap)
- No equivalent to GenerateServerContext for orchestrator sessions (gap)
- `orch version --source` staleness check isn't run automatically (gap)

---

## Structured Uncertainty

**What's tested:**

- ✅ Port 3348 responds, port 3333 does not (verified: curl commands)
- ✅ CLAUDE.md has no web UI startup instructions (verified: grep search)
- ✅ SessionStart hook focuses on workspaces (verified: read full content)
- ✅ GenerateServerContext exists and includes ports (verified: code review and .orch/config.yaml)
- ✅ `orch doctor` and `orch servers status` show useful info (verified: ran commands)
- ✅ `orch version --source` correctly detects staleness (verified: ran command, showed UP TO DATE)
- ✅ Prior kn entries document stale binary pain (verified: kn-64dddf, kn-5afb22)

**What's untested:**

- ⚠️ Whether SessionStart hook enhancement would actually help orchestrators (not tested)
- ⚠️ Whether port 5188 vs 5173 for web is intentional (config says 5188, vite default is 5173)
- ⚠️ Whether ECOSYSTEM.md surfacing would help (exists but may be outdated)

**What would change this:**

- If there's a reason for the 3333 port reference (e.g., historical or alternate config)
- If CLAUDE.md is intentionally minimal and detail should go elsewhere
- If there's a SessionStart extension in progress that wasn't discovered

---

## Implementation Recommendations

### Recommended Approach ⭐

**Four-part fix** - Address each gap with the appropriate mechanism

**Why this approach:**
- Each gap has a natural fix location (skill, CLAUDE.md, hook)
- No architectural changes needed - just filling in existing structures
- Can be done incrementally (bug fixes first, enhancements later)

**Trade-offs accepted:**
- CLAUDE.md changes require manual maintenance
- SessionStart hook adds latency to session start (acceptable per prior investigation)

**Implementation sequence:**
1. **Fix port in orchestrator skill** - Simple find/replace, quick win
2. **Add dev server section to CLAUDE.md** - Document existing behavior
3. **Add stale binary warning to SessionStart hook** - Calls `orch version --source`, surfaces warning if stale
4. **Consider server status in SessionStart** - Surface `orch doctor` or server health summary
5. **Optionally** - Add orchestrator-equivalent of GenerateServerContext to SessionStart

### Alternative Approaches Considered

**Option B: Just fix documentation**
- **Pros:** Minimal code changes
- **Cons:** Doesn't address dynamic surfacing; orchestrators still won't get runtime status
- **When to use instead:** If SessionStart latency is a concern

**Option C: Create new `orch startup` command**
- **Pros:** Explicit, on-demand context surfacing
- **Cons:** Requires orchestrators to remember to run it; passive mechanism
- **When to use instead:** If SessionStart is too heavy for operational context

**Rationale for recommendation:** The four-part approach fixes the immediate bugs (port, CLAUDE.md) while setting up infrastructure for future improvements (SessionStart surfacing). It doesn't require new commands or architectural changes.

---

### Implementation Details

**What to implement first:**
1. Fix port 3333→3348 in `~/.claude/skills/meta/orchestrator/.skillc/` source files, then `skillc build`
2. Add "Development Setup" section to CLAUDE.md with web UI startup instructions
3. Create `stale-binary-warning.py` SessionStart hook - ~30 lines, calls `orch version --source`

**Things to watch out for:**
- ⚠️ Port 5188 in config.yaml vs 5173 vite default - may need alignment
- ⚠️ SessionStart hook is shared across all projects - changes affect everything
- ⚠️ `orch servers list` shows orch-go as "stopped" even when web is running via launchd - status detection may be incomplete

**Areas needing further investigation:**
- Why does `orch servers list` show "stopped" when launchd is running the web server?
- Should there be a unified "orchestrator session context" that matches what workers get?

**Success criteria:**
- ✅ Orchestrator skill references correct port 3348
- ✅ CLAUDE.md has clear "how to start web UI" instructions
- ✅ Orchestrator can find server startup info without trial and error

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` - orchestrator guidance
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - project context
- `~/.claude/hooks/session-start.sh` - SessionStart hook implementation
- `~/.claude/hooks/cdd-hooks.json` - hook configuration
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - spawn context generation
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/config.yaml` - server port config
- `/Users/dylanconlin/Documents/personal/orch-go/web/package.json` - web UI scripts

**Commands Run:**
```bash
# Verify port behavior
curl -s http://127.0.0.1:3333/health  # Nothing
curl -s http://127.0.0.1:3348/health  # {"status":"ok"}

# Check what orchestrators see
grep -E "server|web|port|serve|dashboard" /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

# Check server status
~/bin/orch doctor
~/bin/orch serve status
~/bin/orch servers list
~/bin/orch servers status orch-go

# Check launchd services
launchctl list | grep -i orch
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md` - recommended SessionStart for server health
- **Investigation:** `.kb/investigations/2025-12-25-inv-pressure-over-compensation-surfacing-mechanisms.md` - cataloged 8 surfacing mechanisms
- **Decision:** `.kb/quick/entries.jsonl` (kb-319913) - "SessionStart surfacing is implemented via --session-start flag"

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Discovered Work Check

| Type | Item | Action |
|------|------|--------|
| **Bug** | Port 3333 in orchestrator skill should be 3348 | Create via `bd create` |
| **Documentation** | Missing web UI startup instructions in CLAUDE.md | Included in recommendations |
| **Enhancement** | Add stale binary warning to SessionStart hook | Create via `bd create` |
| **Enhancement** | SessionStart hook could surface server status | Included in recommendations |

---

## Investigation History

**2025-12-28 ~12:30:** Investigation started
- Initial question: What gaps exist in session-start context for orchestrators?
- Context: User reported having to discover web UI startup through trial and error

**2025-12-28 ~12:45:** Found port mismatch
- Orchestrator skill references 3333, but orch serve runs on 3348
- Verified with curl commands

**2025-12-28 ~13:00:** Identified documentation and hook gaps
- CLAUDE.md lacks "how to start web UI" instructions
- SessionStart hook is workspace-focused, not server-focused
- Spawned agents get server context, orchestrators don't

**2025-12-28 ~13:15:** Investigation completed
- Status: Complete
- Key outcome: Four specific gaps identified with actionable fixes
