# Session Synthesis

**Agent:** og-inv-design-analysis-hard-19mar-2503
**Issue:** (ad-hoc, no beads tracking)
**Outcome:** success

---

## Plain-Language Summary

Hard deny hooks in the governance system work perfectly as enforcement — agents physically cannot write to protected files like spawn gates or verification gates. But they create a hidden problem: when agents are blocked, they put code in the wrong place because the denial message says "don't write HERE" but never says "write THERE instead." The scs-sp-8dm agent is a concrete example — it needed to add a concurrency gate, was blocked from `pkg/spawn/gates/`, and put the logic in `pkg/orch/spawn_preflight.go` instead. The best fix is two-part: (1) tell agents BEFORE they start planning that certain paths are protected (inject governance context into SPAWN_CONTEXT.md, just like hotspot context is already injected), and (2) add a brief redirect hint to the denial message as a fallback. This doesn't weaken the governance boundary — it's the difference between a road closure sign that says "ROAD CLOSED" vs one that says "ROAD CLOSED — DETOUR VIA ROUTE 9."

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes are the investigation file and design analysis.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-19-design-analysis-hard-deny-hooks-attractors.md` — Full investigation with 8 findings, 3 attractor patterns compared, recommendation
- `.orch/workspace/og-inv-design-analysis-hard-19mar-2503/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-design-analysis-hard-19mar-2503/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- None (investigation only — no code changes)

---

## Evidence (What Was Observed)

- Current governance hook denial message (`~/.orch/hooks/gate-governance-file-protection.py:105-118`) tells agents WHAT is blocked and WHERE to report, but not WHERE to put code instead
- All 4 spawn gate implementations (`pkg/spawn/gates/{triage,hotspot,agreements,question}.go`) provide guidance on the right path forward — the governance hook is the only enforcement mechanism that blocks without redirecting
- Governance preflight check (`pkg/orch/governance.go:45-74`) detects governance-protected paths at spawn time and prints to stderr, but is NOT injected into SPAWN_CONTEXT.md — agents never see this warning
- scs-sp-8dm agent (commit 61e12c298) displaced concurrency gate logic from `pkg/spawn/gates/` to `pkg/orch/spawn_preflight.go` after being blocked by governance hook
- Hook audit probe (2026-03-12) confirms 63 hook denials with zero outcome tracking
- Gate passability decision (Jan 4) confirms hard deny hooks are intentional human checkpoints, not bugs

---

## Architectural Choices

### Prevention over correction for governance awareness
- **What I chose:** Recommend SPAWN_CONTEXT.md injection as primary solution (prevention), deny message redirect hints as secondary (correction)
- **What I rejected:** Making deny message redirects the primary solution
- **Why:** The architectural-enforcement model's own principle: "Prevention > Detection > Rejection. Each layer further from the source has higher cost." An agent that knows about governance restrictions before planning is strictly better than one that discovers them mid-execution.
- **Risk accepted:** Governance context injection requires changes to governance-protected spawn infrastructure — an architect session will be needed.

### No governance-impl skill
- **What I chose:** Keep the governance boundary absolute for workers
- **What I rejected:** Creating a skill that can write to governance files under controlled conditions
- **Why:** This would be convention-layer trust (L4 in agent-trust model). The whole point of governance protection is structural enforcement. A governance-impl skill fundamentally undermines this.
- **Risk accepted:** All governance file changes must go through orchestrator sessions, which is slower but structurally safe.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-19-design-analysis-hard-deny-hooks-attractors.md` — Full design analysis

### Constraints Discovered
- CONSTRAINT: Governance context injection would require modifying `pkg/spawn/` (governance-protected) — this work must be done in an orchestrator direct session, not by a worker

### Key Insights
1. Gates without attractors create a hidden failure mode: **correct enforcement + wrong architecture**
2. The governance warning at spawn time is invisible to agents — it prints to stderr during spawn but is never injected into SPAWN_CONTEXT.md
3. An attractor does not weaken a governance boundary — it operates in the policy layer (guidance), not the enforcement layer (blocking). Analogous to a detour sign vs removing the road closure.
4. The 63 hook denials with zero outcome tracking is a measurement blind spot — we measure the block but not the architectural displacement

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (architect session)

### Implementation Plan (2 changes)

**Change 1: Inject governance context into SPAWN_CONTEXT.md** (high impact)
- Add governance check results to SPAWN_CONTEXT generation (`pkg/spawn/context.go` or equivalent)
- Similar to existing hotspot injection pattern
- When governance-protected paths are detected in task, include a section: "GOVERNANCE-PROTECTED PATHS: The following paths are protected and cannot be modified by workers. Plan your implementation accordingly."
- **Requires orchestrator session** (touches governance-protected spawn infrastructure)

**Change 2: Add redirect hint to deny message** (moderate impact)
- Modify `~/.orch/hooks/gate-governance-file-protection.py` to append redirect guidance
- Keep it simple — NOT path-specific transformations, just: "Put the code in a non-protected location and document the intended destination in SYNTHESIS.md. The orchestrator will migrate it during review."
- **Requires orchestrator session** (hooks are governance-protected)

**Change 3 (optional): Track displacement outcomes**
- Add `governance.displacement` event type
- Emit when agents report CONSTRAINT via beads comments
- Track what file the agent wrote instead (requires parsing agent's subsequent actions)
- Lower priority — useful for measurement but doesn't fix the problem

---

## Unexplored Questions

- **How many of the 63 hook denials resulted in displaced code?** The empirical audit investigation (`.kb/investigations/simple/2026-03-19-empirical-audit-displaced-code-governance-hooks.md`) is pursuing this question.
- **Should the deny message be different for "agent tried to modify an existing governance file" vs "agent tried to create a new governance file"?** Creating a new file (like concurrency.go in gates/) is more likely to be legitimate displaced work. Modifying an existing file is more likely to be an accidental constraint violation.
- **Would agents actually follow redirect hints?** This is testable — measure displacement rate before and after adding attractors. But given that agents DO follow the existing "report via bd comments" instruction, there's reason to believe they'd follow redirect hints too.

---

## Friction

- `orch kb create investigation --model` flag doesn't exist — had to create investigation file manually. Minor friction.
- No friction otherwise — smooth investigation session.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-design-analysis-hard-19mar-2503/`
**Investigation:** `.kb/investigations/2026-03-19-design-analysis-hard-deny-hooks-attractors.md`
