## Summary (D.E.K.N.)

**Delta:** Orchestrators operate in meta-action space only (spawn, monitor, query) - primitive actions (edit, write, bash) are architecturally restricted, not just discouraged by guidelines.

**Evidence:** 30 years of HRL research convergence (Options Framework, MAXQ, Feudal Networks); existing task-tool-gate.ts demonstrates working registry-level gating; coaching.ts already detects frame collapse.

**Knowledge:** Information hiding reduces temptation; tool restriction prevents the action. Both needed - hiding is psychological barrier, restriction is architectural enforcement. Research shows guidelines fail under cognitive pressure.

**Next:** Three-phase implementation: (1) Prompt-based action space in skill, (2) Output filtering via plugin, (3) Registry-level tool gating.

---

# Decision: Orchestrator Action Space Architectural Constraint

**Date:** 2026-01-28
**Status:** Accepted

**Extracted-From:** `.kb/investigations/2026-01-27-inv-design-information-hiding-tool-restriction.md`
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Context

Orchestrator frame collapse occurs when orchestrators "dive in" to do worker work (editing code, debugging, investigating). This violates the hierarchical boundary and blocks the entire system while the orchestrator has "nose buried in files."

Prior approach: Guidelines in orchestrator skill saying "don't do this." Research and experience show guidelines fail under cognitive pressure - if the orchestrator CAN do something, it eventually WILL under time pressure.

---

## Options Considered

### Option A: Three-Phase Architectural Enforcement
- **Pros:** Defense-in-depth (prompt + detection + enforcement), incrementally deployable, proven patterns exist (task-tool-gate.ts), research-backed
- **Cons:** Orchestrator loses "quick fix" capability, slightly higher latency for trivial tasks

### Option B: Separate Models for Orchestrator vs Worker
- **Pros:** Ultimate enforcement - different model = different capabilities
- **Cons:** Operational complexity, cost (two models), requires significant infrastructure changes

### Option C: Pure Prompt-Based Guidelines
- **Pros:** Easy to implement, no infrastructure changes
- **Cons:** Research shows guidelines fail under cognitive pressure, already proven insufficient

---

## Decision

**Chosen:** Option A - Three-Phase Architectural Enforcement

**Orchestrator Action Space:**

| Allowed (Meta-Actions) | Blocked (Primitive Actions) |
|------------------------|----------------------------|
| `orch spawn/complete/status/review` | `Edit` tool |
| `bd create/show/ready/close` | `Write` tool |
| `kb context/quick` | `bash` (most commands) |
| `git status` (read-only) | `Read` (code files) |
| Read: CLAUDE.md, .kb/*.md, .orch/*.md | Direct file operations |

**Implementation Phases:**

1. **Phase 1 (Immediate):** Explicit CAN/CANNOT in orchestrator skill - DONE (already in skill)
2. **Phase 2 (Short-term):** Information hiding via output filtering in coaching.ts
3. **Phase 3 (Medium-term):** Registry-level tool gating via orchestrator-tool-gate.ts plugin

**Rationale:** Research shows architectural constraints beat guidelines. If you CAN do it, you eventually WILL do it under pressure. Allowlisting meta-actions is safer than blocklisting primitives.

**Trade-offs accepted:**
- Orchestrator cannot do "quick fixes" - must spawn even for trivial changes
- Slightly higher latency for simple tasks
- Research shows benefits outweigh: mixing levels causes more problems than it solves

---

## Structured Uncertainty

**What's tested:**
- ✅ Task-tool-gate pattern works (verified: plugin intercepts tools, injects warnings)
- ✅ Coaching plugin detects frame collapse (verified: isCodeFile() and FrameCollapseState active)
- ✅ Research patterns converge (verified: 30 years across HRL, multi-agent, org psych, LLM agents)

**What's untested:**
- ⚠️ Output filtering at plugin layer (need to verify tool.execute.after can modify returns)
- ⚠️ Allowlist completeness (may miss legitimate orchestrator commands)
- ⚠️ User experience of hard restrictions

**What would change this:**
- If plugin layer cannot intercept/modify tool outputs (different implementation needed)
- If frame collapse persists despite restriction (stronger mechanism needed)
- If orchestrator needs code access for legitimate reasons not yet identified

---

## Consequences

**Positive:**
- Frame collapse becomes architecturally impossible, not just discouraged
- Orchestrator stays available for coordination (not blocked doing worker work)
- Clear boundary between meta-actions and primitive actions
- Defense-in-depth: prompt + detection + enforcement

**Risks:**
- May feel restrictive initially
- Edge cases may require emergency override (should be rare, logged)
- Need good error messages explaining WHY actions are blocked

---

## Success Criteria

- Zero frame collapse incidents (orchestrator never edits code files)
- 100% spawn rate (all implementation work goes through workers)
- Emergency escapes rare (<1% of orchestrator sessions)

---

## References

- **Investigation:** `.kb/investigations/2026-01-27-inv-design-information-hiding-tool-restriction.md`
- **Research:** `.kb/investigations/2026-01-27-inv-research-exists-preventing-hierarchical-controllers.md`
- **Prior Work:** `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md`
- **Principle:** `~/.kb/principles.md` - Authority is Scoping, Perspective is Structural
