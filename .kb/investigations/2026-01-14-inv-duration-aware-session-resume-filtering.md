## Summary (D.E.K.N.)

**Delta:** Add duration-aware filtering to `scanAllWindowsForMostRecent()` to prefer substantive sessions (≥5min) over brief test sessions.

**Evidence:** Brief test sessions (e.g., `test-session/2026-01-13-1547` with 3.3s duration) can override substantive work sessions in resume discovery because current logic only compares timestamps.

**Knowledge:** Duration line format varies: `YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM` (new), `HH:MM → HH:MM` (same-day), or raw seconds (legacy). Parse first format, treat others as invalid (fallback to any).

**Next:** Implement two-candidate tracking (substantive ≥5min, any) and return substantive if exists.

**Promote to Decision:** recommend-no (tactical bugfix, not architectural)

---

# Investigation: Duration Aware Session Resume Filtering

**Question:** How to filter session handoffs by duration to prevent brief test sessions from overriding substantive work?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current scanAllWindowsForMostRecent() only compares timestamps

**Evidence:** Lines 969-1036 in session.go show the function iterates through window directories, finds latest symlink, checks for SESSION_HANDOFF.md, and compares directory timestamps lexicographically to find most recent.

**Source:** `cmd/orch/session.go:969-1036`

**Significance:** No duration filtering exists - a 3-second test session with a newer timestamp will be selected over an 8-hour substantive session.

---

### Finding 2: Duration line has multiple formats in existing handoffs

**Evidence:**
- New format: `**Duration:** 2026-01-14 12:54 → 2026-01-14 21:30`
- Same-day: `**Duration:** 2026-01-14 11:52 → 12:30 (38m)`
- Legacy: `**Duration:** 3.296167s`
- Incomplete: `**Duration:** 2026-01-14 07:29 → {end-time}`

**Source:** `.orch/session/*/*/SESSION_HANDOFF.md` files examined

**Significance:** Need to handle the primary format `YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM` per task spec. Sessions with unparseable duration should fall into "any" bucket.

---

### Finding 3: 5-minute threshold for "substantive" session

**Evidence:** Task specifies ≥5 minutes as threshold for substantive sessions.

**Source:** Task description in SPAWN_CONTEXT.md

**Significance:** This is a reasonable threshold - brief tests typically last seconds, while real work sessions are 30+ minutes.

---

## Implementation Plan

1. **Create `parseDurationFromHandoff()` function:**
   - Read first 20 lines of SESSION_HANDOFF.md
   - Find line matching `**Duration:** YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM`
   - Parse both timestamps and compute duration
   - Return duration in minutes (or -1 if unparseable)

2. **Modify `scanAllWindowsForMostRecent()`:**
   - Track `mostRecentSubstantive` (duration ≥5min) and `mostRecentAny`
   - For each valid handoff, parse duration
   - Update appropriate candidate based on duration
   - Return substantive if exists, else any

---

## References

**Files Examined:**
- `cmd/orch/session.go:969-1036` - scanAllWindowsForMostRecent function
- `.orch/session/*/*/SESSION_HANDOFF.md` - Various handoff format examples
