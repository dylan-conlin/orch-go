## Summary (D.E.K.N.)

**Delta:** Spawn context now includes D.E.K.N. Delta (key findings) for up to 3 most recent related investigations, enabling agents to understand prior work and create lineage references.

**Evidence:** Added `extractDeltaFromInvestigation()` function that parses `**Delta:**` lines from investigation files; all 3 new tests pass; spawn context now shows "**Key finding:**" for each investigation.

**Knowledge:** The spawn context already included investigation titles/paths, but agents lacked the content to understand relevance. By including the one-sentence Delta, agents can now see what was discovered and reference prior work appropriately.

**Next:** Close - implementation complete with tests.

---

# Investigation: Spawn Context Include Related Prior

**Question:** How can spawn context include related prior investigations to help agents create lineage references?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** og-feat-spawn-context-include-28dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Spawn context already includes investigation paths, but no content

**Evidence:** The `FormatContextForSpawn()` function in `pkg/spawn/kbcontext.go` lists investigations under "### Related Investigations" with title and path, but no actual content from the investigation files.

**Source:** `pkg/spawn/kbcontext.go:545-561`

**Significance:** Agents see paths like `/path/to/investigation.md` but have no idea what was discovered without reading the full file. This explains the 0 lineage references finding.

---

### Finding 2: D.E.K.N. format provides extractable key findings

**Evidence:** Investigation files use the pattern `**Delta:** <one-sentence key finding>` which is perfect for extraction. This appears in the first 15-20 lines of compliant investigation files.

**Source:** `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md:7`

**Significance:** The Delta line is designed to be a quick handoff summary - exactly what agents need to understand prior work without reading full investigations.

---

### Finding 3: Limiting to 3 investigations prevents context bloat

**Evidence:** With 500+ investigations, including all matches would explode context. Setting `MaxInvestigationsInContext = 3` surfaces the most recent relevant work without excessive token usage.

**Source:** `pkg/spawn/kbcontext.go:52-54`

**Significance:** The most recent investigations are most likely to be relevant. Older investigations may be superseded or less applicable.

---

## Synthesis

**Key Insights:**

1. **Title + Path isn't enough** - Agents need to understand WHAT was discovered, not just WHERE. The Delta provides this in one sentence.

2. **D.E.K.N. is designed for this** - The Delta/Evidence/Knowledge/Next format already provides extractable summaries - we just needed to surface them.

3. **Recency trumps comprehensiveness** - Showing 3 most recent investigations is better than showing all matches, as recent work is most likely to be relevant.

**Answer to Investigation Question:**

The solution is to extract the `**Delta:**` line from each investigation file and include it in the spawn context's "Related Investigations" section. This was implemented by:
1. Adding `Delta` field to `KBContextMatch` struct
2. Creating `extractDeltaFromInvestigation()` to parse Delta from files
3. Creating `enrichInvestigationsWithDelta()` to populate Delta and limit to 3 investigations
4. Updating `formatKBContextContent()` to include "**Key finding:**" for each investigation

---

## Structured Uncertainty

**What's tested:**

- ✅ Delta extraction works for standard D.E.K.N. format (verified: TestExtractDeltaFromInvestigation passes)
- ✅ Investigations are limited to MaxInvestigationsInContext (verified: TestEnrichInvestigationsWithDelta passes)
- ✅ Delta appears in formatted spawn context (verified: TestFormatContextIncludesDelta passes)

**What's untested:**

- ⚠️ Agents will actually use Delta to create lineage references (requires observing real agent behavior)
- ⚠️ 3 investigations is the right limit (may need adjustment based on usage)

**What would change this:**

- Finding would be wrong if agents still don't create lineage references despite having Delta content available
- Finding would be wrong if Delta extraction fails for non-standard investigation formats

---

## Implementation Recommendations

N/A - Implementation already completed as part of this investigation.

### Changes Made

1. **pkg/spawn/kbcontext.go** - Added:
   - `Delta` field to `KBContextMatch` struct
   - `MaxInvestigationsInContext = 3` constant
   - `extractDeltaFromInvestigation()` function to parse Delta from files
   - `enrichInvestigationsWithDelta()` function to populate Delta and limit count
   - Updated `formatKBContextContent()` to include "**Key finding:**" for investigations

2. **pkg/spawn/kbcontext_test.go** - Added:
   - `TestExtractDeltaFromInvestigation` - tests Delta extraction
   - `TestEnrichInvestigationsWithDelta` - tests enrichment and limiting
   - `TestFormatContextIncludesDelta` - tests formatted output

### Success Criteria

- ✅ Delta extraction works for D.E.K.N. format
- ✅ Investigations limited to 3 most recent
- ✅ Key finding appears in spawn context
- ✅ All tests pass

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` - Main KB context handling code
- `pkg/spawn/context.go` - Spawn context template and generation
- `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md` - Prior investigation that discovered the 0 lineage references problem

**Commands Run:**
```bash
# Build and test
go build ./...
go test ./pkg/spawn/... -v
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md` - Discovered the knowledge fragmentation problem this feature addresses

---

## Investigation History

**2025-12-28:** Investigation started
- Initial question: How to include prior investigations in spawn context
- Context: orch-go-s03z discovered 0 lineage references across 500 investigations

**2025-12-28:** Found existing infrastructure
- KB context already surfaces investigations, just missing content
- D.E.K.N. Delta is perfect for extractable summaries

**2025-12-28:** Investigation and implementation completed
- Status: Complete
- Key outcome: Spawn context now includes Delta (key finding) for up to 3 most recent related investigations
