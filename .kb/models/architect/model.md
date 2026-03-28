# Model: Architect Skill Effectiveness

**Domain:** How the architect skill produces structural decisions that prevent recurring problems
**Last Updated:** 2026-03-18
**Synthesized From:** 4 architect investigations (Feb-Mar 2026), 1 accuracy probe (Mar 2026)

---

## Summary (30 seconds)

The architect skill is the system's structural decision-maker — it designs enforcement layers, failure taxonomies, and simplification plans that implementation skills then execute. Four investigations reveal a consistent pattern: complexity accumulates through locally-correct decisions, and the architect's value is identifying the structural root cause behind multiple symptoms. Accretion enforcement needed 4 layers because a single gate has blind spots. skillc deploy had 4 independent failure modes masquerading as one bug. The daemon's 3 structural problems shared a common root (internal complexity hiding failure modes). Verification had 3 implicit level systems that needed unification. The architect skill works best when it produces testable structural claims with clear implementation phases, ordered by value.

---

## Core Claims (Testable)

### Claim 1: Architect investigations that decompose symptoms into independent failure modes produce higher-value fixes than investigations that treat symptoms as a single bug

The skillc deploy investigation found 4 independent failure modes (exit code, plugin caching, cross-project injection, stale copies). Each had different severity and fix complexity. Treating "stale skills" as one bug would have produced one fix covering at most 1 of 4 modes.

**Test:** Compare fix coverage of architect-decomposed vs single-symptom investigations on the same problem.

**Status:** Supported (3 of 4 investigations demonstrated decomposition)

### Claim 2: Phased implementation plans ordered by value-per-effort produce better adoption than all-at-once designs

The daemon reliability design ordered 3 phases by value: dedup pipeline (highest), scheduler extraction, operational hardening. The accretion enforcement ordered 4 layers: spawn gates (prevention, highest ROI), completion gates, daemon escalation, CLAUDE.md boundaries.

**Test:** Track which phases get implemented. If later phases are frequently abandoned, the ordering was correct (high-value work done first).

**Status:** Supported (all 4 accretion layers now shipped — but all converted to advisory-only after 100% bypass rate over 2-week measurement per decision 2026-03-17. The phased ordering was correct: high-value layers shipped first and remain live. The dedup pipeline was refactored into composable SpawnPipeline gates rather than collapsed to ~60 lines as originally designed.)

### Claim 3: The investigation→architect→implementation sequence prevents architectural violations that direct investigation→implementation produces

Architect investigations produce structural constraints (4-layer enforcement, CAS-like dedup, unified verification levels) that implementation agents wouldn't discover independently. When implementation follows investigation directly, it produces tactical fixes that miss structural opportunities.

**Test:** Compare implementations spawned with vs without architect review for structural quality.

**Status:** Hypothesis (enforced by spawn gate infrastructure, not experimentally validated)

---

## Implications

- **Architect is a coordination skill, not a planning skill.** Its value is preventing 30 agents from each solving the same structural problem differently. The daemon's dedup gauntlet accumulated because each tactical fix was done without architect review. (The gauntlet has since been refactored into composable SpawnPipeline gates in `pkg/daemon/spawn_gate.go`.)
- **Architect output should be implementation issues, not code.** All 4 investigations produced prioritized implementation plans that became beads issues. The architect doesn't implement — it creates the structural attractors that implementation agents follow. However, issue creation itself must be deduped — architects now run a Prior Art Check (git log + bd list) before every `bd create` to avoid creating zombie issues for work already committed by other agents.
- **Architect investigations should be gated by complexity, not urgency.** Simple bugs don't need architecture. The spawn gate correctly exempts architect from hotspot blocking — architects need to read bloated files to design their extraction.

---

## Boundaries

**What this model covers:**
- How the architect skill produces structural decisions
- What makes architect investigations effective (decomposition, phasing, structural root cause)
- The investigation→architect→implementation pipeline

**What this model does NOT cover:**
- The specific enforcement mechanisms designed (see `architectural-enforcement` model)
- Skill content transfer mechanics (see `skill-content-transfer` model)
- Daemon internals (see `daemon-autonomous-operation` model)

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-02-14 | Accretion gravity enforcement | Detection without prevention = zero enforcement. 4-layer design needed. |
| 2026-02-20 | Verification levels | 3 implicit systems unified into V0-V3 levels. "Levels over gates" principle. |
| 2026-02-25 | skillc deploy failures | 1 symptom = 4 independent failure modes. Decomposition multiplied fix value. |
| 2026-03-05 | Daemon unified reliability | Internal complexity hides failure modes. Inside-out simplification > adding more layers. |
| 2026-03-17 | Accretion gates advisory decision | All 4 accretion layers shipped but converted to advisory after 100% bypass rate. Gates signal, don't block. |
| 2026-03-18 | Accuracy audit probe | Model claims verified: decomposition+phasing principles sound; implementation status was stale; dedup not collapsed to ~60 lines but refactored to composable gates. |
| 2026-03-27 | Issue creation dedup probe | Architect skill had zero pre-creation dedup — all dedup was spawn-time (CommitDedupGate). Added Prior Art Check to skill template as layer 1 dedup. |

---

## Probes

- 2026-03-18: [Architect Model Accuracy Audit](probes/2026-03-18-probe-architect-model-accuracy-audit.md) — Core principles (decomposition, phasing) confirmed sound; implementation status was stale (all 4 accretion layers shipped, all advisory); dedup refactored to composable gates not ~60 lines; coaching doesn't detect accretion.
- 2026-03-27: [Issue Creation Dedup Effectiveness](probes/2026-03-27-probe-issue-creation-dedup-effectiveness.md) — Architect skill had zero pre-creation dedup (Defect Class 6 exposure). Added Prior Art Check procedure to skill template as layer 1 dedup, upstream of daemon's CommitDedupGate.

---

## Open Questions

- Does the architect skill's ~4 behavioral norms (near-compliant per skill-content-transfer audit) actually produce better structural output than higher-behavioral-weight skills?
- What is the right trigger for routing work through architect vs direct implementation? Current heuristic: hotspot files and cross-cutting concerns.

## Source Investigations

### 2026-02-14-inv-architect-design-accretion-gravity-enforcement.md

**Delta:** Accretion has detection (hotspot analysis finds 115 bloated files) but zero prevention/enforcement — violates "Gate Over Remind" principle.
**Evidence:** spawn_cmd.go (2,332 lines), session.go (2,166 lines), doctor.go (1,912 lines) all CRITICAL hotspots with zero blocking. Hotspot check at spawn is warning-only (line 834-850).
**Knowledge:** Enforcement requires four layers: spawn-time gates (prevention), completion gates (rejection), daemon escalation routing, CLAUDE.md boundaries (declaration). Tiered thresholds: warn at 800, error at 1,500. Exempt skills: architect, investigation, capture-knowledge, codebase-audit.
**Next:** All 4 layers shipped (spawn gates in `pkg/spawn/gates/hotspot.go`, completion in `pkg/verify/accretion.go`, daemon escalation in `pkg/daemon/architect_escalation.go`, CLAUDE.md boundaries documented). All gates converted to advisory-only (decision 2026-03-17) after 100% bypass rate measurement.

---

### 2026-02-20-inv-architect-verification-levels.md

**Delta:** Three implicit verification systems (spawn tier, checkpoint tier, skill-based auto-skips) consolidated into four-level vocabulary (V0-V3).
**Evidence:** V0 Acknowledge (config/README), V1 Artifacts (investigations/designs), V2 Evidence (features/bugs), V3 Behavioral (UI/user-facing). Replaces 12 skip flags with a single level parameter.
**Knowledge:** "Levels Over Gates" — a single level parameter declared at spawn time selects the right gate set. Common case requires zero flags. Preserves orchestrator override.
**Next:** Verification levels implemented in review_tier.go.

---

### 2026-02-25-inv-architect-skillc-deploy-silent-failures.md

**Delta:** Consolidated 5 prior probes into single failure taxonomy. 4 distinct failure modes causing agents to run with stale skills.
**Evidence:** `skillc deploy` exits 0 on partial failure. Plugin caches skill at init (no hot-reload). Cross-project injection blocked by wrong env var check. 20+ stale copies in `~/.claude/skills/src/`.
**Knowledge:** Silent deploy failure is not one bug but a pipeline with 4 independent failure points. Two require skillc code changes, one requires hook fixes, one is operational hygiene.
**Next:** Issues created for exit code fix, hook spawn detection, stale copy cleanup.

---

### 2026-03-05-inv-architect-unified-reliability-design-orch.md

**Delta:** Daemon's three structural problems (6-layer dedup gauntlet, 625-line loop, operational unreliability) share a common root: internal complexity makes failure modes invisible.
**Evidence:** spawnIssue() 245 lines with 6 dedup layers; runDaemonLoop 697 lines with 12 inline subsystems; beads lacks native CAS.
**Knowledge:** Inside-out simplification: (1) collapse dedup to composable gate pipeline, (2) extract scheduler, (3) launchd management. Beads CAS simulated in Go via fresh-check + update behind local mutex.
**Next:** Phase 1 (dedup pipeline) shipped — refactored into SpawnPipeline with 5 composable gates (`pkg/daemon/spawn_gate.go`, 312 lines) + execution layer (`pkg/daemon/spawn_execution.go`, 277 lines). Not collapsed to ~60 lines as originally designed, but architecturally cleaner and testable.

## Auto-Linked Investigations

- .kb/investigations/2026-02-14-inv-fix-skill-inference-architect-title.md
