<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added /api/agents endpoint verification to orch doctor's checkOrchServe() function to verify dashboard can actually fetch data, not just port open.

**Evidence:** `orch doctor` now shows "Dashboard ready (8 agents)" instead of just "Health endpoint responding", confirmed via testing.

**Knowledge:** Port open + /health OK doesn't guarantee dashboard works. Must verify actual data endpoint (/api/agents) to ensure dashboard functionality.

**Next:** Close - implementation complete, tested, and committed.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Orch Doctor Verify Dashboard Fetch

**Question:** Does `orch doctor` verify that the dashboard can actually fetch agent data from `/api/agents`, or just that the port is open?

**Started:** 2026-01-14 14:30
**Updated:** 2026-01-14 20:25
**Owner:** og-feat-orch-doctor-verify-14jan-70a6
**Phase:** Complete
**Next Step:** N/A - Implementation complete
**Status:** Complete

---

## Findings

### Finding 1: checkOrchServe() only verifies TCP port and /health endpoint

**Evidence:** The current implementation in `checkOrchServe()` does:
1. TCP dial to verify port is listening (line 314)
2. HTTPS GET to /health endpoint (line 326-365)
3. Does NOT verify that /api/agents endpoint works

**Source:** cmd/orch/doctor.go:301-367

**Significance:** The dashboard needs `/api/agents` to function properly, but `orch doctor` doesn't verify this endpoint is working. Port open + /health OK doesn't guarantee dashboard will work.

---

### Finding 2: Dashboard fetches agent data from /api/agents

**Evidence:** The dashboard's primary data fetch is in `web/src/lib/stores/agents.ts:222`:
```typescript
const response = await fetch(`${API_BASE}/api/agents${queryString}`, {
  signal: currentFetchController.signal
});
```

**Source:** web/src/lib/stores/agents.ts:210-250, cmd/orch/serve_agents.go:22-50

**Significance:** This is the critical endpoint that must work for the dashboard to show agent status. If this fails, the dashboard is non-functional even if /health returns 200.

---

### Finding 3: Implementation verified via orch doctor output

**Evidence:** After implementing /api/agents verification:
```
✓ orch serve (port 3348)
  Dashboard ready (8 agents)
```

Before: would show "Health endpoint responding" or "Status: ok"
After: shows "Dashboard ready (N agents)" confirming the full verification chain.

**Source:** `orch doctor` output, cmd/orch/doctor.go:301-392

**Significance:** The new output clearly indicates whether the dashboard can fetch data, not just whether the server is running.

---

## Synthesis

**Key Insights:**

1. **Verification chain gaps** - Previous implementation stopped at /health check, which doesn't verify the actual data endpoint the dashboard depends on.

2. **Dashboard-centric verification** - By verifying /api/agents, orch doctor now confirms what users actually care about: can the dashboard show agent data?

3. **Informative output** - Showing agent count in the output ("Dashboard ready (8 agents)") provides immediate useful information.

**Answer to Investigation Question:**

Prior to this fix, `orch doctor` did NOT verify that the dashboard can fetch agent data from `/api/agents`. It only verified TCP port + /health endpoint. The fix adds verification of the `/api/agents` endpoint, including JSON parsing validation.

---

## Structured Uncertainty

**What's tested:**

- ✅ /api/agents endpoint returns 200 (verified: orch doctor shows "Dashboard ready (8 agents)")
- ✅ JSON parsing of response works (verified: agent count displayed)
- ✅ Verbose mode shows appropriate detail (verified: ran with --verbose)

**What's untested:**

- ⚠️ Behavior when server returns 500 (not tested - would need to break server intentionally)
- ⚠️ Behavior with very slow /api/agents response (timeout increased to 5s but not tested at boundary)

**What would change this:**

- Finding would be wrong if /api/agents can return 200 but with invalid data (mitigated by JSON parsing check)
- Finding would be incomplete if other dashboard endpoints exist that also need verification (currently only /api/agents is critical)

---

## References

**Files Examined:**
- cmd/orch/doctor.go - Main implementation of checkOrchServe()
- cmd/orch/serve_agents.go - /api/agents endpoint handler (AgentAPIResponse structure)

**Commands Run:**
```bash
# Build and test
make build
./build/orch doctor

# Install and verify
make install
orch doctor
```

---

## Investigation History

**2026-01-14 14:30:** Investigation started
- Initial question: Does orch doctor verify dashboard can fetch data?
- Context: Prior investigation identified the gap

**2026-01-14 20:20:** Implementation complete
- Added /api/agents verification to checkOrchServe()
- Tests pass, output shows "Dashboard ready (N agents)"

**2026-01-14 20:25:** Investigation completed
- Status: Complete
- Key outcome: checkOrchServe() now verifies /api/agents endpoint, ensuring dashboard functionality
