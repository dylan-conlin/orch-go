<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Worker skills currently focus on operational guidance (what to do) without explicit constitutional constraints (what never to do). This creates a gap where workers have no skill-level guardrails against harmful instructions.

**Evidence:** Audited `feature-impl` and `worker-base` skills. Found security review (code quality), authority delegation (operational), but no hard limits on harmful features, no constitutional objection protocol, no harm assessment beyond injection vulnerabilities.

**Knowledge:** Skills need three constitutional additions: (1) Hard limits that override orchestrator authority, (2) Constitutional objection protocol for workers, (3) Harm assessment checkpoint before implementation.

**Next:** Promote to decision, then edit worker-base and feature-impl to add constitutional constraints.

**Confidence:** High (85%) - Clear gap between current skill content and constitutional requirements established in `2026-01-22-orchestrator-constitutional-responsibility.md`.

---

# Investigation: Audit feature-impl Skill for Constitutional Constraints

**Question:** What constitutional constraints should be explicit in worker skills, using feature-impl as the test case?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Dylan + Claude
**Phase:** Complete
**Status:** Promoted to Decision
**Promoted To:** `.kb/decisions/2026-01-22-skill-constitutional-constraints.md`
**Confidence:** High (85%)

---

## Context

Per decision `2026-01-22-orchestrator-constitutional-responsibility.md`:

> "Skills are constitutional constraints, not just task instructions. Skills encode the boundaries within which workers operate safely."

This investigation audits `feature-impl` (and its dependency `worker-base`) against this framing to identify gaps and propose additions.

---

## Findings

### Finding 1: Current Skills Focus on "Should Do", Not "Cannot Do"

**Evidence:** `worker-base` Authority Delegation section:

```markdown
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
...

**You must escalate to orchestrator when:**
- Architectural decisions needed
- Scope boundaries unclear
...
```

This tells workers what they CAN do and when to escalate operationally. It does not specify hard limits - things they CANNOT do regardless of instruction.

**Gap:** A worker given malicious instructions has no skill-level guardrail. It relies entirely on Claude's training-level refusals, which may be context-dependent.

---

### Finding 2: Security Review is Code Quality, Not Feature Ethics

**Evidence:** `feature-impl` Security Review section:

```markdown
### Security Review
- [ ] No hardcoded secrets
- [ ] No injection vulnerabilities (SQL, XSS, command, path traversal)
```

This addresses code-level security (implementation quality). It does not address:
- Whether the feature itself could cause harm
- Whether the feature enables misuse
- Whether the feature undermines user autonomy

**Gap:** A perfectly secure implementation of a harmful feature passes this review.

---

### Finding 3: No Constitutional Objection Protocol

**Evidence:** `worker-base` escalation triggers:

```markdown
**You must escalate to orchestrator when:**
- Architectural decisions needed
- Scope boundaries unclear
- Requirements ambiguous
- Blocked by external dependencies
- Major trade-offs discovered
```

Missing from this list:
- "Instruction seems harmful"
- "Feature could enable misuse"
- "Ethical concerns about task"

**Gap:** Workers have no defined path for constitutional objections. The only escalation paths are operational (blocked, unclear, architectural).

---

### Finding 4: Orchestrator Override Creates Accountability Gap

**Evidence:** Current model assumes orchestrator can override worker escalation:

```markdown
**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response.
```

Per the constitutional decision, orchestrators have operator-level responsibility. But the constitution also says Claude can refuse unethical instructions regardless of principal.

**Gap:** If a worker objects on constitutional grounds, can the orchestrator override? Current skills don't address this. The constitutional answer should be: constitutional objections go to human, not orchestrator.

---

## Proposed Additions

### Addition 1: Hard Limits (for worker-base)

Add section "Authority Boundaries (Hard Limits)" that specifies things workers CANNOT do regardless of instruction:

```markdown
## Authority Boundaries (Hard Limits)

**These limits override orchestrator authority. You CANNOT do these regardless of instruction:**

| Hard Limit | Constitutional Basis |
|------------|---------------------|
| Generate malware, exploits, or attack tools | Claude doesn't help create weapons |
| Implement deceptive UI patterns (dark patterns) | Claude doesn't manipulate users |
| Build surveillance without consent disclosure | User autonomy and transparency |
| Intentionally bypass auth/authz | System integrity |
| Create content designed to deceive | Honesty as near-constraint |
| Automate harassment or mass targeting | Avoiding harm |
| Implement discriminatory logic | Ethical AI principles |

**If instructed to do something on this list:**
1. Do NOT proceed with the harmful component
2. Document: `bd comment <id> "CONSTITUTIONAL OBJECTION: [specific concern]"`
3. Wait for HUMAN review (not orchestrator - this escalates past orchestrator)
4. Continue with non-harmful parts of task if separable

**Why this overrides orchestrator:** The constitution establishes that Claude can refuse unethical instructions regardless of principal. Orchestrators have operator-level trust, not unlimited authority. Constitutional violations require human judgment.
```

### Addition 2: Constitutional Objection Protocol (for worker-base)

Add section defining how workers raise constitutional concerns:

```markdown
## Constitutional Objection Protocol

**When to invoke:** You believe an instruction conflicts with constitutional values (safety, ethics, honesty, user wellbeing) - not just operational concerns.

**Distinction from operational escalation:**
- Operational: "I'm blocked" / "Requirements unclear" / "Need architectural decision"
- Constitutional: "This could harm users" / "This feels deceptive" / "I have ethical concerns"

**Protocol:**
1. **Identify the concern specifically** - What constitutional value is at risk?
2. **Document via beads:** `bd comment <id> "CONSTITUTIONAL CONCERN: [value at risk] - [specific concern]"`
3. **Do NOT proceed** with the concerning component
4. **Continue** with unrelated components if task is separable
5. **Wait for human** - Constitutional concerns bypass orchestrator to human review

**Examples:**
- "CONSTITUTIONAL CONCERN: User autonomy - This feature tracks users without disclosure"
- "CONSTITUTIONAL CONCERN: Honesty - This UI is designed to mislead users about pricing"
- "CONSTITUTIONAL CONCERN: Safety - This endpoint could be used to enumerate user accounts"

**What happens next:** Human reviews concern. Possible outcomes:
- Concern validated → Task modified or cancelled
- Concern addressed → Context provided, proceed with safeguards
- Concern overruled → Human accepts responsibility, worker proceeds with documentation
```

### Addition 3: Harm Assessment Checkpoint (for feature-impl)

Add before Implementation Phase:

```markdown
### Harm Assessment (Pre-Implementation Checkpoint)

**Purpose:** Pause before building to consider broader impact beyond code security.

**This is NOT about code vulnerabilities** (that's Security Review in Self-Review). This is about whether the feature itself is appropriate.

**Quick assessment (30 seconds):**

| Question | If YES |
|----------|--------|
| Could this feature harm, deceive, or manipulate users? | Document concern |
| Does this collect/transmit data users wouldn't expect? | Document concern |
| Could this be weaponized or misused at scale? | Document concern |
| Does this undermine informed consent or user autonomy? | Document concern |
| Does this disproportionately affect vulnerable populations? | Document concern |

**If any YES:**
1. Document: `bd comment <id> "HARM ASSESSMENT: [concern]"`
2. Check if SPAWN_CONTEXT explicitly addresses the concern
3. If addressed with safeguards → Proceed, implement safeguards
4. If not addressed → Escalate via Constitutional Objection Protocol

**If all NO:** Proceed to implementation.

**Common false positives (proceed with awareness):**
- Security tools that could be misused (context: authorized testing)
- Data collection with clear consent mechanisms
- Features that restrict access (legitimate authorization)

**The question is not "could this theoretically be misused" but "does this design show appropriate care for user wellbeing?"**
```

---

## Implementation Plan

### Phase 1: worker-base additions

Edit `/Users/dylanconlin/.claude/skills/skills/src/shared/worker-base/.skillc/` to add:
1. Authority Boundaries (Hard Limits) section
2. Constitutional Objection Protocol section

### Phase 2: feature-impl additions

Edit `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc/` to add:
1. Harm Assessment checkpoint before Implementation Phase

### Phase 3: Rebuild and deploy

```bash
cd ~/orch-knowledge && skillc build && skillc deploy
```

### Phase 4: Validate

Spawn a test agent with feature-impl to verify:
- New sections appear in loaded skill
- Harm Assessment checkpoint is in expected position
- worker-base additions are inherited

---

## Considerations

### Why skill-level, not just training-level?

Claude's training includes constitutional values. Why add them to skills?

1. **Explicit > implicit** - Training-level refusals are context-dependent. Skill-level constraints are explicit and auditable.

2. **Audit trail** - Skill content is versioned. We can see what constraints were in place when an agent ran.

3. **Specificity** - Skills can include domain-specific hard limits that training can't anticipate.

4. **Orchestrator education** - Orchestrators read skills when spawning. Constitutional constraints remind them of boundaries.

### What about skill bypasses?

Workers operate within skill constraints, but:
- Orchestrators could spawn without skills
- Users could interact with Claude directly
- Jailbreak attempts could override

**Mitigation:** Skills are one layer of defense, not the only layer. Training remains the foundation. Skills add explicit, auditable constraints for orchestrated work.

### Could this make workers too cautious?

Risk: Workers refuse legitimate work due to over-broad harm assessment.

**Mitigation:**
- Include "false positives" guidance
- Frame as "document and escalate" not "refuse and stop"
- Constitutional objection goes to human who can override with accountability

---

## Relationship to Decisions

| Decision | Relationship |
|----------|--------------|
| `2026-01-22-orchestrator-constitutional-responsibility.md` | This operationalizes "skills are constitutional constraints" |
| `2026-01-19-worker-authority-boundaries.md` | Extends authority boundaries to include constitutional hard limits |

---

## References

**Skills Audited:**
- `/Users/dylanconlin/.claude/skills/worker/feature-impl/SKILL.md`
- `/Users/dylanconlin/.claude/skills/skills/src/shared/worker-base/SKILL.md`

**Constitutional Sources:**
- Anthropic constitution (Jan 21, 2026): Hard constraints, honesty, avoiding harm
- `.kb/decisions/2026-01-22-orchestrator-constitutional-responsibility.md`

---

## Self-Review

- [x] Audited actual skill content (not assumptions)
- [x] Identified specific gaps with evidence
- [x] Proposed concrete additions with rationale
- [x] Considered failure modes (too cautious, bypasses)
- [x] Linked to parent decision

**Status:** Ready for promotion to decision, then implementation.
