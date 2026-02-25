# Synthesis: Enforcement, Accretion, and Verification Design Burst (Feb 19-24)

**Date:** 2026-02-24
**Type:** Cross-investigation synthesis
**Scope:** 14 investigations from the Feb 19-24 design sprint
**Author:** Orchestrator session

---

## The Central Insight

Across 14 investigations spanning 5 days, one pattern dominates: **the system keeps discovering that prompt-level constraints fail under pressure, and the fix is always the same — add infrastructure enforcement alongside the prompt.** This pattern appears in orchestrator compliance (17:1 signal ratio defeat), hotspot gates (--force-hotspot too easy), verification (evidence-exists vs evidence-is-real), and tradeoff visibility (described but not gated).

The system is transitioning from *instruction-based governance* to *infrastructure-based governance*. The investigations document the specific failure modes that drove each transition.

---

## Four Threads

### Thread 1: Enforcement Infrastructure

**Investigations:** Orchestrator skill compliance, Claude Code hooks spike, hotspot gate enforcement, dashboard oscillation, plan mode evaluation

**What happened:** Five investigations converge on one question — how does the system make agents do what they're supposed to, not just know what they're supposed to?

**Key findings:**

| Problem | Root Cause | Solution Implemented |
|---------|-----------|---------------------|
| Orchestrator uses Task tool despite knowing not to | System prompt has 17:1 signal advantage over skill constraints | `--disallowedTools` removes tools at spawn time (1191) |
| Orchestrator uses `bd close` instead of `orch complete` | `bd close` is more salient (shorter, more frequently documented) | PreToolUse hook blocks `bd close` for orchestrator sessions (1191) |
| Workers bypass hotspot gate with `--force-hotspot` | No accountability — override has no precondition | `--architect-ref` requires closed architect issue (1188) |
| Daemon spawns feature-impl into hotspot areas | Skill inference has no hotspot awareness | Daemon escalates to architect for hotspot areas (1189) |
| Dashboard oscillates for claude-backend agents | Tmux used as state (violates two-lane decision) | Phase-based liveness from beads comments (1185) |
| Plan mode considered for feature-impl | Interactive approval, context clearing, observability gap | Rejected — don't integrate (1192) |

**The design principle that emerged:** Identity compliance is additive (layers on top of defaults). Action compliance is subtractive (fights defaults). An agent can believe it's an orchestrator while using worker tools. Testing "what is your role?" tells you nothing about action compliance. Infrastructure must enforce what prompts describe.

**Status:** All five items implemented and verified in the completed issues table from last session. The skill restructuring (Layer 1 of the compliance fix) remains as follow-up work.

### Thread 2: Accretion and Code Health

**Investigations:** extraction.go structure analysis, daemon config extraction, coupling hotspot analysis, progressive skill disclosure

**What happened:** Four investigations map the codebase's growth pressure and design responses.

**Key findings:**

- **extraction.go** (2011 lines): 9 responsibility domains, 22 commits in 28 days, fix-on-fix anti-pattern concentrated in backend resolution. Phased extraction plan: P0 (spawn modes + helpers, -553 lines), P1 (backend + context + beads), P2 (design + inference). Target: extraction.go → spawn_pipeline.go at ~400 lines.

- **Daemon config** (12-file touch surface): Adding one boolean caused a 526K token spiral. Three duplicate Config structs, two copy-pasted plist templates. Three-phase extraction plan: consolidate Config → consolidate plist → add FromUserConfig conversion. Target: 5-7 locations per field (down from 10-12).

- **Coupling hotspot system**: New 4th hotspot type (`coupling-cluster`) using git co-change analysis across architectural layers. Daemon cluster scored 180 (CRITICAL: 25 files, 3 layers, 24 cross-surface commits). Algorithm: filter to cross-surface commits → extract concepts from paths → score by layers × files × frequency.

- **Progressive skill disclosure**: `@section` HTML comment markers in skill sources, `FilterSkillSections()` in loader.go. Estimated 22-29% token reduction for feature-impl spawns. Design complete, not yet implemented.

**The meta-pattern:** Accretion Gravity is the system's dominant failure mode. Files grow because each agent optimizes for its task, not system health. The response is always structural: detection (hotspot system), prevention (accretion gates), and extraction (focused modules).

**Status:** Coupling hotspot analysis implemented (3 test commits landed). extraction.go and daemon config extractions are designed but not started. Progressive skill disclosure is designed but not started.

### Thread 3: Verification as Shared Vocabulary

**Investigations:** Verification levels design, verification infrastructure audit, tradeoff visibility design

**What happened:** Three investigations map the verification system and design its next evolution.

**Key findings:**

- **14 gates exist, all wired and tested** — the system is far more sophisticated than the model documented (which claimed 3 gates). Anti-theater mechanisms in test evidence (22 true-positive patterns, 11 false-positive filters, 116 test cases) are the strongest defense.

- **Sharp boundary at execution**: Everything above "tests actually pass" is verified. Everything at or below is not. The system checks that agents *claim* to have tested, not that tests *actually passed*. Only `go build` executes anything real.

- **Three implicit level systems** already exist (spawn tier, checkpoint tier, skill-based auto-skips). The V0-V3 design unifies them into one concept:
  - **V0 (Acknowledge):** Did agent finish? (Phase Complete only)
  - **V1 (Artifacts):** Are deliverables present? (Synthesis, handoff, constraints)
  - **V2 (Evidence):** Is there evidence of testing? (Test evidence, git diff, build, accretion)
  - **V3 (Behavioral):** Did a human observe it working? (Visual, explain-back, behavioral)

- **Tradeoff visibility** needs three layers: Model Pressure Points (prevent), SYNTHESIS Architectural Choices (capture), completion pipeline surfacing (bring to orchestrator).

- **The daemon_test.go accretion false positive** that blocked 1154 today is exactly the kind of gate friction V0-V3 levels would solve — an investigation agent at V1 would never trigger the accretion gate.

**Status:** Model rewrite completed (1154). V0-V3 levels designed but not implemented. Tradeoff visibility designed but not implemented.

### Thread 4: System Architecture

**Investigations:** Atomic spawn design, beads fork audit

**Key findings:**

- **Atomic spawn** (two-phase): Phase 1 (common: tag beads + write manifest) with rollback function. Phase 2 (per-backend: create session + update manifest) is best-effort. Eliminates the 238-dead-agents bug class. SessionID optional (claude backend has none). Designed in detail, not yet implemented.

- **Beads fork**: 43 commits beyond upstream despite "clean slate" decision to drop it. 6 features actively used by orch-go, ~5 more implicitly depended on. The decision was reversed within 9 days and never superseded. The integration model has 5 wrong file paths, wrong socket path, and a constraint ("never use exec.Command directly") violated 11 times across 7 files.

**Status:** Both are pending implementation. The beads fork needs a superseding decision.

---

## Implementation Priority Matrix

| Design | Value | Effort | Status | Recommendation |
|--------|-------|--------|--------|---------------|
| V0-V3 verification levels | High (eliminates skip-flag ceremony) | Medium (7 files) | Designed | **Next priority** — would have prevented today's daemon_test.go false positive |
| extraction.go P0 extraction | High (isolates instability) | Low (file moves) | Designed | Spawn when ready |
| Daemon config extraction | Medium (prevents spirals) | Low (3 phases) | Designed | Spawn when ready |
| Coupling hotspot analysis | Medium (new detection type) | Low (~200 lines) | Partially implemented | Tests landed, core analysis TODO |
| Progressive skill disclosure | Medium (token savings) | Medium (cross-repo) | Designed | Lower priority |
| Atomic spawn | High (eliminates partial state) | Medium (~575 lines) | Designed | Spawn when ready |
| Beads fork superseding decision | Low (documentation) | Trivial | Needs writing | Can do inline |
| Orchestrator skill restructure (Layer 1) | Medium (salience) | Low (edit skill source) | Designed | Layer 2 shipped, Layer 1 remains |
| Tradeoff visibility layers | Medium (prevents drift) | Medium (model + template + pipeline) | Designed | Spawn when ready |

---

## Decisions to Promote

Several investigations recommend promotion to decisions:

1. **"Plan mode: don't integrate"** — Claude Code plan mode is inappropriate for daemon-spawned agents. Blocks on: interactive approval, context clearing, observability gap.

2. **"Three-layer hotspot enforcement"** — --architect-ref gate + daemon escalation + spawn context injection. Already implemented.

3. **"Phase-based liveness over tmux-as-state"** — Beads phase comments as heartbeat for claude-backend agents. Already implemented.

4. **"Two-layer action compliance: infrastructure + prompt"** — --disallowedTools + PreToolUse hook + skill restructuring. Infrastructure shipped, prompt restructuring pending.

---

## Unexplored Questions

From the investigations' blocking questions:

- **Q (Verification):** Should `--verify-level` be overridable at completion time, or spawn-only? (Recommended: spawn-only, with skip flags as completion override)
- **Q (Tradeoff):** Is the tradeoff visibility gap a symptom of a deeper problem — that feature requests don't flow through architectural models before becoming tasks? (Strategic question, deferred)
- **Q (Beads):** Should fork features be PRed upstream to reduce maintenance? (Medium priority)
- **Q (Coupling):** When to add agent spiral correlation? (After 2+ more confirmed spiral-on-coupling incidents)

---

## Source Investigations

| # | Date | Investigation | Thread |
|---|------|--------------|--------|
| 1 | 02-24 | Plan mode evaluation | Enforcement |
| 2 | 02-24 | Hotspot gate enforcement | Enforcement |
| 3 | 02-24 | Dashboard oscillation / tmux liveness | Enforcement |
| 4 | 02-24 | Claude Code hooks spike | Enforcement |
| 5 | 02-24 | Orchestrator skill behavioral compliance | Enforcement |
| 6 | 02-20 | Verification levels V0-V3 | Verification |
| 7 | 02-20 | Beads fork integration audit | Architecture |
| 8 | 02-20 | Tradeoff visibility design | Verification |
| 9 | 02-20 | Verification infrastructure audit | Verification |
| 10 | 02-20 | Progressive skill disclosure | Accretion |
| 11 | 02-19 | extraction.go structure analysis | Accretion |
| 12 | 02-19 | Coupling hotspot analysis system | Accretion |
| 13 | 02-19 | Daemon config extraction | Accretion |
| 14 | 02-19 | Atomic spawn design | Architecture |
