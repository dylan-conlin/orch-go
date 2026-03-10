# Session Synthesis

**Agent:** og-inv-review-blog-posts-10mar-3f33
**Issue:** orch-go-2bdvb
**Outcome:** success

---

## Plain-Language Summary

I reviewed both published blog posts ("Soft Harness Doesn't Work" and "Building Blind") without reading internal models first, to identify overclaiming. Neither post makes the worst mistake — presenting internal theory as externally validated. Both stay in first-person experiential framing ("I measured," "I was wrong"). The main problem across both posts is **implicit novelty**: describing well-established concepts (affordances, PDCA cycle, Popper's falsificationism, Conway's Law, nudge theory) as personal discoveries without acknowledging the existing literature. This isn't dishonest — it's the natural blind spot of practitioner writing. A domain expert would recognize 60-70% of the conceptual content as restatements. The fix is simple: brief inline acknowledgments, not a literature review. Threshold claims (5+ constraints dilute, 10+ inert) need to be scoped to "in my system" rather than stated as general findings from N=7 skills.

---

## Delta (What Changed)

### Files Created
- `.kb/models/harness-engineering/probes/2026-03-10-probe-blog-post-uncontaminated-claim-review.md` — Per-post claim review with severity ratings and revision suggestions
- `.orch/workspace/og-inv-review-blog-posts-10mar-3f33/VERIFICATION_SPEC.yaml`
- `.orch/workspace/og-inv-review-blog-posts-10mar-3f33/SYNTHESIS.md`

### Files Modified
- `.kb/models/harness-engineering/model.md` — Added probe reference, updated literature context section with specific concept-to-literature mappings

---

## Evidence (What Was Observed)

### "Soft Harness Doesn't Work" (8 flagged items)
- 4 overclaimed: binary taxonomy stated as exhaustive, thresholds from N=7 generalized, prevention > cleanup presented as discovery, "receipts" claimed without methodology transparency
- 2 unsupported: mechanistic causal claim about prompt competition, specific inflection points from small sample
- 2 fine-but-citable: "architecture doing the work of instruction" (Alexander 1977, Norman 1988)

### "Building Blind" (9 flagged items)
- 3 overclaimed: PDCA cycle presented as finding, systems thinking framed as personal acquisition, models-as-hypotheses is Popperian epistemology
- 1 unsupported: "February models work" based on engagement metrics not outcomes
- 4 fine: self-critical evidence, honest framing, good closing
- 1 fine-but-citable: scientific method reference

### Known-Concept Mappings Identified
| Post Concept | Established Source |
|---|---|
| Structural attractors | Affordances (Norman 1988), nudges (Thaler/Sunstein 2008), Conway's Law (1967) |
| Dilution curve | Prompt engineering literature, Miller's Law (1956) |
| Architecture as instruction | Pattern Language (Alexander 1977) |
| Models as hypotheses | Falsificationism (Popper 1934) |
| Build-fail-learn loop | PDCA (Deming 1950s), OODA (Boyd 1960s) |
| Prevention > detection | Poka-yoke (Shingo 1986), shift-left testing |

---

## Architectural Choices

No architectural choices — task was investigation/review.

---

## Knowledge (What Was Learned)

### Key Finding
The posts' strongest quality is also their biggest risk: practitioner voice. Writing from experience naturally omits citations because the author learned concepts through practice, not reading. The fix is additive (inline nods to prior art), not structural (rewriting the posts).

### Recommended Revision Strategy
1. Don't add a literature review — kills the voice
2. Brief inline acknowledgments: "essentially Conway's Law applied to LLM agents"
3. Soften thresholds: "In my system, at 5+" not absolute "At 5+"
4. Add methodology footnote for 265-trial claim
5. Change "works" to observed evidence where outcomes aren't measured

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Probe file exists with all 4 mandatory sections
- [x] Model updated with probe findings
- [x] SYNTHESIS.md created
- [x] VERIFICATION_SPEC.yaml created

---

## Unexplored Questions

- Would the "Coordination Failure" demo post (in .kb/publications/) have the same overclaiming issues? It wasn't in scope but uses similar concepts.
- The harness-engineering extended draft (.kb/publications/harness-engineering-draft.md) likely has higher overclaiming risk than the blog post since it's longer and more theoretical.

---

## Friction

No friction — smooth session. Blog posts were in a separate repo but accessible via absolute paths.

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-review-blog-posts-10mar-3f33/`
**Probe:** `.kb/models/harness-engineering/probes/2026-03-10-probe-blog-post-uncontaminated-claim-review.md`
**Beads:** `bd show orch-go-2bdvb`
