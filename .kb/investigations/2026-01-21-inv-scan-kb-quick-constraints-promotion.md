<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Scanned 149 kb quick constraints and identified 3 strong promotion candidates for principles.md that meet all criteria (tested, generative, not derivable, violation hurts).

**Evidence:** Compared each constraint against existing principles.md content and 4-criteria test; most constraints are implementation-specific or derivable from existing principles.

**Knowledge:** Strong candidates: (1) "AI Explanations Increase Trust Even When Wrong" - new epistemic principle about validation loops, (2) "Critical Paths Need Escape Hatches" - defense-in-depth infrastructure pattern, (3) "Tool Call Examples Corrupt Function Calling" - LLM prompt authoring constraint.

**Next:** Present candidates to orchestrator/Dylan for promotion decision; if approved, add to ~/.kb/.principlec/ and rebuild with principlec build.

**Promote to Decision:** recommend-yes - establishes criteria for constraint→principle promotion and identifies specific candidates

---

# Investigation: Scan KB Quick Constraints for Promotion to Principles

**Question:** Which kb quick constraints (149 total) should be promoted to ~/.kb/principles.md?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None - awaiting orchestrator decision on candidates
**Status:** Complete

---

## Findings

### Finding 1: Most constraints are implementation-specific

**Evidence:** Of 149 constraints reviewed:
- ~60% are orch-go/opencode implementation details (e.g., "OpenCode x-opencode-directory header returns ALL disk sessions")
- ~25% are already covered by existing principles (e.g., "High patch density signals missing coherent model" → Coherence Over Patches)
- ~15% are potential candidates requiring deeper analysis

**Source:** `.kb/quick/entries.jsonl` - full scan of type:constraint entries

**Significance:** The low promotion rate (~2%) is expected - principles.md captures universal patterns, not implementation specifics. Most constraints are correctly captured as project-level kb quick entries.

---

### Finding 2: Three constraints meet all four criteria

**Evidence:** Applied the criteria from principles.md:
1. **Must be tested** - emerged from actual problems
2. **Must be generative** - guides future decisions
3. **Must not be derivable** - from existing principles
4. **Must have teeth** - violation causes real problems

Three constraints pass all four:

| Constraint | ID | Tested | Generative | Not Derivable | Violation Hurts |
|------------|-----|--------|------------|---------------|-----------------|
| AI explanations increase trust even when wrong | kb-aa7174 | ✓ Nate Jones article + validation loops | ✓ Guides verification workflow | ✓ New epistemic insight | ✓ Creates verification loops |
| Critical paths need escape hatches | kb-bf4f55 | ✓ Jan 10 infrastructure incident | ✓ Guides resilient design | ✓ Defense-in-depth pattern | ✓ Cascade failures |
| Tool call examples in prompts corrupt function calling | kb-b7198e | ✓ Gemini investigation | ✓ Guides prompt authoring | ✓ LLM training dynamic | ✓ Tools stop working |

**Source:** Cross-referenced against ~/.kb/principles.md (930 lines) for derivability analysis

**Significance:** These three represent genuinely new insights that would benefit any LLM-based system, not just orch-go.

---

### Finding 3: Several constraints are corollaries of existing principles

**Evidence:** Constraints that SEEM universal but are derivable:
- "Session idle ≠ agent complete" (kb-bef2d9) → Derivable from "Track Actions, Not Just State"
- "High patch density signals missing model" (kb-9865a3) → IS "Coherence Over Patches" principle
- "Ask 'should we' before 'how do we'" (kb-c12998) → IS "Premise Before Solution" principle
- "Separate observations from hypotheses" (kb-cbe677) → IS "Evidence Hierarchy" principle
- "Synthesis is orchestrator work" (kb-0370a3) → IS "Understanding Through Engagement" principle

**Source:** Direct text comparison with existing principles in ~/.kb/principles.md

**Significance:** Many constraints are correctly NOT promoted because they're applications of existing principles to specific situations. The principles system is working as intended - capturing universal patterns while kb quick captures instances.

---

### Finding 4: Two additional candidates require orchestrator judgment

**Evidence:** Constraints that partially meet criteria:

1. **"Outcome Language Preserves Perspective"** (kb-279253)
   - "Orchestrator goals should be outcome-focused not action-focused - action verbs cue worker behavior and cause frame collapse"
   - Tested: ✓ (Jan 15 frame collapse)
   - Question: Is this derivable from "Perspective is Structural" or a new mechanism?

2. **"Replaceable By Design"** (kb-c59f18)
   - "No sacred cows - everything is replaceable at the right cost"
   - Tested: ✓ (ongoing evolution)
   - Question: Is this a meta-principle or obvious good practice?

**Source:** kb-279253, kb-c59f18 from `.kb/quick/entries.jsonl`

**Significance:** These require human judgment on whether they're universal enough for principles.md vs. useful as documented decisions.

---

## Synthesis

**Key Insights:**

1. **The 2% promotion rate validates the kb quick → principles flow** - Most constraints belong at the project level. The high bar for principles (4 criteria) filters appropriately.

2. **LLM-First principles section has gaps** - Two of three strong candidates (AI validation loops, prompt examples) are LLM-specific epistemic insights not covered by existing principles.

3. **System Design principles are comprehensive** - "Critical paths need escape hatches" would fit in System Design section alongside Graceful Degradation.

**Answer to Investigation Question:**

Three constraints should be promoted to principles.md:

1. **"AI Explanations Increase Trust Even When Wrong"** → Add to LLM-First Principles
2. **"Critical Paths Need Escape Hatches"** → Add to System Design Principles
3. **"Tool Call Examples Corrupt Function Calling"** → Add to LLM-First Principles

Two additional candidates require orchestrator judgment:
- "Outcome Language Preserves Perspective" - may be corollary of existing
- "Replaceable By Design" - may be obvious meta-principle

---

## Structured Uncertainty

**What's tested:**

- ✅ All 149 constraints scanned (verified: grep + jq extraction)
- ✅ Existing principles.md reviewed for derivability (verified: read full file)
- ✅ Four-criteria test applied to each candidate (verified: documented in findings)

**What's untested:**

- ⚠️ Whether proposed principles would have prevented incidents (not retroactively testable)
- ⚠️ Whether principles wording is optimal (first draft)
- ⚠️ Cross-project applicability beyond orch-go (assumed but not verified)

**What would change this:**

- Finding that AI validation loop is covered by existing Provenance principle
- Finding that "escape hatches" is covered by Graceful Degradation
- Discovery of better principle wording from reviewing incidents

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add 3 principles via principlec** - Create principle source files in ~/.kb/.principlec/src/ and rebuild

**Why this approach:**
- Follows established principles.md build process
- Preserves provenance chain (constraint → principle)
- Enables future automated promotion workflow

**Implementation sequence:**
1. Create principle source files for each candidate
2. Add provenance row to principles.md provenance table
3. Run `principlec build` to regenerate

### Proposed Principle Texts

**1. AI Validation Loops (LLM-First)**

```markdown
### AI Validation Loops

AI-generated explanations increase trust even when wrong. Verification cannot rely solely on AI explanations of AI work.

**The test:** "Would I accept this verification if the explainer were human?"

**What this means:**

- AI explanations activate the same trust mechanisms as human expert explanations
- Experts can rationalize AI outputs, creating closed verification loops
- Verification must include external evidence, not just explanation quality

**What this rejects:**

- "The agent explained why it works" (explanation is not verification)
- "The reasoning is coherent" (coherence doesn't imply correctness)
- "I reviewed the AI's review" (meta-review is still a closed loop)

**Why this is foundational:** Provenance says trace to external evidence. This principle says *why* internal AI evidence is insufficient - the trust mechanism itself is compromised.
```

**2. Escape Hatches (System Design)**

```markdown
### Escape Hatches

Critical paths need independent secondary paths. When infrastructure can fail, build escape hatches that don't depend on what failed.

**The test:** "If the primary path fails, can I complete this work through another path?"

**What this means:**

- Critical infrastructure needs redundant execution paths
- Escape hatches must be independent of primary infrastructure
- "Independent" means: doesn't share failure modes, provides visibility, can complete work

**What this rejects:**

- Single-path critical infrastructure (all eggs in one basket)
- "We'll fix it when it breaks" (can't fix from inside broken infrastructure)
- Backup paths that share dependencies with primary (correlated failure)

**The pattern:** Primary path (daemon + OpenCode API) + Escape hatch (manual claude CLI spawning)

**Why this is distinct from Graceful Degradation:** GD says core works without optional layers. Escape Hatches says critical paths need *independent alternative paths* for when core itself fails.
```

**3. Prompt Pollution (LLM-First)**

```markdown
### Prompt Pollution

Prompt examples showing tool calls as text train LLMs to output text instead of invoking tools.

**The test:** "Does this prompt example show a tool call as text in the content?"

**What this means:**

- LLMs learn patterns from examples, including anti-patterns
- Examples like `[tool_call: search("query")]` in prompt text are treated as desired output format
- The model mimics the example format instead of using actual function calling API

**What this rejects:**

- "I'll show examples of what tool calls look like" (teaches text mimicry)
- "This helps the model understand the tools" (it learns the wrong thing)
- "The instructions say to use the tools" (examples override instructions)

**Why this matters:** Infrastructure Over Instruction says tools enforce behavior. This principle says prompts can *corrupt* tool behavior by providing conflicting examples.
```

---

## References

**Files Examined:**
- `.kb/quick/entries.jsonl` - All 149 constraint entries scanned
- `~/.kb/principles.md` - Full 930-line principles document for derivability analysis

**Commands Run:**
```bash
# Count total entries
wc -l .kb/quick/entries.jsonl
# Result: 665 total, 149 constraints

# Extract constraint content
grep '"type":"constraint"' .kb/quick/entries.jsonl | jq -r '"\(.id)|\(.content)|\(.reason)"'
```

**Related Artifacts:**
- **Decision:** kb-aa7174 - "Verification cannot rely solely on AI-generated explanations"
- **Decision:** kb-bf4f55 - "Critical paths need independent escape hatches"
- **Decision:** kb-b7198e - "Prompt examples showing tool calls as text train LLMs to output text"
- **Decision:** kb-279253 - "Orchestrator goals should be outcome-focused"
- **Decision:** kb-c59f18 - "No sacred cows - everything is replaceable"

---

## Investigation History

**2026-01-21 23:45:** Investigation started
- Initial question: Which kb quick constraints should be promoted to principles.md?
- Context: Task spawned by orchestrator for kb hygiene

**2026-01-21 23:50:** Scanned all 149 constraints
- Extracted via grep + jq from entries.jsonl
- Categorized by promotion potential

**2026-01-21 23:55:** Applied four-criteria test
- Identified 3 strong candidates + 2 borderline
- Most constraints are implementation-specific or derivable

**2026-01-21 00:00:** Investigation completed
- Status: Complete
- Key outcome: 3 promotion candidates identified with proposed principle text
