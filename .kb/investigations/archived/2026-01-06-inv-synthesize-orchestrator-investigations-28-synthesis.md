<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Synthesized 28 orchestrator investigations into a single authoritative guide covering session lifecycle, three-tier hierarchy, spawnable orchestrators, completion verification, and common problems with fixes.

**Evidence:** Read all 28 investigations from Dec 21, 2025 to Jan 6, 2026, identifying 7 major themes, 8 key decisions, and 5 common problems with documented solutions.

**Knowledge:** The orchestrator system evolved from "interactive only" to "spawnable with registry" via incremental enhancement of existing infrastructure. Key insight: orchestrators ARE structurally spawnable - the gap was verification and tracking, not spawn mechanism.

**Next:** Close - guide created at `.kb/guides/orchestrator-session-management.md`. Future orchestrator investigations should update the guide, not create new scattered artifacts.

---

# Investigation: Synthesize Orchestrator Investigations 28 Synthesis

**Question:** How can 28 orchestrator investigations be consolidated into a single authoritative reference that prevents duplicate work and provides clear guidance?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Seven Major Themes Emerged Across Investigations

**Evidence:** The 28 investigations cluster into seven distinct themes:

| Theme | Count | Key Investigations |
|-------|-------|-------------------|
| Session Boundaries | 4 | 2025-12-21-inv-orchestrator-session-boundaries.md, 2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md |
| Spawnable Orchestrators | 6 | 2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md, 2026-01-04-inv-test-spawnable-orchestrator-infrastructure.md |
| Meta-Orchestrator Architecture | 5 | 2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md, 2026-01-04-design-meta-orchestrator-role-definition.md |
| Completion Lifecycle | 3 | 2025-12-25-design-orchestrator-completion-lifecycle-two.md, 2026-01-05-debug-orch-complete-fails-orchestrator-sessions.md |
| Skill Loading/Context | 4 | 2025-12-23-inv-orchestrator-skill-loading-workers-despite.md, 2025-12-24-inv-orchestrator-skill-says-complete-agents.md |
| Self-Correction/Autonomy | 2 | 2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md, 2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md |
| Communication Patterns | 2 | 2026-01-05-inv-orchestrator-worker-bidirectional-communication-interaction.md |

**Source:** All 28 investigations in `.kb/investigations/*orchestrator*.md`

**Significance:** These themes map to distinct sections in the consolidated guide, ensuring comprehensive coverage without duplication.

---

### Finding 2: Orchestrators ARE Structurally Spawnable (Constraint Was False)

**Evidence:** Multiple investigations (Jan 4, 2026) discovered that the "orchestrators can't be spawned" constraint was false:

> "The orchestrator IS already 'spawnable' via orch session start/end - the constraint that prevented spawning was false. The gap is verification and pattern analysis, not the spawning mechanism itself." - 2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md

Parallel structure exists:
- SPAWN_CONTEXT.md ↔ ORCHESTRATOR_CONTEXT.md (session context)
- SYNTHESIS.md ↔ SESSION_HANDOFF.md (completion artifact)
- `.beads_id` file ↔ `.orchestrator` marker (identity)

**Source:** `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md`, lines 77-95

**Significance:** This unblocked the entire spawnable orchestrator infrastructure that was built Jan 4-5, 2026.

---

### Finding 3: Three-Tier Hierarchy Already Existed Implicitly

**Evidence:** The meta-orchestrator role definition investigation found:

> "Three-tier hierarchy is descriptive, not prescriptive - Dylan already operates as meta-orchestrator, orchestrators delegate to workers. Making this explicit adds infrastructure, not concepts." - 2026-01-04-design-meta-orchestrator-role-definition.md

| Tier | Role | Scope | Key Artifact |
|------|------|-------|--------------|
| Meta-orchestrator | Dylan | Cross-project, multi-session | Epic progress, handoffs |
| Orchestrator | Claude agent | Single project, single focus | SESSION_HANDOFF.md |
| Worker | Spawned agent | Single issue | SYNTHESIS.md |

**Source:** `.kb/investigations/2026-01-04-design-meta-orchestrator-role-definition.md`, lines 36-55

**Significance:** The "meta-orchestrator" concept didn't require new systems - it required documenting the implicit hierarchy Dylan was already operating in.

---

### Finding 4: Frame Collapse is the Critical Failure Mode

**Evidence:** Investigation 2026-01-04-inv-meta-orchestrator-level-collapse showed spawned meta-orchestrators immediately doing worker-level work:

> "Spawned meta-orchestrators collapse to worker behavior because ORCHESTRATOR_CONTEXT.md frames them with task-completion goals rather than interactive session management guidance."

The root cause: template framing ("work toward goal") overrides skill content ("delegate, never implement").

**Source:** `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md`, lines 15-19

**Significance:** Framing is stronger than skill guidance. Goal specificity prevents frame collapse - vague goals cause exploration → investigation → debugging (level collapse).

---

### Finding 5: Session Registry Replaced Beads for Orchestrators

**Evidence:** Investigation 2026-01-05-inv-design-orchestrator-session-lifecycle-without-beads found:

> "Orchestrators aren't issues being worked on - they're interactive sessions with Dylan. Beads is for tracking work items, not collaborative sessions."

The session registry at `~/.orch/sessions.json` was created to track orchestrator sessions without beads overhead.

**Source:** `.kb/investigations/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md`, lines 45-56

**Significance:** Semantic alignment matters - beads tracks work items (spawn→task→complete), sessions track conversations (start→interact→end).

---

### Finding 6: Signal Ratio Impacts Skill Effectiveness

**Evidence:** Investigation 2025-12-24-inv-orchestrator-skill-says-complete-agents found why "Always Act (Silent)" guidance for completions was ignored:

> "Signal ratio is 4:1 against autonomy (56 'ask permission' patterns vs 13 'act autonomously' patterns), plus an internal contradiction between 'Always Act (Silent)' and 'Propose-and-Act' examples."

**Source:** `.kb/investigations/2025-12-24-inv-orchestrator-skill-says-complete-agents.md`, lines 7-9

**Significance:** LLMs resolve ambiguous/conflicting guidance by falling back to training defaults. Skill documents need balanced signal ratios.

---

### Finding 7: Five Common Problems Have Documented Fixes

**Evidence:** Across investigations, five problems appeared multiple times with documented solutions:

1. **Frame collapse** - Fix: Specific goals, WHICH vs HOW test
2. **orch complete fails for orchestrators** - Fix: Registry-first lookup (implemented)
3. **Workspace name collision** - Fix: 4-char random hex suffix (implemented)
4. **Skill loads for workers despite audience** - Fix: Move ORCH_WORKER check to config hook (implemented)
5. **Spawned orchestrator tries orch session end** - Fix: Template change to "wait" (implemented)

**Source:** Various investigations, implementation commits in Jan 2026

**Significance:** These are settled problems. The guide captures fixes to prevent re-investigation.

---

## Synthesis

**Key Insights:**

1. **Evolution over revolution** - The orchestrator system evolved via incremental enhancement of existing infrastructure. Every "new capability" (spawnable orchestrators, session registry, tier-based verification) was an extension of existing patterns, not replacement.

2. **Framing trumps content** - Template framing sets agent behavioral mode before skill content is processed. "Work toward goal" = task completion mode. This is why frame collapse happens despite explicit skill guardrails.

3. **Semantic alignment matters** - Using beads (issue tracker) for sessions (conversations) created friction. Creating a purpose-built session registry aligned semantics with mechanics.

4. **The three-tier hierarchy is complete** - Worker → Orchestrator → Meta-orchestrator, each completed by the level above. The infrastructure now supports spawning at every level.

**Answer to Investigation Question:**

The 28 investigations have been consolidated into `.kb/guides/orchestrator-session-management.md` which provides:
- Architecture diagram showing three-tier hierarchy
- Session type and boundary definitions
- Key concepts table
- Common problems with fixes
- Settled decisions (don't re-investigate)
- File location reference
- Debugging checklist

Future work on orchestrator sessions should consult and update this guide rather than creating new scattered investigations.

---

## Structured Uncertainty

**What's tested:**

- ✅ Read all 28 investigations completely
- ✅ Identified 7 major themes with counts
- ✅ Verified session registry, orchestrator context, and completion verification exist in codebase
- ✅ Guide created at `.kb/guides/orchestrator-session-management.md`

**What's untested:**

- ⚠️ Whether the guide is comprehensive enough for all orchestrator debugging (needs real-world usage)
- ⚠️ Whether the "settled decisions" are actually settled (future contradictions possible)
- ⚠️ Whether frame collapse prevention is effective with updated templates

**What would change this:**

- Finding would need update if new orchestrator failure modes emerge
- Finding would need update if meta-orchestrator automation becomes viable (currently Dylan-only)
- Finding would need update if beads tracking becomes valuable for orchestrators again

---

## Implementation Recommendations

**Purpose:** The investigation deliverable IS the guide. No further implementation needed.

### Recommended Approach ⭐

**Guide-first maintenance** - Future orchestrator investigations should update the guide as their primary artifact.

**Why this approach:**
- Single source of truth prevents duplicate investigations
- New learnings immediately benefit all agents
- Debugging checklist prevents unnecessary spawns

**Trade-offs accepted:**
- Guide may grow large over time (mitigated by clear sections)
- Needs periodic refresh (mitigated by "last verified" date)

**Implementation sequence:**
1. ✅ Guide created with 12 major sections
2. Update orchestrator skill to reference the guide
3. Add guide path to kb context orchestrator results

### Alternative Approaches Considered

**Option B: Create multiple focused guides**
- **Pros:** Smaller, more focused documents
- **Cons:** Risk of fragmentation, need to search multiple docs
- **When to use instead:** If guide exceeds 500 lines

**Option C: Archive investigations, keep only guide**
- **Pros:** Cleaner .kb/investigations directory
- **Cons:** Loses historical context and evolution
- **When to use instead:** If disk space becomes a concern

**Rationale for recommendation:** Single comprehensive guide with clear sections provides best discoverability while preserving depth.

---

## References

**Files Examined:**
- All 28 `.kb/investigations/*orchestrator*.md` files
- `pkg/spawn/orchestrator_context.go` - Context generation
- `pkg/session/registry.go` - Session registry
- `cmd/orch/complete_cmd.go` - Completion flow
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill

**Commands Run:**
```bash
# Find all orchestrator investigations
glob .kb/investigations/*orchestrator*.md

# Create investigation file
kb create investigation synthesize-orchestrator-investigations-28-synthesis

# Create guide
kb create guide orchestrator-session-management
```

**Related Artifacts:**
- **Guide:** `.kb/guides/orchestrator-session-management.md` - The synthesized guide (deliverable)
- **Decision:** `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Frame shift concept
- **Decision:** `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md` - Lifecycle model

---

## Investigation History

**2026-01-06 16:30:** Investigation started
- Initial question: How to synthesize 28 orchestrator investigations?
- Context: kb synth suggestion to consolidate accumulated investigations

**2026-01-06 16:45:** Read all investigations
- Identified 7 major themes
- Found 8 key decisions
- Documented 5 common problems with fixes

**2026-01-06 17:00:** Guide created
- `.kb/guides/orchestrator-session-management.md` written
- 12 major sections covering architecture, concepts, problems, decisions

**2026-01-06 17:15:** Investigation completed
- Status: Complete
- Key outcome: Single authoritative guide created from 28 investigations
