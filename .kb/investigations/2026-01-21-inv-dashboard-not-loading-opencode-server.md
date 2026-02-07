## Summary (D.E.K.N.)

**Delta:** Dashboard services aren't running - no processes on ports 4096/3348/5188 - and cannot be started from Claude Code's Linux sandbox environment.

**Evidence:** `lsof -i :4096 -i :3348 -i :5188` returned no processes; opencode binary returns "Exec format error" (darwin-arm64 binary on Linux x86_64).

**Knowledge:** Claude Code runs in a Linux sandbox, so dashboard services must be started from the user's macOS terminal, not through the agent.

**Next:** User should run `~/bin/orch-dashboard start` from their macOS terminal.

**Promote to Decision:** recommend-no - This is an environment constraint, not a pattern worth codifying.

---

# Investigation: Dashboard Not Loading - OpenCode Server Refusing Connections

**Question:** Why is the dashboard not loading and OpenCode server refusing connections on port 4096?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Claude agent (og-debug-dashboard-not-loading-21jan-ea12)
**Phase:** Complete
**Next Step:** None - user action required
**Status:** Complete

---

## Findings

### Finding 1: No services running on dashboard ports

**Evidence:**
```bash
$ lsof -i :4096 -i :3348 -i :5188
No processes found on these ports
```

All three dashboard services are stopped:
- Port 4096: OpenCode server (not running)
- Port 3348: orch API server (not running)
- Port 5188: Web UI (not running)

**Source:** `lsof -i :4096 -i :3348 -i :5188` command

**Significance:** This explains why the dashboard won't load - there's nothing to connect to. The services were not started or have crashed.

---

### Finding 2: orch-dashboard script exists but overmind not available

**Evidence:**
```bash
$ /Users/dylanconlin/bin/orch-dashboard start
→ Starting dashboard services...
→ Starting overmind...
env: 'overmind': No such file or directory
```

The script exists at `~/bin/orch-dashboard` and attempts to use overmind as the process manager, but overmind is not available in the current environment.

**Source:** `/Users/dylanconlin/bin/orch-dashboard:91` - `overmind start -D`

**Significance:** The orch-dashboard script requires overmind process manager to orchestrate the three services (defined in Procfile).

---

### Finding 3: Architecture mismatch - Linux sandbox vs macOS binaries

**Evidence:**
```bash
$ ~/.bun/bin/opencode --version
/bin/bash: cannot execute binary file: Exec format error

$ /Users/dylanconlin/Documents/personal/opencode/.../opencode-darwin-arm64/bin/opencode --version
/bin/bash: cannot execute binary file: Exec format error
```

Environment info shows:
- Platform: linux
- OS Version: Linux 6.8.0-64-generic

But binaries are compiled for darwin-arm64 (macOS ARM).

**Source:** Environment info from spawn context; opencode binary path includes `darwin-arm64`

**Significance:** This is the root cause of why services cannot be started from within Claude Code. The agent runs in a Linux x86_64 sandbox, but all the user's binaries (opencode, overmind) are macOS ARM binaries.

---

## Synthesis

**Key Insights:**

1. **Services stopped, not crashed** - No orphan processes exist. The services were simply never started or were cleanly stopped. This is a fresh state, not a crash recovery scenario.

2. **Architecture boundary** - Claude Code runs in a sandboxed Linux environment separate from the user's macOS host. This creates a fundamental constraint: agents cannot start/stop macOS-native services.

3. **User action required** - The dashboard must be started from the user's actual macOS terminal using `~/bin/orch-dashboard start` or equivalent.

**Answer to Investigation Question:**

The dashboard is not loading because all three services (OpenCode on 4096, orch API on 3348, web UI on 5188) are not running. The services cannot be started from within Claude Code because this agent runs in a Linux sandbox while the service binaries are compiled for macOS ARM. The user needs to run `~/bin/orch-dashboard start` from their macOS terminal to start the services.

---

## Structured Uncertainty

**What's tested:**

- ✅ No processes on ports 4096/3348/5188 (verified: lsof command returned empty)
- ✅ orch-dashboard script exists (verified: found at ~/bin/orch-dashboard)
- ✅ Overmind not available in sandbox (verified: "No such file or directory" error)
- ✅ Binary architecture mismatch (verified: "Exec format error" on opencode binary)

**What's untested:**

- ⚠️ Services will start successfully on macOS (user needs to verify)
- ⚠️ No underlying system issues preventing services from running
- ⚠️ Overmind is actually installed on the macOS host

**What would change this:**

- If user's actual terminal can't run overmind either, then overmind needs to be installed
- If services crash after starting, root cause is different

---

## Implementation Recommendations

**Purpose:** Enable the user to restore dashboard functionality.

### Recommended Approach ⭐

**Start services from macOS terminal** - Run `~/bin/orch-dashboard start` from the user's actual terminal

**Why this approach:**
- Direct path to resolution
- Uses existing infrastructure (orch-dashboard script handles cleanup and startup)
- Addresses the root cause (services not running)

**Trade-offs accepted:**
- Requires user manual action
- Agent cannot verify success directly

**Implementation sequence:**
1. Open macOS terminal
2. Run `~/bin/orch-dashboard start`
3. Verify dashboard loads at http://localhost:5188

### Alternative Approaches Considered

**Option B: Start services individually**
- **Pros:** Doesn't require overmind
- **Cons:** More complex, doesn't benefit from orch-dashboard's orphan cleanup
- **When to use instead:** If overmind is not installed on macOS host

Commands for manual start:
```bash
# Terminal 1: OpenCode
~/.bun/bin/opencode serve --port 4096

# Terminal 2: orch API
cd ~/Documents/personal/orch-go && orch serve

# Terminal 3: Web UI
cd ~/Documents/personal/orch-go/web && bun run dev
```

**Rationale for recommendation:** The orch-dashboard script is purpose-built for this, handles edge cases (orphan processes, stale sockets), and is the documented approach.

---

## References

**Files Examined:**
- `/Users/dylanconlin/bin/orch-dashboard` - Dashboard management script
- `/Users/dylanconlin/Documents/personal/orch-go/Procfile` - Service definitions
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - Project instructions

**Commands Run:**
```bash
# Check ports
lsof -i :4096 -i :3348 -i :5188

# Attempt service start
/Users/dylanconlin/bin/orch-dashboard start

# Verify binary architecture
~/.bun/bin/opencode --version
```

**Related Artifacts:**
- **Guide:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md` - Dashboard operations

---

## Investigation History

**2026-01-21 ~09:XX:** Investigation started
- Initial question: Why is dashboard not loading?
- Context: Spawned to diagnose OpenCode connection refused on port 4096

**2026-01-21 ~09:XX:** Root cause identified
- All services stopped (ports empty)
- Architecture mismatch discovered (Linux sandbox vs macOS binaries)

**2026-01-21 ~09:XX:** Investigation completed
- Status: Complete
- Key outcome: User must run `~/bin/orch-dashboard start` from macOS terminal
