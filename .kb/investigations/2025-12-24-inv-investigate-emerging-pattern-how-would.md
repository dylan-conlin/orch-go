<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** "How would the system recommend..." reveals a new usage pattern - treating the knowledge system as a semantic oracle that synthesizes recommendations from prior investigations, which `kb context` cannot do (keyword match only).

**Evidence:** `kb context "swarm"` found the 2025-12-23 progressive disclosure investigation containing the recommendation, but `kb context "swarm map sorting"` and `kb context "how should dashboard present agents"` returned nothing - demonstrating kb context's keyword-matching limitation.

**Knowledge:** The pattern signals a desire for semantic query answering ("what should we do about X?") beyond current keyword retrieval, which would require LLM synthesis over the knowledge base.

**Next:** Document as insight for future knowledge system evolution; no immediate implementation needed - current pattern (human orchestrator + kb context) works but suggests future enhancement opportunity.

**Confidence:** High (85%) - Pattern clearly observed; implementation complexity for semantic queries is high.

---

# Investigation: Emerging Pattern - "How Would the System Recommend..."

**Question:** What does the "how would the system recommend we sort the swarm map?" question pattern reveal about knowledge system evolution, and is this related to kb reflect?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** og-inv-investigate-emerging-pattern-24dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: The question was answered via kb context with keyword matching

**Evidence:** Dylan asked "how would the system recommend we sort the swarm map?" The answer came from `kb context "swarm"` which surfaced:

1. kn decision: "Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management"
2. Investigation: `2025-12-23-inv-design-question-should-swarm-dashboard.md`

The investigation (308 lines) contained a thorough analysis recommending progressive disclosure with three sections (Active/Recent/Archive) and specific implementation details.

**Source:** 
- `kb context "swarm"` output showing constraints, decisions, and 22+ investigations
- `.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md`

**Significance:** The answer existed in the knowledge base but was only accessible because "swarm" was a keyword in both the question and the artifacts. The framing "how would the system recommend" implies semantic understanding, but the actual discovery was keyword-based.

---

### Finding 2: kb context fails on semantic queries

**Evidence:** Tested several natural language query variants:

```bash
kb context "swarm map sorting"           # No context found
kb context "how should dashboard present agents"  # No context found
kb context "swarm sort dashboard"        # No context found
```

But keyword-based queries succeeded:

```bash
kb context "swarm"               # 22+ investigations, 2 constraints, 1 decision
kb context "dashboard"           # 17 investigations, 1 constraint, 2 decisions
kb context "progressive disclosure"  # 6 investigations, 4 decisions
```

**Source:** Direct command testing during investigation

**Significance:** The "how would the system recommend" framing treats the knowledge system as if it can answer natural language questions about recommendations. In reality, `kb context` is a keyword search - it surfaces artifacts containing the query terms but cannot synthesize recommendations.

---

### Finding 3: kb reflect and kb context serve different purposes

**Evidence:** Examined both commands:

**kb reflect** (from `pkg/daemon/reflect.go` and help output):
- Purpose: Surface patterns requiring human attention
- Types: synthesis, promote, stale, drift, open
- Detection: Finds investigation clusters, stale decisions, potential drift
- Output: "Consider synthesizing..." suggestions

**kb context** (from help output):
- Purpose: Get relevant context for a topic
- Aggregates: kn entries (constraints, decisions) + kb artifacts (investigations)
- Output: Lists of matching artifacts grouped by type

Neither command can:
- Synthesize a recommendation from multiple sources
- Answer "what should we do about X?"
- Provide semantic understanding of intent

**Source:**
- `kb reflect --help`
- `kb context --help`
- `.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md`

**Significance:** The "how would the system recommend" pattern implies a third capability - **semantic query answering** - that sits between retrieval (kb context) and reflection (kb reflect). This would require LLM synthesis.

---

### Finding 4: The current workflow works but has friction

**Evidence:** When Dylan asked the question, the workflow was:

1. Dylan asks: "How would the system recommend we sort the swarm map?"
2. Orchestrator runs: `kb context "swarm"` (keyword match)
3. Finds 22+ investigations including the dashboard design question
4. Reads the investigation artifact (308 lines)
5. Synthesizes: "The prior investigation recommends progressive disclosure..."

This worked because:
- Dylan's question contained "swarm" keyword
- A prior investigation existed on exactly this topic
- The orchestrator could read and summarize

This would fail if:
- No investigation existed (no prior thought on the topic)
- Keywords didn't match (different terminology)
- Question was more abstract ("how should we handle list views?")

**Source:** Analysis of the actual interaction pattern

**Significance:** The human orchestrator is performing the semantic synthesis step. The question "how would the system recommend" is really "help me find if we've thought about this before and what we concluded."

---

## Synthesis

**Key Insights:**

1. **The framing reveals user mental model** - Dylan asked "how would the system recommend" treating the knowledge system as an oracle that can synthesize recommendations. This is aspirational but not current reality - the orchestrator/human performs the synthesis.

2. **kb reflect is about maintenance, not recommendations** - kb reflect surfaces patterns requiring attention (synthesis needed, stale decisions, drift) but doesn't answer "what should we do?" questions. The question pattern is more like "kb consult" or "kb recommend".

3. **The gap is semantic query answering** - Between retrieval (kb context) and reflection (kb reflect) sits a missing capability: taking a natural language question and synthesizing a recommendation from multiple prior investigations. This would require LLM processing of the knowledge base.

**Answer to Investigation Question:**

The "how would the system recommend..." pattern reveals that:

1. **What made it natural to ask:** Dylan has externalized significant knowledge into .kb/ investigations and kn entries. After enough decisions are documented, asking "what does the system think?" becomes natural - treating accumulated knowledge as institutional wisdom.

2. **How the answer was found:** `kb context "swarm"` surfaced the prior investigation, and the orchestrator synthesized the recommendation by reading it. This required: (a) keyword matching, (b) human interpretation, (c) prior investigation existing.

3. **Relationship to kb reflect:** Not directly related. kb reflect identifies maintenance needs (synthesis opportunities, stale decisions) while the question pattern asks for recommendations. They're complementary but different:
   - `kb reflect`: "What needs attention in our knowledge base?"
   - Implicit "kb recommend": "What would our knowledge base suggest we do about X?"

4. **Implications for evolution:** The pattern suggests a future capability - semantic query answering over the knowledge base. This would require:
   - LLM processing of queries
   - Retrieval-augmented generation (RAG) over .kb/ and .kn/
   - Synthesis of multiple sources into a recommendation
   - High implementation complexity

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The pattern is clearly observable - Dylan's question, kb context's limitations, and the gap between retrieval and recommendation are all testable and validated. The interpretation of what this means for system evolution is reasonable inference.

**What's certain:**

- ✅ kb context uses keyword matching, not semantic understanding
- ✅ The "recommend" pattern requires human synthesis of retrieved artifacts
- ✅ kb reflect serves maintenance needs, not recommendation queries
- ✅ The workflow currently works with human-in-loop orchestrator

**What's uncertain:**

- ⚠️ Whether semantic query answering is worth the complexity to implement
- ⚠️ How often this pattern occurs (is it frequent enough to optimize?)
- ⚠️ Whether improved keyword coverage would be sufficient without LLM

**What would increase confidence to Very High (95%):**

- Track frequency of "how would the system..." questions over a week
- Test more query variants to understand failure patterns
- Prototype RAG-based recommendation to validate complexity

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Document insight, no immediate implementation** - Record this pattern as an evolutionary insight for the knowledge system but don't implement semantic query answering now.

**Why this approach:**
- Current workflow works (orchestrator + kb context)
- LLM-based semantic queries would add significant complexity
- The pattern frequency is unknown - may be rare
- Better to accumulate more examples before investing

**Trade-offs accepted:**
- Human synthesis remains required
- Some queries will miss due to keyword mismatch
- "System recommend" framing remains aspirational

**Implementation sequence:**
1. Record this investigation as documentation of the pattern
2. Continue using current workflow (orchestrator + kb context)
3. Collect more examples of semantic query patterns
4. Revisit when pattern frequency justifies investment

### Alternative Approaches Considered

**Option B: Implement kb recommend with RAG**
- **Pros:** Direct answer to "what should we do about X?"
- **Cons:** Significant implementation (LLM integration, RAG pipeline, cost)
- **When to use instead:** If semantic queries become frequent (>10/week)

**Option C: Improve kb context with synonyms/embeddings**
- **Pros:** Lighter than full RAG, improves keyword matching
- **Cons:** Still doesn't synthesize recommendations
- **When to use instead:** If keyword mismatch is the main pain point

**Rationale for recommendation:** Pattern is interesting but not frequent enough to justify implementation complexity. Document for future reference.

---

### Implementation Details

**What to implement first:**
- None - this is a documentation-only recommendation
- Externalize the insight via kn command

**Things to watch out for:**
- ⚠️ If Dylan asks "how would the system recommend..." frequently, revisit
- ⚠️ If keyword mismatches cause missed retrievals, consider embeddings

**Success criteria:**
- ✅ Investigation documents the pattern for future reference
- ✅ Insight is externalized for session amnesia resilience
- ✅ Pattern is trackable if frequency increases

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md` - kb reflect design
- `.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md` - The actual recommendation that answered Dylan's question
- `pkg/daemon/reflect.go` - ReflectSuggestions implementation

**Commands Run:**
```bash
# kb context keyword matching tests
kb context "swarm"                              # Found 22+ investigations
kb context "swarm map sorting"                  # No context found
kb context "how should dashboard present agents" # No context found
kb context "dashboard"                          # Found 17 investigations
kb context "progressive disclosure"             # Found 6 investigations

# kb reflect capabilities
kb reflect --help
kb reflect
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md` - kb reflect design
- **Investigation:** `.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md` - The source of the recommendation

---

## Self-Review

- [x] Real test performed (tested kb context with multiple query variants)
- [x] Conclusion from evidence (based on observed behavior)
- [x] Question answered (pattern analyzed, relationship to kb reflect explained)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Discovered Work:**
- No bugs/issues discovered
- Insight worth capturing: "How would the system recommend..." pattern

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-24 (start):** Investigation started
- Initial question: What does the "how would the system recommend we sort the swarm map?" question pattern reveal?
- Context: Dylan asked this question, treating the knowledge system as an oracle

**2025-12-24 (analysis):** Tested kb context behavior
- Discovered kb context uses keyword matching only
- Found that "swarm" keyword surfaced the investigation containing the answer
- Tested failure cases with semantic queries

**2025-12-24 (synthesis):** Pattern documented
- kb reflect is for maintenance, not recommendations
- Gap exists between retrieval (kb context) and recommendation (human synthesis)
- Current workflow works but reveals future enhancement opportunity

**2025-12-24:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Pattern reveals desire for semantic query answering over knowledge base; no immediate implementation needed
