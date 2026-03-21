---
status: accepted
blocks:
  - keywords:
      - hotspot enforcement
      - force-hotspot
      - architect gate
      - investigation implementation sequence
      - accretion enforcement
      - hotspot bypass
    patterns:
      - "**/spawn/gates/hotspot*"
      - "**/daemon/architect_escalation*"
---

## Summary (D.E.K.N.)

**Delta:** Hotspot enforcement uses three layers: (1) spawn gate requiring architect-ref for --force-hotspot bypass, (2) daemon skill escalation routing feature-impl to architect when targeting hotspot areas, (3) spawn context injection providing advisory hotspot info to agents.

**Evidence:** The orch-go-1182/1183 incident where --force-hotspot was used without architectural review, producing two reverted implementations. Investigation `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` designed the three-layer approach. All three layers are now implemented and tested.

**Knowledge:** Single-layer enforcement leaves gaps: spawn gates alone don't cover daemon-driven spawns; advisory guidance alone violates "Gate Over Remind." Multi-layer enforcement with each layer addressing a different gap produces comprehensive coverage without single points of failure.

**Next:** Monitor for false positives (legitimate hotspot overrides blocked by missing architect issues). Consider adding automatic architect-finder coverage metrics. CLAUDE.md Accretion Boundaries section should be updated to reference `--architect-ref` requirement.

---

# Decision: Three-Layer Hotspot Enforcement with Architect-Gated Override

**Date:** 2026-02-26
**Status:** Accepted
**Deciders:** Dylan (via orchestrator), architect investigation
**Context Issues:** orch-go-1196 (promotion), orch-go-1187 (architect design), orch-go-1182/1183 (reverted implementations), orch-go-1184 (architect that caught violations)
**Source Investigation:** `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md`
**Supersedes:** Quick entries kb-3d169f (force-hotspot requires architect-ref)

---

## Context

The `--force-hotspot` flag allowed workers to bypass the CRITICAL hotspot spawn gate unconditionally. In the orch-go-1182/1183 incident:

1. An investigation correctly identified a root cause in a hotspot area
2. A worker spawned with `--force-hotspot` and implemented directly, violating the two-lane decision
3. A second worker deepened the violation with a cache workaround
4. An architect finally caught both violations and recommended a completely different approach

**Cost:** 3 spawn cycles, a regression, and reverted work. The investigation-to-architect-to-implementation sequence was bypassed because `--force-hotspot` had no accountability.

**Principles driving this decision:**
- "Gate Over Remind" — Advisory guidance is insufficient; hard gates required
- "Infrastructure Over Instruction" — Code/tools enforce behavior, not prompts
- "Escape Hatches" — Critical paths need independent secondary paths (can't remove --force-hotspot)
- "Skills own domain behavior, spawn owns orchestration infrastructure"

---

## Options Considered

### Option A: Spawn Gate Only
- **Pros:** Simplest implementation, addresses direct failure mode
- **Cons:** Daemon gap remains (daemon-driven spawns skip hotspot gate), no advisory routing for investigations

### Option B: Remove --force-hotspot Entirely
- **Pros:** Simplest enforcement, zero bypass possible
- **Cons:** Violates "Escape Hatches" principle; blocks emergency infrastructure work

### Option C: Three-Layer Enforcement
- **Pros:** Comprehensive coverage (spawn gate + daemon + advisory), preserves escape hatches with accountability, each layer addresses a different gap
- **Cons:** More implementation surface area, three code paths to maintain

---

## Decision

**Chosen:** Option C — Three-Layer Enforcement with Architect-Gated Override

### Layer 1: Spawn Gate Enhancement (Hard Enforcement)

`pkg/spawn/gates/hotspot.go:CheckHotspot()`

When `--force-hotspot` is passed for a blocking skill (feature-impl, systematic-debugging):
1. Requires `--architect-ref <issue-id>` flag
2. Verifies the referenced issue exists (via `bd show`)
3. Verifies the issue is an architect type
4. Verifies the issue is status=closed (architect completed review)
5. Blocks spawn with descriptive error if any check fails

Additionally, auto-detection: when no `--force-hotspot` is passed, the gate searches for a prior closed architect review covering the critical files. If found, bypass is automatic with no flags needed.

**Files:** `pkg/spawn/gates/hotspot.go`, `cmd/orch/spawn_cmd.go` (--architect-ref flag), `pkg/orch/extraction.go` (plumbing)

### Layer 2: Daemon Hotspot Routing (Gap Closure)

`pkg/daemon/architect_escalation.go`

When daemon infers skill for a feature/task issue:
1. Runs hotspot check against inferred target files
2. If task targets ANY hotspot file and inferred skill is feature-impl/systematic-debugging, escalates to architect
3. Checks for prior closed architect reviews before escalating (auto-bypass)
4. Logs escalation events

**Files:** `pkg/daemon/architect_escalation.go`, `pkg/daemon/architect_escalation_test.go`

### Layer 3: Spawn Context Injection (Advisory)

`pkg/spawn/context.go` + `cmd/orch/spawn_cmd.go`

When `orch spawn` detects hotspots:
1. Injects `HotspotArea: true` and `HotspotFiles: [list]` into SPAWN_CONTEXT.md
2. Investigation agents see this context and include hotspot-aware routing in recommendations
3. Non-blocking — advisory only, supporting correct follow-up routing

**Files:** `pkg/spawn/context.go` (SpawnConfig fields), `cmd/orch/spawn_cmd.go` (injection)

**Rationale:** Each layer covers a gap the others can't:
- Layer 1 prevents manual bypass without proof of architect review
- Layer 2 prevents daemon from auto-spawning feature-impl in hotspot areas
- Layer 3 helps investigation agents recommend the correct follow-up path

**Trade-offs accepted:**
- More implementation surface than single-layer (three code paths)
- Daemon needs hotspot detection dependency (exposed as pkg-level function)
- Auto-detection may produce false positives if architect issue titles don't match well
- Slightly more friction on legitimate hotspot overrides (mitigated by ~30 min architect spawns)

---

## Structured Uncertainty

**What's tested:**
- Layer 1 spawn gate with architect-ref verification (unit tests in `pkg/spawn/gates/hotspot_test.go`)
- Layer 2 daemon escalation logic (unit tests in `pkg/daemon/architect_escalation_test.go`)
- Layer 3 context injection (spawn context template includes hotspot section)
- Auto-detection of prior architect reviews bypasses gate without flags
- The orch-go-1182 failure mode: `--force-hotspot` without `--architect-ref` now produces a blocking error

**What's untested:**
- End-to-end flow where daemon escalates to architect, architect completes, then daemon re-infers feature-impl with auto-bypass
- False positive rate of auto-detection (how often does it find the wrong architect issue?)
- Behavior when architect spawns are themselves rate-limited or fail

**What would change this:**
- If architect spawns become unavailable (all LLM providers down) — would need emergency bypass
- If false positive rate of auto-detection exceeds 10% — would need tighter matching
- If hotspot detection thresholds change significantly — may need to recalibrate which layers trigger

---

## Consequences

**Positive:**
- The orch-go-1182 failure mode is impossible without an architect having reviewed the area first
- Daemon-driven spawns are covered (Layer 2 closes the gap Layer 1 alone leaves open)
- Escape hatch preserved: `--force-hotspot --architect-ref` still works for legitimate overrides
- Advisory routing (Layer 3) prevents the problem from reaching gates in the first place
- Auto-detection reduces friction for areas with prior architect review

**Risks:**
- Three layers create three maintenance surfaces; if hotspot detection changes, all three need updating
- Auto-detection trusts `bd` output for architect issue matching — if beads data is stale, bypass may fail
- Daemon escalation adds a dependency on hotspot detection at spawn time, slightly increasing spawn latency

## Evidence

### Source Investigation
- `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` — Full design with decision forks and trade-off analysis

### Implementation
- `pkg/spawn/gates/hotspot.go` — Layer 1 (spawn gate with architect-ref verification)
- `pkg/spawn/gates/hotspot_test.go` — Layer 1 tests
- `pkg/daemon/architect_escalation.go` — Layer 2 (daemon skill escalation)
- `pkg/daemon/architect_escalation_test.go` — Layer 2 tests
- `pkg/spawn/context.go:561-562` — Layer 3 (HotspotArea/HotspotFiles in SpawnConfig)
- `cmd/orch/spawn_cmd.go:648-649` — Layer 3 (injection from hotspot result)

### Quick Entries (superseded by this decision)
- kb-3d169f: "force-hotspot requires architect-ref with verified closed architect issue"

## Auto-Linked Investigations

- .kb/investigations/2026-03-09-design-extraction-plan-three-near-critical.md
