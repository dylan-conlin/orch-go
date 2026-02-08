<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `kb reflect --type refine` to detect kn entries that refine existing principles in ~/.kb/principles.md.

**Evidence:** Successfully detects entries like kn-15e53a (refines Pressure Over Compensation), kn-c12998 (refines Premise Before Solution). Tested with real kn data - found 10+ refinement candidates in orch-go.

**Knowledge:** Principle refinement detection uses keyword matching (2+ terms required) against known principle keywords. Works with project-local .kn/ entries.

**Next:** None - implementation complete. Consider adding tests for refine functionality in kb-cli.

---

# Investigation: Principle Refinement Detection - Surface kn Entries

**Question:** How can we detect kn entries that refine or add nuance to existing principles?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kn Entries Contain Principle References

**Evidence:** Examining kn entries in orch-go/.kn/entries.jsonl shows many entries reference principles by name or use principle terminology. Examples:
- `kn-147574`: "Reactive approach applies to reversible failures..." - mentions "Pressure Over Compensation"
- `kn-c3f086`: "Detection accelerates pressure, prevention relieves it..." - uses pressure/compensation terminology
- `kn-c12998`: "Ask 'should we' before 'how do we'..." - relates to "Premise Before Solution"

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl`

**Significance:** kn entries often refine principles by adding edge cases, clarifications, or implementation details that aren't captured in the original principle.

### Finding 2: Principles Have Distinctive Keywords

**Evidence:** Each principle in ~/.kb/principles.md has unique terminology:
- "Session Amnesia": amnesia, session, memory, externalize, persist, discoverable
- "Pressure Over Compensation": pressure, compensation, failure, gap, system learns
- "Evidence Hierarchy": evidence, hierarchy, primary, secondary, code is truth
- "Gate Over Remind": gate, remind, enforce, block, cannot proceed

**Source:** `~/.kb/principles.md`

**Significance:** Keyword matching (requiring 2+ matches) can reliably identify kn entries that refine specific principles.

### Finding 3: Implementation Requires Cross-Repository Data

**Evidence:** Principles are stored globally (~/.kb/principles.md) while kn entries are per-project (.kn/entries.jsonl). Detection needs to:
1. Read principles from global location
2. Check project-local kn entries first, fallback to global
3. Match entries against principle keywords

**Source:** Code analysis of kb reflect and kn storage patterns

**Significance:** The implementation must handle multiple data sources and gracefully degrade when either is missing.

---

## Synthesis

**Key Insights:**

1. **Keyword Matching Works Well** - Requiring 2+ keyword matches filters out false positives while catching real refinements. Single-keyword matches would be too noisy.

2. **Recent Entries Matter Most** - Sorting by creation date (newest first) surfaces the most recent refinements, which are more likely to be actionable.

3. **Decision and Constraint Types Are Most Relevant** - Questions and attempts rarely refine principles. Only considering "decision" and "constraint" types improves signal quality.

**Answer to Investigation Question:**

Principle refinement detection works by:
1. Extracting principle names and keywords from ~/.kb/principles.md
2. Reading kn entries from project .kn/entries.jsonl
3. For each active decision/constraint entry, checking if content/reason contains 2+ keywords from any principle
4. Surfacing matches with the principle name and matched terms

The implementation was added to kb-cli (`kb reflect --type refine`) and integrated into orch-go daemon reflect.

---

## Structured Uncertainty

**What's tested:**

- ✅ Detection finds entries matching "Pressure Over Compensation" (verified with real data)
- ✅ Detection finds entries matching "Session Amnesia" (verified with real data)
- ✅ JSON output format works correctly
- ✅ Existing reflect tests still pass

**What's untested:**

- ⚠️ Global reflect mode (--global flag) with refine type
- ⚠️ Edge cases with principles containing overlapping keywords
- ⚠️ Performance with very large kn files (>1000 entries)

**What would change this:**

- If principles are restructured to not use distinctive terminology, keyword matching would fail
- If kn entry format changes, JSON parsing would need updates

---

## Implementation Recommendations

### Recommended Approach ⭐

**Keyword-based matching with principle extraction** - Parse principles.md to extract named principles and their key terms, then match kn entries against these terms.

**Why this approach:**
- Leverages existing structure of principles.md
- Simple to implement and maintain
- Works with current data formats

**Trade-offs accepted:**
- Hardcoded keyword lists require manual updates when principles change
- 2-keyword minimum may miss some legitimate refinements

**Implementation sequence:**
1. Add RefineCandidate type to reflect.go
2. Implement findRefineCandidates function
3. Integrate with existing Reflect() function
4. Update text and JSON output formatters

### Alternative Approaches Considered

**Option B: Semantic/embedding-based matching**
- **Pros:** More robust to paraphrasing
- **Cons:** Requires embedding model, complex infrastructure
- **When to use instead:** If keyword matching produces too many false positives

**Option C: Manual tagging of kn entries**
- **Pros:** High precision
- **Cons:** Requires user discipline, not automatic
- **When to use instead:** If detection quality is critical

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Source of principle definitions
- `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` - kn entries data

**Commands Run:**
```bash
# Test the implementation
kb reflect --type refine --limit 10
kb reflect --type refine --format json
```

**Related Artifacts:**
- **Decision:** `~/.kb/decisions/2025-12-25-pressure-over-compensation.md` - Example of a principle

---

## Investigation History

**2025-12-26 18:00:** Investigation started
- Initial question: How to detect kn entries that refine principles?
- Context: Need to surface principle refinements for inclusion in principles.md

**2025-12-26 18:30:** Implementation complete
- Added `kb reflect --type refine` to kb-cli
- Updated orch-go daemon/reflect.go to handle new type
- Tested with real data from orch-go

**2025-12-26 18:45:** Investigation completed
- Status: Complete
- Key outcome: Principle refinement detection implemented and working
