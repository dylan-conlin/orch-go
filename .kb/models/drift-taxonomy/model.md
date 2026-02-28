# Model: Drift Taxonomy

**Domain:** System coherence / Cross-boundary state management
**Last Updated:** 2026-02-28
**Synthesized From:**
- `.kb/investigations/2026-02-27-design-claude-config-dir-drift-elimination.md` — Config drift analysis and symlink solution
- `.kb/decisions/2026-02-14-model-staleness-detection.md` — Detect-Annotate-Queue design for model drift
- `.kb/models/architectural-enforcement/probes/2026-02-27-probe-config-dir-drift-scope.md` — Config drift as enforcement gap
- `kb-cli/.kb/agreements/` (5 YAML contracts) — Cross-project interface drift prevention
- `.kb/investigations/archived/2026-01-07-inv-orch-status-surface-drift-metrics.md` — Work alignment drift metrics
- `.kb/models/architectural-enforcement/model.md` — Drift as failure mode within enforcement

---

## Summary (30 seconds)

Drift is **duplicated state that silently diverges**. The system has four distinct drift domains — config, model, work alignment, and cross-boundary interface — each with different source/consumer relationships, failure signatures, and prevention strategies. The unifying insight is that all drift shares a common anatomy: a source of truth, a consumer that can fall out of sync, a **silent** failure mode (you don't know it happened), and a resolution that favors **structural elimination over reminders**. Detection at the point of consumption beats periodic sweeps because the harm happens when stale state is *used*, not when it *exists*.

---

## Core Mechanism

### The Anatomy of Drift

Every drift instance has four components:

```
Source of Truth  ──(should sync)──>  Consumer
       │                                │
       │  change happens here           │  staleness lives here
       │                                │
       └── detection point ─────────────┘
              (where do you notice?)
```

**Why drift is dangerous:** The failure mode is always **silent**. The consumer operates on stale state without knowing it. No error, no crash, no warning — just wrong behavior. An agent spawned with a stale model acts on outdated understanding. A personal session runs without hooks. A daemon drops a new `kb reflect` field. Each produces plausible-looking output that's subtly wrong.

### The Four Drift Domains

| Domain | Source of Truth | Consumer | What Drifts | Failure Signature |
|--------|----------------|----------|-------------|-------------------|
| **Config** | `~/.claude/` (work config) | `~/.claude-personal/` (personal config) | settings.json, CLAUDE.md, skills/, hooks/ | Personal sessions silently run without hooks, instructions, skills |
| **Model** | Code reality (Go files, git history) | `.kb/models/*/model.md` | File references, function names, architecture descriptions | Agents act on outdated understanding of how the system works |
| **Work Alignment** | Focus statement (`~/.orch/focus.json`) | Active agent work | Whether spawned work maps to strategic intent | Effort spent on work that doesn't serve the current goal |
| **Cross-Boundary** | Producer project (e.g., kb-cli output format) | Consumer project (e.g., orch-go daemon parser) | Interface contracts between independent codebases | Data silently dropped, stale instructions compete with tools |

### Detection Strategies (Ranked)

Three detection strategies exist, in order of effectiveness:

1. **At consumption** — Detect when stale state is about to be *used*. Catches drift at the moment of harm. Examples: spawn-time staleness annotations, `kb agreements check` in SessionStart.

2. **Periodic sweep** — Scan for drift on a schedule. Catches drift that accumulates between consumption checks. Examples: `kb reflect --type model-drift`, daemon model-drift reflection loop.

3. **At production** — Detect when the source changes. Theoretically best, but impractical — most source changes don't affect consumers, creating noise. Example: git hooks on every commit (rejected as too noisy).

**The consumption principle:** Detection should happen as close to the point of harm as possible. Spawn-time detection > periodic reflection > commit-time hooks.

### Prevention Strategies (Ranked)

Four prevention strategies, in order of strength:

1. **Structural elimination** — Make drift impossible by construction. Single source of truth, no copies. Example: symlinks for config drift (both configs are literally the same file).

2. **Executable contracts** — Machine-checked invariants that fail visibly when violated. Example: `kb agreements check` with YAML contracts and runnable checks.

3. **Annotate-and-surface** — Don't prevent drift, but ensure consumers *know* they're using stale state. Example: staleness annotations in SPAWN_CONTEXT.md.

4. **Document-and-remind** — Write down what should stay in sync. Always fails under cognitive load. Example: "remember to update both configs" (this is what we replaced).

**The elimination principle:** Prefer strategies that make drift impossible over strategies that detect it after the fact. Structural elimination > executable contracts > annotations > documentation.

### Critical Invariants

1. **Drift is always silent.** If it produced errors, it would be a bug, not drift. The defining characteristic is that stale state produces plausible-looking output. This means detection must be *active* — you must go looking for drift, because drift won't come to you.

2. **Detection at consumption beats detection at production.** The source changes frequently; only some changes affect the consumer. Checking at consumption is precise (this specific stale state is about to cause harm). Checking at production is noisy (most changes are irrelevant to this consumer).

3. **Prevention strength must match failure cost.** Config drift (wrong hooks → security gap) got structural elimination. Model drift (outdated context → suboptimal agent behavior) got annotate-and-surface. Work alignment drift (misallocated effort) got manual review. The stronger the prevention, the more it constrains; match to actual risk.

4. **Cross-boundary drift requires explicit contracts.** Within a single project, the compiler, tests, and linters catch inconsistencies. Across project boundaries, nothing enforces coherence unless you build it. This is why agreements exist — they're the only mechanism for cross-project drift.

---

## Why This Fails

### 1. Reminder-Based Prevention (Systemic, Observed Repeatedly)

**What happens:** "Remember to update both configs" / "Remember to update the model" / "Remember to check the interface." Under cognitive load, the reminder is forgotten.

**Root cause:** Reminders are additive to the task at hand. The person doing the work (changing a hook, refactoring code, adding a kb reflect type) is focused on that change, not on downstream consumers.

**Evidence:** Config drift accumulated across 5 files over weeks before the Task tool guard incident exposed it. Model drift accumulated to ~50% staleness rate across 24 models. Both were "known" but not acted on.

**Fix:** Gate Over Remind — structural elimination, executable contracts, or at minimum active detection. Never rely on humans or agents remembering to check.

### 2. Periodic-Only Detection (Observed: Model Drift Pre-Feb 2026)

**What happens:** Drift is detected on a schedule (weekly reflection, monthly audit) but harm occurs continuously between sweeps. During the gap, agents spawn with stale models and act on wrong information.

**Root cause:** The detection cadence is decoupled from the consumption cadence. Models are consumed at every spawn (dozens/day), but checked monthly.

**Evidence:** Model staleness was ~50% when the spawn-time detection was added. The staleness events JSONL showed stale models being served to agents repeatedly before the periodic sweep caught them.

**Fix:** Add consumption-time detection. Periodic sweeps remain valuable for catching drift that consumption-time checks miss, but they can't be the only layer.

### 3. Silent Drop at Interface Boundaries (Observed: kb-reflect-output-passthrough)

**What happens:** Producer adds a new field/type/format. Consumer parses with the old schema. New data is silently dropped — no error, no warning.

**Root cause:** No shared schema enforcement across project boundaries. Each project has its own types, and Go's JSON unmarshalling silently ignores unknown fields by default.

**Evidence:** The `kb-reflect-output-passthrough` agreement was created specifically because this happened — kb-cli added a reflection type, orch-go's `kbReflectOutput` struct didn't have the field, data was lost.

**Fix:** Executable contracts (agreements) that compare producer output keys against consumer struct fields. The `comm -23` check in the agreement catches missing fields.

### 4. Stale Shadow (Observed: skill-action-compliance, verification-model-accuracy)

**What happens:** Documentation/instructions describe a system that no longer exists. Consumers follow the stale instructions and produce wrong behavior or confusion.

**Root cause:** Documentation is a copy of understanding, not the understanding itself. When the system changes, the copy doesn't automatically update.

**Evidence:** Worker skills referenced `orch spawn` (a command workers can't run). Verification model described gates that had been renamed. Both produced confused agent behavior.

**Fix:** Agreements with `failure_mode: stale-shadow` checks that verify referenced artifacts still exist and match.

---

## Constraints

### Why Not Eliminate All Drift Structurally?

**Constraint:** Structural elimination (single source of truth, no copies) only works when the consumer needs *exactly* the same state as the source. When the consumer needs a *derived* or *interpreted* version, copies are inherent.

**Implication:** Config drift was eliminable (both accounts need the same settings.json). Model drift is not — models are human-authored interpretations of code, not copies of code. You can't symlink a model to the source code it describes.

**This enables:** Choosing the right prevention strategy per domain instead of force-fitting one approach
**This constrains:** Model drift and cross-boundary drift will always need detection + remediation, never full elimination

### Why Not Auto-Fix Detected Drift?

**Constraint:** Models require synthesis (orchestrator work). Cross-boundary contracts require understanding both sides. Auto-fixing creates review obligations faster than they can be processed (Verification Bottleneck).

**Implication:** Detection creates *awareness*, not resolution. Resolution is always a human/orchestrator decision.

**This enables:** Detection can run at high frequency without overwhelming the system
**This constrains:** There will always be a backlog of detected-but-unresolved drift; the system must manage this backlog (backpressure limits, circuit breakers)

### Why Cross-Boundary Drift Needs Its Own Mechanism?

**Constraint:** Within a project, the compiler, tests, and type system enforce coherence. Across projects, there is no shared compilation unit — each project builds independently.

**Implication:** Cross-boundary drift is invisible to all single-project tools. `go build`, `go test`, linters — none of them can tell you that kb-cli's output format changed in a way that orch-go's parser doesn't handle.

**This enables:** Agreements as a dedicated cross-boundary mechanism
**This constrains:** Every cross-project interface needs an explicit agreement, or it will drift silently. Coverage is currently thin (5 agreements for ~20+ interfaces).

---

## Evolution

**Jan 7, 2026:** First drift awareness — `orch status` gained session metrics (time since spawn, spawn count) for detecting orchestrator behavioral drift. Work alignment drift surfaced as a concept.

**Jan 8, 2026:** Doc drift prevention — `orch doctor --docs` for CLI documentation drift detection. Established the "passive tracking + doctor surfacing" pattern.

**Feb 14, 2026:** Model drift formalized — Decision: Detect-Annotate-Queue with `code_refs` blocks, spawn-time detection, and `kb reflect --type model-drift`. First systematic treatment of model-code drift.

**Feb 20, 2026:** Model drift probes — Three major probes (agent-lifecycle, spawn-architecture, orchestration-cost-economics) documented ~50% staleness rate and classified drift as CONTRADICTS + EXTENDS. Revealed that core *concepts* survive drift but *implementation details* don't.

**Feb 27, 2026:** Config drift elimination — Symlinks structurally eliminated config drift between `~/.claude/` and `~/.claude-personal/`. Established structural elimination as the gold standard.

**Feb 27, 2026:** Agreements system — `kb agreements` Phase 1-3 implemented. Five cross-project contracts seeded. Introduced failure mode taxonomy (silent-drop, stale-shadow, unchecked-assumption). First mechanism for cross-boundary drift.

**Feb 28, 2026:** Drift taxonomy model created — Synthesized four domains into unified framework. This model.

**Feb 28, 2026:** `orch drift` command fixed — Replaced raw ID dump with skill-grouped alignment analysis. Work alignment drift now has real detection.

---

## References

**Investigations:**
- `.kb/investigations/2026-02-27-design-claude-config-dir-drift-elimination.md` — Config drift root cause and symlink solution
- `.kb/investigations/archived/2026-01-08-inv-drift-prevention-auto-track-cli.md` — CLI doc drift tracking pattern
- `.kb/investigations/archived/2026-01-07-inv-orch-status-surface-drift-metrics.md` — Work alignment drift metrics

**Decisions informed by this model:**
- `.kb/decisions/2026-02-14-model-staleness-detection.md` — Detect-Annotate-Queue for model drift

**Related models:**
- `.kb/models/architectural-enforcement/model.md` — Drift as enforcement gap (Gate Over Remind principle)
- `.kb/models/entropy-spiral/model.md` — Drift as entropy accumulation

**Primary Evidence (Verify These):**
<!-- code_refs: machine-parseable file references for staleness detection -->
- `pkg/spawn/staleness_events.go` — Staleness event recording at spawn time
- `pkg/spawn/kbcontext.go` — Spawn-time model staleness detection
- `pkg/daemon/model_drift_reflection.go` — Daemon periodic model drift issue creation
- `cmd/orch/focus.go:186-205` — `orch drift` command (work alignment drift)
- `cmd/orch/drift_test.go` — Drift command tests
<!-- /code_refs -->

**Cross-project evidence:**
- `kb-cli/.kb/agreements/*.yaml` — Five seeded cross-boundary contracts
- `kb-cli/cmd/kb/agreements.go` — Agreements check/list implementation
