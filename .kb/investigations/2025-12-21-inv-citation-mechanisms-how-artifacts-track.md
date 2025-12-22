---
linked_issues:
  - orch-go-ws4z.7
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Citation mechanisms are already minimal and sufficient: content parsing via grep for discovery, structured `## Related Artifacts` sections for explicit links, and `linked_issues` frontmatter for beads bi-directionality.

**Evidence:** Tested inbound link discovery: `rg "artifact-name" .kb/` finds 2 citers in <100ms. Found 51 files contain artifact references (37% of 138 artifacts). Top-cited artifacts have 6 references each.

**Knowledge:** Content-based parsing (grep) is simpler, more flexible, and already working than frontmatter links would be. Frontmatter adds maintenance burden without discovery benefit since kb search already indexes content.

**Next:** Add `kb cited-by <artifact>` command as thin wrapper around grep. No new data structure needed.

**Confidence:** High (85%) - Tested actual discovery mechanisms; uncertainty around whether high citation count truly signals "load-bearing."

---

# Investigation: Citation Mechanisms - How Artifacts Track References

**Question:** What's the minimal citation mechanism (frontmatter links vs content parsing)? How to track inbound links ('what cites this decision')? How to surface load-bearing artifacts (high citation count = foundational)?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Three existing citation mechanisms already exist

**Evidence:** The system already has three citation mechanisms:

1. **`linked_issues` frontmatter** - Used by `kb link` to create bidirectional links between kb artifacts and beads issues. Example from `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md`:
   ```yaml
   ---
   linked_issues:
     - orch-go-4kwt.6
   ---
   ```

2. **`## Related Artifacts` section** - Template-defined section at end of investigation files for explicit artifact-to-artifact references:
   ```markdown
   **Related Artifacts:**
   - **Decision:** [Path to related decision document] - [How it relates]
   - **Investigation:** [Path to related investigation] - [How it relates]
   ```

3. **Inline content references** - Artifacts reference each other in body text using relative paths like `.kb/investigations/2025-12-21-inv-design-single-agent-review-command.md`

**Source:** 
- `kb link --help`
- `/Users/dylanconlin/.kb/templates/INVESTIGATION.md:214-217`
- `rg "\.kb/(investigations|decisions)/" .kb/ | head -30`

**Significance:** No new mechanism needed for basic citation - the question is about discovery (inbound links) and surfacing importance (citation count).

---

### Finding 2: Inbound link discovery works via content parsing (grep)

**Evidence:** Tested finding "what cites this artifact":

```bash
# Find inbound citations to model-handling investigation
ARTIFACT="2025-12-21-inv-model-handling-conflicts-between-orch"
rg -l "$ARTIFACT" .kb/ | grep -v "$ARTIFACT.md"
```

Result: Found 2 citing files in <100ms:
- `.kb/investigations/2025-12-21-inv-fix-buildspawncommand-pass-model-flag.md`
- `.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md`

**Source:** Bash command output (tested live)

**Significance:** Content-based inbound link discovery is:
- Fast (<100ms for 138 artifacts)
- Requires no maintenance (automatic as artifacts are written)
- Finds all references (explicit Related sections + inline mentions)
- Simple implementation (`grep`/`rg` wrapper)

---

### Finding 3: 51 files contain artifact references (37% of artifacts cite others)

**Evidence:** Scoped the citation surface area:

```bash
rg -c "\.kb/(investigations|decisions)/[^)\"' >]+\.md" .kb/ | wc -l
# Result: 51
```

Of ~138 total investigation/decision files, 51 contain explicit paths to other artifacts.

**Source:** `rg` count on `.kb/` directory

**Significance:** Citations are common but not universal. Most artifacts are standalone investigations that don't build on prior work.

---

### Finding 4: Load-bearing artifacts identifiable by citation count

**Evidence:** Identified most-cited artifacts via content grep:

```bash
grep -roh "2025-12-[0-9][0-9]-[a-z0-9-]*\.md" .kb/ | sort | uniq -c | sort -rn | head -15
```

Top 5 most-cited:
| Citations | Artifact |
|-----------|----------|
| 6 | `2025-12-21-inv-model-handling-conflicts-between-orch.md` |
| 6 | `2025-12-20-inv-test-concurrent-spawn-capability.md` |
| 6 | `2025-12-18-sdk-based-agent-management.md` |
| 5 | `2025-12-21-inv-add-tmux-fallback-orch-status.md` |
| 5 | `2025-12-20-inv-test.md` |

**Source:** Grep with frequency counting

**Significance:** High citation count correlates with artifacts that established patterns (SDK management, concurrent spawn testing) or that addressed widely-felt problems (model conflicts). These are "load-bearing" - changing them would require updates to citers.

---

### Finding 5: Frontmatter links add maintenance burden without discovery benefit

**Evidence:** Compared two approaches for artifact-to-artifact links:

**Option A: Frontmatter links (like `linked_issues`)**
```yaml
---
cites:
  - .kb/decisions/2025-12-18-sdk-based-agent-management.md
---
```
- Pros: Machine-parseable, structured
- Cons: Must be maintained manually, easy to forget, duplicates inline references, adds YAML complexity

**Option B: Content parsing (status quo)**
- Pros: Zero maintenance, already captures all references, uses existing tools (`kb search`, `rg`)
- Cons: Slightly slower for large corpora (not an issue at current scale)

**Source:** Analysis of existing `kb link` behavior and template structure

**Significance:** Since `kb search` already indexes content, frontmatter links for artifact-to-artifact references would duplicate without adding value. The `linked_issues` frontmatter is special because it creates links TO beads (external system), not within kb.

---

## Synthesis

**Key Insights:**

1. **Content parsing wins for simplicity** - A `kb cited-by <artifact>` command as a thin grep wrapper is the minimal solution. No new data structures, no maintenance burden, instant results.

2. **Load-bearing artifacts are discoverable now** - Citation count via grep already works. A `kb top-cited` command could formalize this if needed, but it's a one-liner today.

3. **Frontmatter is for cross-system links** - The `linked_issues` pattern exists because beads is a separate system. Artifact-to-artifact links within kb don't need this pattern.

4. **37% citation rate is healthy** - Not every artifact needs to cite others. Standalone investigations are valid.

**Answer to Investigation Question:**

**Minimal citation mechanism:** Content parsing via grep. The template already has `## Related Artifacts` section for explicit links; these are parsed via content search. No new frontmatter needed.

**Inbound link tracking:** `rg "<artifact-filename>" .kb/ | grep -v "<artifact-filename>.md"` - A `kb cited-by` command could wrap this.

**Surfacing load-bearing artifacts:** `grep -roh "2025-12-[0-9][0-9]-*.md" .kb/ | sort | uniq -c | sort -rn | head -10` - A `kb top-cited` command could wrap this.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Content parsing is proven technology (grep/rg). The patterns are already in use. The only uncertainty is whether "high citation count" truly indicates load-bearing importance or just reflects recency/test artifacts.

**What's certain:**

- ✅ Content parsing finds all references (tested with known artifacts)
- ✅ Inbound link discovery is fast (<100ms at current scale)
- ✅ Existing template structure captures outbound links
- ✅ `linked_issues` frontmatter is for beads integration, not kb-internal links

**What's uncertain:**

- ⚠️ Whether high citation count correlates with importance vs. just recency
- ⚠️ Whether scale will become a problem (138 files is small)
- ⚠️ Whether agents actually use Related Artifacts section consistently

**What would increase confidence to 95%:**

- Validate that highly-cited artifacts are actually foundational (not just test noise)
- Monitor citation patterns over 30 days to see if patterns hold
- Survey actual agent usage of Related Artifacts section

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Content-based citation discovery via thin CLI wrappers** - Add two commands to kb:

1. `kb cited-by <artifact>` - Wrapper around `rg "<basename>" .kb/`
2. `kb top-cited [--limit N]` - Wrapper around grep frequency count

**Why this approach:**
- Zero new data structures
- Zero maintenance burden (no frontmatter to keep in sync)
- Uses battle-tested tools (grep/rg)
- Immediately useful without migration

**Trade-offs accepted:**
- Slight overhead on each query (re-scans files)
- Not as fast as index-based search at scale (acceptable at current 138 files)

**Implementation sequence:**
1. Add `kb cited-by` command (10 lines of shell)
2. Add `kb top-cited` command (10 lines of shell)
3. Consider: Should these be in kb CLI or just documented as patterns?

### Alternative Approaches Considered

**Option B: Frontmatter citation links**
- **Pros:** Structured, fast lookup
- **Cons:** Maintenance burden, duplicates content references, adds complexity
- **When to use instead:** If corpus grows to 10,000+ artifacts and grep becomes slow

**Option C: Citation index (SQLite/JSON)**
- **Pros:** Fast queries, complex graph analysis possible
- **Cons:** Requires build step, sync issues, adds infrastructure
- **When to use instead:** If we need citation graph visualization or complex queries

**Rationale for recommendation:** Current scale (138 files) doesn't justify infrastructure. Content parsing is "good enough" and eliminates maintenance burden.

---

## References

**Files Examined:**
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Context on artifact types
- `.kb/decisions/2025-12-21-single-agent-review-command.md` - Example of Related section usage
- `~/.kb/templates/INVESTIGATION.md:214-217` - Template Related Artifacts section
- `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md` - Example of linked_issues usage

**Commands Run:**
```bash
# Find inbound citations
rg -l "2025-12-21-inv-model-handling-conflicts-between-orch" .kb/

# Count files with artifact references
rg -c "\.kb/(investigations|decisions)/[^)\"' >]+\.md" .kb/ | wc -l

# Find most-cited artifacts
grep -roh "2025-12-[0-9][0-9]-[a-z0-9-]*\.md" .kb/ | sort | uniq -c | sort -rn | head -15

# Test kb link help
kb link --help
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Defines 5+3 artifact taxonomy
- **Investigation:** `.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md` - Documents kb link mechanism

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-21 ~14:00:** Investigation started
- Initial question: What's the minimal citation mechanism? How to track inbound links? How to surface load-bearing artifacts?
- Context: Part of amnesia-resilient artifact architecture epic (orch-go-ws4z)

**2025-12-21 ~14:30:** Core findings complete
- Discovered three existing mechanisms (linked_issues, Related section, inline refs)
- Tested inbound link discovery via grep (<100ms)
- Found 51/138 artifacts contain cross-references

**2025-12-21 ~14:45:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Content parsing is the minimal mechanism; no new infrastructure needed
