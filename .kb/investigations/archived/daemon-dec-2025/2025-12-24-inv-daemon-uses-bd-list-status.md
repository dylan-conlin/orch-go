<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon's ListOpenIssues() uses `bd list --status open` which excludes `in_progress` issues; should use `bd ready --json` instead.

**Evidence:** `bd ready --help` shows it returns "open or in_progress" issues without blockers; confirmed `in_progress` issue (orch-go-s1i2) with `triage:ready` label would be missed by current implementation.

**Knowledge:** The daemon's label filtering (triage:ready) is correct, but the underlying issue fetch excludes valid work candidates.

**Next:** Change ListOpenIssues() at pkg/daemon/daemon.go:313 from `bd list --status open --json` to `bd ready --json`.

**Confidence:** Very High (95%) - clear bug with straightforward fix.

---

# Investigation: Daemon Uses Bd List Status

**Question:** Why does the daemon miss in_progress issues that have triage:ready label?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent (orch-go-d0x9)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: ListOpenIssues uses wrong bd command

**Evidence:** At pkg/daemon/daemon.go:313, the function uses:
```go
cmd := exec.Command("bd", "list", "--status", "open", "--json")
```

This filters to only `status=open`, excluding `status=in_progress` issues.

**Source:** pkg/daemon/daemon.go:310-325

**Significance:** Issues that are `in_progress` with `triage:ready` label (meaning an agent can resume/work on them) are excluded from daemon processing.

---

### Finding 2: bd ready returns both open and in_progress

**Evidence:** From `bd ready --help`:
```
Show ready work (no blockers, open or in_progress)
```

The `bd ready` command is specifically designed for exactly what the daemon needs - issues that are workable.

**Source:** `bd ready --help`

**Significance:** Using `bd ready` instead of `bd list --status open` would correctly include both open and in_progress issues.

---

### Finding 3: Real in_progress issues exist with triage:ready

**Evidence:** Running `bd list --status in_progress --json` showed:
- `orch-go-s1i2` with `status: in_progress` and `labels: ["triage:ready"]`

This issue would be missed by the current daemon implementation but SHOULD be picked up.

**Source:** `bd list --status in_progress --json` output in orch-go directory

**Significance:** This is a real-world case where the bug causes work to be missed.

---

## Synthesis

**Key Insights:**

1. **Command mismatch** - The daemon was designed to work with `triage:ready` labeled issues, but the underlying data fetch is too restrictive.

2. **bd ready is purpose-built** - The beads CLI has a dedicated `bd ready` command that encapsulates the exact semantics needed (open OR in_progress, no blockers).

3. **Simple fix** - Just change the exec.Command from `bd list --status open --json` to `bd ready --json`.

**Answer to Investigation Question:**

The daemon misses in_progress issues because `ListOpenIssues()` at pkg/daemon/daemon.go:313 uses `bd list --status open` which explicitly filters to only `open` status. The fix is to use `bd ready` which returns both `open` and `in_progress` issues.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Clear root cause identified with straightforward fix. Both the problem and solution are well-documented in bd CLI help.

**What's certain:**

- ✅ `bd list --status open` excludes in_progress issues
- ✅ `bd ready` includes both open and in_progress issues
- ✅ Real in_progress issues with triage:ready exist and would be missed

**What's uncertain:**

- ⚠️ Whether any downstream code assumes status=open (review suggests not - daemon filters by label, not status)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Replace bd list with bd ready** - Single line change at pkg/daemon/daemon.go:313

**Why this approach:**
- `bd ready` is designed exactly for this use case
- Includes both open and in_progress without blockers
- Simpler than adding `--status open,in_progress` (which bd list doesn't support)

**Implementation sequence:**
1. Change the exec.Command from `bd list --status open --json` to `bd ready --json`
2. Run tests to verify no breakage
3. Rename function from ListOpenIssues to ListReadyIssues (semantic accuracy)

### Alternative Approaches Considered

**Option B: Use bd list with multiple calls**
- **Pros:** More explicit about statuses included
- **Cons:** Two commands = more latency, more code, bd list doesn't support comma-separated statuses
- **When to use instead:** Never - bd ready is the right tool

**Rationale for recommendation:** bd ready exists specifically for this use case and is more maintainable.

---

## References

**Files Examined:**
- pkg/daemon/daemon.go:310-325 - ListOpenIssues function with bug
- pkg/daemon/daemon_test.go - Tests use mock, don't call real bd

**Commands Run:**
```bash
# Checked bd ready behavior
bd ready --help

# Checked bd list behavior
bd list --help

# Confirmed in_progress issues with triage:ready exist
bd list --status in_progress --json
```

---

## Investigation History

**2025-12-24:** Investigation started
- Initial question: Why does daemon miss in_progress issues with triage:ready label?
- Context: Spawned from beads issue orch-go-d0x9

**2025-12-24:** Root cause identified
- ListOpenIssues() uses `bd list --status open` which excludes in_progress
- `bd ready` is the correct command to use

**2025-12-24:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Simple fix - replace `bd list --status open` with `bd ready`
