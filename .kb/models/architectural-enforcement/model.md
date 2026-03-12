# Model: Architectural Enforcement

**Domain:** Quality gates / Accretion prevention / Architect routing
**Last Updated:** 2026-03-06
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

**Empirical validation (Mar 2026 probe):** Files above the 800-line threshold (context.go at 981, daemon.go at 921) have disproportionately high fix density AND coupling. The threshold accurately predicts which files generate the most bug-fix work. Post-extraction fix-density declines: spawn_cmd.go (1171→551 lines) showed declining fix rate; extraction.go (22 fixes) dissolved when extracted into 8 files.

### Hotspot Dimensions

Hotspot severity is determined by three independent dimensions that compound:

| Dimension | What It Measures | Detection |
|-----------|-----------------|-----------|
| **Bloat** (line count) | File complexity/size | `wc -l`, `orch hotspot` |
| **Fix-density** | Bug frequency | `git log --grep fix:` per file |
| **Coupling** | Co-change frequency | Files that always change together |

**Triple hotspots** (all three dimensions elevated) are the highest-priority targets for architect intervention. As of Mar 2026:
- **pkg/daemon/daemon.go** — 921 lines, 13 fixes/28d, 26 co-changes with cmd/orch/daemon.go
- **pkg/spawn/context.go** — 981 lines, 9 fixes/28d, 15 co-changes with spawn_cmd.go

**Trajectory matters:** Most fix-density is burst-driven (concentrated feature pushes over 1-2 weeks), not steady churn. 28-day windows can overstate hotspot severity if a burst has already subsided. A declining hotspot shouldn't receive the same intervention priority as a rising one.

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

5. **Shared configuration must have a single source of truth.** Parallel config directory copies (`~/.claude/` vs `~/.claude-personal/`) are the config-level equivalent of the code-level "instruction-based enforcement fails under pressure" invariant. Agents running from `cc personal` were silently missing hooks, skills, and global instructions — indistinguishable from correct operation. Symlinks structurally eliminated this. (Source: probe `2026-02-27-probe-config-dir-drift-scope.md`)

6. **Constraint type determines enforcement mechanism.** Behavioral constraints in prompt dilute at 5+ co-resident items and become inert at 10+. Mapping by type: hard behavioral (31 items) → infrastructure deny hooks; soft behavioral (~28 items) → infrastructure coaching hooks; judgment behavioral (~28 items) → prompt-budgeted ≤4 per section, or reformulated as knowledge; knowledge (~64 items) → prompt (survives dilution at 10+). The orchestrator skill had 87 behavioral constraints in prompt — ~83 were non-functional. (Source: probe `2026-03-02-probe-layered-constraint-enforcement-design.md`; evidence: `.kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md`)

7. **Hotspot severity requires multi-dimensional assessment.** Line count alone underestimates hotspot risk. Files with high bloat + high fix-density + high coupling ("triple hotspots") require highest-priority architect intervention. Coupling clusters (files that always change together) amplify accretion effects — a fix in one cascades to coupled files. Trajectory matters: burst-driven fix-density that's declining shouldn't receive the same priority as steady/rising churn. (Source: probe `2026-03-11-probe-fix-density-hotspot-trajectory-overlap.md`)

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

### 5. Constraint Dilution at Scale (Extended: Mar 2026)

**What happens:** Orchestrator skill has 87 behavioral constraints in prompt. Empirical test (`.kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md`) showed bare parity at 10 competing constraints. At 87, ~83 of the behavioral constraints are non-functional — agents comply with them at the same rate as unconstrained.

**Root cause:** System prompt > user prompt hierarchy, and concentration of behavioral signals past 10 causes mutual dilution. Each constraint competes with every other.

**Evidence:** Delegation prohibition scored 1/8 (= bare), anti-sycophancy scored 3/8 (= bare), reconnection framing scored 0-1/8 (= bare) on v3 skill with 50+ constraints.

**Fix:** (1) Hard behavioral constraints → hooks (deny). (2) Soft behavioral constraints → coaching hooks (allow + context). (3) Judgment behavioral constraints → prompt-budgeted ≤4 per section, or reformulated as knowledge ("the pattern is Y" instead of "don't do X"). Knowledge constraints survive dilution at 10+ and have no effective budget limit until ~50+.

**Behavioral → Knowledge reformulation:** Some constraints phrased as prohibitions can be rewritten as norms, moving them from the dilution-prone behavioral bucket to the resilient knowledge bucket. Example: "Don't ask 'want me to complete them?'" → "Orchestrators auto-complete agents at Phase: Complete." Same intent, different transfer mechanism.

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

**Mar 2026 (probe merges):** Three probes incorporated.
- `2026-02-27-probe-config-dir-drift-scope.md` — Identified config drift as an enforcement gap: personal sessions (`cc personal`) were silently running without global CLAUDE.md, skills, and hooks. Parallel config directories are duplicated state with no enforcement mechanism. Structural elimination (symlinks) applied. New invariant added (Invariant 5).
- `2026-03-02-probe-layered-constraint-enforcement-design.md` — Audited all 151 orchestrator skill constraints (87 behavioral, 64 knowledge). Existing hooks cover ~7 of 31 hard-enforceable behavioral constraints; 24 need new hooks. Hook API is sufficient (no new mechanism types needed). ~28 judgment constraints cannot move to infrastructure. ~20 behavioral constraints can be reformulated as knowledge. New invariant added (Invariant 6), new failure mode §5 added.

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

### Merged Probes

| Probe | Date | Verdict | Key Finding |
|-------|------|---------|-------------|
| `probes/2026-02-27-probe-config-dir-drift-scope.md` | 2026-02-27 | Extends | Config drift between parallel Claude config dirs is the same enforcement gap as silent toolchain failures. Personal sessions ran without hooks/skills/CLAUDE.md. Symlinks structurally eliminated it. New invariant: shared config must have a single source of truth. |
| `probes/2026-03-02-probe-layered-constraint-enforcement-design.md` | 2026-03-02 | Confirms + Extends | 87 behavioral constraints in orchestrator skill prompt; dilution evidence shows ~83 are non-functional. Hook API is sufficient for all enforcement types. Constraint taxonomy: hard behavioral → deny hooks; soft → coaching hooks; judgment → prompt-budgeted ≤4; knowledge → prompt (resilient). Behavioral → knowledge reformulation technique identified. |
| `probes/2026-03-11-probe-fix-density-hotspot-trajectory-overlap.md` | 2026-03-11 | Confirms + Extends | 800-line threshold empirically validated: files above it (context.go 981, daemon.go 921) have disproportionate fix density AND coupling. Post-extraction fix-density declines confirm extraction as effective intervention. New dimension: coupling clusters compound bloat + fix-density into "triple hotspots." Burst vs steady fix patterns affect intervention priority — 28-day windows can overstate post-burst hotspots. |
