<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation template has lineage headers in kb-cli source, but decision and research templates are missing them.

**Evidence:** Examined kb-cli/cmd/kb/create.go lines 63-66 (investigation has lineage), lines 256-324 (decision missing lineage), lines 367-455 (research missing lineage).

**Knowledge:** Runtime templates (~/.kb/templates/) have lineage headers but source templates (kb-cli) don't, causing drift between source and runtime.

**Next:** Add lineage headers to decision and research templates in kb-cli/cmd/kb/create.go, then rebuild kb binary.

**Confidence:** High (90%) - Source code clearly shows the gap

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Add Lineage Headers to KB-CLI Templates

**Question:** Where should lineage headers (extracted-from, supersedes, superseded-by) be added in kb-cli templates?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-feat-add-lineage-headers-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Investigation template already has lineage headers in kb-cli source

**Evidence:** In ~/Documents/personal/kb-cli/cmd/kb/create.go lines 63-66:
```go
<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]
```

**Source:** ~/Documents/personal/kb-cli/cmd/kb/create.go:15-253 (investigationTemplate const)

**Significance:** Investigation template is already correctly implemented in the source of truth.

---

### Finding 2: Decision template is missing lineage headers in kb-cli source

**Evidence:** Examined decisionTemplate const (lines 256-324) - no lineage block present between "**Status:** Draft" and the Context section.

**Source:** ~/Documents/personal/kb-cli/cmd/kb/create.go:256-324 (decisionTemplate const)

**Significance:** Decision template needs lineage headers added to match investigation template pattern.

---

### Finding 3: Research template is missing lineage headers in kb-cli source

**Evidence:** Examined researchTemplate const (lines 367-455) - no lineage block present between "**Status:** In Progress" and Requirements section.

**Source:** ~/Documents/personal/kb-cli/cmd/kb/create.go:367-455 (researchTemplate const)

**Significance:** Research template needs lineage headers added for consistency across all artifact types.

---

### Finding 4: Runtime templates have lineage headers but source doesn't

**Evidence:** Verified ~/.kb/templates/DECISION.md and RESEARCH.md both have lineage headers (modified 2025-12-22 18:10), but kb-cli source code doesn't match.

**Source:** 
- ~/.kb/templates/DECISION.md:18-21
- ~/.kb/templates/RESEARCH.md:19-22
- Modified timestamps: 2025-12-22 18:10:08 to 18:10:26

**Significance:** There's drift between runtime templates and source templates - source needs to be updated so future kb builds include lineage headers.

---

## Synthesis

**Key Insights:**

1. **Template drift between source and runtime** - Runtime templates have lineage headers but source code doesn't, meaning any new kb binary builds will revert to the old templates without lineage.

2. **Investigation template is the reference implementation** - Lines 63-66 of create.go show the correct format: HTML comment explaining "fill only when applicable", followed by three fields (Extracted-From, Supersedes, Superseded-By).

3. **Consistent placement across templates** - Lineage headers should go after the metadata block (Status field) and before the main content sections, maintaining the same format as investigation template.

**Answer to Investigation Question:**

Lineage headers need to be added to two places in ~/Documents/personal/kb-cli/cmd/kb/create.go:
1. **decisionTemplate** (line ~273): Add after "**Status:** Draft" line
2. **researchTemplate** (line ~384): Add after "**Status:** In Progress" line

Both should use the exact format from investigationTemplate (lines 63-66) to ensure consistency. After adding, rebuild kb binary to ensure future template usage includes lineage headers.

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add lineage headers to kb-cli source templates** - Update decisionTemplate and researchTemplate constants in create.go to include lineage headers, then rebuild binary.

**Why this approach:**
- Ensures source of truth (kb-cli) matches runtime templates
- Future kb builds will include lineage headers automatically
- Maintains consistency across all artifact types (investigation, decision, research)

**Trade-offs accepted:**
- Requires rebuild of kb binary after changes
- Need to update kb-cli repository, not just runtime templates

**Implementation sequence:**
1. Edit ~/Documents/personal/kb-cli/cmd/kb/create.go - Add lineage block to decisionTemplate and researchTemplate
2. Rebuild kb binary - Run `cd ~/Documents/personal/kb-cli && go build -o kb ./cmd/kb`
3. Verify changes - Run `kb create decision test-decision` and check for lineage headers

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
- Edit decisionTemplate in create.go (lines 256-324) - Add lineage block after line 271 "**Status:** Draft"
- Edit researchTemplate in create.go (lines 367-455) - Add lineage block after line 383 "**Status:** In Progress"
- Use exact format from investigationTemplate lines 63-66

**Things to watch out for:**
- ⚠️ String concatenation in Go - Ensure proper backtick escaping for multiline strings
- ⚠️ Exact formatting matters - Must match investigation template format (HTML comment + three bold fields)
- ⚠️ Placement matters - Must go after Status field but before first content section

**Areas needing further investigation:**
- None - implementation is straightforward

**Success criteria:**
- ✅ decisionTemplate includes lineage block (create.go updated)
- ✅ researchTemplate includes lineage block (create.go updated)
- ✅ kb binary rebuilt successfully
- ✅ `kb create decision test` produces file with lineage headers
- ✅ `kb create research test` produces file with lineage headers

---

## References

**Files Examined:**
- ~/Documents/personal/kb-cli/cmd/kb/create.go - Template source code (lines 15-455)
- ~/.kb/templates/INVESTIGATION.md - Runtime investigation template (has lineage)
- ~/.kb/templates/DECISION.md - Runtime decision template (has lineage)
- ~/.kb/templates/RESEARCH.md - Runtime research template (has lineage)

**Commands Run:**
```bash
# Check kb templates path
kb templates path
# Output: /Users/dylanconlin/.kb/templates

# List templates
kb templates list

# Check which templates have lineage
grep -l "Lineage\|Extracted-From" /Users/dylanconlin/.kb/templates/*.md

# Check file modification times
stat -f "%Sm %N" -t "%Y-%m-%d %H:%M:%S" /Users/dylanconlin/.kb/templates/*.md
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-22-inv-design-knowledge-system-project-extraction.md - Parent investigation that recommended lineage headers
- **Beads Issue:** orch-go-hkkh - Add lineage headers to investigation/decision templates

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
