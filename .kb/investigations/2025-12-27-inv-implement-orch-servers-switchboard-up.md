<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `orch servers up/down/status` switchboard commands that use launchd for native processes and Docker for containers, reading from `.orch/servers.yaml`.

**Evidence:** All 26 tests pass in pkg/servers; commands build successfully; lifecycle.go provides Up/Down/Status functions with per-server-type handling.

**Knowledge:** The switchboard pattern (up/down/status) is cleaner than individual start/stop for multi-server projects; launchd provides native macOS service management with auto-restart.

**Next:** Test with a real project's servers.yaml to validate end-to-end lifecycle management.

---

# Investigation: Implement Orch Servers Switchboard Up

**Question:** How should `orch servers up/down/status` commands manage per-project servers using launchd and Docker?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Prior Schema Already Defined

**Evidence:** The `pkg/servers/servers.go` package already defines the servers.yaml schema with Server, HealthCheck, and Duration types. It supports three server types: command, docker, and launchd.

**Source:** `pkg/servers/servers.go:361-396`, prior investigation `2025-12-27-inv-define-servers-yaml-schema-per.md`

**Significance:** No schema design needed - focused purely on lifecycle management implementation.

---

### Finding 2: Existing Commands Use tmuxinator

**Evidence:** The existing `orch servers start/stop` commands use tmuxinator to manage tmux sessions. This works but is heavy for simple server lifecycle management.

**Source:** `cmd/orch/servers.go:231-260` (runServersStart)

**Significance:** The new up/down commands provide a lighter alternative using launchd for native processes and Docker for containers, while preserving tmuxinator as a "legacy" option.

---

### Finding 3: Launchd Provides Native macOS Service Management

**Evidence:** macOS launchd supports:
- Auto-restart on failure (KeepAlive)
- Log redirection (StandardOutPath/StandardErrorPath)
- Environment variable configuration
- Working directory specification

**Source:** launchd documentation, existing `pkg/servers/servers.go:90-328` plist generation code

**Significance:** Using launchd for command-type servers gives us production-grade service management without external dependencies.

---

## Synthesis

**Key Insights:**

1. **Type-based dispatch** - Different server types need different lifecycle management: command → launchd, docker → docker commands, launchd → launchctl.

2. **Graceful handling** - Operations should report per-server success/failure rather than failing fast, allowing partial success when some servers work.

3. **Status checking** - launchd status can be checked via `launchctl print` looking for "state = running".

**Answer to Investigation Question:**

The switchboard uses a type-based dispatch pattern:
- `up`: Loads servers.yaml, iterates servers, dispatches to startCommandServer/startDockerServer/startLaunchdServer based on type
- `down`: Same pattern but calls stop functions in reverse dependency order
- `status`: Checks each server's status using launchctl print (command) or docker ps (docker)

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles successfully (verified: `go build ./...`)
- ✅ All 26 pkg/servers tests pass (verified: `go test ./pkg/servers/...`)
- ✅ Up/Down/Status return empty results for missing servers.yaml (verified: unit tests)
- ✅ Status correctly identifies stopped servers (verified: unit tests)

**What's untested:**

- ⚠️ Actual launchd service bootstrap/bootout (requires real services)
- ⚠️ Docker container lifecycle (requires Docker daemon)
- ⚠️ Health check execution (schema only, runtime not implemented)

**What would change this:**

- If launchctl commands differ on older macOS versions
- If Docker API changes in future versions

---

## Implementation Recommendations

**Purpose:** This was an implementation task, not just investigation.

### Implemented Approach ⭐

**Lifecycle package with type dispatch** - Added `pkg/servers/lifecycle.go` with Up/Down/Status functions.

**Why this approach:**
- Clean separation from schema code
- Type-based dispatch is extensible
- Per-server results allow partial success

**Trade-offs accepted:**
- Health checks not yet executed at runtime (schema only)
- Dependency ordering is simple (reverse order for stop)

**Implementation sequence:**
1. ✅ Created lifecycle.go with Up/Down/Status functions
2. ✅ Added up/down subcommands to cmd/orch/servers.go
3. ✅ Enhanced status to accept optional project argument
4. ✅ Added lifecycle tests

---

### Implementation Details

**What was implemented:**

- `pkg/servers/lifecycle.go`: Up(), Down(), Status() functions with type dispatch
- `cmd/orch/servers.go`: serversUpCmd, serversDownCmd with --project-dir flag
- Enhanced serversStatusCmd to accept optional project for per-server status
- gen-plist command integrated with servers.yaml
- Duration MarshalYAML for proper serialization

**Files changed:**
- `pkg/servers/lifecycle.go` (new)
- `pkg/servers/lifecycle_test.go` (new)
- `pkg/servers/servers.go` (added MarshalYAML)
- `cmd/orch/servers.go` (added up/down commands, enhanced status)

---

## References

**Files Examined:**
- `pkg/servers/servers.go` - Existing schema and plist generation
- `cmd/orch/servers.go` - Existing server commands
- `pkg/port/port.go` - Port registry for context

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification
go test ./pkg/servers/... -v
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-define-servers-yaml-schema-per.md` - Prior schema work

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: How to implement orch servers up/down/status
- Context: Manual control for dev servers per project using launchd/Docker

**2025-12-27:** Implementation completed
- Added lifecycle.go with Up/Down/Status
- Added up/down commands to CLI
- Enhanced status command
- All 26 tests pass
