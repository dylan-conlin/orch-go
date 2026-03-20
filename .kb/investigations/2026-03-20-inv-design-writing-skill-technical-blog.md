## Summary (D.E.K.N.)

**Delta:** A standalone `technical-writer` skill with 4 phases (Story Discovery → Draft → Composition Review → Revision) is the right design — it treats writing as a compositional correctness problem where self-audit with quote-based evidence is the composition-level gate that grammar/clarity checks miss.

**Evidence:** Analysis of 5 existing skills (investigation, architect, experiential-eval, research, feature-impl), skillc detection pattern capabilities (regex/contains/negation/OR only — no AND, no position, no semantic), the writing-style model (4 stance primers, INERT status), and the compositional correctness gap pattern (component gates pass, composition fails).

**Knowledge:** Composition-level quality cannot be fully tested with skillc's current detection patterns — they measure symptoms (turn language present, first-person voice) but not structure (turn in first third, abstraction after story). The gap requires a structural deliverable: a composition self-audit artifact with quote-based evidence, which CAN be validated at completion time.

**Next:** Create `skills/src/worker/technical-writer/.skillc/` with skill manifest and template. Write 3-4 contrastive test scenarios. Update writing-style model from INERT to TESTABLE.

**Authority:** architectural — New skill creation crosses skill system boundaries and establishes a new pattern (composition self-audit as gate).

---

# Investigation: Design Writing Skill for Technical Blog Posts

**Question:** How should a composition-level writing skill be structured, tested, and connected to the existing skill system?

**Started:** 2026-03-20
**Updated:** 2026-03-20
**Owner:** orch-go-npm1s
**Phase:** Complete
**Next Step:** None — design complete, ready for implementation
**Status:** Complete
**Model:** writing-style

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/threads/2026-03-20-writing-soft-harness-from-primers.md` | extends | Yes — thread poses 4 open questions, this investigation answers them | None |
| `.kb/models/writing-style/model.md` | extends | Yes — model is INERT (primers untested), this design makes them testable | None |
| `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` | deepens | Yes — identifies compositional correctness gap as cross-domain pattern | None |
| `.kb/models/harness-engineering/probes/2026-03-20-probe-scs-ai-part-builder-compositional-correctness.md` | confirms | Yes — DFM/LED/agent scales all show same pattern | None |
| Behavioral grammars model (constraint dilution at 5+) | constrains | Yes — primers kept to 4 items by design | None |

---

## Findings

### Finding 1: Writing is a Compositional Correctness Problem — Not a Clarity Problem

**Evidence:** The harness engineering blog post has well-written individual sections. Grammar is correct. Clarity is adequate. Tables are accurate. The failure is at the composition level:
- Framework appears before reader cares about the problem
- Self-correction ("gates haven't bent the curve yet") buried in section 4 of 6
- 11 tables presented as narrative when they're reference material
- No emotional voice despite emotionally rich source material (1,625 lost commits)

This exactly mirrors the compositional correctness gap identified across 3 scales: DFM (operations pass, assembly fails), LED gates (geometry passes, function fails), agent coordination (commits pass, codebase degrades).

**Source:** `feedback_writing_style.md`, `.kb/models/writing-style/model.md`, `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md`

**Significance:** The skill must gate at composition level, not component level. A grammar/clarity skill would pass the failed blog post. The skill needs structural checks: Where is the turn? Does abstraction follow story? Is emotional voice present?

---

### Finding 2: skillc Detection Patterns Are Necessary But Insufficient for Composition

**Evidence:** skillc supports 4 detection pattern types:
- `regex:PATTERN` — full regex, case-insensitive
- `response contains X|Y|Z` — alternation (OR), case-insensitive
- `response does not contain X` — negation
- Plain substring — case-insensitive

What CAN be detected:
- Turn markers present: `response contains I was wrong|I realized|turned out|I thought`
- Emotional markers: `response contains felt like|the moment|surprised|frustrated`
- First-person voice: `regex:\b(I|we|my|our)\b`
- Framework-first avoidance: `response does not contain ## Taxonomy|Related Work|Prior Art`
- Self-correction language: `response contains wrong|mistake|assumed`

What CANNOT be detected:
- Turn placement (first third vs last third) — requires position awareness
- Abstraction ordering (story before framework) — requires sequence detection
- Table density (is each table earned?) — requires counting and context
- Semantic composition quality — requires understanding

**Source:** `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/09-contradiction-detection.yaml` (concrete detection pattern examples), exploration agent analysis of skillc testing framework

**Significance:** Testing strategy must be two-tiered: proxy detection patterns for measurable symptoms + structural artifact validation for composition quality. An LLM-as-judge capability would close the gap but requires new skillc infrastructure.

---

### Finding 3: Self-Audit with Quote-Based Evidence Is the Composition Gate

**Evidence:** Existing skills use structured self-review as quality gates:
- Investigation: D.E.K.N. summary (Delta, Evidence, Knowledge, Next)
- Research: Structured uncertainty (tested vs untested vs falsifiable)
- Feature-impl: VERIFICATION_SPEC.yaml with specific test evidence

The pattern: force the agent to explicitly confront quality dimensions by requiring evidence, not assertions. "Did you include a turn?" → "Quote the turn sentence and its position."

Applied to writing composition:
1. "Quote the sentence where the author was wrong. What fraction of the piece precedes it?"
2. "What is the first framework/abstraction element? What story precedes it?"
3. "How many data tables? For each, quote the experience that earned it."
4. "Quote one sentence of emotional honesty."
5. "Read the opening paragraph: is this a story, a question, or a framework label?"

If any answer reveals a composition failure, revision is required before completion.

**Source:** Investigation skill D.E.K.N. pattern, research skill structured uncertainty, feature-impl VERIFICATION_SPEC pattern

**Significance:** This is the key design insight. Composition quality isn't testable by regex, but it IS enforceable by requiring the writer to produce evidence of composition quality as a structured deliverable. The evidence artifact is the gate.

---

### Finding 4: Standalone Skill, Not Shared Dependency or Modifier

**Evidence:** Analysis of which skills produce writing:
- investigation → internal probes/investigations (audience: other agents)
- architect → internal investigations with recommendations (audience: orchestrator)
- research → internal investigations (audience: orchestrator)
- experiential-eval → internal evaluations (audience: orchestrator)

All existing writing-producing skills create **internal artifacts** for agent/orchestrator consumption. The writing primers are specifically for **external publications** targeting human readers (HN, blog readers, technical community).

Injecting publication-quality stance into investigation/architect would be counterproductive — those skills should optimize for clarity and completeness for AI consumers, not narrative arc for human readers.

A shared dependency (like `writer-base`) has no consumers — no existing skill writes publications.

**Source:** `skills/src/worker/experiential-eval/SKILL.md`, `skills/src/worker/architect/.skillc/`, exploration agent analysis of 5 skills

**Significance:** The technical-writer skill must be standalone. The only connection to existing skills: it may consume outputs from investigation/architect as source material for a publication.

---

### Finding 5: The Primers Work as Phase 1 Context, Not Phase-Spanning Rules

**Evidence:** The writing-style model explicitly notes: "These are attention primers — they shift what the writer notices, not what the writer must do." The behavioral grammars model shows constraint dilution at 5+ items.

If the primers are injected as rules throughout all phases, they compete with phase-specific instructions for attention. If they're injected as Phase 1 context (Story Discovery), they prime the writer's attention before drafting begins — then the structural phases (Composition Review) enforce the outcomes the primers produce.

Structure:
- **Phase 1 (Story Discovery):** Primers active. Writer identifies turn, experiences, emotions BEFORE drafting.
- **Phase 2 (Draft):** Primers recede. Writer follows the story map from Phase 1.
- **Phase 3 (Composition Review):** Primers operationalized as audit questions. Quote-based evidence required.
- **Phase 4 (Revision):** Addresses composition failures.

**Source:** `.kb/models/writing-style/model.md` lines 33-34, behavioral grammars model (constraint dilution)

**Significance:** The primers aren't rules to follow; they're a lens to use during story discovery. The composition review phase is where enforcement happens — but through structural audit questions, not the primers themselves.

---

## Synthesis

**Key Insights:**

1. **Composition gates require structural evidence, not pattern matching.** Just as the compositional correctness gap shows component gates missing assembly-level failures, regex detection misses composition-level writing quality. The solution is the same: gate at the right abstraction level, using a composition self-audit artifact with quote-based evidence.

2. **Story Discovery as explicit phase eliminates the framework-first default.** The failed blog post opened with framework because the writer started writing before identifying the story. Phase 1 (Story Discovery) forces story identification before any drafting — the turn, the experiences, the emotional moments are mapped before a word of the post is written.

3. **Testing splits into two tiers that measure different things.** Tier 1 (proxy detection patterns) measures whether the skill shifts output distribution — does the agent use more first-person voice, more turn language, more emotional markers when the skill is loaded? Tier 2 (composition audit artifact) measures whether the composition actually works — are the quotes real, is the turn actually in the first third, does a story actually precede the first abstraction?

**Answer to Investigation Question:**

The skill should be a standalone `technical-writer` skill with 4 phases, tested at two tiers, producing a blog post draft plus a composition self-audit artifact. It is not a modifier for existing skills (those serve different audiences) and it is not two skills (the self-review must happen within the same session to access draft context). The composition self-audit with quote-based evidence is the key innovation — it operationalizes the primers as structural checkpoints without adding behavioral rules that would dilute.

---

## Structured Uncertainty

**What's tested:**

- ✅ Existing skills (investigation, architect, research, experiential-eval, feature-impl) produce internal artifacts — confirmed by reading all 5 SKILL.md files
- ✅ skillc detection patterns support regex/contains/negation/OR only — confirmed by reading contrastive test scenarios and skillc source
- ✅ The writing-style model has INERT status (primers untested) — confirmed by reading model.md
- ✅ D.E.K.N. / structured uncertainty / VERIFICATION_SPEC patterns successfully enforce self-review in existing skills — confirmed by reading investigation/research/feature-impl skills
- ✅ Compositional correctness gap manifests at 3+ scales (DFM, LED, agent coordination) — confirmed by reading harness-engineering investigation

**What's untested:**

- ⚠️ Whether Story Discovery as Phase 1 actually prevents framework-first drafting (hypothesis — needs real publication attempt)
- ⚠️ Whether composition self-audit catches composition failures vs becoming checkbox theater (could be validated by applying to the harness engineering draft rewrite)
- ⚠️ Whether proxy detection patterns show measurable lift in contrastive testing (needs actual skillc test runs)
- ⚠️ Whether 4 phases is the right granularity or whether draft + review could collapse to 2 phases (needs experimentation)

**What would change this:**

- If self-audit becomes checkbox theater (writer quotes irrelevant sentences to pass), the audit questions need to be more specific or an LLM-as-judge tier is required
- If proxy detection patterns show zero lift between bare and variant, the skill may not be changing behavior at the level detectable by tokens — would need different measurement approach
- If the skill is used for non-blog-post writing (documentation, READMEs, proposals), the primers may not apply — current design is blog-post-specific

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create standalone `technical-writer` skill | architectural | New skill creation, establishes composition-audit pattern |
| 4-phase structure (Story Discovery → Draft → Review → Revision) | implementation | Internal skill design, follows existing phase patterns |
| Composition self-audit as required deliverable | architectural | New artifact type, may influence completion pipeline |
| Proxy detection patterns for contrastive testing | implementation | Uses existing skillc infrastructure |
| LLM-as-judge as future skillc enhancement | strategic | New testing infrastructure capability |

### Recommended Approach: Standalone 4-Phase Skill with Composition Self-Audit

**Why this approach:**
- Treats writing as composition problem (matches root cause analysis)
- Story Discovery phase prevents framework-first default
- Self-audit with quote evidence is enforceable without new infrastructure
- Standalone prevents contamination of internal-artifact skills

**Trade-offs accepted:**
- No real-time composition feedback during drafting (only post-draft audit)
- Proxy detection patterns are weak for composition quality (Tier 2 artifact validation compensates)
- LLM-as-judge deferred (requires new skillc infrastructure)

**Implementation sequence:**

1. **Create skill skeleton** — `skills/src/worker/technical-writer/.skillc/` with manifest, template, phases
2. **Write contrastive test scenarios** — 3-4 scenarios measuring proxy indicators (turn markers, emotional voice, framework avoidance)
3. **Apply to harness engineering rewrite** — First real test of the skill on the existing failed draft
4. **Measure and iterate** — Compare rewritten draft against original using composition audit

### Alternative Approaches Considered

**Option B: Shared Dependency (`writer-base`)**
- **Pros:** Other skills could inherit writing quality guidance
- **Cons:** No existing skills write publications; would add irrelevant context to investigation/architect skills; primer stance conflicts with internal-artifact optimization
- **When to use instead:** If multiple skills begin producing external publications

**Option C: Two Skills (draft + review)**
- **Pros:** Separation of concerns; review skill could be reused independently
- **Cons:** Requires cross-agent context transfer (draft content); heavyweight for a task completable in one session; loses composition-in-context advantage
- **When to use instead:** If drafts are produced by humans (non-agent) and only review is agent-assisted

**Option D: LLM-as-Judge Gate**
- **Pros:** Semantic composition evaluation; could validate ordering, density, arc quality
- **Cons:** Requires new skillc infrastructure; adds latency and cost; overkill for initial version
- **When to use instead:** When self-audit proves insufficient (checkbox theater detected)

**Rationale for recommendation:** Option A is the minimum viable composition gate. It requires no new infrastructure (uses existing skill/skillc patterns), addresses the root cause (composition-level enforcement), and produces a testable artifact (the composition audit). Options B-D are either wrong audience, wrong architecture, or premature optimization.

---

### Implementation Details

**What to implement first:**
- Skill manifest (`skill.yaml`) with deliverables, verification, spawn config
- Story Discovery phase template (primer context + story map structure)
- Composition Review phase (5 audit questions with quote-evidence format)
- 3 contrastive test scenarios (turn markers, emotional voice, framework avoidance)

**Things to watch out for:**
- ⚠️ Skill token budget: 4 primers + 4 phases + composition audit template may exceed typical 5000-token budget. May need progressive disclosure (front-load Phase 1, hide reference details).
- ⚠️ Defect Class 0 (Scope Expansion): Skill is blog-post-specific. Don't pre-build for documentation, READMEs, or presentations.
- ⚠️ The composition audit could become ceremony if questions are too vague. Quote-based evidence is critical — "yes/no" answers defeat the purpose.

**Areas needing further investigation:**
- How does the harness engineering draft change when the skill is applied? (First real validation)
- Can the completion pipeline validate composition audit artifacts? (May need `orch complete` awareness)
- Should the skill embed source material (investigation outputs, measurement data) or reference it?

**Success criteria:**
- ✅ Skill produces a draft that opens with story, not framework
- ✅ Turn appears in first third of draft
- ✅ Contrastive tests show measurable lift on proxy indicators (turn markers, emotional voice)
- ✅ Composition audit artifact exists with quote-based evidence (not checkbox answers)
- ✅ First real application (harness engineering rewrite) produces a structurally different piece

---

## References

**Files Examined:**
- `skills/src/worker/experiential-eval/SKILL.md` — Lightweight skill structure example
- `skills/src/worker/architect/.skillc/` — Complex skill with 5 phases, fork navigation
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/09-contradiction-detection.yaml` — Detection pattern examples, variant testing
- `.kb/models/writing-style/model.md` — 4 primers, INERT status, design rationale
- `.kb/threads/2026-03-20-writing-soft-harness-from-primers.md` — Open questions this investigation answers
- `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` — Compositional correctness gap across 3 scales
- `pkg/claims/architect_output.go` — ARCHITECT_OUTPUT.yaml format reference

**Related Artifacts:**
- **Model:** `.kb/models/writing-style/model.md` — Parent model, should be updated from INERT to TESTABLE when skill is created
- **Thread:** `.kb/threads/2026-03-20-writing-soft-harness-from-primers.md` — Thread that motivated this investigation
- **Investigation:** `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` — Compositional correctness gap generalization

---

## Investigation History

**2026-03-20:** Investigation started
- Initial question: How should a composition-level writing skill be structured, tested, and connected to the existing skill system?
- Context: 4 writing primers exist but are INERT (untested). Failed blog post diagnosed as compositional correctness problem. Thread poses design questions.

**2026-03-20:** Exploration complete — 5 forks navigated
- Forks: standalone vs modifier, one vs two skills, enforcement mechanism, testing strategy, artifact format
- Substrate consulted: principles (skill domain behavior), models (behavioral grammars, skill-content-transfer, compositional correctness), decisions (skills own domain behavior)

**2026-03-20:** Investigation completed
- Status: Complete
- Key outcome: 4-phase standalone skill with composition self-audit is the right design. Testing splits into proxy detection patterns (Tier 1) and structural artifact validation (Tier 2). LLM-as-judge deferred as future enhancement.
