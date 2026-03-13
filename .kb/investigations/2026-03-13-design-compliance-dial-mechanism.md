## Summary (D.E.K.N.)

**Delta:** The daemon has 8 compliance mechanisms that can be cleanly separated into "always-on safety" (dedup, circuit breakers, rate limits) and "tunable compliance" (extraction gate, architect escalation, verification threshold, verify levels, review tiers), enabling a 4-level compliance dial.

**Evidence:** Read all daemon compliance code: spawn_gate.go (5 dedup gates), verification_tracker.go (pause after N completions), extraction.go (hotspot extraction), architect_escalation.go (skill routing), invariants.go (self-check), issue_selection.go (label filtering), spawn_execution.go (pipeline), plus the full verify package (V0-V3 levels, 23 gates, 4 review tiers).

**Knowledge:** The verify package already has a level system (V0-V3) and review tiers (Auto/Scan/Review/Deep). Compliance-as-dial extends this pattern to daemon-side gates with per-(skill, model) resolution. The key architectural insight is that compliance level should be resolved per-spawn, not per-daemon — different skill+model combos have different reliability profiles.

**Next:** Implement in 3 phases: (1) ComplianceConfig type + resolution logic, (2) gate awareness + verify integration, (3) measurement feedback loop.

**Authority:** architectural - Cross-component design affecting daemon, verify, spawn, and config packages

---

# Investigation: Compliance Dial Mechanism Design

**Question:** How should daemon compliance be made tunable so that as models improve, enforcement overhead can be dialed down without removing safety mechanisms?

**Started:** 2026-03-13
**Updated:** 2026-03-13
**Owner:** architect
**Phase:** Complete
**Next Step:** None — design ready for implementation
**Status:** Complete
**Model:** daemon-autonomous-operation

---

## Findings

### Finding 1: Clear Safety vs Compliance Boundary

**Evidence:** Analysis of all daemon mechanisms reveals a clean split between mechanisms that prevent system-level failures (duplicates, cascades, resource exhaustion) and mechanisms that prevent quality failures (bad code, architectural drift, unmaintained knowledge).

**Always-On Safety (never tunable):**

| Mechanism | Purpose | Why always-on |
|-----------|---------|---------------|
| Dedup pipeline L1-L5 (spawn_gate.go) | Prevent duplicate spawns | Duplicates waste resources, create confusion |
| PID lock L0 | Prevent concurrent daemons | Multiple daemons corrupt shared state |
| Beads status update L6 (fail-fast) | Primary dedup gate | Feb 14 incident: 10 duplicates when this failed |
| CompletionFailureTracker | Pause on broken completion | Prevents orphaning agents silently |
| BeadsCircuitBreaker | Exponential backoff on beads failures | Prevents cascade failures |
| Rate limiter (MaxSpawnsPerHour) | Prevent runaway spawning | Resource protection |
| Pool/capacity management | Limit concurrent agents | Resource protection |
| Spawn tracker disk persistence | Survive restarts | Dedup across daemon lifecycle |

**Tunable Compliance:**

| Mechanism | Current behavior | What it prevents |
|-----------|-----------------|------------------|
| Extraction gate (extraction.go) | Active for files >1500 lines | Quality: agent works on unmaintainable file |
| Architect escalation (architect_escalation.go) | Active for all hotspot types | Quality: code changes without architectural review |
| VerificationTracker (verification_tracker.go) | Pause after 3 completions | Oversight: unchecked autonomous operation |
| Verify levels V0-V3 (pkg/verify/) | V1-V3 per skill | Quality: agents skip required deliverables |
| Review tiers Auto-Deep (pkg/verify/) | Scan-Deep per skill | Oversight: orchestrator review depth |
| InvariantChecker (invariants.go) | Active, threshold 3 | Safety-adjacent: catches daemon bugs |
| SYNTHESIS.md requirement | Required for full tier | Knowledge: captured learnings |
| Phase reporting enforcement | Required | Oversight: orchestrator visibility |

**Source:** pkg/daemon/spawn_gate.go, pkg/daemon/verification_tracker.go, pkg/daemon/extraction.go, pkg/daemon/architect_escalation.go, pkg/daemon/invariants.go, pkg/verify/

**Significance:** The separation is clean enough that we can build a compliance dial that adjusts the "tunable" column without touching the "always-on" column. This is the fundamental architectural insight.

---

### Finding 2: Verify Package Already Has Leveled Infrastructure

**Evidence:** The verify package implements V0-V3 verification levels and 4 review tiers (Auto/Scan/Review/Deep), with per-skill defaults and per-issue-type minimums. This is exactly the pattern we need for compliance-as-dial.

- V0 (Acknowledge): Phase Complete only — skills: issue-creation, capture-knowledge
- V1 (Artifacts): V0 + deliverable/constraint checks — skills: investigation, architect, research
- V2 (Evidence): V1 + test evidence, build, git diff — skills: feature-impl, systematic-debugging
- V3 (Behavioral): V2 + visual verification — skills: debug-with-playwright
- Review tiers: Auto (no human) → Scan (glance) → Review (full) → Deep (behavioral)

The compliance dial needs to cap these levels, not replace them. At higher compliance, the full skill defaults apply. At lower compliance, a ceiling is imposed.

**Source:** pkg/verify/level.go, pkg/spawn/verify_level.go, pkg/spawn/review_tier.go

**Significance:** We don't need to invent a new leveling system — we cap the existing one. This reduces implementation scope significantly.

---

### Finding 3: Compliance Should Be Per-Spawn, Not Per-Daemon

**Evidence:** Different (skill, model) combinations have dramatically different reliability profiles:
- opus+feature-impl has high completion rates (production-proven)
- GPT-5.2-codex has 67-87% stall rates on protocol-heavy skills
- sonnet+investigation may be reliable for simple tasks but unreliable for complex ones

The daemon's existing `InferModelFromSkill()` already makes per-skill model decisions. Compliance level should follow the same pattern.

**Source:** daemon-autonomous-operation model (Section: "Model Incompatibility Stall"), pkg/daemon/skill_inference.go

**Significance:** A single global compliance dial is too coarse. The config needs per-skill, per-model, and per-(skill+model) override capability with a global default.

---

## Synthesis

**Key Insights:**

1. **Safety and compliance are cleanly separable** — Safety mechanisms prevent system failures (dedup, cascades, resource exhaustion) and should never be tunable. Compliance mechanisms prevent quality failures and can be relaxed as models improve.

2. **The verify package already has the right abstraction** — V0-V3 levels and review tiers are exactly the per-spawn compliance dial. The daemon-side gates (extraction, architect escalation) need the same pattern applied.

3. **Per-spawn resolution is the right granularity** — Global compliance is too coarse. Per-(skill, model) resolution matches how reliability actually varies across the system.

**Answer to Investigation Question:**

Make compliance tunable by introducing a `ComplianceLevel` type (Strict/Standard/Relaxed/Autonomous) that resolves per-spawn based on (skill, model) with a global default. Each level defines a configuration surface: which daemon-side gates are active, what verify level ceiling applies, what review tier is used, and what verification threshold governs human oversight. Always-on safety mechanisms are excluded from the dial entirely.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 8 compliance mechanisms identified and categorized (verified: read every source file)
- ✅ Verify package already has V0-V3 levels and 4 review tiers (verified: read level.go, review_tier.go)
- ✅ Safety/compliance boundary is clean — no mechanism straddles both categories (verified: traced all code paths)

**What's untested:**

- ⚠️ Whether 4 compliance levels is the right number (could be 3 or 5)
- ⚠️ Whether per-spawn resolution introduces too much config complexity
- ⚠️ Whether the measurement feedback loop metrics are the right ones for auto-adjustment
- ⚠️ Whether InvariantChecker belongs in "safety" or "compliance" (it catches daemon bugs, which is safety-adjacent)

**What would change this:**

- If a compliance mechanism was discovered that can't cleanly be made per-spawn (currently all can)
- If the existing verify level system proved insufficient for the compliance dial (currently it's a natural fit)
- If always-on safety mechanisms were found to have significant performance overhead worth tuning (currently they don't)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| ComplianceLevel type + resolution | architectural | Cross-component design: daemon, verify, spawn, config |
| Gate compliance-awareness | implementation | Each gate's modification is scoped to its own package |
| Measurement feedback loop | architectural | Connects events, completion, and config systems |

### Recommended Approach ⭐

**Compliance-as-Dial with Per-Spawn Resolution** — Add a `ComplianceConfig` to daemon config that resolves a `ComplianceLevel` per (skill, model) tuple, then use that level to gate daemon-side checks and cap verify-side levels.

**Why this approach:**
- Builds on existing verify package infrastructure (V0-V3, review tiers)
- Clean safety/compliance separation means no risk of disabling critical safety gates
- Per-spawn resolution matches actual reliability variance across skill+model combos
- Backward compatible: default level = Strict = current behavior

**Trade-offs accepted:**
- Config complexity increases (per-skill, per-model, per-combo overrides)
- No auto-adjustment in Phase 1 (manual level setting only)

---

# COMPLIANCE LEVEL SCHEMA

## Four Compliance Levels

### Level 0: Strict (current behavior, default)

Use when: Untested model/skill combos, new skills, early-stage system.

| Mechanism | Behavior |
|-----------|----------|
| Extraction gate | Active (files >1500 lines trigger extraction) |
| Architect escalation | Active (all hotspot types route to architect) |
| Verification threshold | 3 (pause after 3 unverified completions) |
| Verify level | Skill default (V1-V3 depending on skill) |
| Review tier | Skill default (Scan-Deep depending on skill) |
| InvariantChecker | Active (threshold 3) |
| SYNTHESIS.md | Required for full-tier spawns |
| Phase reporting | Required (flagged as unresponsive if missing) |

### Level 1: Standard

Use when: Proven model/skill combos with >90% completion rate and >85% gate pass rate.

| Mechanism | Behavior |
|-----------|----------|
| Extraction gate | Active (unchanged — file size is objective, not model-dependent) |
| Architect escalation | Active only for fix-density hotspots (skip bloat-size, churn) |
| Verification threshold | 8 (more autonomous operation between reviews) |
| Verify level | Capped at V1 for knowledge skills, V2 for impl skills |
| Review tier | Scan for impl skills (down from Review) |
| InvariantChecker | Active (threshold 5) |
| SYNTHESIS.md | Required for full-tier spawns |
| Phase reporting | Required |

### Level 2: Relaxed

Use when: High-confidence model/skill combos with >95% completion rate and >95% gate pass rate.

| Mechanism | Behavior |
|-----------|----------|
| Extraction gate | Active (unchanged) |
| Architect escalation | Disabled |
| Verification threshold | 20 (effectively overnight-capable) |
| Verify level | Capped at V1 for all skills |
| Review tier | Auto for knowledge skills, Scan for impl skills |
| InvariantChecker | Active (threshold 10) |
| SYNTHESIS.md | Optional (not required for completion) |
| Phase reporting | Advisory only (not enforced) |

### Level 3: Autonomous

Use when: Production-grade model/skill combos with >99% completion rate, or when maximum throughput is needed.

| Mechanism | Behavior |
|-----------|----------|
| Extraction gate | Active (unchanged — this is cheap and objective) |
| Architect escalation | Disabled |
| Verification threshold | 0 (disabled — no verification pause) |
| Verify level | V0 for all skills (Phase Complete only) |
| Review tier | Auto for all skills |
| InvariantChecker | Disabled |
| SYNTHESIS.md | Not required |
| Phase reporting | Advisory only |

**Design decision: Extraction gate stays active at all levels.** Unlike other compliance mechanisms, extraction is objective (file size is measurable, not a judgment call) and cheap (one hotspot check per spawn). Even the best model can't produce good code in a 2000-line file — this is a physical constraint, not a trust issue.

---

# RUNTIME CONFIG

## Config Format (~/.orch/config.yaml)

```yaml
daemon:
  # Existing fields preserved
  max_agents: 5
  max_spawns_per_hour: 20
  verification_pause_threshold: 5  # Manual override (wins over compliance-derived)

  # NEW: Compliance configuration
  compliance:
    # Global default level
    default: strict  # strict | standard | relaxed | autonomous

    # Per-skill overrides
    skills:
      feature-impl: standard
      investigation: standard
      architect: strict          # Always strict for design work
      systematic-debugging: standard
      capture-knowledge: relaxed
      issue-creation: autonomous

    # Per-model overrides
    models:
      opus: standard
      sonnet: strict
      haiku: strict
      flash: strict

    # Per-(skill+model) overrides (highest precedence)
    combos:
      "opus+feature-impl": relaxed
      "opus+investigation": standard
      "opus+architect": strict
      "sonnet+feature-impl": standard
```

## Resolution Order

```
combo(skill+model) > skill > model > default
```

In Go:

```go
func (c *ComplianceConfig) Resolve(skill, model string) ComplianceLevel {
    // 1. Check combo (highest precedence)
    key := model + "+" + skill
    if level, ok := c.Combos[key]; ok {
        return level
    }
    // 2. Check skill
    if level, ok := c.Skills[skill]; ok {
        return level
    }
    // 3. Check model
    if level, ok := c.Models[model]; ok {
        return level
    }
    // 4. Global default
    return c.Default
}
```

## Interaction with Existing Config

The compliance level DERIVES values for existing config fields. Manual overrides in the config win over compliance-derived values:

| Existing Field | Compliance-Derived | Manual Override |
|---------------|-------------------|-----------------|
| `verification_pause_threshold` | Strict=3, Standard=8, Relaxed=20, Autonomous=0 | If set explicitly, wins |
| `invariant_violation_threshold` | Strict=3, Standard=5, Relaxed=10, Autonomous=0 | If set explicitly, wins |
| `invariant_check_enabled` | Strict/Standard/Relaxed=true, Autonomous=false | If set explicitly, wins |

New compliance-only fields (no existing equivalent):

| Field | Type | Per-spawn? | Description |
|-------|------|------------|-------------|
| `verify_level_cap` | V0-V3 | Yes | Maximum verify level for this compliance level |
| `review_tier_cap` | Auto-Deep | Yes | Maximum review tier |
| `architect_escalation_enabled` | bool | Yes | Whether hotspot escalation fires |
| `synthesis_required` | bool | Yes | Whether SYNTHESIS.md is mandatory |
| `phase_enforcement` | required/advisory | Yes | Whether missing phases block completion |

---

# THE 80/20 ARCHITECTURE

## What the Daemon Looks Like at Compliance Level "Autonomous" (20% compliance)

### Spawn Pipeline

```
┌──────────────────────────────────────────────────────────────┐
│  Daemon OnceExcluding (Autonomous compliance)                │
│                                                              │
│  ALWAYS-ON SAFETY (unchanged):                               │
│  1. Check CompletionFailureTracker                           │
│  2. Check RateLimiter                                        │
│  3. NextIssueExcluding (label filtering, dependency check)   │
│  4. InferSkill + InferModel                                  │
│  5. Dedup pipeline (L1-L5 all gates)                         │
│  6. Pool slot acquisition                                    │
│  7. Beads status update (fail-fast L6)                       │
│  8. Spawn                                                    │
│                                                              │
│  SKIPPED (compliance = autonomous):                          │
│  ✗ VerificationTracker pause check (threshold = 0)           │
│  ✗ Extraction gate (still runs — objective, not trust-based) │
│  ✗ Architect escalation                                      │
│  ✗ InvariantChecker                                          │
└──────────────────────────────────────────────────────────────┘
```

### Completion Pipeline

```
┌──────────────────────────────────────────────────────────────┐
│  CompletionOnce (Autonomous compliance)                      │
│                                                              │
│  ALWAYS-ON:                                                  │
│  1. Detect Phase: Complete                                   │
│  2. Record in VerificationTracker (no pause, but count)      │
│  3. Label daemon:ready-review → auto-close                   │
│                                                              │
│  AT AUTONOMOUS LEVEL:                                        │
│  - Verify level: V0 (Phase Complete check only)              │
│  - Review tier: Auto (no orchestrator review)                │
│  - SYNTHESIS.md: Not checked                                 │
│  - Test evidence: Not checked                                │
│  - Git diff: Not checked                                     │
│  - Build/vet: Not checked                                    │
│                                                              │
│  RESULT: Agent reports Phase: Complete → auto-closed.         │
│  No human in the loop.                                        │
└──────────────────────────────────────────────────────────────┘
```

### Which Gates Remain (Always-On Safety)

| Gate | Category | Why always-on |
|------|----------|---------------|
| SpawnTrackerGate (L1) | Dedup | Prevents double-spawn within TTL |
| SessionDedupGate (L2) | Dedup | Prevents spawn when session exists |
| TitleDedupMemoryGate (L3) | Dedup | Prevents content-duplicate spawns |
| TitleDedupBeadsGate (L4) | Dedup | Prevents cross-restart content dupes |
| FreshStatusGate (L5) | Dedup | Catches TOCTOU races |
| Beads status update (L6) | Dedup | Primary persistent dedup gate |
| PID lock (L0) | Process | Single instance enforcement |
| CompletionFailureTracker | Health | Prevents orphaning on broken completion |
| BeadsCircuitBreaker | Health | Prevents cascade on beads failure |
| RateLimiter | Resource | Prevents runaway spawning |
| WorkerPool | Resource | Limits concurrent agents |

### How the Daemon Struct Changes

```go
type Daemon struct {
    // ... all existing fields unchanged ...

    // NEW: Compliance configuration for per-spawn level resolution.
    // When nil, defaults to ComplianceStrict (current behavior).
    ComplianceConfig *ComplianceConfig
}
```

The ComplianceConfig is consulted in OnceExcluding after skill and model inference:

```go
func (d *Daemon) OnceExcluding(skip map[string]bool) (*OnceResult, error) {
    // ... existing safety checks (completion health, rate limit) ...

    // Resolve compliance level for this (skill, model) combo
    level := ComplianceStrict // default
    if d.ComplianceConfig != nil {
        level = d.ComplianceConfig.Resolve(skill, inferredModel)
    }

    // Verification pause: only check at Strict/Standard/Relaxed
    // (Autonomous has threshold=0 so IsPaused() is always false)
    if d.VerificationTracker != nil && d.VerificationTracker.IsPaused() {
        // ... existing pause logic ...
    }

    // Extraction gate: always runs (objective, not trust-based)
    if d.HotspotChecker != nil {
        extraction := CheckExtractionNeeded(issue, d.HotspotChecker)
        // ... existing extraction logic ...
    }

    // Architect escalation: only at Strict, or Standard for fix-density
    if level <= ComplianceStandard && d.HotspotChecker != nil {
        escalation := CheckArchitectEscalation(issue, skill, d.HotspotChecker,
            d.PriorArchitectFinder)
        if level == ComplianceStandard && escalation != nil {
            // Only escalate for fix-density hotspots at Standard
            if escalation.HotspotType != "fix-density" {
                escalation = nil
            }
        }
        // ... apply escalation if non-nil ...
    }

    // ... rest of spawn (dedup pipeline, pool, status update, spawn) ...
}
```

The compliance level is also passed through to the spawn context so the verify package knows the ceiling:

```go
// In spawn context generation
spawnConfig.ComplianceLevel = level
// This sets verify level cap and review tier in SPAWN_CONTEXT.md
```

---

# MEASUREMENT FEEDBACK LOOP

## Metrics That Indicate Reliability

Track per-(skill, model) combo:

| Metric | What it measures | Source | Threshold for upgrade |
|--------|-----------------|--------|----------------------|
| **Completion rate** | % of spawns that reach Phase: Complete | events.jsonl (session.spawned → agent.completed) | >90% for Standard, >95% for Relaxed, >99% for Autonomous |
| **Gate pass rate** | % of completions that pass all verify gates | verify package results logged at completion | >85% for Standard, >95% for Relaxed, >99% for Autonomous |
| **Human override rate** | % of auto-completed work that human rejects or reverts | orch complete outcomes | <5% for Standard, <2% for Relaxed, <1% for Autonomous |
| **Rework rate** | % of completed issues that spawn follow-up fix issues | beads dependency graph | <10% for Standard, <5% for Relaxed, <2% for Autonomous |
| **Phase timeout rate** | % of agents flagged unresponsive | daemon phase timeout detection | <10% for Standard, <5% for Relaxed |

## Connection to Harness Measurement Work

The existing harness work already tracks:
- Agent completion outcomes (success/partial/blocked/failed)
- Skill inference accuracy
- Model-specific stall rates

The compliance feedback loop extends this with:
1. **Per-combo metrics store** at `~/.orch/compliance-metrics.jsonl`
2. **Compliance suggestion command**: `orch compliance suggest` — reads metrics, recommends level changes
3. **Optional auto-adjustment** (future): daemon checks metrics at startup and adjusts compliance levels automatically

## Feedback Loop Cycle

```
┌──────────────────────────────────────────────────────────────┐
│  Feedback Loop                                               │
│                                                              │
│  1. Daemon spawns at current compliance level                │
│  2. Agents complete (or fail)                                │
│  3. Completion metrics recorded per (skill, model)           │
│  4. Periodically: orch compliance suggest                    │
│     - "opus+feature-impl: 97% completion, 96% gates pass    │
│       → recommend upgrade from Standard to Relaxed"          │
│  5. Dylan reviews suggestion, updates config                 │
│  6. Daemon picks up new config on next poll                  │
│                                                              │
│  SAFETY: Compliance can only be LOWERED manually (no auto).  │
│  Auto-adjustment is future work and requires Dylan approval. │
└──────────────────────────────────────────────────────────────┘
```

## Metric Collection Points

| Event | Where to collect | What to record |
|-------|-----------------|----------------|
| Spawn success | spawnIssue() in spawn_execution.go | skill, model, compliance_level, issue_id |
| Spawn failure | spawnIssue() error path | skill, model, compliance_level, failure_reason |
| Completion success | CompletionOnce() | skill, model, compliance_level, gates_passed, gates_failed |
| Completion failure | CompletionOnce() error path | skill, model, compliance_level, failure_reason |
| Human override | orch complete with changes | skill, model, compliance_level, override_type |
| Verification pause | VerificationTracker.IsPaused() | compliance_level, completions_since_verification |

---

# MIGRATION PATH

## Phase 1: Config + Resolution (1-2 PRs)

**Goal:** Add ComplianceLevel type and config parsing. No behavioral changes.

1. Create `pkg/daemonconfig/compliance.go`:
   - `ComplianceLevel` type (Strict/Standard/Relaxed/Autonomous)
   - `ComplianceConfig` struct with Default, Skills, Models, Combos
   - `Resolve(skill, model) ComplianceLevel` method
   - `DeriveVerificationThreshold(level) int` helper
   - `DeriveInvariantThreshold(level) int` helper

2. Add `Compliance ComplianceConfig` to `daemonconfig.Config`

3. Parse compliance section from `~/.orch/config.yaml` in config loading

4. Wire `ComplianceConfig` into `Daemon` struct initialization

5. Add `orch compliance status` command showing current levels

**Behavioral change:** None. Default = Strict = current behavior.

**Test:** Verify `Resolve()` with all precedence levels. Verify backward compatibility (no compliance config = Strict).

## Phase 2: Gate Awareness (2-3 PRs)

**Goal:** Make daemon-side and verify-side gates compliance-aware.

PR 2a — Daemon-side gates:
1. Architect escalation in `OnceExcluding()`: check compliance level before running `CheckArchitectEscalation()`
2. VerificationTracker: derive threshold from compliance level (with manual override)
3. InvariantChecker: derive threshold from compliance level (with manual override)

PR 2b — Verify-side integration:
1. Pass compliance level through spawn context (add to `SpawnConfig`)
2. In verify package: cap verify level based on compliance level
3. In verify package: cap review tier based on compliance level
4. SYNTHESIS.md requirement: make conditional on compliance level

PR 2c — Phase enforcement:
1. Phase reporting: make enforcement level configurable (required/advisory)
2. Phase timeout detection: respect compliance level

**Behavioral change:** Yes, but only for non-default compliance levels. Users who don't set compliance config see no change.

**Test:** For each gate, test at each compliance level. Verify default (Strict) matches current behavior exactly.

## Phase 3: Measurement + Feedback (2-3 PRs)

**Goal:** Collect metrics and surface upgrade recommendations.

PR 3a — Metric collection:
1. Add compliance level to spawn events in events.jsonl
2. Add gate results to completion events
3. Create `~/.orch/compliance-metrics.jsonl` store

PR 3b — Analysis and suggestions:
1. Add `orch compliance suggest` command
2. Reads metrics, computes per-(skill, model) rates
3. Recommends level changes based on thresholds
4. Output: human-readable table with current level, metrics, suggested level

PR 3c (optional, future) — Auto-adjustment:
1. Daemon reads metrics at startup
2. If a combo consistently meets upgrade thresholds, suggest in logs
3. Auto-upgrade requires explicit opt-in flag (`compliance.auto_adjust: true`)
4. Auto-DOWNGRADE is automatic (if metrics deteriorate, tighten compliance)

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Daemon struct, OnceExcluding, safety check ordering
- `pkg/daemon/spawn_gate.go` — SpawnPipeline, 5 gate implementations
- `pkg/daemon/verification_tracker.go` — VerificationTracker, pause/resume mechanism
- `pkg/daemon/extraction.go` — Extraction gate, hotspot detection
- `pkg/daemon/architect_escalation.go` — Architect escalation, hotspot routing
- `pkg/daemon/invariants.go` — InvariantChecker, self-check assertions
- `pkg/daemon/spawn_execution.go` — spawnIssue, buildSpawnPipeline, rollback
- `pkg/daemon/issue_selection.go` — NextIssueExcluding, label filtering
- `pkg/daemonconfig/config.go` — Config struct, DefaultConfig
- `pkg/verify/` — Full verification package (V0-V3, 23 gates, review tiers, escalation levels)
- `.kb/models/daemon-autonomous-operation/model.md` — Daemon model (39+ investigations)

**Related Artifacts:**
- **Model:** `.kb/models/daemon-autonomous-operation/model.md` — Parent model for daemon behavior
- **Guide:** `.kb/guides/daemon.md` — Procedural daemon guide
