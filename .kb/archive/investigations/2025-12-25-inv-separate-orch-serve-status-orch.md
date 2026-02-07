<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `orch serve status` subcommand to check if the API server is running, separating infrastructure monitoring from project server status.

**Evidence:** Tests pass, `orch serve status` correctly reports running/not running status, `orch serve --help` shows the status subcommand.

**Knowledge:** `orch serve` is infrastructure (persistent monitoring), not a project dev server. It should not be mixed with `orch servers` (project-specific, tmux-managed).

**Next:** Close this issue. Users should manually run `orch port release orch-go api` to remove false coupling from existing installs.

**Confidence:** High (90%) - Implementation tested, but release of existing port allocation is manual.

---

# Investigation: Separate Orch Serve Status from Orch Servers

**Question:** How do we separate the `orch serve` API server status from project servers managed by `orch servers`?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: orch serve and orch servers have different operational patterns

**Evidence:** 
- `orch serve` runs a persistent HTTP API server on port 3348 for monitoring
- `orch servers` manages ephemeral tmux-based dev servers per project
- The API server is infrastructure, not project-specific

**Source:** 
- `cmd/orch/serve.go` - API server implementation
- `cmd/orch/servers.go` - tmux-based server management
- Prior investigation: `.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md`

**Significance:** Mixing these in status output creates semantic confusion - a project can show "running" (tmux session exists) while the API is down, or vice versa.

---

### Finding 2: Simple health check is sufficient for status

**Evidence:**
- `orch serve` already exposes `/health` endpoint
- A 2-second timeout HTTP GET is fast and reliable
- No need for PID file management or process listing

**Source:** 
- `cmd/orch/serve.go:109-113` - health check endpoint
- `cmd/orch/serve.go:87-131` - new `runServeStatus` function

**Significance:** The simplest approach is best - check if the API responds, report status accordingly.

---

## Synthesis

**Key Insights:**

1. **Separation is semantic, not just technical** - The issue isn't just about checking ports, it's about clearly distinguishing infrastructure from project resources.

2. **Subcommand pattern works well** - Adding `orch serve status` as a subcommand keeps backward compatibility while adding the new capability.

3. **Documentation is key** - The command help now explicitly states that `orch serve` is "orchestration infrastructure (persistent monitoring), NOT a project dev server."

**Answer to Investigation Question:**

We separated the status by adding `orch serve status` as a subcommand that checks the API server health independently of `orch servers`. The implementation uses a simple HTTP health check against `/health` endpoint with a 2-second timeout. Users can now check API status with `orch serve status` while `orch servers list/status` remains focused on project dev servers.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The implementation is tested and working. The only remaining step is manual: users with existing orch-go port allocations should run `orch port release orch-go api` to remove the false coupling.

**What's certain:**

- ✅ `orch serve status` correctly reports running/not running
- ✅ Tests pass for new functionality
- ✅ Backward compatibility maintained (`orch serve` still starts server)

**What's uncertain:**

- ⚠️ Users need to manually release the false port allocation
- ⚠️ Future projects initialized with `orch init` will still allocate API ports (which is correct for projects that have their own API)

---

## Implementation

**Implemented:**
1. Added `DefaultServePort` constant (3348)
2. Added `serveStatusCmd` subcommand
3. Added `runServeStatus()` function with health check
4. Updated command help to document infrastructure nature
5. Added tests for new status command

**Not implemented (deferred):**
- PID file tracking (not needed - health check is sufficient)
- Uptime reporting (would require more complex state tracking)
- Automatic release of false port allocations (requires manual action)

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added status subcommand and runServeStatus function
- `cmd/orch/serve_test.go` - Added tests for new functionality

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md` - Prior design investigation
- **Issue:** `orch-go-k0mg` - Beads tracking issue

---

## Investigation History

**2025-12-25 10:45:** Investigation started
- Initial question: How to separate orch serve status from orch servers
- Context: From architect investigation recommending separation

**2025-12-25 11:00:** Implementation complete
- Added `orch serve status` subcommand
- Tests passing
- Status: Complete
- Key outcome: Infrastructure status now separate from project server status
