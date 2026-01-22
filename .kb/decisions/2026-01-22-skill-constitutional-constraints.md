# Decision: Skill Constitutional Constraints

**Date:** 2026-01-22
**Status:** Accepted
**Context:** Per `2026-01-22-orchestrator-constitutional-responsibility.md`, skills are constitutional constraints, not just task instructions. This decision specifies what constitutional content skills must include.

## Decision

**Worker skills must include three constitutional components:**

1. **Hard Limits** - Actions workers cannot take regardless of instruction
2. **Constitutional Objection Protocol** - How workers raise ethical concerns
3. **Harm Assessment** (implementation skills) - Pre-implementation checkpoint for feature ethics

## Hard Limits (worker-base)

Workers CANNOT do these regardless of instruction:

| Hard Limit | Constitutional Basis |
|------------|---------------------|
| Generate malware, exploits, or attack tools | Claude doesn't create weapons |
| Implement deceptive UI patterns (dark patterns) | Claude doesn't manipulate users |
| Build surveillance without consent disclosure | User autonomy and transparency |
| Intentionally bypass authentication/authorization | System integrity |
| Create content designed to deceive | Honesty as near-constraint |
| Automate harassment or mass targeting | Avoiding harm |
| Implement discriminatory logic | Ethical AI principles |

**Key principle:** These limits override orchestrator authority. Constitutional violations require human judgment, not orchestrator override.

## Constitutional Objection Protocol (worker-base)

**Trigger:** Worker believes instruction conflicts with constitutional values (safety, ethics, honesty, user wellbeing).

**Distinction from operational escalation:**
- Operational: "I'm blocked" / "Requirements unclear" / "Need decision"
- Constitutional: "This could harm users" / "This feels deceptive" / "Ethical concern"

**Protocol:**
1. Identify specific constitutional value at risk
2. Document: `bd comment <id> "CONSTITUTIONAL CONCERN: [value] - [concern]"`
3. Do NOT proceed with concerning component
4. Continue with unrelated components if separable
5. Wait for HUMAN review (bypasses orchestrator)

**Why bypass orchestrator:** The constitution says Claude can refuse unethical instructions regardless of principal. Orchestrators are Claude - they cannot authorize constitutional violations. Human judgment required.

## Harm Assessment (feature-impl and similar)

**Purpose:** Pre-implementation checkpoint for feature ethics, distinct from code security review.

**Quick assessment:**

| Question | Action if YES |
|----------|---------------|
| Could this harm, deceive, or manipulate users? | Document concern |
| Does this collect/transmit unexpected data? | Document concern |
| Could this be weaponized at scale? | Document concern |
| Does this undermine informed consent? | Document concern |
| Disproportionate impact on vulnerable populations? | Document concern |

**If concerns identified:**
1. Document via `bd comment`
2. Check if SPAWN_CONTEXT addresses with safeguards
3. If addressed → Proceed with safeguards
4. If not addressed → Escalate via Constitutional Objection Protocol

**If no concerns:** Proceed to implementation.

## Why Skill-Level Constraints

| Reason | Explanation |
|--------|-------------|
| Explicit > implicit | Training refusals are context-dependent; skill constraints are explicit and auditable |
| Audit trail | Skill versions show what constraints were in place when agent ran |
| Specificity | Domain-specific limits training can't anticipate |
| Orchestrator education | Orchestrators read skills - constraints remind them of boundaries |

## Implementation

### Files to Edit

**worker-base:** Add Hard Limits and Constitutional Objection Protocol sections
- Source: `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/`

**feature-impl:** Add Harm Assessment checkpoint before Implementation Phase
- Source: `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc/`

### Deployment

```bash
cd ~/orch-knowledge && skillc build && skillc deploy
```

### Validation

Spawn test agent to verify:
- New sections appear in loaded skill
- Harm Assessment in correct position
- worker-base additions inherited by dependent skills

## Failure Modes and Mitigations

| Failure Mode | Mitigation |
|--------------|------------|
| Workers too cautious, refuse legitimate work | Include "false positives" guidance; frame as "escalate" not "refuse" |
| Orchestrators spawn without skills | Skills are one layer; training remains foundation |
| Human overrides constitutional concern | Human accepts documented accountability |
| Skill constraints too broad | Specific examples and "common false positives" list |

## Relationship to Other Decisions

| Decision | Relationship |
|----------|--------------|
| `2026-01-22-orchestrator-constitutional-responsibility.md` | Parent decision establishing skills as constitutional constraints |
| `2026-01-19-worker-authority-boundaries.md` | Extended to include constitutional hard limits |

## References

- `.kb/investigations/2026-01-22-audit-feature-impl-skill-constitutional-constraints.md` - Investigation this decision promotes
- Anthropic constitution (Jan 21, 2026) - Hard constraints, honesty, avoiding harm sections
