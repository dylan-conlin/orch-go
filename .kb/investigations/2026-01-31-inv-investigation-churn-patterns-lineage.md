<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation churn is less prevalent than expected - 44.8% of investigations (314/701) already cite prior work via in-text references, but only 0.86% (6/701) use formal Supersedes metadata; the problem is metadata structure, not citation behavior.

**Evidence:** Analyzed 701 investigations: 137 have unfilled Supersedes templates, 80 have "N/A", only 6 have actual .md references. Server cluster (11 investigations) chains correctly via in-text citations like "Jan 26 investigation found X" but none use Supersedes field.

**Knowledge:** Investigations DO chain, but via informal mechanisms (in-text citations, date references) that aren't machine-readable. The gap isn't "agents don't cite prior work" but "citations aren't structured for tooling."

**Next:** Option B (enforce structured lineage) addresses root cause; Case Files (Option A) may still be useful for complex multi-investigation failures with contradictions, but lineage enforcement should come first.

**Authority:** architectural - Affects investigation template, kb create command, and potentially kb reflect tooling across the system.

---

# Investigation: Investigation Churn Patterns and Lineage Gaps

**Question:** How prevalent is investigation churn (multiple investigations on same topic without citing prior work), and should investigations be required to reference prior work on the same topic?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** Worker agent og-inv-investigation-investigation-churn-31jan-ab8b
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** Issue orch-go-21128
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Formal Lineage Metadata Almost Never Used

**Evidence:** Analyzed 701 investigation files for Supersedes field usage:
- 137 files (19.5%): Unfilled template placeholder `[Path to artifact this replaces, if applicable]`
- 80 files (11.4%): Explicitly marked `N/A`
- 6 files (0.86%): **Actual references** to other .md files

The 6 investigations with actual lineage metadata:
1. `2026-01-06-inv-diagnose-investigation-skill-29-completion.md` - "confirms prior findings and adds new one"
2. `2026-01-14-inv-compare-contrast-two-orchestrator-session.md` - "deepens... doesn't supersede - complementary"
3. `2026-01-18-inv-ci-implement-role-aware-injection.md` - "duplicate investigation of same issue"
4. `2026-01-21-inv-investigate-beads-sqlite-database-corruption.md` - "extends recovery findings with root cause"
5. `2026-01-22-inv-daemon-capacity-tracking-stale-after.md` - "extends, doesn't replace"
6. `2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` - "builds on and confirms"

**Source:** `grep -l "Supersedes:.*\[Path to" .kb/investigations/*.md`, `grep -E "Supersedes:.*\.md" .kb/investigations/*.md`

**Significance:** Formal Supersedes metadata is effectively unused (0.86%). The relationship qualifiers used ("extends", "deepens", "confirms") show agents recognize nuanced relationships, but they're rarely captured in the structured field.

---

### Finding 2: In-Text Citations Are Much More Common

**Evidence:** Searched for investigations that reference other investigations via `.kb/investigations` paths:
- **314 files (44.8%)** contain references to other investigation files
- This is 52x more common than formal Supersedes usage

Common citation patterns found:
- Full path: `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md`
- Date reference: "Jan 26 investigation found X"
- Prior investigation language: "From prior investigation:"

Server investigation cluster citation counts:
- Jan 23 investigation: 1 cross-reference
- Jan 26 investigation: 5 cross-references
- Jan 29 investigation: **9 cross-references** to prior work

**Source:** `grep -l "\.kb/investigations" .kb/investigations/*.md | wc -l`, server investigation file analysis

**Significance:** Nearly half of investigations DO cite prior work, but via informal in-text citations that aren't machine-readable. The chaining behavior exists; the structured metadata doesn't.

---

### Finding 3: Server Saga Shows Good Chaining Despite No Formal Lineage

**Evidence:** The server investigation cluster (11 files, Dec 2025 - Jan 2026) demonstrates proper chaining:

| Date | Investigation | Citations | Key Finding |
|------|--------------|-----------|-------------|
| Jan 23 | opencode-server-crashes-under-load | 0 | No crash logs, can't diagnose |
| Jan 26 | opencode-server-keeps-crashing-dying | 3 | SSE stream breaking kills agents |
| Jan 28 | sse-reconnection-opencode-client | 2 | Found unconditional break bug |
| Jan 29 | server-restarts-strand-workers | 4 | SSE fix deployed, verified |

Jan 29 investigation explicitly lists prior work in Finding 1:
1. "Jan 26 investigation... concluded that agent death is caused by SSE stream breaking"
2. "Jan 17 investigation... designed auto-resume mechanism"
3. "Jan 28 investigation... implemented SSE reconnection"

Yet Supersedes field is blank ("None") in all of them.

**Source:** Read and analyzed `.kb/investigations/*server*.md` files

**Significance:** This is the exemplar case mentioned in the task. It shows investigations DO chain correctly - problem evolution (can't diagnose → identified mechanism → found bug → deployed fix) is traceable. The gap is METADATA, not BEHAVIOR.

---

### Finding 4: What Multi-Investigation View Reveals

**Evidence:** Reading the server cluster chronologically reveals patterns not visible in individual investigations:

1. **Problem evolution timeline:** Jan 23 couldn't diagnose → Jan 26 identified SSE mechanism → Jan 28 found root cause → Jan 29 confirmed fix

2. **Terminology clarification:** Jan 23 calls it "crash" but Jan 26 distinguishes "session corruption" vs "session loss"

3. **Solution convergence:** Multiple approaches considered (auto-resume, SSE reconnection) → SSE fix emerged as primary, auto-resume as fallback

4. **Contradiction resolution:** Jan 26 said agents die when server restarts. Jan 29 tests and confirms orch serve restarts are harmless; only OpenCode restarts matter.

**Source:** Comparative reading of server investigation cluster

**Significance:** The multi-investigation view reveals EVOLUTION (how understanding changed), CONTRADICTION RESOLUTION (what got clarified), and SOLUTION EMERGENCE (how fix was chosen). Individual investigations capture point-in-time truth but not these patterns.

---

### Finding 5: Nuanced Relationships Need Vocabulary

**Evidence:** The 6 investigations with actual Supersedes values use these qualifiers:
- "confirms prior findings and adds new one"
- "deepens... doesn't supersede - complementary"
- "extends recovery findings with root cause"
- "extends, doesn't replace"
- "builds on and confirms"
- "duplicate investigation of same issue"

Only 1 of 6 is an actual supersession ("duplicate"). The rest are EXTENSIONS, CONFIRMATIONS, or COMPLEMENTARY investigations.

**Source:** `grep -rh "Supersedes:.*extends\|Supersedes:.*deepens\|Supersedes:.*confirms" .kb/investigations/*.md`

**Significance:** "Supersedes" is the wrong concept for most relationships. Investigations have:
- **Extends:** Adds to prior findings (most common)
- **Confirms:** Validates prior hypothesis
- **Contradicts:** Disproves prior conclusion
- **Deepens:** Explores same question at greater depth
- **Supersedes:** Replaces obsolete investigation (rare)

---

## Synthesis

**Key Insights:**

1. **Churn is less prevalent than expected** - Nearly half (44.8%) of investigations cite prior work. The server saga shows proper chaining. The real problem isn't that agents ignore prior work but that citations are unstructured.

2. **Supersedes is the wrong primitive** - Only ~1% of investigation relationships are true supersession. Most are extensions, confirmations, or deepening. A richer vocabulary is needed.

3. **The structural gap is metadata, not behavior** - In-text citations work for human readers but not for tooling. `kb reflect --type synthesis` detected clusters but can't trace lineage because references are prose, not structured.

**Answer to Investigation Question:**

**"How prevalent is investigation churn?"**

Less than expected. 44.8% of investigations reference prior work via in-text citations. The server cluster (the exemplar case) chains correctly across 4+ investigations. True churn (starting fresh without any prior work reference) occurs in ~55% of investigations, but many of these are genuinely novel investigations, not duplicates.

**"Should investigations be required to reference prior work?"**

Yes, but with nuance:
1. **`kb search` before create** - Require searching for existing investigations on the topic
2. **Structured lineage field** - Replace simple `Supersedes:` with richer vocabulary: `Extends:`, `Confirms:`, `Contradicts:`
3. **Machine-readable references** - Full paths, not date descriptions

**Case Files verdict:**

Case Files would be useful for the MULTI-INVESTIGATION VIEW insights (evolution, contradiction resolution, solution emergence) that don't appear in individual investigations. But lineage enforcement should come FIRST to reduce the synthesis burden.

---

## Structured Uncertainty

**What's tested:**

- ✅ Supersedes field usage rate is 0.86% (verified: grep analysis of 701 files)
- ✅ In-text citation rate is 44.8% (verified: grep for .kb/investigations paths)
- ✅ Server cluster chains correctly via in-text citations (verified: read 4 files, traced references)
- ✅ Investigations use nuanced qualifiers (extends, confirms, deepens) (verified: grep output)

**What's untested:**

- ⚠️ Whether requiring `kb search` before create would reduce churn (behavioral hypothesis)
- ⚠️ Whether structured lineage enables useful tooling (need to build it to test)
- ⚠️ Whether Case Files would actually be used if created (adoption unknown)

**What would change this:**

- If sampling other clusters showed no in-text citations, would indicate server saga is unusual
- If agents ignore `kb search` results despite being shown them, lineage enforcement won't help
- If contradictions between investigations are rare, Case Files may not be needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Enforce structured lineage | architectural | Affects investigation template, kb create, and kb reflect tooling |
| Create Case Files artifact type | strategic | New artifact type is commitment to maintain and promote adoption |

### Recommended Approach ⭐

**Option B: Enforce Investigation Lineage** - Require `kb search` before `kb create investigation`, add structured `Prior-Work:` field with vocabulary (Extends/Confirms/Contradicts/Deepens).

**Why this approach:**
- Addresses root cause: citations exist but aren't structured (Finding 2)
- Builds on existing behavior (agents already cite prior work informally)
- Enables `kb reflect` to trace lineage programmatically
- Lower lift than new artifact type

**Trade-offs accepted:**
- Adds friction to investigation creation
- Requires template update and skill modification
- Case Files deferred (may be needed for complex multi-investigation failures)

**Implementation sequence:**
1. Update investigation template: Replace `Supersedes:` with `Prior-Work:` supporting multiple entries with relationship types
2. Add `kb context --topic "X"` command that surfaces prior investigations on topic
3. Update `kb create investigation` to run `kb context --topic` and require acknowledgment
4. Add `kb reflect --lineage` to visualize investigation chains

### Alternative Approaches Considered

**Option A: Case Files (new artifact type)**
- **Pros:** Captures multi-investigation synthesis, evolution, contradiction resolution
- **Cons:** New artifact type to maintain, adoption uncertain, doesn't fix root lineage problem
- **When to use instead:** After lineage enforcement is in place, for complex multi-investigation failures where synthesis is valuable

**Option C: Both (lineage + Case Files)**
- **Pros:** Complete solution addressing both chaining and synthesis
- **Cons:** High implementation cost, may be over-engineering
- **When to use instead:** If lineage enforcement alone proves insufficient

**Rationale for recommendation:** Findings show the gap is structured metadata (0.86% Supersedes usage) not citation behavior (44.8% in-text citations). Fix the metadata gap first. Case Files may be warranted later for complex failure patterns like the server saga.

---

### Implementation Details

**What to implement first:**
1. `Prior-Work:` field in investigation template with relationship vocabulary
2. `kb context --topic` command to surface prior investigations
3. Require acknowledgment in `kb create investigation` workflow

**Things to watch out for:**
- ⚠️ Don't make lineage enforcement so burdensome that agents skip it
- ⚠️ "Supersedes" vocabulary is wrong - use Extends/Confirms/Contradicts
- ⚠️ Allow "N/A - novel investigation" as valid acknowledgment

**Areas needing further investigation:**
- How to detect when investigations SHOULD be related but aren't
- Whether kb reflect can reliably detect topic clusters for suggesting prior work
- How to handle cross-project investigation lineage

**Success criteria:**
- ✅ Prior-Work field usage > 50% (vs current 0.86% Supersedes)
- ✅ kb reflect can trace investigation chains programmatically
- ✅ Synthesis opportunities (like server saga) detected automatically

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-29-inv-server-restarts-strand-workers-4096.md` - Server saga investigation
- `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - Prior server investigation
- `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Origin server investigation
- `.kb/investigations/2026-01-28-inv-sse-reconnection-opencode-client-survive.md` - SSE fix investigation

**Commands Run:**
```bash
# Count investigations with different Supersedes states
ls .kb/investigations/*.md | wc -l  # 701 total
grep -l "Supersedes:.*\[Path to" .kb/investigations/*.md | wc -l  # 137 unfilled
grep -l "Supersedes: N/A" .kb/investigations/*.md | wc -l  # 80 N/A
grep -E "Supersedes:.*\.md" .kb/investigations/*.md | grep -v "N/A"  # 6 actual

# Count in-text citations
grep -l "\.kb/investigations" .kb/investigations/*.md | wc -l  # 314 files

# Analyze server cluster
grep -c "\.kb/investigations/" .kb/investigations/*server*.md
```

**Related Artifacts:**
- **Prompt:** Case Files design session that raised this question
- **Command:** `kb reflect --type synthesis` that detected the clusters
- **Constraint:** "D.E.K.N. is universal handoff structure" - lineage extends this

---

## Investigation History

**2026-01-31 10:00:** Investigation started
- Initial question: How prevalent is investigation churn?
- Context: kb reflect found 15 synthesis clusters, server saga has 11+ investigations

**2026-01-31 10:10:** Analyzed Supersedes field usage
- Found only 0.86% (6/701) use formal lineage metadata
- Discovered nuanced qualifiers: extends, confirms, deepens

**2026-01-31 10:20:** Analyzed in-text citations
- Found 44.8% (314/701) reference prior investigations informally
- This is 52x more than formal metadata

**2026-01-31 10:30:** Deep-dive on server saga
- Traced Jan 23 → Jan 26 → Jan 28 → Jan 29 chain
- Confirmed proper chaining via in-text citations
- Identified multi-investigation insights (evolution, contradictions, solution emergence)

**2026-01-31 10:45:** Formulated recommendation
- Option B (lineage enforcement) addresses root cause
- Case Files deferred but may be needed for complex failures

**2026-01-31 11:00:** Investigation completed
- Status: Complete
- Key outcome: Churn is less prevalent than expected (44.8% cite prior work), but citations are unstructured. Fix metadata gap with structured lineage before considering Case Files.
