<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Post-synthesis archival should move synthesized investigations to `.kb/investigations/synthesized/{guide-name}/` subdirectories, preserving provenance while removing them from synthesis detection.

**Evidence:** 667 investigations currently exist; 59 dashboard investigations remain despite dashboard.md guide synthesis; existing `archived/` pattern handles empty/test files; synthesis detection correctly excludes topics with guides but can't distinguish synthesized-from vs merely-related investigations.

**Knowledge:** The key insight is that archival is not about deletion or marking - it's about creating a clear provenance chain between source investigations and their synthesis (guide). Moving to `synthesized/{guide-name}/` makes the relationship explicit and discoverable.

**Next:** Implement `kb archive --synthesized-into {guide}` command that moves investigations to subdirectory and optionally adds reference section to the guide.

**Promote to Decision:** recommend-yes - This establishes the investigation lifecycle pattern (active → complete → synthesized → archived) that will govern knowledge management across projects.

---

# Investigation: Post-Synthesis Investigation Archival Workflow

**Question:** How should investigations be archived after their knowledge has been synthesized into a Guide, such that synthesis opportunities stop recurring while preserving provenance?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Implement feat-042 (kb archive command) and feat-043 (meta-topic exclusions)
**Status:** Complete

---

## Findings

### Finding 1: Synthesis Detection Works but Investigations Accumulate

**Evidence:** 
- 667 total investigations in `.kb/investigations/`
- 59 investigations contain "dashboard" in filename
- `dashboard.md` guide exists (synthesized Jan 6-7)
- Synthesis detection correctly shows 0 "dashboard" opportunities (guide exists)
- Yet all 59 dashboard investigations remain in place

**Source:** 
- `find .kb/investigations -name "*.md" | wc -l` → 667
- `ls .kb/investigations | grep -i dashboard | wc -l` → 59
- `orch status` output shows synthesis detection excluding "dashboard" topic

**Significance:** The synthesis detection system works correctly (topics with guides are excluded), but the source investigations that were synthesized are never cleaned up. This leads to directory bloat and makes it harder to find recent, relevant investigations.

---

### Finding 2: Existing Archival Pattern Already in Use

**Evidence:** 
- `.kb/investigations/archived/` exists with 40 files
- `orch clean` command has `archiveEmptyInvestigations()` function at `cmd/orch/clean_cmd.go:752`
- Archives empty/template investigations that were never filled out
- Uses move-to-subdirectory pattern (not header markers, not deletion)

**Source:** 
- `ls .kb/investigations/archived/ | wc -l` → 40
- Code review of `clean_cmd.go` lines 750-845

**Significance:** There's already precedent for the subdirectory archival pattern. The empty investigation archival is a different concern (unfilled templates) but validates the approach.

---

### Finding 3: Meta-Topics Pollute Synthesis Detection

**Evidence:** 
- 35 investigations contain "investigation" in filename
- These are about the investigation system itself (meta-topic)
- Synthesis detection shows "35 investigations on 'investigation' without synthesis"
- Creating a guide called "investigation.md" would be confusing (too meta)

**Source:** 
- `orch status` output showing synthesis opportunities
- `ls .kb/investigations | grep -i investigation | head -20` showing files like:
  - `2025-12-22-inv-how-do-investigation-files-become-stale.md`
  - `2026-01-03-inv-empty-investigation-templates-agents-dying.md`

**Significance:** Some keywords are meta-topics about the knowledge system itself, not domain topics. These should be excluded from synthesis detection to reduce noise.

---

### Finding 4: Status Field Already Tracks Completion

**Evidence:**
- 394 investigations have `Status: Complete` in their headers
- Investigation template includes `**Status:** [In Progress/Complete/Paused]`
- Status field could theoretically be extended to include "Synthesized"

**Source:**
- `grep -l "Status:.*Complete" .kb/investigations/2025-12*.md | wc -l` → 394

**Significance:** Header-based status tracking exists but is limited - it requires parsing file content rather than using filesystem structure. Adding "Synthesized" status wouldn't help synthesis detection (which only reads filenames/paths).

---

## Synthesis

**Key Insights:**

1. **Archival is about provenance, not deletion** - The goal isn't to delete synthesized investigations or hide them, but to create a clear link between source investigations and their synthesis artifact (the guide). Moving to `synthesized/{guide-name}/` makes this relationship explicit.

2. **Subdirectory pattern is already validated** - The existing `archived/` pattern for empty investigations proves this approach works. It's discoverable, reversible, and keeps related files together.

3. **Meta-topics need explicit exclusion** - Keywords like "investigation", "synthesis", "artifact" are about the knowledge system itself. They shouldn't trigger synthesis opportunities because they're not domain topics.

4. **Gate on synthesis, not on cleanup** - The archival should happen as part of the synthesis workflow (when creating a guide), not as a separate cleanup task. This ensures the relationship is captured while context is fresh.

**Answer to Investigation Question:**

Post-synthesis investigation archival should use a **subdirectory pattern with guide-name organization**: `.kb/investigations/synthesized/{guide-name}/`. This approach:
- Makes provenance explicit (investigations in `synthesized/dashboard/` were synthesized into `dashboard.md`)
- Is already excluded by synthesis detection (which skips subdirectories)
- Preserves full file content for reference
- Is reversible (can move back if guide is deleted)
- Follows existing `archived/` precedent

Meta-topics like "investigation", "synthesis", "artifact" should be added to an exclusion list in `TopicKeywords`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Synthesis detection excludes topics with guides (verified: dashboard not in opportunities)
- ✅ Subdirectory pattern works for archival (verified: `archived/` has 40 files)
- ✅ 667 investigations exist, creating directory bloat (verified: find command)

**What's untested:**

- ⚠️ Performance impact of 600+ subdirectory files (not measured)
- ⚠️ Whether `kb context` searches synthesized/ subdirectory (needs verification)
- ⚠️ Whether guides should reference source investigations (design choice)

**What would change this:**

- If investigations need to remain searchable via `kb context`, subdirectory exclusion logic would need updating
- If the guide update process is too manual, we'd need tighter kb CLI integration

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Subdirectory archival with guide reference** - Move synthesized investigations to `.kb/investigations/synthesized/{guide-name}/` and optionally add a "Sources" section to the guide.

**Why this approach:**
- Creates explicit provenance chain (investigations → guide)
- Follows existing `archived/` pattern (validated approach)
- Automatically excluded by synthesis detection (already ignores subdirectories)
- Preserves original content for deep reference

**Trade-offs accepted:**
- Creates additional subdirectory structure
- Requires manual invocation (not automatic post-synthesis)

**Implementation sequence:**
1. Add `synthesized/` directory handling to synthesis_opportunities.go (exclude from scanning)
2. Create `kb archive --synthesized-into {guide}` command
3. Update guides to optionally include "Sources" section listing archived investigations

### Alternative Approaches Considered

**Option B: Header marker (status: Synthesized)**
- **Pros:** No file movement, simpler
- **Cons:** Synthesis detection would need to parse file content (expensive); status field exists but isn't used for detection
- **When to use instead:** If directory structure becomes problematic

**Option C: Deletion after synthesis**
- **Pros:** Simplest, cleanest directories
- **Cons:** Loses provenance; investigations may have value beyond the synthesis
- **When to use instead:** For truly redundant investigations (test runs, duplicates)

**Option D: Symlink-based organization**
- **Pros:** Keeps files in place, creates virtual organization
- **Cons:** Symlinks are fragile, not well-supported in all tools, git handling is complex
- **When to use instead:** Never recommended for this use case

**Rationale for recommendation:** Option A (subdirectory) is the only approach that creates explicit provenance while being automatically detected by the existing synthesis system. The `archived/` pattern proves it works.

---

### Implementation Details

**What to implement first:**
1. Add meta-topics to exclusion list in `synthesis_opportunities.go`:
   ```go
   var MetaTopicExclusions = []string{
       "investigation",
       "synthesis", 
       "artifact",
       "skill",
   }
   ```
2. Create `synthesized/` directory structure
3. Build `kb archive` command

**Things to watch out for:**
- ⚠️ `kb context` may need updating to include/exclude synthesized/ directory based on search intent
- ⚠️ Moving files mid-git-history creates potential confusion (consider using --follow in git log)
- ⚠️ Guide "Sources" section should be auto-generated, not manually maintained

**Areas needing further investigation:**
- Should the archival happen during `kb reflect` workflow?
- What's the right threshold for archiving (all related investigations, or only directly-used ones)?
- Should there be a "promote back to active" path?

**Success criteria:**
- ✅ Synthesis opportunities stop showing topics that have been synthesized
- ✅ Archived investigations are discoverable via guide reference
- ✅ `kb context` can optionally search synthesized/ when needed
- ✅ `.kb/investigations/` directory contains only active/unsynthesized work

---

## References

**Files Examined:**
- `pkg/verify/synthesis_opportunities.go` - Current synthesis detection logic
- `cmd/orch/clean_cmd.go` - Existing archival pattern for empty investigations
- `.kb/guides/dashboard.md` - Example synthesized guide (58 source investigations)
- `.kb/investigations/archived/` - Existing archival directory (40 files)

**Commands Run:**
```bash
# Count total investigations
find .kb/investigations -name "*.md" | wc -l  # 667

# Count dashboard investigations
ls .kb/investigations | grep -i dashboard | wc -l  # 59

# Check existing archived
ls .kb/investigations/archived/ | wc -l  # 40

# Check synthesis opportunities
./orch-go status  # Shows 35 "investigation" topic, 0 "dashboard"
```

**Related Artifacts:**
- **Guide:** `.kb/guides/dashboard.md` - Example of synthesis target
- **Investigation:** `2026-01-06-inv-dashboard-auto-discover-investigation-synthesis.md` - Related investigation
- **Decision:** None yet - this investigation proposes creating one

---

## Investigation History

**2026-01-07 22:24:** Investigation started
- Initial question: How to archive investigations after synthesis to stop recurring opportunities?
- Context: 667 investigations, synthesis updates guides but doesn't archive sources

**2026-01-07 22:45:** Findings gathered
- Confirmed synthesis detection works (dashboard excluded)
- Found existing archived/ pattern with 40 files
- Identified meta-topic pollution issue (35 "investigation" investigations)

**2026-01-07 23:00:** Investigation completed
- Status: Complete
- Key outcome: Recommend subdirectory archival to `synthesized/{guide-name}/` with optional guide Sources section
