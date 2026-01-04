<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Agents create investigation files via `kb create investigation` early in their lifecycle, but if they die before filling content, empty template files accumulate - 28 out of 471 investigation files had placeholder content.

**Evidence:** Found 28 files containing `[Brief, descriptive title]` placeholder text. Implemented and tested `orch clean --investigations` which successfully archived 22 empty files.

**Knowledge:** The problem is cleanup, not prevention - early file creation signals agent activity. The solution is archiving (not deleting) to preserve any partial context for future analysis.

**Next:** Close - feature implemented, tested, and verified working.

---

# Investigation: Empty Investigation Templates from Agents Dying Early

**Question:** Why do empty investigation template files accumulate and what should be done about them?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** og-debug-empty-investigation-templates-03jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Spawn context instructs agents to create investigation files immediately

**Evidence:** The SPAWN_CONTEXT.md template at `pkg/spawn/context.go:131` contains:
```
2. **SET UP investigation file:** Run `kb create investigation {{.InvestigationSlug}}` to create from template
```

**Source:** `pkg/spawn/context.go:131`

**Significance:** This design means any agent that starts (and creates the file) but dies before doing meaningful work leaves an empty template file behind. This is intentional - early file creation signals that an agent has started working.

---

### Finding 2: Significant number of empty investigation files existed

**Evidence:** Scan found 28 files containing `[Brief, descriptive title]` placeholder text out of 471 total investigation files (~6% abandonment rate).

**Source:** `grep -rl "Brief, descriptive title" .kb/investigations --include="*.md" | wc -l`

**Significance:** While not a huge percentage, this creates noise in the knowledge base. Empty files appear in searches, clutter listings, and waste cognitive overhead when scanning for real investigations.

---

### Finding 3: Existing `orch clean` command handles similar cleanup

**Evidence:** `orch clean` already cleans:
- tmux windows for completed agents (`--windows`)
- Phantom tmux windows (`--phantoms`) 
- Orphaned OpenCode disk sessions (`--verify-opencode`)

**Source:** `cmd/orch/main.go:1438-1473`

**Significance:** The cleanup pattern already exists. Adding `--investigations` flag maintains consistency with the existing approach.

---

## Synthesis

**Key Insights:**

1. **Early file creation is a feature** - It signals agent activity and provides a place to track progress. Changing to delayed creation would lose this visibility.

2. **Archive, don't delete** - Empty files might still contain partial insights (D.E.K.N. summary, metadata). Archiving preserves them while cleaning up active listings.

3. **Heuristic detection works** - Checking for multiple placeholder patterns (e.g., `[Brief, descriptive title]`) reliably identifies unfilled templates with minimal false positives.

**Answer to Investigation Question:**

Empty investigation files accumulate because agents are instructed to create them early but may die before filling them. The solution is a cleanup command (`orch clean --investigations`) that archives files still containing template placeholders. This was implemented and tested successfully.

---

## Structured Uncertainty

**What's tested:**

- ✅ Placeholder detection identifies empty files (verified: 22 files matched heuristics)
- ✅ Archive operation preserves files (verified: files moved to .kb/investigations/archived/)
- ✅ Build compiles without errors (verified: `go build ./cmd/orch/...` succeeded)
- ✅ Existing clean tests pass (verified: `go test ./cmd/orch/... -run Clean`)

**What's untested:**

- ⚠️ False positive rate on edge cases (files that mention placeholder text in documentation)
- ⚠️ Long-term archive growth (may need periodic cleanup of archived files)

**What would change this:**

- If false positive rate is high (>5%), would need stricter heuristics
- If archived files are never useful, could switch to deletion

---

## Implementation Recommendations

### Recommended Approach ⭐

**Archive empty investigations** - Added `--investigations` flag to `orch clean` that moves empty template files to `.kb/investigations/archived/`.

**Why this approach:**
- Non-invasive to existing spawn workflow
- Preserves files for potential future analysis
- Consistent with existing `orch clean` patterns
- Dry-run mode for safety

**Trade-offs accepted:**
- Requires periodic manual cleanup (vs automatic prevention)
- Archived files may accumulate (acceptable given low volume)

**Implementation sequence:**
1. Add `--investigations` flag to clean command (done)
2. Implement `isEmptyInvestigation()` heuristic checker (done)
3. Implement `archiveEmptyInvestigations()` mover (done)
4. Test with dry-run then actual archive (done)

### Alternative Approaches Considered

**Option B: Delayed file creation**
- **Pros:** Prevents empty files from existing
- **Cons:** Loses visibility into agent startup, changes spawn flow significantly
- **When to use instead:** If empty file volume becomes unmanageable

**Option C: Delete instead of archive**
- **Pros:** Simpler, no archived folder to manage
- **Cons:** Loses any partial information in files
- **When to use instead:** If archived files prove to have zero value

**Rationale for recommendation:** Archiving is safest first approach. Can always upgrade to deletion if archiving proves to add no value.

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template, shows early file creation instruction
- `cmd/orch/main.go` - Existing clean command structure

**Commands Run:**
```bash
# Count total investigation files
find .kb/investigations -type f -name "*.md" | wc -l
# Result: 471

# Find files with placeholder content
grep -rl "Brief, descriptive title" .kb/investigations --include="*.md" | wc -l
# Result: 28

# Test dry-run
./orch clean --investigations --dry-run

# Execute archive
./orch clean --investigations
# Result: Archived 22 empty investigation files
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-debug-empty-investigation-templates-03jan/`

---

## Investigation History

**2026-01-03 20:10:** Investigation started
- Initial question: Why do empty investigation templates accumulate?
- Context: Task mentioned "cleanup command or delayed file creation" as options

**2026-01-03 20:15:** Root cause identified
- Agents create files early but may die before filling content
- 28 files (6% of 471) contained placeholder text

**2026-01-03 20:25:** Solution implemented
- Added `--investigations` flag to `orch clean`
- Implemented archive functionality to `.kb/investigations/archived/`

**2026-01-03 20:30:** Investigation completed
- Status: Complete
- Key outcome: Implemented and tested `orch clean --investigations` that archives empty template files
