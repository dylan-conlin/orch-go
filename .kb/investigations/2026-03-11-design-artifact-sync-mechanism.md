<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed artifact sync mechanism to address recurring drift where agents ship features but don't update referencing artifacts (CLAUDE.md, orchestrator skill, guides). Evaluated per-agent gate vs periodic sync agent. Recommends periodic sync agent with completion-time diff capture — extends the existing `pkg/modeldrift` pattern to cover all artifact classes.

**Evidence:** 10+ recent commits added commands/flags/events (--explore, orch review synthesize, exploration-judge, harness report, gate_decision events) without updating CLAUDE.md or guides. Prior soft fixes (skill checklists, completion reminders) haven't held — consistent with harness model prediction that convention without gate erodes. The existing `pkg/modeldrift` already implements detect→aggregate→issue for .kb/models/ but covers zero other artifact classes.

**Knowledge:** Per-agent gates fail because the gatable party (feature-impl agent) can't reasonably update cross-cutting artifacts (CLAUDE.md, orchestrator skill). The sync agent pattern (OpenAI's "garbage collection") decouples feature work from artifact maintenance. The modeldrift package provides the architectural precedent: detect at spawn/complete time, aggregate events, spawn dedicated maintenance agents.

**Next:** Implementation in 3 phases: (1) artifact manifest + change-scope tagging at completion, (2) `orch sync` command that diffs recent commits against artifact inventory, (3) daemon integration for periodic sync sweeps.

**Authority:** architectural — cross-component change affecting completion pipeline, spawn context, and artifact maintenance workflow

---

# Investigation: Design Artifact Sync Mechanism

**Question:** How should the system keep referencing artifacts (CLAUDE.md, skills, guides, models) in sync with feature work, given that soft conventions (checklists, reminders) have consistently failed?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Architect agent (orch-go-w052w)
**Phase:** Complete
**Next Step:** Create implementation issues
**Status:** Complete

---

## The Problem

Agents ship features but don't update the artifacts that reference those features. This session alone shipped `--explore` flag, `orch review synthesize`, `exploration-judge` skill, and gate signal-vs-noise fixes — none updated CLAUDE.md or the orchestrator skill.

**Evidence from last 20 commits:**

| Commit | Added | Should Have Updated | Updated? |
|--------|-------|--------------------|-|
| `0f397a147` | `--explore` flag | CLAUDE.md Commands | No |
| `44fbe47ef` | `exploration-judge` skill | CLAUDE.md skills docs | No |
| `7bb5bc4cc` | `orch review synthesize` | CLAUDE.md Commands | No |
| `956fffc78` | `orch harness report` | CLAUDE.md Commands | No |
| `32ab78cb1` | Dashboard `/harness` page | `.kb/guides/dashboard.md` | No |
| `54b95acae` | `beads_id` on `gate_decision` events | CLAUDE.md Event Types | No |
| `5a3203260` | `accretion.snapshot` event | CLAUDE.md Event Types | No |

**Why soft fixes fail:** The harness-engineering model predicts this: "constraint-type content in context-type containers dilutes at scale." Skill checklists saying "update CLAUDE.md" are soft harness. No gate enforces it. Under time pressure, agents skip it.

---

## Two Designs Evaluated

### Option 1: Per-Agent Completion Gate

**Mechanism:** At `orch complete` time, detect that the agent modified files covered by an artifact (e.g., `cmd/orch/` → CLAUDE.md), block completion until the referencing artifact is updated.

**Pros:**
- Immediate — no drift lag
- Clear accountability (the agent that changed code must update docs)
- Extends existing gate pipeline (`pkg/verify/check.go`)

**Cons:**
- **High false positive rate.** Most `cmd/orch/` changes don't need CLAUDE.md updates (internal refactors, bug fixes). The gate can't distinguish "added new command" from "fixed bug in existing command."
- **Wrong gatable party.** Feature-impl agents shouldn't edit the orchestrator skill or CLAUDE.md. That's a different skill scope — an agent implementing `--explore` in spawn logic doesn't have context about what the orchestrator skill should say about exploration.
- **Scope explosion.** To know which artifacts reference which code, you need a mapping (artifact → files). CLAUDE.md references ~50 files. The gate would fire on nearly every commit to `cmd/orch/` or `pkg/`.
- **Violates "gate must be passable by gated party"** (from hotspot enforcement design). The agent being gated can't reasonably satisfy the gate without becoming a different kind of agent.

**Verdict:** Rejected as primary mechanism. The FP rate and wrong-party problems are structural, not tunable.

### Option 2: Periodic Sync Agent

**Mechanism:** A dedicated agent (or command) periodically reviews recent commits against an artifact inventory, detects drift, and produces update PRs. OpenAI's "garbage collection" pattern.

**Pros:**
- **Decoupled.** Feature agents stay focused on features. Sync agent has full context of all artifacts.
- **Batched.** Multiple drifted artifacts updated in one coherent pass (better than per-agent fragments).
- **Right expertise.** The sync agent reads CLAUDE.md, reads the git diff, and produces a coherent update. Feature-impl agents don't need to understand artifact structure.
- **Extends existing pattern.** `pkg/modeldrift` already does detect→aggregate→issue for models. This generalizes it to all artifact classes.

**Cons:**
- **Lag.** Drift persists until the next sync sweep (hours to a day).
- **No per-agent accountability.** The agent that caused drift isn't the one that fixes it.
- **Requires artifact inventory.** Need a machine-readable list of artifacts and what they cover.

**Verdict:** Recommended. The lag is acceptable (agents don't reference each other's work within the same day). The accountability loss is offset by reliability — a dedicated sync agent actually updates artifacts, while per-agent gates produce FP-laden nudges that get ignored.

---

## Recommended Design: Periodic Sync Agent + Change-Scope Capture

### Architecture

```
Feature Agent                     Completion Pipeline
    │                                    │
    ├── commits code ──────────────────► git diff captured
    │                                    │
    │                                    ├── change-scope tags emitted
    │                                    │   (e.g., "new-command", "new-event",
    │                                    │    "new-flag", "new-skill")
    │                                    │
    │                                    ▼
    │                              ~/.orch/artifact-drift.jsonl
    │
    │
Sync Agent (periodic)
    │
    ├── reads artifact-drift.jsonl
    ├── reads artifact inventory (ARTIFACT_MANIFEST.yaml)
    ├── for each drifted artifact:
    │   ├── reads current artifact content
    │   ├── reads recent git log for context
    │   └── produces update
    └── commits updates, reports via bd comment
```

### Component 1: Change-Scope Tagging (Completion Time)

At `orch complete`, after the existing gate pipeline, analyze the git diff to classify what kind of change was made. This is **advisory** (not blocking).

**Change-scope categories:**

| Category | Detection Heuristic |
|----------|-------------------|
| `new-command` | New `*_cmd.go` file, or new `AddCommand()` call in existing file |
| `new-flag` | New `Flags().String/Bool/Int` call in `cmd/orch/` |
| `new-event` | New event type constant in `pkg/events/` |
| `new-skill` | New directory in `skills/src/` |
| `new-package` | New directory in `pkg/` |
| `api-change` | Modified handler signatures in `serve*.go` |
| `config-change` | Modified config structs or YAML schemas |

**Output:** Append to `~/.orch/artifact-drift.jsonl`:
```json
{
  "timestamp": "2026-03-11T14:30:00Z",
  "beads_id": "orch-go-abc12",
  "skill": "feature-impl",
  "change_scopes": ["new-command", "new-flag"],
  "files_changed": ["cmd/orch/review_cmd.go"],
  "commit_range": "abc123..def456"
}
```

This is lightweight — just diff analysis and event emission. Same pattern as `spawn.StalenessEvent` used by `pkg/modeldrift`.

### Component 2: Artifact Manifest

A checked-in file that maps artifacts to what they document:

```yaml
# ARTIFACT_MANIFEST.yaml
artifacts:
  - path: CLAUDE.md
    sections:
      - name: Commands
        covers: ["cmd/orch/*_cmd.go"]
        triggers: [new-command, new-flag]
      - name: Event Types
        covers: ["pkg/events/"]
        triggers: [new-event]
      - name: Key Packages
        covers: ["pkg/*/"]
        triggers: [new-package]
      - name: Spawn Flow
        covers: ["pkg/spawn/"]
        triggers: [new-flag, config-change]

  - path: skills/src/meta/orchestrator/SKILL.md
    sections:
      - name: Completion Lifecycle
        covers: ["pkg/verify/", "cmd/orch/complete_cmd.go"]
        triggers: [new-command, api-change]
      - name: Spawn Awareness
        covers: ["pkg/spawn/", "cmd/orch/spawn_cmd.go"]
        triggers: [new-flag, new-skill]

  - path: .kb/guides/spawn.md
    covers: ["pkg/spawn/", "cmd/orch/spawn_cmd.go"]
    triggers: [new-flag, config-change]

  - path: .kb/guides/completion.md
    covers: ["pkg/verify/", "cmd/orch/complete_cmd.go"]
    triggers: [new-command, api-change]

  - path: .kb/guides/dashboard.md
    covers: ["cmd/orch/serve*.go", "web/src/"]
    triggers: [api-change, new-command]
```

This is the "artifact inventory" that doesn't exist today. It's small, static, and human-maintained (updated when new artifacts are added, which is rare).

### Component 3: `orch sync` Command

A new command that:

1. Reads `~/.orch/artifact-drift.jsonl` for unprocessed events
2. Cross-references against `ARTIFACT_MANIFEST.yaml`
3. For each affected artifact, reports what's stale and why
4. Optionally spawns a sync agent to produce updates

```bash
# Dry run — show what's drifted
orch sync --dry-run

# Output:
# CLAUDE.md:Commands — 3 new commands since last update (orch review synthesize, orch harness report, --explore flag)
# CLAUDE.md:Event Types — 2 new events (accretion.snapshot, gate_decision with beads_id)
# .kb/guides/dashboard.md — 1 new page (/harness)

# Spawn sync agent to fix drift
orch sync --fix

# Runs a light-tier agent with artifact-sync skill that:
# - Reads each drifted artifact
# - Reads the relevant commits
# - Produces minimal, targeted updates
# - Commits with "docs: sync artifacts (orch sync)"
```

### Component 4: Daemon Integration

The daemon already runs periodic tasks (model drift reflection, friction accumulation, knowledge health). Add artifact sync as another periodic sweep:

```go
// In daemon.go periodic tasks
if time.Since(lastArtifactSync) > 24*time.Hour {
    result := artifactsync.Check(manifestPath, driftEventsPath)
    if len(result.DriftedArtifacts) > 0 {
        // Create beads issue for sync, or auto-spawn if configured
        bd.Create("Artifact sync: N artifacts drifted", ...)
    }
    lastArtifactSync = time.Now()
}
```

**Cadence:** Daily. Artifact drift is annoying but not urgent — agents receiving slightly stale CLAUDE.md still function. Daily sync keeps lag bounded without creating verification bottleneck.

---

## Why Not Both (Gate + Sync)?

The gate adds value only if it can be **advisory without being ignorable**. The existing `guarded.go` protocol is exactly that — and it hasn't worked. Adding another advisory signal to the completion pipeline won't change the outcome.

The sync agent works because it's a **different actor** with a **different skill**. Feature-impl agents build features. Sync agents maintain artifacts. This follows the existing pattern: feature-impl agents don't run `kb reflect`, don't update models, don't maintain CLAUDE.md. Dedicated maintenance agents do.

One possible hybrid: at completion time, emit a **non-blocking warning** when the diff touches artifact-covered files:

```
⚠ Artifact sync advisory: This commit touches cmd/orch/ — CLAUDE.md:Commands may need update.
  (Will be checked by next `orch sync` sweep)
```

This is informational only. The sync agent is the enforcement mechanism.

---

## Comparison with modeldrift

| Aspect | modeldrift (existing) | artifact sync (proposed) |
|--------|----------------------|-------------------------|
| **Artifacts covered** | `.kb/models/` only | CLAUDE.md, skills, guides, models |
| **Detection trigger** | Spawn time (code_refs check) | Completion time (diff analysis) |
| **Event format** | `StalenessEvent` → JSONL | `DriftEvent` → JSONL |
| **Aggregation** | Domain-grouped, threshold-based | Artifact-grouped, scope-based |
| **Remediation** | Creates beads issue → human assigns | `orch sync --fix` → spawns sync agent |
| **Backpressure** | CircuitBreaker (5 open issues) | Same pattern |

The artifact sync mechanism generalizes modeldrift. Long-term, modeldrift could be subsumed into this system — model staleness is just another artifact class where the detection heuristic is `code_refs:` block comparison instead of change-scope tagging.

---

## Implementation Sequence

### Phase 1: Change-Scope Capture (Low effort, immediate value)

- Add diff analysis to completion pipeline that classifies changes
- Emit `DriftEvent` to `~/.orch/artifact-drift.jsonl`
- Create `ARTIFACT_MANIFEST.yaml` with initial artifact inventory
- **Value:** Starts collecting data. Even without sync agent, `orch sync --dry-run` shows what's drifted.

### Phase 2: `orch sync` Command (Medium effort, core value)

- Implement `orch sync --dry-run` (read events, cross-reference manifest, report)
- Implement `orch sync --fix` (spawn sync agent with artifact-sync skill)
- Create `artifact-sync` skill (light-tier, reads drifted artifacts + commits, produces updates)
- **Value:** On-demand artifact synchronization. Orchestrator can run `orch sync --fix` after completing a batch of agents.

### Phase 3: Daemon Integration (Low effort, automation)

- Add periodic artifact sync check to daemon
- Create beads issues for drifted artifacts (with dedup against existing)
- Optional: auto-spawn sync agent when drift exceeds threshold
- **Value:** Fully automated. Drift is detected and remediated without human initiation.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Manifest becomes stale itself | Manifest is small (~30 lines) and changes rarely (new artifacts are rare). Include manifest freshness in `orch sync` self-check. |
| Sync agent produces low-quality updates | Start with `--dry-run` only. Sync agent updates are reviewed like any other PR. |
| Change-scope heuristics have high FP | Start conservative (only detect obvious patterns like new files). Tune based on FP data. |
| Daily cadence too slow for active dev | Orchestrator can run `orch sync --dry-run` manually at any time. Daemon cadence is floor, not ceiling. |
| Sync agent edits CLAUDE.md incorrectly | CLAUDE.md is `guarded.go` protected. Sync agent uses `artifact-sync` skill with specific CLAUDE.md editing guidance. |

---

## Structured Uncertainty

**What's tested:**
- ✅ Drift is real and quantified (10+ recent commits with missing artifact updates)
- ✅ Soft fixes haven't worked (checklists, completion reminders — multiple attempts over months)
- ✅ `pkg/modeldrift` pattern works for model-class artifacts (shipped, running in production)
- ✅ Per-agent gates are structurally problematic (wrong gatable party, high FP — analyzed against hotspot gate precedent)

**What's untested:**
- ⚠️ Whether change-scope heuristics can reliably classify diffs (not benchmarked)
- ⚠️ Whether a sync agent can produce quality CLAUDE.md updates without human guidance
- ⚠️ Whether daily cadence is sufficient (could be too slow for active dev periods)
- ⚠️ ARTIFACT_MANIFEST.yaml maintenance burden (unknown — small file but requires updates when artifacts change)

**What would change this:**
- If sync agent updates are consistently wrong → fall back to creating issues instead of auto-fixing
- If change-scope classification is too noisy → simplify to file-path matching only (artifact covers `cmd/orch/`, commit touched `cmd/orch/`, flag it)
- If daily cadence causes problems → add `orch sync` as post-completion hook (still decoupled from the feature agent, just triggered more frequently)

---

## References

**Files examined:**
- `pkg/modeldrift/modeldrift.go` — existing drift detection for models
- `pkg/verify/check.go` — completion verification pipeline
- `pkg/daemon/daemon.go` — daemon spawn and periodic task pipeline
- `pkg/spawn/kbcontext.go` — spawn-time model serving
- `cmd/orch/guarded.go` — advisory protocols for protected files
- `.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md` — prior model staleness design
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — hotspot gate precedent
- `.kb/models/harness-engineering/model.md` — hard vs soft harness framework

**Related decisions:**
- `.kb/decisions/2026-02-14-model-staleness-detection.md` — model-specific staleness (this design generalizes it)
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — gate design precedent

---

## Investigation History

**2026-03-11:** Investigation started. Analyzed 10+ recent commits for drift evidence. Evaluated per-agent gate vs periodic sync agent. Recommended periodic sync agent with change-scope capture at completion time. Design extends existing `pkg/modeldrift` pattern to all artifact classes.
