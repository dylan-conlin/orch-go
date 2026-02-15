<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Guide already complete (verified 2026-01-08); archived 14 investigations to `.kb/investigations/synthesized/code-extraction-patterns/`; discovered kb reflect bug scanning archived/synthesized directories.

**Evidence:** Compared guide References (13 investigations listed) against kb reflect output; moved files and verified they exist in synthesized/; kb reflect still reports 13 despite archival.

**Knowledge:** kb reflect's synthesis detection scans archived/ and synthesized/ directories, causing false positives for already-processed investigation clusters.

**Next:** File bug report for kb reflect to exclude synthesized/ and archived/ from synthesis scanning.

**Promote to Decision:** recommend-no - Bug fix, not architectural decision.

---

# Investigation: Synthesize Extract Investigation Cluster (13 Investigations)

**Question:** What patterns emerge from the 13 extraction investigations flagged by kb reflect, and how should they be consolidated?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** og-arch-synthesize-extract-investigation-17jan-be9f
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** `.kb/investigations/synthesized/code-extraction-patterns/2026-01-08-inv-synthesize-extract-investigations-13-synthesis.md`

---

## Findings

### Finding 1: Guide Already Complete and Up-to-Date

**Evidence:** `.kb/guides/code-extraction-patterns.md` (last verified 2026-01-08) contains:
- 5 extraction phases covering Go packages, Svelte components, feature tabs, and TypeScript services
- All 13 code extraction investigations listed in References section
- Line count benchmarks from actual extractions
- Debugging checklist and common problems

**Source:** `.kb/guides/code-extraction-patterns.md:1-339`

**Significance:** No new patterns need to be added to the guide. The synthesis task is about archival, not guide creation.

---

### Finding 2: kb reflect Includes Archived and Synthesized Directories

**Evidence:** After moving 14 investigations to `.kb/investigations/synthesized/code-extraction-patterns/`, kb reflect still reports:
```json
{
  "topic": "extract",
  "count": 13,
  "investigations": [
    "2025-12-26-inv-implement-kb-extract-command-cross.md",  // archived/
    "2026-01-04-inv-extract-clean-cmd-go-main.md",  // archived/
    "2026-01-03-inv-extract-serve-agents-go-serve.md",  // synthesized/
    // ... 10 more from synthesized/
  ]
}
```

**Source:** `kb reflect --type synthesis --format json`

**Significance:** This is a bug - investigations in archived/ and synthesized/ directories should be excluded from synthesis detection since they've already been processed.

---

### Finding 3: 5 "Extract" Investigations Are About Different Topics

**Evidence:** These investigations have "extract" in their names but are NOT about code extraction patterns:
1. `2025-12-22-inv-design-knowledge-system-project-extraction.md` - Cross-project knowledge migration
2. `2025-12-22-inv-add-lineage-headers-extracted-supersedes.md` - Lineage metadata for templates
3. `2025-12-23-inv-implement-constraint-extraction-verification-orch.md` - Skill constraint extraction
4. `2025-12-26-inv-implement-kb-extract-command-cross.md` - kb extract command (archived)
5. `2026-01-17-inv-investigate-failed-extract-session-id.md` - Debugging session ID parsing

**Source:** Read file headers and summaries

**Significance:** These should NOT be included in code-extraction-patterns synthesis. The keyword-based matching in kb reflect creates false positives.

---

### Finding 4: Archival Structure Works Correctly

**Evidence:** Successfully moved 14 investigations:
- 12 code extraction investigations (serve.go, main.go, Svelte, SSE)
- 2 prior synthesis investigations

To: `.kb/investigations/synthesized/code-extraction-patterns/`

Files are accessible and git tracks the moves.

**Source:** `ls .kb/investigations/synthesized/code-extraction-patterns/` (14 files)

**Significance:** The synthesized/ directory structure works; the issue is kb reflect's scanning logic.

---

## Synthesis

**Key Insights:**

1. **Guide was already complete** - Prior synthesis (2026-01-08) did the work correctly. This investigation confirmed completeness rather than adding content.

2. **kb reflect has a scanning bug** - It includes archived/ and synthesized/ directories in synthesis detection, causing false positives for already-processed clusters.

3. **Keyword matching creates false positives** - Investigations with "extract" in their names but about different topics (knowledge extraction, constraint extraction) are incorrectly grouped with code extraction patterns.

**Answer to Investigation Question:**

The 13 extraction investigations don't need additional synthesis - the guide is already complete and up-to-date. The appropriate action was archival, which has been completed (14 files moved to synthesized/). However, kb reflect will continue to flag this cluster until its scanning logic is fixed to exclude synthesized/ and archived/ directories.

---

## Structured Uncertainty

**What's tested:**

- ✅ Guide contains all 13 code extraction patterns (verified: compared References section to kb reflect list)
- ✅ Archival moved files successfully (verified: `ls .kb/investigations/synthesized/code-extraction-patterns/` shows 14 files)
- ✅ kb reflect still detects cluster after archival (verified: `kb reflect --type synthesis` output)

**What's untested:**

- ⚠️ Whether a fix to kb reflect would resolve the false positives
- ⚠️ Whether other investigation clusters have the same archival issue

**What would change this:**

- Finding would be wrong if kb reflect's scanning logic was intended to include synthesized/
- Finding would be incomplete if there are other archival mechanisms that properly exclude files

---

## Implementation Recommendations

### Recommended Approach ⭐

**File bug report for kb reflect** - The command should exclude `.kb/investigations/archived/` and `.kb/investigations/synthesized/` from synthesis scanning.

**Why this approach:**
- Addresses root cause (scanning wrong directories)
- Enables proper archival workflow
- Fixes false positives for other clusters too

**Trade-offs accepted:**
- Requires code change to kb CLI
- Until fixed, kb reflect will continue showing stale clusters

**Implementation sequence:**
1. Create beads issue for kb reflect bug
2. Fix scanning logic to exclude archived/ and synthesized/ directories
3. Verify extract cluster no longer appears in kb reflect output

### Alternative Considered

**Manual verification** - Ignore kb reflect for synthesized clusters, manually verify guide completeness

- **Pros:** No code change needed
- **Cons:** Repeated false alarms, cognitive overhead, doesn't scale
- **When to use:** Only as temporary workaround while bug is fixed

---

## References

**Files Examined:**
- `.kb/guides/code-extraction-patterns.md` - Verified guide completeness
- `.kb/investigations/synthesized/code-extraction-patterns/2026-01-08-inv-synthesize-extract-investigations-13-synthesis.md` - Prior synthesis
- 14 code extraction investigations (now in synthesized/)

**Commands Run:**
```bash
# Check synthesis opportunities
kb reflect --type synthesis --format json

# Archive investigations
mkdir -p .kb/investigations/synthesized/code-extraction-patterns
mv .kb/investigations/*extract*.md .kb/investigations/synthesized/code-extraction-patterns/

# Verify archival
ls .kb/investigations/synthesized/code-extraction-patterns/
```

**Related Artifacts:**
- **Guide:** `.kb/guides/code-extraction-patterns.md` - The authoritative reference
- **Prior Synthesis:** `.kb/investigations/synthesized/code-extraction-patterns/2026-01-08-inv-synthesize-extract-investigations-13-synthesis.md`

---

## Investigation History

**2026-01-17 ~19:25:** Investigation started
- Initial question: Synthesize 13 extraction investigations per kb reflect
- Context: kb reflect flagged synthesis opportunity

**2026-01-17 ~19:30:** Found guide already complete
- Read guide and prior synthesis
- Determined no new content needed

**2026-01-17 ~19:33:** Archived 14 investigations
- Moved to `.kb/investigations/synthesized/code-extraction-patterns/`
- Discovered kb reflect still detects cluster

**2026-01-17 ~19:40:** Identified kb reflect bug
- Bug: scans archived/ and synthesized/ directories
- Filed as constraint to surface

**2026-01-17 ~19:45:** Investigation completed
- Status: Complete
- Key outcome: Guide verified complete; archival done; kb reflect bug identified
