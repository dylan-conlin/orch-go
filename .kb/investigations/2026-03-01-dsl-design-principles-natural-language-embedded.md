# DSL Design Principles: Natural-Language-Embedded Domain Languages

**TLDR:** There is a rich lineage of "documents that function as language specifications for intelligent readers" — from Knuth's literate programming (1984) through Controlled Natural Languages (1995), RFC 2119 modal keywords (1997), Gherkin/BDD (2008), to the current convergence point: Strands Agent SOPs (2025) and Agent Behavioral Contracts (2025). The pattern works when it respects three principles: domain alignment, graduated formality, and constraint-based execution. It fails when it drifts toward general-purpose ambition or abandons the domain vocabulary that gave it power.

## D.E.K.N. Summary

- **Delta:** Mapped the full lineage from literate programming to modern agent SOPs; identified the design principles that make natural-language-embedded DSLs effective; found direct precedents for "a document that functions as a language specification for an intelligent reader"
- **Evidence:** Seven distinct precedent systems analyzed (literate programming, CNLs, RFC 2119, Gherkin, Design by Contract, Agent SOPs, Agent Behavioral Contracts); Strands Agent SOPs are the closest existing precedent to the orch-go skill/CLAUDE.md pattern
- **Knowledge:** Three core principles (domain alignment, graduated formality, constraint-based execution); five failure modes; the "specification spectrum" framework for positioning any document-as-DSL
- **Next:** Apply findings to evaluate orch-go's CLAUDE.md and skill system against these principles; consider whether the skill format should adopt RFC 2119 keywords more systematically

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

**Status:** Complete
**Question:** What are the design principles for effective DSLs, especially those embedded in natural language? Are there precedents for documents that function as language specifications for intelligent readers?

---

## Finding 1: The Specification Spectrum

DSLs exist on a spectrum from fully formal to fully natural:

```
Formal ◄──────────────────────────────────────────────► Natural
  │                                                         │
  SQL, RegEx    Gherkin    RFC 2119    Agent SOPs    Prose
  │             │          │           │              │
  Machine-      Constrained Graduated  Structured     Ambiguous
  parseable     vocabulary  formality  natural lang
```

The interesting zone — and the one most relevant to the question — is the middle-right: **structured natural language with formalized constraint vocabulary**. This is where a document can function as both human-readable prose AND a behavioral specification for an intelligent reader (human expert or LLM).

### Key insight

The traditional DSL literature (Fowler, Mernik et al.) draws a hard line between **internal DSLs** (embedded in a host language's syntax) and **external DSLs** (custom parsers). But there's a third category emerging that neither framework anticipated: **documents that are specifications for cognitive processors** — readers who parse meaning, not syntax. This category includes CLAUDE.md files, skill definitions, system prompts, and Agent SOPs.

---

## Finding 2: Seven Precedent Systems

### 2a. Literate Programming (Knuth, 1984)

Donald Knuth's literate programming introduced the idea that a program should be written as a document for human readers, with code embedded within prose. The document IS the program, but organized for human comprehension rather than compiler requirements.

**Relevance:** First precedent for "document as specification." But literate programs still needed formal compilation — the natural language was commentary, not specification.

**Limitation:** The natural language explains but doesn't constrain. A reader could ignore the prose and extract just the code.

### 2b. Controlled Natural Languages — Attempto Controlled English (1995)

ACE is a subset of English with restricted syntax that maps unambiguously to first-order logic. You write what looks like English ("Every customer who buys a product gets a discount") but it's formally parseable.

**Relevance:** Proved that natural language CAN be formal specification — if you constrain the vocabulary and grammar sufficiently. ACE texts can be automatically translated into logic, verified, and queried.

**Limitation:** ACE reads like stilted English. The constraint on vocabulary makes it powerful for machines but awkward for humans. It solves ambiguity by removing expressiveness.

### 2c. RFC 2119 Modal Keywords (1997)

RFC 2119 defined MUST, SHOULD, MAY (and their negations) as formal requirement levels embedded in otherwise natural-language specifications. This is the most widely adopted "graduated formality" system in computing — nearly every internet standard uses it.

**Relevance:** This is the purest example of "a document that functions as a language specification for an intelligent reader." RFC authors write English prose, but the capitalized keywords carry precise, formally-defined obligation levels. The reader (human implementer) must understand both the prose context and the keyword formality.

**Why it works:**
- Minimal vocabulary (10 keywords)
- Clear graduation (absolute requirement → recommendation → optional)
- Embedded in natural language, not replacing it
- The keywords ADD precision to prose rather than constraining it

### 2d. Gherkin / BDD (2008)

Cucumber's Given/When/Then syntax bridges domain experts and developers. Specifications are written in constrained natural language that's both human-readable and machine-executable.

**Relevance:** Proved that natural-language specifications can be executable. The key insight: you don't need full NLP — a small constrained vocabulary (Given, When, Then, And, But) plus domain-specific step definitions creates a powerful specification language.

**Design principle:** The constrained keywords provide STRUCTURE; the natural language within steps provides MEANING. Neither alone is sufficient.

### 2e. Design by Contract / Eiffel (Meyer, 1986)

Bertrand Meyer's Design by Contract embeds formal specifications (preconditions, postconditions, invariants) within code. The specification IS the documentation IS the runtime check.

**Relevance:** Introduced the three-layer contract model (pre/post/invariant) that shows up in every subsequent specification system, including Agent Behavioral Contracts.

### 2f. Strands Agent SOPs (AWS, 2025)

**This is the closest direct precedent to the question.** Agent SOPs are markdown documents that function as behavioral specifications for AI agents. They use:
- Natural language overview (human-readable context)
- Parameterized inputs (reusability)
- Numbered steps (sequential workflow)
- RFC 2119 keywords in constraint subsections (graduated formality)
- Multi-modal distribution (same document → MCP tools, skills, system prompts)

Amazon teams use thousands of SOPs internally. The format emerged organically from internal builder communities because it made agent behavior "both more reliable and easier to evolve."

**Why it works:**
- **Dual audience:** Readable by humans, executable by LLMs
- **Graduated formality:** Natural language for context, RFC 2119 for constraints
- **Parameterization:** Same spec, different contexts
- **Progressive disclosure:** Steps build complexity incrementally

### 2g. Agent Behavioral Contracts (2025)

Academic formalization of the gap between natural language prompts and formal software contracts. Defines contracts as C = (Preconditions, Invariants, Governance, Recovery) with probabilistic satisfaction guarantees.

**Relevance:** Acknowledges that LLM-based agents can't provide deterministic contract satisfaction. Introduces "(p,δ,k)-satisfaction" — contracts hold with probability p, within tolerance δ, recovering within k steps. This is the mathematical foundation for why natural-language specifications can work with AI: you don't need perfect compliance, you need bounded drift.

---

## Finding 3: Three Design Principles for Natural-Language-Embedded DSLs

Across all seven precedent systems, three principles consistently determine success:

### Principle 1: Domain Alignment

The language's vocabulary, structure, and abstractions must match the mental model of the domain expert. This is the oldest DSL principle (Mernik et al., 2005) and the most consistently validated.

**What this means for document-as-DSL:** The document's sections, keywords, and organizational patterns should mirror how a practitioner thinks about the domain. RFC 2119 works because implementers already think in terms of "must do this, should do that." Gherkin works because testers already think in terms of "given this setup, when this happens, then expect this."

**Anti-pattern:** Importing vocabulary from a different domain. A specification written in the language of formal methods will fail as a DSL for operations engineers, even if it's technically precise.

### Principle 2: Graduated Formality

Effective natural-language DSLs don't try to be uniformly formal OR uniformly natural. They use **natural language for context and explanation** and **constrained vocabulary for behavioral specification**.

**The RFC 2119 pattern:** Most of the document is prose. Specific obligation points use MUST/SHOULD/MAY. The reader's attention is directed to constraint points by the keyword formality standing out from surrounding prose.

**The Agent SOP pattern:** Overview sections are conversational. Constraint subsections within steps use RFC 2119 keywords. Parameters are structured. The formality gradient matches the precision requirement.

**Anti-pattern:** Uniform formality (everything is MUST — readers stop distinguishing) or uniform informality (everything is prose — readers can't identify requirements).

### Principle 3: Constraint-Based Execution

Rather than specifying every step procedurally, effective document-DSLs define constraints that bound the reader's behavior while leaving implementation details flexible.

**Why this matters for intelligent readers:** An LLM (or human expert) doesn't need step-by-step instructions for things within their competence. What they need is: what MUST be true, what SHOULD be avoided, and what the boundaries are. The specification defines the constraint space; the reader fills in the execution.

**This is the key difference from traditional DSLs:** A traditional DSL specifies procedure. A document-DSL for an intelligent reader specifies constraints.

---

## Finding 4: Five Failure Modes

### Failure 1: General-Purpose Drift

The most common DSL failure. The language starts domain-specific, then accumulates features until it's a poor general-purpose language. Matt Rickard: "The future problem space is unpredictable. If you design a DSL that perfectly fits the problems today, it will be obsolete quickly."

**For document-DSLs:** This manifests as the specification trying to cover every edge case, growing from a focused behavioral contract into an encyclopedia. The CLAUDE.md that tries to specify everything becomes as useless as the one that specifies nothing.

### Failure 2: Vocabulary Orphaning

The DSL's vocabulary drifts from the domain's actual terminology. Keywords that made sense at design time become jargon that neither domain experts nor implementers recognize.

**For document-DSLs:** When a specification uses terms like "Phase: Planning" but the actual workflow doesn't have discrete phases, or when constraint keywords are used so loosely they lose their meaning.

### Failure 3: Abstraction Mismatch

The DSL operates at the wrong level of abstraction — too high (vague platitudes) or too low (micromanaging steps the reader already knows).

**Addy Osmani's finding:** Analysis of 2,500+ AI agent specification files found the most common problem was being "too vague." But the second most common was over-specification that the agent couldn't follow because it conflicted with the agent's own judgment about implementation.

### Failure 4: Ecosystem Isolation

Traditional DSL failure: the language can't integrate with existing tools. For document-DSLs: the specification format can't be consumed by multiple systems. Agent SOPs solved this with multi-modal distribution (same markdown → MCP tools, Cursor commands, system prompts).

### Failure 5: Formality Uniformity

Either everything is a hard constraint (MUST fatigue — readers treat everything as suggestion) or nothing is (readers can't identify actual requirements). RFC 2119 explicitly warns: "These terms MUST only be used where it is actually required for interoperation or to limit behavior which has potential for causing harm."

---

## Finding 5: The Emerging Pattern — Document as Language Specification

The question "Are there precedents for 'a document that functions as a language specification for an intelligent reader'?" has a clear answer: **yes, and the pattern is converging rapidly**.

The convergence point has these characteristics:

1. **Markdown as substrate** — Not XML, not YAML, not custom syntax. Markdown because it's readable by humans, parseable by machines, and universally understood.

2. **RFC 2119 keywords for graduated formality** — The 30-year-old standard is finding new life as the constraint vocabulary for AI agent specifications.

3. **Section-based progressive disclosure** — Overview → Parameters → Steps → Constraints, mirroring how both humans and LLMs process hierarchical information.

4. **Constraint-based rather than procedural** — Specifying boundaries and obligations rather than step-by-step procedures.

5. **Dual audience** — Written for humans to read and AI to execute, with both audiences requiring different things from the same document (humans need context and reasoning; AI needs structure and constraints).

6. **Probabilistic compliance** — Unlike traditional formal specifications that demand deterministic satisfaction, document-DSLs for AI accept bounded deviation. Agent Behavioral Contracts formalize this as (p,δ,k)-satisfaction.

---

## Test Performed

This is a research investigation, not a code investigation. The "test" was systematic search across academic literature, industry practice, and current open-source projects to identify precedent systems and extract principles. Sources consulted:

- Academic: Mernik et al. (2005), Knuth (1984), Meyer (1986), Fuchs et al. (1995), Agent Behavioral Contracts (2025)
- Industry: Martin Fowler's DSL work, Cucumber/Gherkin, RFC 2119, Strands Agent SOPs
- Current practice: Addy Osmani's analysis of 2,500+ agent specs, Amazon's internal SOP adoption

## Conclusion

The concept of "a document that functions as a language specification for an intelligent reader" is not novel — it has a lineage stretching back to Knuth's literate programming. What IS novel is the convergence of three factors:

1. **LLMs as readers** — For the first time, the "intelligent reader" is a machine that processes natural language with near-human comprehension, making document-DSLs executable without compilation.

2. **RFC 2119 as universal constraint vocabulary** — The keywords designed for human implementers in 1997 work equally well for LLM agents in 2026.

3. **Markdown as universal substrate** — The format is simple enough for any reader (human or AI) and structured enough for multi-modal distribution.

The orch-go skill system (SKILL.md files with structured sections, constraint keywords, and progressive disclosure) is an instance of this pattern, as are CLAUDE.md files. The Strands Agent SOPs formalize the same intuition with explicit RFC 2119 adoption and parameterization.

**The design principles that make these work:**
- Domain alignment (vocabulary matches practitioner mental model)
- Graduated formality (natural language for context, constrained keywords for obligations)
- Constraint-based execution (define boundaries, not procedures)

**The failure modes to watch for:**
- General-purpose drift (specification tries to cover everything)
- MUST fatigue (overuse of constraint keywords dilutes their meaning)
- Abstraction mismatch (too vague or too prescriptive)
