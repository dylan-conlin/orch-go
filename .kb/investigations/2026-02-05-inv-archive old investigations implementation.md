<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** kb tooling uses recursive directory search, so archiving to `.kb/investigations/archive/` preserves full discoverability while removing files from active pool.

**Evidence:** SearchArtifacts in kb-cli/cmd/kb/search.go:256 uses filepath.Walk which recursively searches subdirectories; tested with existing synthesized/ archives which remain searchable.

**Knowledge:** Age-based archival should use filename date prefix (YYYY-MM-DD) for age calculation, not mtime; implement as `kb archive --older-than 60d` command extending existing archive infrastructure.

**Next:** Implement age-based archival in kb-cli by adding --older-than flag and age parsing logic; currently 0 investigations >60 days old so immediate impact is zero but feature needed for future.

**Authority:** implementation - Extends existing kb archive pattern with new flag, no architectural changes, follows established archive subdirectory convention.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Archive old investigations implementation

**Question:** How should we archive old investigations (>60 days) while preserving discoverability via kb tooling?

**Started:** 2026-02-05 23:18
**Updated:** 2026-02-05 23:18
**Owner:** Worker agent (orch-go-21305)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->

**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| ------------- | ------------ | -------- | --------- |
| N/A           | -            | -        | -         |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: kb search and kb context use recursive discovery

**Evidence:** `SearchArtifacts` function in kb-cli uses `filepath.Walk(dir, ...)` which recursively walks through all subdirectories under `.kb/investigations/`, `.kb/decisions/`, and `.kb/models/`. This means files in subdirectories like `.kb/investigations/archive/` or `.kb/investigations/synthesized/` are still discovered.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/search.go:256` - Line 254 comment explicitly says "Search each directory (recursively for subdirectories)"

**Significance:** Archiving investigations to a subdirectory (e.g., `.kb/investigations/archive/`) will NOT break discoverability. They remain searchable via `kb search` and `kb context`.

---

### Finding 2: Current kb archive command is synthesis-based, not age-based

**Evidence:** The existing `kb archive` command (in `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/archive.go`) archives investigations that have been synthesized into a guide, moving them to `.kb/investigations/synthesized/{guide-name}/`. It matches by topic keyword in filename, not by age.

**Source:** `kb archive --help` output and `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/archive.go:18-30`

**Significance:** We need a NEW archival mechanism for age-based archival. The existing `kb archive --synthesized-into` serves a different purpose (lineage preservation after synthesis).

---

### Finding 3: Currently 0 investigations are >60 days old

**Evidence:** Running `find .kb/investigations -type f -name "*.md" -mtime +60` returns 0 results. Earliest investigation by filename date is `2025-12-19` (Dec 19, 2025). Today is Feb 5, 2026, so oldest investigations are only ~48 days old. Total investigations: 767.

**Source:** `find .kb/investigations -type f -name "*.md" | wc -l` → 767; `ls .kb/investigations/ | head -20` shows earliest dates of Dec 19-20, 2025

**Significance:** The immediate impact of implementing age-based archival will be zero (no files to archive). However, the feature will be needed in the future as investigations age beyond 60 days.

---

### Finding 4: Investigation naming convention includes date prefix

**Evidence:** All investigation files follow the pattern `YYYY-MM-DD-{type}-{slug}.md` (e.g., `2025-12-19-inv-cli-orch-spawn-command.md`). This date prefix can be parsed to determine investigation age, which is more reliable than file modification time (mtime).

**Source:** `ls .kb/investigations/ | head -20` shows consistent YYYY-MM-DD prefix pattern

**Significance:** We should parse the filename date prefix (first 10 characters) rather than relying on `mtime` for age calculation. This is more reliable because mtime can change during git operations or file moves.

---

### Finding 5: Existing archive preserves only directory structure, not search optimization

**Evidence:** The synthesis-based archive moves files to `.kb/investigations/synthesized/{guide-name}/` subdirectories. The `findMatchingInvestigations` function (line 172-206 in archive.go) reads the investigations directory NON-recursively (skips subdirectories), meaning synthesized investigations won't be re-archived.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/archive.go:172-206` - `os.ReadDir(investigationsDir)` with `entry.IsDir()` skip logic

**Significance:** Archiving to a subdirectory removes investigations from the "active pool" for synthesis detection, but they remain discoverable via search. This is the correct behavior for age-based archival as well.

---

## Synthesis

**Key Insights:**

1. **Subdirectory archival preserves discoverability** - kb tooling uses recursive directory traversal (`filepath.Walk`), so moving investigations to subdirectories like `.kb/investigations/archive/` maintains full searchability via `kb search` and `kb context`. This is already proven by the existing synthesis-based archive mechanism.

2. **Filename date prefix is the source of truth for age** - All investigations have `YYYY-MM-DD` date prefixes in their filenames, which is more reliable than `mtime` for age calculation. Parse the first 10 characters of the filename to determine investigation age.

3. **Age-based archival is orthogonal to synthesis-based archival** - The existing `kb archive --synthesized-into` serves lineage preservation (connecting investigations to guides). Age-based archival serves decluttering (removing old investigations from the active pool). These are separate concerns and should remain separate commands.

**Answer to Investigation Question:**

Archive old investigations (>60 days) by moving them to `.kb/investigations/archive/` subdirectory. Parse the filename date prefix (`YYYY-MM-DD`) to determine age. This preserves discoverability via kb tooling (which searches recursively) while removing them from the "active pool" (synthesis detection and directory listings scan only top-level files). Implement as a new `kb archive --older-than 60d` command or similar, separate from the existing synthesis-based archive.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation                              | Authority                                  | Rationale                                                                                                 |
| ------------------------------------------- | ------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**

- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"

- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Add `kb archive --older-than <duration>` command** - Extend kb-cli archive.go to support age-based archival alongside existing synthesis-based archival.

**Why this approach:**

- Reuses existing archive infrastructure (move logic, directory creation)
- Keeps age-based and synthesis-based archival separate (different flags, different use cases)
- Archive destination `.kb/investigations/archive/` preserves discoverability via recursive search
- Filename date parsing is reliable and already validated by investigation naming convention

**Trade-offs accepted:**

- Won't archive investigations immediately (0 files >60 days old currently)
- Manual invocation required (not automated) - could add to kb reflect later if needed
- Only supports investigations initially (not decisions/models), though could extend later

**Implementation sequence:**

1. Add age calculation function that parses filename date prefix (YYYY-MM-DD)
2. Add `--older-than` flag to kb archive command with duration parsing (e.g., "60d")
3. Filter investigations by age in ArchiveInvestigations function
4. Move matching files to `.kb/investigations/archive/` subdirectory
5. Add tests to verify age calculation and archival behavior

### Alternative Approaches Considered

**Option B: [Alternative approach]**

- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**

- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**

- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**

- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**

- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**

- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**

- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**

```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**

- [Link or reference] - [What it is and relevance]

**Related Artifacts:**

- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started

- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]

- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed

- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
