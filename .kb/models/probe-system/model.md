# Model: Probe System

**Created:** 2026-03-20
**Updated:** 2026-03-20 (probed 2026-03-20)
**Status:** Active
**Synthesized from:** 8 investigations (Feb 13 - Mar 11, 2026)

## Summary

Probes are the knowledge system's falsifiability mechanism. They are lightweight (30-50 line) artifacts that test specific claims in existing models against reality, producing verdicts (confirms, extends, contradicts). The probe system has three layers: the **methodology** (how probes work epistemically), the **infrastructure** (routing, commit pipeline, verdict parsing), and the **feedback loop** (merge back into parent model).

The Feb 8, 2026 decision established probes as the default artifact when a model exists for a domain, replacing full investigations for confirmatory work. This shifted the knowledge system from accumulating 300-line investigations (414:29 ratio) to producing focused probes that feed models directly.

## Core Concepts

### Probe vs Investigation Decision Boundary

Decided at spawn time based on `kb context` model detection:

| Condition | Artifact | Rationale |
|-----------|----------|-----------|
| Model exists for domain (`HasInjectedModels=true`) | Probe | Confirmatory — test model claims |
| No model exists | Investigation | Novel exploration — build understanding |

**Routing is automatic** since Feb 14: spawn infrastructure detects model presence via `HasInjectedModels` field (populated from `KBContextFormatResult`) and conditionally shows probe vs investigation template in DELIVERABLES. Before this, routing was manual orchestrator judgment — a source of invisible friction.

### Probe Structure

Four mandatory sections (from `.orch/templates/PROBE.md`):
1. **Question** — What model claim are we testing?
2. **What I Tested** — Commands/code run (actual execution, not code review)
3. **What I Observed** — Actual output
4. **Model Impact** — Verdict: confirms/contradicts/extends which invariant

The template also includes frontmatter fields (`claim:`, `verdict:`) for machine-readable metadata.

### Two Verdict Formats

Probes use two verdict formats that both must be parsed:

| Format | Example | Source |
|--------|---------|--------|
| Structured | `**Verdict:** extends — model needs condition 5` | Real probes (e.g., completion-verification) |
| Checkbox | `- [x] **Confirms** invariant: gate catches defects` | PROBE.md template |

Both are parsed by `pkg/verify/probe_verdict.go` during `orch complete`.

### Probe Directory Structure

Probes live under their parent model: `.kb/models/{model-name}/probes/`

This makes discovery structural (browse a model's probes/ directory) rather than search-dependent. Agent B sees Agent A's recent probes without keyword search.

## Infrastructure

### Verdict Parsing (`orch complete`)

Added Feb 13. During completion, probes in `.kb/models/*/probes/` are scanned and their Model Impact verdicts surfaced in completion output. Matching is via spawn time comparison (workspace `.spawn_time` vs probe file modification time), avoiding git dependency.

**Key files:** `pkg/verify/probe_verdict.go` (ProbeVerdict type, parsing, workspace matching, formatting)

### Automatic Routing (spawn)

Added Feb 14. Three components wired together:
1. `pkg/spawn/kbcontext.go` — `hasInjectedModelContent()` detects model presence, populates `HasInjectedModels` and `PrimaryModelPath`
2. `pkg/spawn/config.go` — `HasInjectedModels` field on spawn.Config
3. `pkg/spawn/context.go` — DELIVERABLES template conditionally shows probe vs investigation instructions

### Commit Pipeline

Probe files in nested `.kb/models/{name}/probes/` directories were intermittently failing to commit because agents lacked explicit staging instructions. Fix (Feb 14): added git status checkpoint to worker-base Session Complete Protocol — agents now verify all `.kb/` files are committed before reporting Phase: Complete.

**Root cause:** Spawn context said "ensure committed" without specifying staging patterns. Agents using individual `git add` commands missed nested directories.

## Probing as Methodology

Beyond the infrastructure, the investigations reveal how probing works as an epistemic practice:

### Pattern: Probe Against Model Claims

The strongest probes test **specific falsifiable claims** in a model. Examples from these investigations:

- **macOS click-freeze probe** — Tested Phase 2 service state claims. Found skhd/yabai states were inverted from model. Model said "skhd re-enabled", reality: skhd disabled. This invalidated the elimination narrative.
- **Friction gate inventory** — Tested whether completion gates catch real defects. Found only 3 of 12 have healthy bypass:fail ratios. 73.4% of bypasses from 3 systemic patterns.
- **Knowledge accretion falsifiability** — Tested whether the theory is predictive or merely descriptive. Found: conditionally predictive, needs a 5th condition (non-trivial composition).
- **Exploration mode routing** — Tested whether adding skill content degrades routing accuracy. Found: 0/10 scenarios misrouted, knowledge content doesn't dilute.

### Pattern: Verification Requires Execution, Not Review

Probes that execute commands and observe real output are more valuable than probes that only review code. The macOS probe ran `launchctl print-disabled`, `pgrep`, `memory_pressure` to capture actual state. The friction gate probe analyzed 7,029 events from `events.jsonl`. Code-review-only probes miss the gap between what code says and what actually happens.

### Pattern: Probe Scope Discipline

Effective probes stay scoped to their model claim. The exploration mode probe could have expanded into a full skill content analysis but stayed focused: "Does adding 8 lines degrade existing routing?" — structural trace, token budget measurement, done. Scope creep turns probes into investigations, defeating the purpose.

## Failure Modes

### 1. Unmerged Probes (Knowledge Loop Break)

Probes that sit in `.kb/models/*/probes/` without being merged into the parent `model.md` break the feedback loop. The model stays stale while evidence accumulates in probe files. Worker-base skill now requires probe-to-model merge before Phase: Complete.

### 2. Investigation Masquerading as Probe

When agents write 200+ line investigations but file them as probes, the lightweight format is lost. The probe template is intentionally rigid (4 sections, 30-50 lines) to prevent this.

### 3. Commit Failures

Before the git status checkpoint fix, probe files in nested directories were silently left uncommitted. `orch complete` verifies commits exist but doesn't create them — agents are responsible for staging.

### 4. Model-less Domain Mis-routing

If `kb context` doesn't find a model for the domain, routing defaults to investigation. This is correct behavior, but means the first exploration of a topic always produces an investigation, not a probe. The probe pattern only activates once a model exists.

## Key Metrics (from Investigations)

- **414 investigations : 29 models** — ratio before probe system (Feb 8). As of Mar 20: 292 investigations : 47 models (ratio improved from 14.3:1 to 6.2:1)
- **Two verdict formats** — both must be parsed (Feb 13)
- **48 friction gates** — inventoried across spawn/completion/daemon, only 3 completion gates have healthy ratios
- **73.4% bypass rate** — from 3 systemic patterns (skill-class blindness, model blindness, blanket override)
- **0/10 misroutes** — exploration mode addition produced no false-positive routing

## Open Questions

1. Should probes support multiple models in one spawn? (currently uses first match)
2. Is the probe file modification time matching reliable across all edge cases?
3. What's the actual probe-to-model merge compliance rate? (no metrics collected)
4. Should there be a "probe failed to merge" escalation path beyond the worker-base requirement?

## Probes

- 2026-03-20: Knowledge Decay Verification — "weakens" verdict not implemented (only confirms/contradicts/extends parsed); template has gained frontmatter fields; investigation:model ratio improved from 14.3:1 to 6.2:1

## Evidence (Synthesized Investigations)

| Date | Investigation | Key Finding |
|------|-------------|-------------|
| 2026-02-13 | add-probe-verdict-parsing | Two verdict formats (structured + checkbox); spawn-time matching for workspace-probe association |
| 2026-02-13 | probe-capture-macos-service-state | Model Phase 2 claims inverted (skhd/yabai); demonstrates probe methodology against model claims |
| 2026-02-13 | probe-inventory-friction-gates | 48 gates across 3 subsystems; only build gate has healthy ratio; 73.4% bypasses from 3 systemic patterns |
| 2026-02-14 | fix-probe-commit-pipeline | Agents lack git staging instructions for nested probe directories; fixed with git status checkpoint |
| 2026-02-14 | probe-vs-investigation-routing | Automatic routing via HasInjectedModels; infrastructure existed but wasn't wired |
| 2026-02-17 | understand-openai-gpt-plugin | End-to-end model string tracing; example of systematic probe methodology across system boundaries |
| 2026-03-10 | probe-falsifiability-knowledge-accretion | Theory is conditionally predictive; needs 5th condition (non-trivial composition); 15+ counterexamples examined |
| 2026-03-11 | probe-exploration-mode-routing | Knowledge content additions don't dilute; 0/10 scenarios misrouted; structural analysis confirms model prediction |

## Related

- **Decision:** `.kb/decisions/2026-02-08-model-centric-probes-replace-investigations.md`
- **Template:** `.orch/templates/PROBE.md`
- **Infrastructure:** `pkg/verify/probe_verdict.go`, `pkg/spawn/kbcontext.go`, `pkg/spawn/context.go`
- **Models that use probes extensively:** completion-verification, harness-engineering, agent-lifecycle-state-model, knowledge-accretion, skill-content-transfer
