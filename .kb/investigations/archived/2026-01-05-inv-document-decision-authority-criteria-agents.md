## Summary (D.E.K.N.)

**Delta:** Created decision-authority.md guide defining when agents can decide autonomously vs escalate, with decision tree and examples.

**Evidence:** Analyzed SPAWN_CONTEXT.md template (context.go:108-124), spawn.md, and orchestrator skill patterns.

**Knowledge:** The core distinction is strategic/user-facing/irreversible → escalate vs tactical/internal/reversible → agent decides. Uncertainty defaults to escalation.

**Next:** Close - guide created and referenced from spawn.md and SPAWN_CONTEXT template.

---

# Investigation: Document Decision Authority Criteria Agents

**Question:** When should agents decide autonomously vs escalate to orchestrator/human?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SPAWN_CONTEXT.md had authority section but no criteria

**Evidence:** Lines 108-124 of context.go define the AUTHORITY section with lists of what agents can/cannot decide, but no underlying criteria or decision tree.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:108-124`

**Significance:** Agents had rules but no framework for edge cases. This led to overly cautious behavior.

---

### Finding 2: Spawn.md mentioned authority but didn't define it

**Evidence:** spawn.md line 94 says "Authority levels (what agent can decide vs escalate)" but links nowhere.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawn.md:94`

**Significance:** Documentation gap - the concept was named but never defined.

---

### Finding 3: Implicit patterns exist in orchestrator skill

**Evidence:** Orchestrator skill has patterns like "ABSOLUTE DELEGATION RULE" and escalation triggers, but these are orchestrator-focused, not agent-focused.

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md`

**Significance:** Needed agent-specific guidance, not just orchestrator rules.

---

## Synthesis

**Key Insights:**

1. **Six-axis decision framework** - The core distinction maps to six axes: Strategic↔Tactical, User-facing↔Internal, Irreversible↔Reversible, Cross-boundary↔Single-scope, Resource-commitment↔Within-budget, Ambiguous↔Clear.

2. **Decision tree provides actionable guidance** - A flowchart format lets agents quickly determine escalation without reading full documentation.

3. **Examples bridge theory to practice** - Concrete examples of "agent decides" vs "escalate" cases prevent misinterpretation.

**Answer to Investigation Question:**

Agents should decide when changes are tactical, internal, reversible, single-scope, within-budget, and have clear trade-offs. Escalate when any of these is violated. Uncertainty defaults to escalation because the cost of unnecessary escalation (minutes) is far less than wrong decisions (hours of rework).

---

## Structured Uncertainty

**What's tested:**

- Analyzed existing SPAWN_CONTEXT template and spawn.md (verified: read files)
- Guide created and referenced from both locations (verified: edited files)
- Build compiles with template changes (verified: go build runs)

**What's untested:**

- Agent behavior with new guide (not benchmarked - would need agent sessions)
- Orchestrator satisfaction with criteria (needs human review)

**What would change this:**

- Finding that agents still over-escalate despite guide → need more examples
- Finding that agents under-escalate → criteria too loose

---

## Implementation Recommendations

### Recommended Approach: Reference-based integration

Guide lives in `.kb/guides/decision-authority.md`. SPAWN_CONTEXT.md template references it with one-line pointer. spawn.md references it in context.

**Why this approach:**
- Guide is authoritative source (single point of truth)
- SPAWN_CONTEXT stays concise (already long)
- Can update guide without changing code

**Implementation sequence:**
1. Create guide ✓
2. Update spawn.md reference ✓
3. Update SPAWN_CONTEXT template pointer ✓

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT template
- `.kb/guides/spawn.md` - Spawn documentation
- `~/.kb/principles.md` - Meta-orchestration principles

**Files Created:**
- `.kb/guides/decision-authority.md` - New guide

**Files Modified:**
- `.kb/guides/spawn.md` - Added reference to new guide
- `pkg/spawn/context.go` - Added pointer to guide in AUTHORITY section

---

## Investigation History

**2026-01-05 08:00:** Investigation started
- Initial question: Document decision authority criteria for agents
- Context: Gap identified - spawn.md mentions authority but doesn't define

**2026-01-05 08:15:** Investigation completed
- Status: Complete
- Key outcome: Created decision-authority.md guide with decision tree, examples, and integration into spawn workflow
