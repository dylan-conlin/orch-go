<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented hotspot warnings in daemon preview to flag issues touching high-churn areas before auto-spawning.

**Evidence:** 23 unit tests pass covering HotspotWarning, PreviewResult integration, FormatHotspotWarnings, and GitHotspotChecker.

**Knowledge:** Hotspot detection integrates with daemon via HotspotChecker interface; shells out to `orch hotspot --json` for analysis; gracefully degrades if command unavailable.

**Next:** Close - implementation complete with all tests passing.

---

# Investigation: Add Hotspot Warnings Orch Daemon

**Question:** How to integrate hotspot detection into daemon preview to warn before auto-spawning on high-churn areas?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing hotspot command provides JSON output

**Evidence:** `orch hotspot --json` outputs structured JSON with hotspots array containing path, type, score, and recommendation fields.

**Source:** cmd/orch/hotspot.go:299-307 (outputJSON function)

**Significance:** Can reuse existing analysis logic by shelling out to `orch hotspot --json` rather than duplicating git analysis code.

---

### Finding 2: PreviewResult can be extended with warnings

**Evidence:** Added `HotspotWarnings []HotspotWarning` to PreviewResult struct along with helper methods `HasHotspotWarnings()` and `HasCriticalHotspots()`.

**Source:** pkg/daemon/daemon.go:80-105

**Significance:** Preview results now carry hotspot context that CLI can display.

---

### Finding 3: HotspotChecker interface enables testing

**Evidence:** Created HotspotChecker interface with single method `CheckHotspots(projectDir string) ([]HotspotWarning, error)`. MockHotspotChecker for tests, GitHotspotChecker for production.

**Source:** pkg/daemon/hotspot.go:25-29, pkg/daemon/hotspot_checker.go

**Significance:** Clean separation allows easy testing and alternative implementations.

---

## Structured Uncertainty

**What's tested:**

- ✅ HotspotWarning IsCritical() returns true for score >= 10 (verified: TestHotspotWarning_IsCritical)
- ✅ PreviewResult HasHotspotWarnings() and HasCriticalHotspots() work correctly (verified: unit tests)
- ✅ FormatHotspotWarnings produces expected output (verified: TestFormatHotspotWarnings_*)
- ✅ Daemon.Preview() integrates hotspot checking (verified: TestDaemon_Preview_WithHotspotChecker)
- ✅ Graceful degradation when checker returns error (verified: TestDaemon_Preview_HotspotCheckerError)

**What's untested:**

- ⚠️ End-to-end with real orch hotspot command (requires orch binary built)
- ⚠️ Performance impact on daemon preview (not benchmarked)
- ⚠️ Hotspot detection for issues that don't have clear file targets (description-based matching)

---

## References

**Files Created/Modified:**
- pkg/daemon/hotspot.go - HotspotWarning struct, HotspotChecker interface, FormatHotspotWarnings
- pkg/daemon/hotspot_checker.go - GitHotspotChecker implementation
- pkg/daemon/hotspot_test.go - Unit tests (23 tests)
- pkg/daemon/daemon.go - Added HotspotChecker field to Daemon, HotspotWarnings to PreviewResult
- cmd/orch/daemon.go - Wired hotspot checking into preview and dry-run commands

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test run
go test ./pkg/daemon/... -run "Hotspot|Preview" -v
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-04-design-patch-density-architect-escalation.md - Original design for hotspot detection
- **Investigation:** .kb/investigations/2026-01-04-inv-implement-orch-hotspot-cli-command.md - orch hotspot command implementation

---

## Investigation History

**2026-01-04 09:15:** Investigation started
- Initial question: How to add hotspot warnings to daemon preview?
- Context: Epic yz3d task 4 - integrate hotspot detection into spawn preview

**2026-01-04 09:30:** Implementation complete
- Created HotspotWarning struct and HotspotChecker interface
- Extended PreviewResult with HotspotWarnings
- Implemented GitHotspotChecker (shells to orch hotspot --json)
- Updated daemon preview CLI to display warnings
- All 23 tests passing
