# Model: Architectural Enforcement

**Domain:** Quality gates / Accretion prevention / Architect routing
**Last Updated:** 2026-02-26
**Synthesized From:**
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Four-layer enforcement design
- `.kb/investigations/2026-02-14-inv-soften-strategic-first-hotspot-gate.md` — Gate severity calibration
- `.kb/investigations/2026-02-25-inv-architect-skillc-deploy-silent-failures.md` — Toolchain reliability
- `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` — Three-layer architect routing
- `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md` — Cross-investigation synthesis

---

## Summary (30 seconds)

The system enforces architectural quality through **multi-layer gate mechanisms** that operate at four enforcement points: spawn-time (prevent bad work from starting), completion-time (reject violations after the fact), real-time coaching (correct agents mid-session), and declarative boundaries (make rules explicit in loaded context). The fundamental tension is **gate strength vs. false positive rate** — gates that are too strict create bypass culture (`--force-hotspot` used reflexively), while gates that are too lenient get ignored (warnings without teeth). The system has converged on a **tiered enforcement model**: warnings at moderate thresholds (800 lines), hard gates at critical thresholds (1,500 lines), with skill-based exemptions for knowledge-producing work (architect, investigation) and net-negative-delta escape for extraction work. The investigation-to-architect-to-implementation sequence is enforced through infrastructure (spawn gates, daemon routing), not instructions (prompts, skill guidance).

---

## Core Mechanism

### The Four Enforcement Layers

Architectural enforcement operates through four complementary layers, each addressing a different failure window:

| Layer | When | What It Catches | How |
|-------|------|----------------|-----|
| **Spawn gates** | Before work starts | Planned accretion (task targets hotspot file) | Block feature-impl/debugging spawns in CRITICAL hotspot areas |
| **Completion gates** | After work done | Unplanned accretion (agent modified hotspot during impl) | Reject +50 lines to files >800 lines |
| **Real-time coaching** | During work | Mid-session drift into accretion | Escalating warnings when editing large files |
| **Declarative boundaries** | Always (loaded context) | Ignorance of constraints | CLAUDE.md documents accretion rules explicitly |

These layers are complementary, not redundant. Spawn gates prevent planned accretion. Completion gates catch unplanned accretion that escaped spawn gates. Coaching provides in-flight correction. CLAUDE.md makes rules discoverable.

### The Investigation-Architect-Implementation Sequence

For hotspot areas, the enforced sequence is:

```
Investigation (find root cause)
    → Architect (design solution)
        → Implementation (build solution)
```

Three mechanisms enforce this:

1. **Spawn gate**: `--force-hotspot` requires `--architect-ref <closed-architect-issue>` — proves an architect reviewed the area before implementation proceeds.
2. **Daemon routing**: When daemon infers skill for a feature/task targeting hotspot files, escalates to `architect` instead of `feature-impl`.
3. **Spawn context injection**: Hotspot status injected into SPAWN_CONTEXT.md so investigation agents recommend architect follow-up in their "Next" section.

### Threshold Calibration

Two calibrated thresholds govern enforcement severity:

| Threshold | Enforcement | Rationale |
|-----------|-------------|-----------|
| **800 lines** | Warning (completion gate), coaching trigger | Moderate bloat — files growing but manageable. Warning provides learning signal. |
| **1,500 lines** | Hard gate (spawn blocking, completion error) | CRITICAL — file has accreted beyond healthy range. Requires extraction before feature additions. |

The +50 line delta captures a typical single-feature addition. Net-negative deltas (extraction work) always pass regardless of file size.

### Skill-Based Exemptions

Not all skills accrete. Knowledge-producing skills need to READ hotspot files to do their job:

| Skill Category | Spawn Gate | Completion Gate | Coaching |
|---------------|------------|-----------------|----------|
| **feature-impl, systematic-debugging** | Blocked at CRITICAL | Error at CRITICAL, warn at moderate | Active |
| **architect, investigation, capture-knowledge, codebase-audit** | Exempt | Subject to (rarely triggers — these don't write impl code) | Exempt |

### Critical Invariants

1. **Gates must be infrastructure-enforced, not instruction-reliant.** Prompts fail under pressure (17:1 system-prompt signal advantage drowns skill constraints). Code-level enforcement is the only reliable approach.

2. **Gates must be passable by the gated party.** An architect analyzing a 2,000-line file to design its decomposition cannot be blocked from that file. Exemptions are required for legitimate work patterns.

3. **Escape hatches must exist but with accountability.** `--force-hotspot` is preserved (per "Escape Hatches" principle) but requires `--architect-ref` proof that architectural review happened. Removing the escape hatch entirely violates the principle that critical paths need independent secondary paths.

4. **Toolchain reliability underlies all enforcement.** If `skillc deploy` exits 0 on partial failure, agents run with stale skills that may not contain enforcement guidance. Silent toolchain failures propagate through the entire system.

---

## Why This Fails

### 1. Gate Calibration Death Spiral (Observed: Feb 2026)

**What happens:** Gate set too strict → high false positive rate → users add `--force` reflexively → gate becomes meaningless → no enforcement.

**Root cause:** The original strategic-first hotspot gate blocked ALL non-architect, non-daemon spawns in hotspot areas. Build fixes, investigations, and low-risk work were all blocked, requiring `--force-hotspot`. This trained the bypass reflex.

**Evidence:** Investigation 2026-02-14 (soften-strategic-first-hotspot-gate) documented that blocking is "too aggressive — prevents productive work and creates friction bypass patterns."

**Fix:** Tiered enforcement (warning at 800, hard gate at 1,500) with skill-based exemptions. The hotspot gate was converted from blocking to warning-only as an interim step, then re-strengthened with the `--architect-ref` accountability requirement.

**Lesson:** The fix for an ignored gate is never "make it louder." It's "make it more precise" — fire less often, fire correctly.

### 2. Silent Toolchain Failures (Observed: Feb 2025)

**What happens:** Agents run with stale skills because `skillc deploy` exits 0 on partial failure.

**Root cause:** Four independent failure points in the skillc deploy pipeline:
- Deploy exits 0 on partial failure (CRITICAL — no programmatic error detection)
- Plugin init-time caching (HIGH — OpenCode reads skill once at startup, never re-reads)
- Cross-project injection blocked (HIGH — CLAUDE_CONTEXT conflation with ORCH_SPAWNED)
- Stale copy accumulation (LOW — old deployment locations never cleaned)

**Evidence:** Feature-impl `src/` copy has checksum `047ddb2689b3` (Jan 7) while canonical has `76a3920c0fe9` (Feb 25) — 7 weeks stale.

**Impact:** An agent spawned with a 7-week-stale skill doesn't know about new enforcement rules, new gates, or changed procedures. All prompt-level enforcement is undermined.

**Fix:** (1) `skillc deploy` must exit non-zero on any failure, (2) `--verify` post-deploy validation, (3) fix ORCH_SPAWNED env var, (4) one-time stale copy cleanup.

### 3. Instruction-Based Enforcement Under Pressure (Systemic)

**What happens:** Agent knows the rule but violates it anyway because system prompt signals overwhelm skill constraints.

**Root cause:** Identity compliance is additive (layers on top of defaults) but action compliance is subtractive (fights defaults). An agent can believe it's an orchestrator while using worker tools. Testing "what is your role?" tells you nothing about action compliance.

**Evidence:** Orchestrator uses `bd close` despite skill saying to use `orch complete` because `bd close` is shorter and more frequently documented in system prompt. 17:1 signal ratio advantage.

**Fix:** Infrastructure enforcement — `--disallowedTools` removes tools at spawn time, PreToolUse hooks block forbidden commands. Code removes the option; prompts describe it.

### 4. Post-Facto Rejection Waste (Design Flaw)

**What happens:** Agent spends hours adding 200 lines to a 2,000-line file, then completion gate rejects the work.

**Root cause:** Completion gates are detection (after work), not prevention (before work). The agent's time is already spent.

**Mitigation:** Spawn gates prevent the work from starting. Coaching plugin catches drift mid-session. But for unplanned accretion (agent didn't plan to edit the hotspot file but did), the completion gate is the last defense, and rejection does waste the work.

**Lesson:** Prevention > Detection > Rejection. Each layer further from the source has higher cost.

---

## Constraints

### Why Multi-Layer Instead of One Gate?

**Constraint:** Each enforcement layer addresses a different failure window that the others miss.

**Implication:** No single gate can catch all violation modes. Spawn gates miss unplanned accretion. Completion gates waste work. Coaching can't block, only warn. CLAUDE.md can be ignored.

**This enables:** Comprehensive coverage across all violation modes
**This constrains:** Cannot simplify to a single enforcement point without creating gaps

### Why 800/1,500 Thresholds?

**Constraint:** Thresholds are calibrated to the existing codebase — 800 = moderate bloat (healthy range ceiling from extraction guide), 1,500 = CRITICAL (hotspot.go already uses these values).

**Implication:** Introducing different thresholds would create confusion ("hotspot says 800, gate says 1,000").

**This enables:** Consistent vocabulary across hotspot detection, spawn gates, completion gates, and coaching
**This constrains:** Cannot change thresholds in one layer without updating all four

### Why Architect Exemption Is Necessary?

**Constraint:** Architect agents must read and analyze hotspot files to design their decomposition. Blocking them from hotspot files defeats the purpose.

**Implication:** Knowledge-producing skills bypass spawn gates. They remain subject to completion gates but rarely trigger them (their deliverables are investigations and decisions, not features).

**This enables:** Architect review of the exact files that need architectural attention
**This constrains:** Knowledge-producing agents can technically add code to hotspot files; mitigated by their skill's deliverable expectations

### Why Toolchain Reliability Is a Prerequisite?

**Constraint:** All prompt-level enforcement (skill guidance, CLAUDE.md boundaries, spawn context injection) depends on the toolchain correctly deploying current skill content.

**Implication:** If `skillc deploy` silently fails, agents run with stale enforcement rules. The entire prompt-level enforcement layer is undermined.

**This enables:** Clear priority ordering: fix toolchain first, then add enforcement layers
**This constrains:** Cannot trust prompt-level enforcement until toolchain reliability is verified

---

## Evolution

**Feb 14, 2026:** Investigation found accretion gravity has detection (hotspot analysis) without prevention (enforcement gates). Designed four-layer enforcement architecture.

**Feb 14, 2026:** Strategic-first hotspot gate converted from blocking to warning-only after probe documented high false-positive rate for build fixes and investigations.

**Feb 24, 2026:** Three-layer architect routing designed after orch-go-1182/1183 incident showed agents jumping from investigation directly to implementation, bypassing architect review. `--force-hotspot` strengthened with `--architect-ref` requirement.

**Feb 25, 2026:** skillc deploy failures investigated — consolidated 5 prior probes into failure taxonomy. Identified 4 independent failure points causing 7-week-stale skills.

**Feb 26, 2026:** Model created synthesizing all investigations into coherent architectural enforcement understanding.

---

## References

**Investigations:**
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Four-layer enforcement design (spawn gates, completion gates, coaching, CLAUDE.md)
- `.kb/investigations/2026-02-14-inv-soften-strategic-first-hotspot-gate.md` — Gate calibration: blocking → warning-only
- `.kb/investigations/2026-02-25-inv-architect-skillc-deploy-silent-failures.md` — Toolchain failure taxonomy (4 failure modes)
- `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` — Three-layer architect routing with `--architect-ref`
- `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md` — Cross-investigation synthesis (14 investigations)

**Related models:**
- `.kb/models/completion-verification/model.md` — How completion gates (Layer 2) work in detail
- `.kb/models/spawn-architecture/model.md` — How spawn gates (Layer 1) work in detail
- `.kb/models/coaching-plugin/model.md` — How coaching (Layer 3) works

**Primary Evidence (Verify These):**
- `cmd/orch/spawn_cmd.go:830-860` — Hotspot check integration at spawn time
- `cmd/orch/hotspot.go:36-486` — Hotspot analysis (800/1500 thresholds)
- `pkg/verify/check.go` — Completion accretion gate (`GateAccretion`)
- `pkg/spawn/gates/hotspot.go` — Spawn gate with `--architect-ref` verification
- CLAUDE.md "Accretion Boundaries" section — Declarative enforcement layer

**Decisions informed by this model:**
- CLAUDE.md "Accretion Boundaries" section — Files >1,500 lines require extraction before feature additions
- Three-layer hotspot enforcement (--architect-ref + daemon escalation + spawn context injection)
- `.kb/decisions/2026-02-26-two-layer-action-compliance.md` — Infrastructure + prompt enforcement for orchestrator action constraints
