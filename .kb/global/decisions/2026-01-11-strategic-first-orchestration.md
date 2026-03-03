# Decision: Strategic-First Orchestration

**Date:** 2026-01-11
**Status:** Accepted
**Context:** Coaching plugin had 8 bugs and 2 abandonments from tactical fixes; architect found root cause in one session

## Decision

Add "Strategic-First Orchestration" as a Meta principle: In areas with patterns (hotspots, persistent failures, infrastructure), strategic approach is the default. Tactical requires justification.

## Context

The coaching plugin injection system demonstrated the cost of defaulting to tactical debugging when strategic analysis was needed:

**The symptoms:**
- 8 bugs in the 'serve' area over 2 days (Jan 10-11)
- 2 abandoned debugging attempts (agents hit complexity wall)
- 8+ commits fixing specific symptoms: worker detection, message content detection, timing issues
- Investigation files created but never filled (agents gave up without understanding)
- Issue abandoned 2x, suggesting agents couldn't see the pattern

**Each tactical fix was correct:**
- Agent A: Fixed worker detection logic
- Agent B: Fixed message content detection
- Agent C: Fixed timing issues
- Agent D: Fixed restart behavior

**But the root cause persisted:**
- Metrics are persistent (survive restart)
- Session state is ephemeral (lost on restart)
- Injection is coupled to observation (can only inject while actively observing)
- After restart: metrics show problems, but injection can't fire (no session state)

**Resolution required architect:**
- One session identified architectural coupling as root cause
- Recommended separating observation (plugin) from intervention (daemon)
- Would eliminate entire class of "injection doesn't fire after X" bugs
- Followed "Coherence Over Patches" principle correctly

## Rationale

**The pattern we keep seeing:**

We consistently face choices between tactical (fix symptom) and strategic (understand pattern) approaches:

| Approach | Time Cost | Outcome |
|----------|-----------|---------|
| **Tactical** | 3+ debugging agents, 8+ commits, days | Symptoms fixed, root cause persists |
| **Strategic** | 1 architect agent, 1 session, hours | Root cause identified, class of bugs eliminated |

Strategic is almost always:
- **Faster total time** (1 architect vs 3+ debugging attempts)
- **More effective** (fixes real problem vs surface bug)
- **Preventative** (coherent system vs patches)

Yet we keep defaulting to tactical because:
- Immediate action feels productive ("fix it now")
- Strategic requires slowing down ("let's understand first")
- Warnings are ignorable ("consider architect" → ignored)
- No forcing function prevents tactical in patterned areas

**Why existing principles didn't prevent this:**

| Principle | What It Says | Why It Wasn't Enough |
|-----------|--------------|---------------------|
| Coherence Over Patches | After 5+ fixes, recommend redesign | Recommendation, not requirement |
| Premise Before Solution | Validate premise first | Applies to questions, not spawning |
| Reflection Before Action | Build the process | Doesn't gate tactical spawns |

The gap: No principle said "you cannot spawn debugging to a hotspot - architect required."

**Why this is a Meta principle (not System Design):**

This isn't about a specific architectural choice. It's about how we evolve our approach to problem-solving:
- Recognize patterns (hotspots, persistent failures)
- Default to strategic in patterned areas
- Make tactical the exception that requires justification
- Apply principles without asking permission

This is meta-level discipline about when to use which approach.

## Operational Changes Required

**1. Make HOTSPOT a gate:**
- Current: Warning the user can ignore
- New: Blocking error, require `--force` to override
- When spawning to hotspot area, refuse tactical debugging
- Require architect skill first

**2. Update orchestrator skill:**
- Current: "Consider architect" (optional suggestion)
- New: "Architect required" (mandatory requirement)
- Change language from suggestions to requirements
- Orchestrator applies principles, doesn't ask permission

**3. Add infrastructure detection:**
- Current: Warning about circular dependency, user proceeds
- New: Auto-apply `--backend claude` for infrastructure work
- Detect work on orchestration system itself (paths, keywords)
- Infrastructure needs escape hatch (can't use what you're fixing)

**4. Daemon applies strategic-first:**
- Auto-spawns from triage:ready use strategic-first logic
- Hotspot areas → architect
- Persistent failures → architect
- Infrastructure → escape hatch

## Signals That Trigger Strategic-First

| Signal | Threshold | Strategic Requirement |
|--------|-----------|----------------------|
| **HOTSPOT** | 5+ bugs in same area (4 weeks) | Architect required (refuse debugging) |
| **Persistent failure** | 2+ abandons on same issue | Auto-spawn architect to investigate pattern |
| **Infrastructure work** | Paths in .orch/, orch CLI, spawn.py | Auto-apply --backend claude |
| **Investigation clustering** | 3+ investigations on topic without synthesis | Synthesis required before more spawns |

## Relationship to Other Principles

**Coherence Over Patches:** That principle says "after 5+ fixes, recommend redesign." Strategic-First *enforces* this - in hotspot areas, tactical is blocked.

**Premise Before Solution:** Before debugging (solution), verify the premise (is tactical appropriate here?). Strategic-First makes this explicit for patterned areas.

**Reflection Before Action:** Build the process that detects patterns (hotspot detection), then use the process to require strategic approach (gate on hotspots).

**Pressure Over Compensation:** Don't compensate for tactical failures by spawning more debugging. Require strategic approach to create pressure to understand the pattern.

## Why This Is Foundational

The entire orchestration system exists to leverage agent capabilities effectively. When we spawn tactical agents in patterned areas:
- We waste agent time fixing symptoms
- Root causes persist
- Same bugs recur
- Trust in the system erodes

Strategic-first maximizes ROI on agent work by ensuring we solve the right problems. In patterned areas, tactical is almost never the right approach.

The math is clear: 1 architect session (hours) vs 3+ debugging sessions (days). Yet we keep choosing the slow path because the fast path requires slowing down first.

This principle makes the fast path the default.

## Evidence

**Coaching plugin case (Jan 11, 2026):**
- 8 bugs, 2 abandonments, days of work
- 1 architect session found root cause in hours
- Would have eliminated entire class of bugs
- Investigation file: `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md`
- Synthesis: `.orch/workspace/og-arch-review-design-coaching-11jan-f74a/SYNTHESIS.md`

**Previous examples:**
- Dashboard status logic: 10+ conditions, 37% fix commits, architect designed Priority Cascade Model
- Multiple "same area keeps breaking" patterns throughout orch-go development

## Success Criteria

Strategic-first orchestration is working when:
- ✅ Hotspot areas refuse tactical spawns (blocking, not warning)
- ✅ Persistent failures trigger architect automatically
- ✅ Infrastructure work auto-applies escape hatch
- ✅ Orchestrator applies principles without asking permission
- ✅ Fewer abandonments in patterned areas (architect finds root cause)
- ✅ Faster time-to-resolution in patterned areas (1 strategic session vs multiple tactical attempts)

## Implementation Status

**Created:** Jan 11, 2026
**Status:** Principle documented, operational changes pending

**Next steps:**
1. Implement hotspot gate in spawn logic (refuse tactical in hotspot areas)
2. Update orchestrator skill guidance (architect required → not suggested)
3. Add infrastructure detection to spawn/triage
4. Integrate with daemon (auto-spawn architect for persistent failures)
5. Measure impact (fewer abandons in hotspots, faster resolution time)
