<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `orch daemon preview` now shows per-issue rejection reasons when issues can't be spawned.

**Evidence:** Smoke test shows output like "orch-go-78jw: status is in_progress" - previously showed only "No spawnable issues".

**Knowledge:** Silent filtering in `NextIssue()` caused invisible daemon behavior; surfacing rejection reasons follows "Gate Over Remind" principle.

**Next:** Close - fix implemented and verified with unit tests and smoke test.

---

# Investigation: Daemon Show Rejection Reasons Preview

**Question:** Why does daemon preview only show "no spawnable issues" without explaining WHY each issue was rejected?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Agent og-debug-daemon-show-rejection-04jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Preview() silently called NextIssue() which filtered without explanation

**Evidence:** In `pkg/daemon/daemon.go:343-383`, `Preview()` simply called `NextIssue()` and returned "No spawnable issues in queue" if nil was returned. The actual filtering logic with reasons existed only in `NextIssueExcluding()` (lines 169-244) behind `if d.Config.Verbose` debug output.

**Source:** pkg/daemon/daemon.go:343-383, pkg/daemon/daemon.go:169-244

**Significance:** Orchestrators had no visibility into why issues were being rejected, leading to debugging sessions like session ses_474f where the actual cause (null type in JSON) was invisible.

---

### Finding 2: Six distinct rejection reasons exist in the daemon

**Evidence:** The code in `NextIssueExcluding()` checks for:
1. Empty/missing type (line 193 - `IsSpawnableType`)
2. Non-spawnable type like epic/chore (line 193)
3. Status is blocked (line 200)
4. Status is in_progress (line 207)
5. Missing required label (line 214)
6. Has blocking dependencies (line 221)

**Source:** pkg/daemon/daemon.go:169-244, pkg/daemon/skill_inference.go:10-17

**Significance:** All rejection scenarios were already detected but not exposed to the user. The fix surfaces these reasons in preview output.

---

### Finding 3: Output format matches expected spec

**Evidence:** After fix, `orch daemon preview` outputs:
```
Rejected issues:
  orch-go-78jw: status is in_progress (already being worked on)
  orch-go-eysk: type 'epic' not spawnable (must be bug/feature/task/investigation)
  orch-go-eysk.4: missing label 'triage:ready'
```

**Source:** Smoke test output

**Significance:** Matches the expected format from the spawn context spec.

---

## Synthesis

**Key Insights:**

1. **Silent filtering is an anti-pattern for system visibility** - The daemon silently accepted then silently rejected issues, violating "Gate Over Remind" and "Surfacing Over Browsing" principles.

2. **Per-issue rejection reasons require iterating all issues, not just finding first spawnable** - Changed `Preview()` to iterate all issues and collect rejection reasons instead of returning on first match.

3. **Empty type check before `IsSpawnableType()` gives clearer message** - "missing type (required for skill inference)" is more actionable than "type '' not spawnable".

**Answer to Investigation Question:**

The daemon preview showed only "no spawnable issues" because `Preview()` delegated filtering to `NextIssue()` which returned nil on no match. The verbose debug output existed but wasn't accessible to users. The fix adds a `checkRejectionReason()` method that `Preview()` calls for each issue, collecting a `RejectedIssues` slice that is then formatted for display.

---

## Structured Uncertainty

**What's tested:**

- Preview returns rejected issues with reasons (unit test: TestDaemon_Preview_ShowsRejectionReasons)
- Missing label rejection surfaced correctly (unit test: TestDaemon_Preview_ShowsMissingLabelRejection)
- FormatRejectedIssues handles empty and populated slices (unit tests)
- Smoke test: orch daemon preview shows rejection reasons

**What's untested:**

- N/A - all scenarios covered

**What would change this:**

- Finding would be wrong if rejection reasons are not accurate (e.g., misreporting status)

---

## Implementation Recommendations

**Purpose:** N/A - fix already implemented.

### Recommended Approach (Implemented)

Added `RejectedIssue` struct, modified `Preview()` to collect all issues with rejection reasons, and formatted output in CLI.

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Core daemon logic, Preview(), NextIssue()
- `pkg/daemon/skill_inference.go` - IsSpawnableType()
- `cmd/orch/daemon.go` - CLI commands

**Commands Run:**
```bash
# Build and install
make install

# Run tests
go test ./pkg/daemon/... -v -run "TestDaemon_Preview_Shows" -timeout 60s

# Smoke test
orch daemon preview
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-debug-daemon-show-rejection-04jan/SYNTHESIS.md`
- **Beads:** `bd show orch-go-78jw`

---

## Investigation History

**2026-01-04 22:30:** Investigation started
- Initial question: Why does daemon preview not show rejection reasons?
- Context: Orchestrator spent ~15 turns debugging invisible daemon state (session ses_474f)

**2026-01-04 22:45:** Root cause identified
- Preview() silently delegated to NextIssue() without surfacing rejection reasons
- All rejection scenarios already existed but were only in verbose debug output

**2026-01-04 23:00:** Fix implemented and verified
- Added RejectedIssue struct and checkRejectionReason() method
- Unit tests passing, smoke test confirmed fix works
- Status: Complete
