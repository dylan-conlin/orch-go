# Clarifying Questions Phase

**Purpose:** Surface all ambiguities BEFORE design work begins.

**When you're in this phase:** Investigation (if any) is complete. Before starting design, explicitly identify and ask about any unclear requirements, edge cases, or integration concerns.

**Why this exists:** Asking questions during design is often too late - design decisions may already be influenced by assumptions. This phase creates a hard stop to surface ambiguities before investing effort in design.

---

## Deliverables

- **Questions documented:** All clarifying questions communicated via `bd comment`
- **Answers received:** Orchestrator or user has answered all blocking questions
- **No ambiguities:** Ready to proceed with design with clear understanding

---

## Workflow

### 1. Review What You Know

Before identifying gaps, summarize your current understanding:

**If investigation phase preceded this:**
- Read investigation file findings
- Note key architectural constraints discovered
- Identify integration points found
- List dependencies identified

**If no investigation phase:**
- Review SPAWN_CONTEXT requirements
- Note explicit constraints provided
- Identify stated scope boundaries

### 2. Identify Question Categories

Systematically consider each category for potential ambiguities:

| Category | Questions to Consider |
|----------|----------------------|
| **Edge Cases** | Empty inputs? Maximum limits? Concurrent access? Null/undefined handling? |
| **Error Handling** | What should happen when X fails? Retry behavior? User-facing error messages? |
| **Integration Points** | How does this connect to existing systems? API contracts? Data flow? |
| **Backward Compatibility** | Will this break existing functionality? Migration needed? Deprecation strategy? |
| **Performance** | Expected load? Response time requirements? Resource constraints? |
| **Security** | Authentication requirements? Authorization rules? Data sensitivity? |
| **Scope Boundaries** | What's explicitly out of scope? Deferred to future work? |

### 3. Document Questions

**Report questions via beads comment:**

```bash
bd comments add <beads-id> "QUESTION: [question with context and default assumption]"
```

**Example:**
```bash
bd comments add <beads-id> "QUESTION: Edge case - What should happen with empty input? Default assumption: return empty result"
bd comments add <beads-id> "QUESTION: Integration - Should auth middleware apply to this endpoint? Default assumption: yes, standard auth"
```

**Include default assumptions** - this allows orchestrator to quickly confirm or correct rather than answering from scratch.

### 4. Ask Questions Using Directive-Guidance Pattern

**CRITICAL: Use directive-guidance, not quiz-style questions.**

Clarifying questions are about **confirming intent**, not testing knowledge. There is no "wrong" answer - the user's response defines the requirement.

**Pattern reference:** `~/.orch/patterns/directive-guidance.md`

**❌ DON'T present neutral options (quiz-style):**
```
"How should we handle flag conflicts?"
  1. Option A
  2. Option B
  3. Option C
```
This feels like a quiz where the user might give a "wrong" answer.

**✅ DO state your recommendation with reasoning:**
```
"I'm planning to error if both --json and --format are specified, since
explicit errors are clearer than magic precedence rules. Does that match
what you want, or would you prefer different behavior?"
```
This confirms intent - user can agree or redirect.

**When using the question tool:**

The `question` tool allows you to ask the user questions during execution. Use it to gather preferences, clarify ambiguities, or get decisions on implementation choices.

**Tool interface:**
```json
{
  "questions": [
    {
      "question": "Complete question text",
      "header": "Short label (max 12 chars)",
      "options": [
        {"label": "Option text (1-5 words)", "description": "Explanation of choice"}
      ]
    }
  ]
}
```

**Usage notes:**
- Users can always select "Other" to provide custom input
- If you recommend a specific option, make it the first option and add "(Recommended)" to the label

**Example question tool usage:**
```json
{
  "questions": [{
    "question": "I'm planning to return a 429 error for rate limit violations. Does that work, or would you prefer different behavior?",
    "header": "Rate Limit",
    "options": [
      {"label": "429 error (Recommended)", "description": "Clear feedback, standard HTTP semantics"},
      {"label": "Queue requests", "description": "Better UX but adds complexity"},
      {"label": "Drop silently", "description": "Simple but user gets no feedback"}
    ]
  }]
}
```

**For complex or open-ended questions**, report via `bd comments add <beads-id> "AWAITING_ANSWERS: [details]"`.

**Do NOT proceed to design until questions are answered.**

### 5. Record Answers

When orchestrator responds, acknowledge via beads:

```bash
bd comments add <beads-id> "Answers received: [summary]. Impact on design: [brief notes]"
```

### 6. Move to Design Phase

Once all questions resolved:

1. Report phase transition: `bd comments add <beads-id> "Phase: Design "Questions resolved, proceeding with design"`

2. Output: "✅ Clarifying questions resolved, moving to Design phase"

---

## When Questions Are Not Needed

**Skip this phase (or complete quickly) when:**
- SPAWN_CONTEXT is highly detailed and explicit
- Following well-established patterns with no ambiguity
- Orchestrator pre-answered likely questions in spawn prompt
- Investigation phase already surfaced and resolved ambiguities

**Even then, quickly verify:** "Are there any edge cases, error handling, or integration concerns I should ask about?"

If genuinely nothing unclear → Document "No clarifying questions - requirements are clear" and proceed.

---

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Ask before design** | Questions during design means rework; questions before design saves time |
| **Confirm intent, don't quiz** | State your recommendation, ask if it matches intent - there's no "wrong" answer |
| **Default assumptions** | Always state what you'll assume - enables quick confirmation vs open-ended questions |
| **Structured categories** | Systematic review prevents missing important ambiguities |
| **Block on answers** | Don't proceed to design with unresolved ambiguities |
| **Document impact** | When answer received, note how it affects design approach |

---

## Completion Criteria

Before moving to Design phase, verify:

- [ ] All question categories reviewed (edge cases, errors, integration, compatibility, etc.)
- [ ] Questions communicated via `bd comment`
- [ ] Orchestrator answered all blocking questions
- [ ] Answers acknowledged via `bd comment`
- [ ] No remaining ambiguities that would affect design
- [ ] Reported via beads: `bd comments add <beads-id> "Phase: Design "Questions resolved"`

**If ANY box unchecked, clarifying questions phase is NOT complete.**

**Exception:** If genuinely no questions exist, report via `bd comments add <beads-id> "Phase: Design "No clarifying questions needed"` and proceed.
