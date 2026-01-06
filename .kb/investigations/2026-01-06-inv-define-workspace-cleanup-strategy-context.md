## Summary (D.E.K.N.)

**Delta:** Implemented workspace cleanup strategy with `orch clean --stale` for archiving old workspaces and `orch doctor --sessions` for detecting zombie sessions.

**Evidence:** Tested `--stale` flag - found 132 workspaces >7 days old eligible for archival; tested `--sessions` - cross-referenced 273 workspaces with 600 sessions.

**Knowledge:** File-based completion detection (SYNTHESIS.md, .tier, .beads_id) is fast; beads API calls are slow and should be avoided in bulk operations.

**Next:** Close issue - implementation complete. Users can run `orch clean --stale` to archive old workspaces and `orch doctor --sessions` to detect orphans.

---

# Investigation: Define Workspace Cleanup Strategy

**Question:** What should the cleanup strategy be for completed workspaces and OpenCode sessions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Workspace State Analysis

**Evidence:** 
- 295 workspaces in orch-go, ~13MB total disk usage
- Oldest workspaces from Dec 23, 2025 (~14 days old)
- 132 workspaces older than 7 days (44%)
- Light tier dominates (218/288 = 76%)

**Source:** `.orch/workspace/` directory analysis

**Significance:** Disk usage is minimal (13MB), so aggressive cleanup isn't urgent. A 7-day retention policy is reasonable - keeps recent work accessible while archiving old completed work.

---

### Finding 2: Session Cross-Reference

**Evidence:**
- 600 OpenCode sessions across projects
- 273 workspaces have .session_id files
- 511 sessions without corresponding workspaces (expected - orchestrator/interactive sessions)
- 0 workspaces with missing sessions

**Source:** `orch doctor --sessions` implementation

**Significance:** Sessions without workspaces are normal (orchestrator sessions don't have workspaces). Workspaces with missing sessions would indicate OpenCode restart or session cleanup - currently none found.

---

### Finding 3: Performance Considerations

**Evidence:**
- Original implementation with beads API calls took >2 minutes for 295 workspaces
- File-based detection (SYNTHESIS.md, .tier, .beads_id) completes in <1 second
- Beads daemon startup adds 5+ seconds per call when cold

**Source:** Testing `orch clean --stale` implementation

**Significance:** Bulk operations must avoid individual beads API calls. File-based indicators are sufficient for determining workspace completion status.

---

## Implementation

### Added: `orch clean --stale` flag

Archives completed workspaces older than N days (default: 7) to `.orch/workspace/archived/`.

**Completion detection (file-based for speed):**
1. SYNTHESIS.md exists → completed full-tier spawn
2. .tier = "light" → no SYNTHESIS.md required by design
3. .beads_id exists → tracked spawn (was a real agent)

**Usage:**
```bash
orch clean --stale              # Archive workspaces >7 days old
orch clean --stale --stale-days 14  # Archive workspaces >14 days old
orch clean --stale --dry-run    # Preview what would be archived
```

### Added: `orch doctor --sessions` flag

Cross-references workspaces and OpenCode sessions to detect:
- Sessions without workspaces (usually fine - orchestrator/interactive)
- Workspaces with missing sessions (indicates session cleanup needed)

**Usage:**
```bash
orch doctor --sessions  # Run cross-reference check
```

---

## References

**Files Modified:**
- `cmd/orch/clean_cmd.go` - Added `--stale` and `--stale-days` flags, `archiveStaleWorkspaces()` function
- `cmd/orch/doctor.go` - Added `--sessions` flag, `runSessionsCrossReference()` function

**Prior Investigation:**
- `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Established tier system understanding

---

## Investigation History

**2026-01-06 11:15:** Investigation started
- Initial question: Define cleanup strategy for workspaces and sessions
- Context: Prior investigation found "orphaned workspaces" was a misunderstanding of tier system

**2026-01-06 12:30:** Implementation complete
- Added `orch clean --stale` for workspace archival
- Added `orch doctor --sessions` for cross-reference checking
- Optimized to use file-based detection (avoid slow beads API calls)

**2026-01-06 12:45:** Investigation completed
- Status: Complete
- Key outcome: Cleanup strategy defined and implemented with two new CLI features
