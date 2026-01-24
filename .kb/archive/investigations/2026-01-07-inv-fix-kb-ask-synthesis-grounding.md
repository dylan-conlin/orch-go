<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `kb ask` was failing to find context for natural language questions because it passed raw questions to `kb context`, which expects keywords not sentences.

**Evidence:** `kb context "what is kb ask for?"` returned 0 results, but `kb context "kb ask"` returned 2 decisions and 2 investigations. After fix, questions work correctly.

**Knowledge:** The `kb context` CLI does keyword matching, not semantic search. Natural language questions contain stopwords that dilute the search, causing no matches.

**Next:** Close - implementation complete with tests passing.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: Fix Kb Ask Synthesis Grounding

**Question:** Why does `kb ask` fail to synthesize answers even when relevant context exists?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Architect Agent (og-arch-fix-kb-ask-07jan-c951)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kb context returns nothing for natural language questions

**Evidence:** 
```bash
$ kb context "what is kb ask for?" --format json | jq '{constraints: .constraints|length, decisions: .decisions|length, investigations: .investigations|length}'
{
  "constraints": 0,
  "decisions": 0,
  "investigations": 0
}

$ kb context "kb ask" --format json | jq '{constraints: .constraints|length, decisions: .decisions|length, investigations: .investigations|length}'
{
  "constraints": 0,
  "decisions": 2,
  "investigations": 2
}
```

**Source:** Manual testing with kb CLI

**Significance:** The issue is not in `kb ask` synthesis logic but in how it queries `kb context`. The `kb context` command expects keywords, not full questions.

---

### Finding 2: Question words dilute keyword search

**Evidence:** The words "what", "is", "the", "for" are common stopwords that don't match any domain-relevant content. When combined with actual keywords like "kb" and "ask", the combined query fails to match anything.

**Source:** Analysis of `runKBContext` function at cmd/orch/kb.go:203

**Significance:** Passing raw questions directly to `kb context` is fundamentally broken. Need preprocessing to extract keywords.

---

### Finding 3: Progressive fallback improves robustness

**Evidence:** After implementing keyword extraction and fallback strategies:
```bash
$ orch kb ask "what is kb ask for?"
🔍 Searching knowledge base for: what is kb ask for?
   Found: 0 constraints, 2 decisions, 3 investigations
🤖 Synthesizing answer...
[Correct answer synthesized]
```

**Source:** Testing with fixed implementation

**Significance:** Extracting keywords from questions and trying multiple query strategies makes `kb ask` work with natural language.

---

## Synthesis

**Key Insights:**

1. **Keyword extraction solves the grounding problem** - By filtering stopwords from questions, we extract the domain-relevant terms that `kb context` can match.

2. **Progressive fallback improves reliability** - When combined keywords fail, trying individual keywords (longest first) often finds relevant context.

3. **LLM handles irrelevant matches well** - Even when keyword extraction finds broadly matching content (e.g., "life" matching "lifecycle"), the LLM correctly identifies when context doesn't answer the question.

**Answer to Investigation Question:**

`kb ask` failed because it passed natural language questions directly to `kb context`, which expects keyword queries. The fix adds:
1. `extractKeywords()` - Removes stopwords to get domain terms
2. `runKBContextWithFallback()` - Tries keywords, then individual terms, then original question

---

## Structured Uncertainty

**What's tested:**

- ✅ Keyword extraction correctly filters stopwords (unit tests pass)
- ✅ Questions like "what is kb ask for?" now find context (verified: manual testing)
- ✅ Complex questions like "how should I handle spawning?" work (verified: 123 investigations found)
- ✅ All existing tests pass (verified: go test ./...)

**What's untested:**

- ⚠️ Edge cases with very short queries (1-2 character keywords)
- ⚠️ Non-English text handling
- ⚠️ Performance impact of multiple kb context calls in fallback

**What would change this:**

- Finding would be wrong if kb context adds semantic search (would then handle questions directly)
- Finding would be wrong if LLM hallucinated answers from irrelevant context (tested: it correctly says "no relevant context")

---

## Implementation Recommendations

### Recommended Approach ⭐

**Keyword extraction with progressive fallback** - Already implemented

**Why this approach:**
- Fixes root cause (stopwords diluting search)
- Minimal changes (100 lines added)
- Backward compatible (original question still used for synthesis prompt)

**Trade-offs accepted:**
- Multiple kb context calls in fallback (acceptable for ~5-10s latency budget)
- Simple stopword list (works for English technical questions)

**Implementation sequence:**
1. Add stopwords map and extractKeywords function
2. Add runKBContextWithFallback with multi-strategy search
3. Update runKBAsk to use keywords for context, original question for synthesis

### Alternative Approaches Considered

**Option B: Modify kb CLI to support semantic search**
- **Pros:** Would solve problem at root
- **Cons:** Requires changes to kb-cli project, more scope
- **When to use instead:** If keyword extraction proves insufficient for many question types

**Option C: Use LLM to extract keywords from question**
- **Pros:** More intelligent keyword extraction
- **Cons:** Adds LLM call latency, complexity
- **When to use instead:** If simple stopword filtering misses important patterns

**Rationale for recommendation:** Simple stopword filtering is sufficient for technical questions and adds minimal complexity.

---

## References

**Files Examined:**
- cmd/orch/kb.go:147-224 - runKBAsk and runKBContext functions
- cmd/orch/kb_test.go - Existing test patterns

**Commands Run:**
```bash
# Test kb context behavior
kb context "what is kb ask for?" --format json | jq '{c:.constraints|length, d:.decisions|length, i:.investigations|length}'
kb context "kb ask" --format json | jq '{c:.constraints|length, d:.decisions|length, i:.investigations|length}'

# Test fix
go run ./cmd/orch kb ask "what is kb ask for?"
go run ./cmd/orch kb ask "how should I handle spawning?"

# Run tests
go test ./cmd/orch/... -v -run "TestExtractKeywords|TestHasResults"
```

**Related Artifacts:**
- **Decision:** kn-3e8bd6 - "Investigations already do semantic query answering... Consider 'kb ask' for mini-investigations without artifact overhead"
- **Decision:** orch kb ask uses polling instead of SSE streaming for LLM response

---

## Investigation History

**2026-01-07 13:00:** Investigation started
- Initial question: Why does kb ask fail to synthesize even when context exists?
- Context: 8x gap frequency for "ask command inline" in orch learn

**2026-01-07 13:15:** Root cause identified
- kb context expects keywords not natural language questions
- Stopwords dilute search causing no matches

**2026-01-07 13:30:** Implementation complete
- Added extractKeywords() with stopword filtering
- Added runKBContextWithFallback() with progressive strategies
- Added unit tests for new functions

**2026-01-07 13:45:** Investigation completed
- Status: Complete
- Key outcome: kb ask now works with natural language questions via keyword extraction
