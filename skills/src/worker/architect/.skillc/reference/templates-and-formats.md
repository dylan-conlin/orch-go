# Architect Reference: Templates and Formats

**Parent skill:** architect
**Purpose:** Detailed templates, format examples, and interactive mode patterns. Referenced from the main skill for progressive disclosure.

---

## Fork Documentation Format

```markdown
### Fork: [Decision Question]

**Options:**
- A: [Description]
- B: [Description]

**Substrate says:**
- Principle: [constraint from principles.md]
- Model: [relevant model constraint]
- Decision: [precedent if exists]

**Unknown:** [Any uncertainty that needs spiking]
```

---

## Question Generation Format

```markdown
## Blocking Questions

> **Hard cap: 3-7 questions maximum.** If you have more, you're either bikeshedding or the scope is too large.

### Q1: [Question text]
- **Authority:** implementation | architectural | strategic
- **Subtype:** factual | judgment | framing
- **What changes based on answer:** [Impact on design]
```

**Authority classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → `authority:implementation`
- Reaches to other components/agents → `authority:architectural`
- Reaches to values/direction/irreversibility → `authority:strategic`

**Subtype guidance:**
- `factual`: Can be answered by checking substrate, code, or external docs
- `judgment`: Requires evaluating tradeoffs between valid options
- `framing`: Questions the premise or direction itself

---

## Synthesis Format

```markdown
### Fork: [Decision Question]

**SUBSTRATE:**
- Principle: [X] says...
- Model: [Y] constrains...
- Decision: [Z] established...

**RECOMMENDATION:** [Option] because [reasoning from substrate]

**Trade-off accepted:** [What we're sacrificing]
**When this would change:** [Conditions that would alter recommendation]
```

---

## Recommendations Section Format

```markdown
## Recommendations

**RECOMMENDED:** [Approach name]
- **Why:** [Key reasons based on exploration]
- **Trade-off:** [What we're accepting and why that's OK]
- **Expected outcome:** [What this achieves]

**Alternative: [Other approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended given context]
- **When to choose:** [Conditions where this makes sense]

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves recurring issues (3+ prior investigations on same topic)
- This decision establishes constraints future agents might violate
- Future spawns might conflict with this decision

**Suggested blocks keywords:**
- [keyword that describes the problem domain]
- [how someone might describe this in a spawn task]
```

---

## Verification Specification Template

**When to include:** Designs that will be implemented by agents benefit from explicit verification criteria.

**Output location:** `.kb/specifications/YYYY-MM-DD-{feature}-VERIFICATION-SPEC.md`

### Full Format

```markdown
# Verification Specification: [Feature Name]

**Design Document:** [path to investigation file]

---

## Observable Behaviors

> What can be seen when this feature is working correctly?

### Primary Behavior
[One sentence describing the main observable behavior]

### Secondary Behaviors (if applicable)
- [Additional observable behaviors]

---

## Acceptance Criteria

> Pass/fail conditions for each behavior.

### AC-001: [Criterion Name]
**Behavior:** [Which observable behavior this verifies]
**Condition:** [Testable condition - MUST/SHOULD/MAY verb + measurable outcome]
**Threshold:** [Numeric threshold if applicable, or "Boolean pass/fail"]

---

## Failure Modes

> What breaks this feature and how to diagnose?

### FM-001: [Failure Name]
**Symptom:** [What agent/user observes when this fails]
**Root Cause:** [Why this happens]
**Diagnostic:** [How to confirm this is the cause]
**Fix:** [How to resolve]

---

## Evidence Requirements

| Criterion | Evidence Type | Artifact |
|-----------|---------------|----------|
| AC-001 | [test output / screenshot / log] | [artifact path or description] |
```

### Simplified Format (for simple features)

```markdown
# Verification Specification: [Feature Name]

## Observable Behavior
[What can be seen when working correctly - one sentence]

## Acceptance Criterion
[Testable pass/fail condition - one criterion]

## Failure Mode
**Symptom:** [What you see when broken]
**Fix:** [How to resolve]

## Evidence
[What artifact proves it works: test output / screenshot / etc.]
```

---

## Interactive Mode Patterns

### Phase 1: Understanding (Interactive)

- Ask ONE question at a time to refine the idea
- Always include your recommendation with reasoning
- Present alternatives naturally in your question
- Gather: Purpose, constraints, success criteria

**Example (natural conversation with recommendation):**
```
"I recommend storing auth tokens in httpOnly cookies - they're secure against XSS
attacks and work well with server-side rendering. What's your preference?

Other options to consider:
- localStorage: More convenient but vulnerable to XSS
- sessionStorage: More secure but clears on tab close
- Server-side sessions: Most secure but requires Redis/session store

What matters most for your use case - security, convenience, or compatibility?"
```

### Phase 2: Exploration (Interactive)

- Use natural conversation with recommendation (question tool as fallback)
- Propose 2-3 approaches with your recommendation
- For each: Core architecture, trade-offs, complexity assessment
- Lead with recommendation and reasoning

**Example:**
```
"Based on your requirements and existing infrastructure,
I recommend the **Hybrid approach with background jobs**. Here's why:

Recommended: Hybrid with background jobs
- Async processing reliability without operational complexity
- Integrates with existing Sidekiq setup
- Moderate complexity - team already knows this pattern

Alternative 1: Event-driven with message queue
- Most scalable but high operational complexity

Alternative 2: Direct API calls with retry
- Simplest but less reliable

Which approach resonates with you?"
```

### Question Tool Interface

**Use the question tool only if:**
- Dylan seems overwhelmed by options
- Need to force explicit choice
- Structured comparison would clarify

```json
{
  "questions": [{
    "question": "Complete question text",
    "header": "Short label (max 12 chars)",
    "options": [
      {"label": "Option (1-5 words)", "description": "Explanation"}
    ]
  }]
}
```
- Make recommended option first with "(Recommended)" in label
- Users can always select "Other" for custom input
