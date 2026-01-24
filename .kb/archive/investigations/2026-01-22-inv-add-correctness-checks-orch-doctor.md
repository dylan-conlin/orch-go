<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added 3 correctness checks to orch doctor: beads PRAGMA integrity_check, registry-tmux reconciliation, docker container spawn test.

**Evidence:** All tests pass (go test ./cmd/orch/ -run TestCheck), live orch doctor run shows checks working (detected 101 stale registry entries).

**Knowledge:** OpenCode check already tested API (not just port) via ListSessions(). Correctness checks are distinct from liveness - they verify data integrity and consistency.

**Next:** Close - implementation complete and tested.

**Promote to Decision:** recommend-no (tactical fix, follows existing patterns)

---

# Investigation: Add Correctness Checks Orch Doctor

**Question:** How to add correctness checks (not just liveness) to orch doctor?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode check already tests API, not just port

**Evidence:** `checkOpenCode()` at `cmd/orch/doctor.go:276-298` calls `client.ListSessions("")` - this is an API correctness check, not just a port liveness check.

**Source:** `cmd/orch/doctor.go:286-288`

**Significance:** One of the requested checks (OpenCode API test) was already implemented. No changes needed for this one.

---

### Finding 2: Beads uses SQLite database at .beads/beads.db

**Evidence:** Directory listing shows `.beads/beads.db` (4.1 MB), plus evidence of past corruption (`backup-corrupted-*` directories).

**Source:** `ls -la .beads/` output

**Significance:** PRAGMA integrity_check can be run via sqlite3 CLI. Past corruption evidence validates the need for this check.

---

### Finding 3: Registry stores tmux window IDs for reconciliation

**Evidence:** `pkg/registry/registry.go` Agent struct has `TmuxWindow string` field. `pkg/tmux/tmux.go` has `WindowExistsByID()` function.

**Source:** `pkg/registry/registry.go:50`, `pkg/tmux/tmux.go:1004-1025`

**Significance:** All components exist to implement registry-tmux reconciliation by checking if active agents' tmux windows still exist.

---

## Synthesis

**Key Insights:**

1. **Correctness vs liveness is a useful distinction** - orch doctor previously checked "is it running?" but missed "is it working correctly?" Adding PRAGMA integrity_check, registry reconciliation, and docker test closes this gap.

2. **Fail-loud principle (kb-42d29d)** - All new checks log at WARN level if they fail, use actionable error messages, and don't block other checks.

3. **Optional services handled gracefully** - Docker check returns Running=true if Docker isn't installed (it's optional). Beads check returns Running=true if no .beads/ exists.

**Answer to Investigation Question:**

Implemented 3 correctness checks:
1. `checkBeadsIntegrity()` - Runs PRAGMA integrity_check on .beads/beads.db
2. `checkRegistryReconciliation()` - Compares active registry entries against tmux windows
3. `checkDockerBackend()` - Tests trivial container spawn if docker is available

The OpenCode API check was already implemented (ListSessions call, not just port check).

---

## Structured Uncertainty

**What's tested:**

- ✅ All correctness checks compile (go build ./cmd/orch/)
- ✅ All correctness check tests pass (go test ./cmd/orch/ -run TestCheck)
- ✅ Live orch doctor run shows checks working (detected 101 stale registry entries, docker test passed)

**What's untested:**

- ⚠️ Performance impact on --watch mode (checks run every 30s)
- ⚠️ Behavior with corrupted beads database (would need to artificially corrupt DB)

**What would change this:**

- If PRAGMA integrity_check returns non-"ok" for valid databases
- If WindowExistsByID returns wrong results

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Add correctness checks to existing doctor flow** - Reuse ServiceStatus pattern, add checks after liveness checks, include in --watch and --daemon modes.

**Why this approach:**
- Follows existing patterns in doctor.go
- Non-blocking (checks run independently)
- Fail-loud with actionable messages

**Implementation sequence:**
1. Added new check functions (`checkBeadsIntegrity`, `checkRegistryReconciliation`, `checkDockerBackend`)
2. Integrated into main `runDoctor()` flow
3. Added to --watch and --daemon modes
4. Updated command help text
5. Added tests

---

## References

**Files Modified:**
- `cmd/orch/doctor.go` - Added 3 correctness check functions and integrated into all modes
- `cmd/orch/doctor_test.go` - Added tests for correctness checks

**Commands Run:**
```bash
# Verify compilation
go build ./cmd/orch/

# Run tests
go test ./cmd/orch/ -run TestCheck -v

# Live test
make install && orch doctor
```

---

## Investigation History

**2026-01-22 20:00:** Investigation started
- Initial question: Add correctness checks to orch doctor
- Context: orch doctor passed but missed beads corruption, registry drift, docker issues

**2026-01-22 20:05:** Implementation complete
- Added 3 correctness checks
- All tests passing
- Live orch doctor confirms checks working
