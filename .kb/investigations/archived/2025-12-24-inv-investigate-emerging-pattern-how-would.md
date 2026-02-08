---
linked_issues:
  - orch-go-99lk
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** "How would the system recommend..." isn't a new capability desire - it's the investigation skill already. The real insight is wanting this SYNCHRONOUSLY without spawn overhead.

**Evidence:** Every `orch spawn investigation "how does X work?"` does exactly this: kb context → read code → synthesize recommendation. The pattern frequency isn't rare - it's every investigation spawn.

**Knowledge:** The gap isn't "semantic query answering" - that exists. The gap is latency: spawn (30s+) vs inline answer (5s). A lightweight `kb ask` could do mini-investigations without artifact overhead.

**Next:** Consider `kb ask "question"` command - LLM reads kb context + relevant files, returns synthesis, no artifact created. Trade-off: loses externalization benefit.

**Confidence:** High (90%) - Reframe is clearly correct; implementation approach needs design.

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

### Finding 5: Investigations already do semantic query answering (REFRAME)

**Evidence:** Dylan's follow-up insight: "Aren't investigations already doing exactly this? When we spawn 'how does X work?' or 'should we do Y?', the agent searches kb context, reads code, and synthesizes a recommendation."

This is correct. The investigation skill workflow is:
1. `kb context "topic"` - find prior knowledge
2. Read relevant code/files
3. Run tests if needed
4. Synthesize recommendation
5. Produce artifact

This IS semantic query answering. Every investigation spawn does it.

**Source:** Dylan's follow-up insight and reflection on investigation skill workflow

**Significance:** The original framing ("pattern reveals desire for semantic queries") was wrong. The desire isn't for a NEW capability - it's for the EXISTING capability to be FASTER and SYNCHRONOUS. The friction is:

| Current | Desired |
|---------|---------|
| `orch spawn investigation "question"` | `kb ask "question"` |
| 30+ seconds to spawn | 5 seconds inline |
| Creates .kb/investigations/ artifact | No artifact (ephemeral) |
| Full investigation rigor | Quick synthesis |
| Async (spawn and wait) | Sync (immediate answer) |

The trade-off: `kb ask` loses externalization (no artifact for future sessions), but gains speed for ephemeral questions.

---

## Synthesis

**Key Insights:**

1. **The capability exists, the latency is the problem** - Investigations already do semantic query answering (kb context → read → synthesize). The "how would the system recommend" pattern isn't asking for new capability - it's asking for the existing capability to be FASTER and INLINE.

2. **Spawn overhead is the friction** - Every investigation requires: spawn setup (~5s), agent startup (~10s), artifact creation, commit. For quick questions, this overhead is excessive. Dylan wanted a 5-second answer, not a 30-second spawn.

3. **Trade-off: Speed vs Externalization** - A hypothetical `kb ask` command could provide fast inline answers but would lose the externalization benefit. Investigation artifacts persist for future sessions; inline answers are ephemeral.

**Answer to Investigation Question:**

The "how would the system recommend..." pattern reveals that:

1. **What made it natural to ask:** Dylan has externalized significant knowledge into .kb/ investigations and kn entries. The investigation skill already answers these questions - spawning an agent that searches, reads, and synthesizes.

2. **How the answer was found:** The current path is `orch spawn investigation "question"` which does exactly what Dylan wanted - but with 30+ second overhead. The desire was for INLINE/SYNCHRONOUS synthesis.

3. **Relationship to kb reflect:** Not directly related. kb reflect is for maintenance (synthesis opportunities, stale decisions). The question pattern is already served by investigation skill - just with spawn latency.

4. **Implications for evolution:** Consider a lightweight `kb ask` command:
   - Runs kb context + reads top results + synthesizes answer
   - Returns inline (5 seconds, not 30+)
   - NO artifact created (ephemeral answer)
   - Trade-off: loses externalization benefit

   Design question: When is speed worth losing the artifact? Perhaps:
   - Ephemeral questions → `kb ask` (fast, no artifact)
   - Questions worth preserving → `orch spawn investigation` (slower, creates artifact)

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

- ⚠️ Implementation approach: kb-native (Go) or shell wrapper?
- ⚠️ LLM cost per query - is it acceptable for frequent use?
- ⚠️ When to use ask vs spawn - needs clear heuristic

**What would increase confidence to Very High (95%):**

- Prototype `kb ask` to validate latency (<10s target)
- Test answer quality vs full investigation spawn
- Define clear ask vs spawn heuristic

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Consider `kb ask` command for inline mini-investigations** - Lightweight command that does kb context → read → synthesize without artifact overhead.

**Why this approach:**
- Pattern frequency is HIGH (every investigation does this)
- Real desire is speed, not new capability
- Spawn overhead (30s+) is excessive for quick questions
- Trade-off (speed vs externalization) is acceptable for ephemeral questions

**Trade-offs accepted:**
- No artifact created (answer is ephemeral)
- Future sessions won't have access to the synthesis
- Need heuristic for "ask vs spawn" decision

**Implementation sequence:**
1. Design `kb ask "question"` command interface
2. Implement: kb context → read top N results → LLM synthesis → return answer
3. Add `--save` flag to optionally create investigation artifact
4. Document when to use `kb ask` vs `orch spawn investigation`

### Alternative Approaches Considered

**Option B: Keep current workflow (spawn for everything)**
- **Pros:** All answers externalized, consistent workflow
- **Cons:** 30+ second overhead for quick questions, spawn fatigue
- **When to use instead:** If externalization is always valuable (maybe not)

**Option C: Improve kb context with LLM summary**
- **Pros:** Enhances existing command, no new command to learn
- **Cons:** Mixes retrieval and synthesis in one command, unclear responsibility
- **When to use instead:** If simpler enhancement is preferred

**Rationale for recommendation:** The pattern is frequent (every investigation) and the friction is real (spawn overhead). `kb ask` directly addresses the desire for synchronous answers.

---

### Implementation Details

**What to implement first:**
- Define `kb ask "question"` CLI interface
- Decide: kb-native (Go) or shell wrapper around existing tools?
- LLM integration: which model? (probably same as opencode default)

**Things to watch out for:**
- ⚠️ Cost: every `kb ask` is an LLM call (unlike kb context which is grep)
- ⚠️ Answer quality: mini-investigation may miss nuance vs full spawn
- ⚠️ User confusion: when to ask vs spawn? Need clear guidance.

**Potential implementation:**
```bash
kb ask "how should we sort the swarm map?"
# 1. Run kb context "swarm map sort" (keyword search)
# 2. Read top 3 matching artifacts
# 3. LLM prompt: "Given this context, answer: {question}"
# 4. Return synthesis inline

# Optional: save the answer
kb ask "how should we sort the swarm map?" --save
# Creates .kb/investigations/2025-12-24-ask-swarm-map-sort.md
```

**Success criteria:**
- ✅ `kb ask` returns useful synthesis in <10 seconds
- ✅ Clear guidance on ask vs spawn decision
- ✅ Optional `--save` for questions worth preserving

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

**2025-12-24 (follow-up):** Dylan provided reframe
- Key insight: Investigations ALREADY do exactly this (kb context → read → synthesize)
- Real desire is SYNCHRONOUS/INLINE answers without spawn overhead
- New recommendation: Consider lightweight `kb ask` command for mini-investigations
- Updated D.E.K.N. and added Finding 5
