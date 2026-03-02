# Probe: Decidability Graph Knowledge Recovery Assessment

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The decidability-graph model existed on `entropy-spiral-feb2026` but was removed during cleanup. Does this model describe concepts that are still active in the current system, and what knowledge was lost versus preserved through other artifacts?

---

## What I Tested

Compared knowledge artifacts between `entropy-spiral-feb2026` branch and `master`:

```bash
# Read the full decidability-graph model from entropy branch
git show entropy-spiral-feb2026:.kb/models/decidability-graph.md

# Compare .kb/models/, .kb/decisions/, .kb/guides/ between branches
diff <(git ls-tree -r --name-only entropy-spiral-feb2026 -- .kb/models/ | sort) \
     <(git ls-tree -r --name-only HEAD -- .kb/models/ | sort)

# Check which decisions referenced by decidability model exist on master
for dec in worker-authority-boundaries question-subtype-encoding-labels \
           decidability-graph-substrate-options recommendation-authority-classification; do
  git show HEAD:.kb/decisions/2026-01-*-${dec}.md 2>/dev/null
done

# Checked master coverage of decidability concepts:
# - .kb/guides/decision-authority.md (agent authority guide)
# - .kb/decisions/2026-01-18-questions-as-first-class-entities.md (question type)
# - worker-base skill (authority delegation in SPAWN_CONTEXT)
# - orchestrator-session-lifecycle model (orchestrator engagement)
```

---

## What I Observed

### Decidability Graph Model Assessment (~470 lines, high quality)

**Core concepts:**
1. **Node taxonomy** (Work/Question/Gate) — classifies work items by resolution type
2. **Edge authority** (daemon/orchestrator/Dylan) — who can traverse which edges
3. **Context-scoping as irreducible function** — hierarchy is about who scopes context, not reasoning capability
4. **Graph dynamics** — questions can fracture, collapse, or reframe, changing graph structure
5. **Frontier representation** — daemon reports graph state, not just agent states
6. **Intra-task authority** — when sub-decisions within Work nodes are worker vs escalation

**Empirical validation:** Dogfooded 2026-01-19 with real beads question entities. Discovered friction (bd close requires Phase: Complete for questions, answered status doesn't unblock).

**Probe on branch:** One high-quality probe (`2026-02-09-context-scoping-reducibility-daemon-vs-manual.md`) analyzing 1,015 classified spawns, finding daemon-driven inference is only 42% of spawns with 49% skill divergence.

### What's Preserved on Master (Partial Coverage)

| Decidability Concept | Master Coverage | Gap |
|---------------------|-----------------|-----|
| Agent authority delegation | `decision-authority.md` guide + worker-base skill | ✅ Tactical authority well-covered |
| Questions as blocking entities | `questions-as-first-class-entities.md` decision | ✅ Core decision preserved |
| Strategic orchestrator model | `strategic-orchestrator-model.md` decision | ✅ Base authority division preserved |
| Work/Question/Gate node taxonomy | **None** | ❌ Unified framework lost |
| Edge authority (who can traverse) | **None** | ❌ Not captured |
| Context-scoping insight | SPAWN_CONTEXT constraint only | ⚠️ Mentioned but not explained |
| Graph dynamics (fracture/collapse) | **None** | ❌ Not captured |
| Frontier as observable state | **None** | ❌ Not captured |
| Intra-task authority boundaries | **None** | ❌ Not captured |
| Question subtypes (factual/judgment/framing) | **None** (decision missing) | ❌ Not captured |
| Worker node-creation vs edge-creation boundary | **None** (decision missing) | ❌ Not captured |
| Resolution typing determining daemon/orchestrator routing | **None** | ❌ Not captured |

### Other Missing Artifacts

**58 decisions missing from master.** Key ones:
- `worker-authority-boundaries` — Workers create nodes, orchestrator creates edges
- `question-subtype-encoding-labels` — factual/judgment/framing via labels
- `decidability-graph-substrate-options` — Extend beads with authority edges
- `recommendation-authority-classification` — Classify investigation outputs by authority
- `orchestrator-constitutional-responsibility` — Hard limits on orchestrator
- `skill-constitutional-constraints` — Hard limits on skills

**~15 models missing from master.** Notable:
- `completion-lifecycle.md` (74 lines) — May be superseded by completion-verification model
- `cross-project-visibility.md` (371 lines) — Cross-project daemon architecture
- `sse-connection-management.md` (384 lines) — SSE resilience patterns
- `dashboard-agent-status.md` (231 lines) — Dashboard field audit
- `current-model-stack.md` (203 lines) — Model/provider mapping
- `agent-state-architecture-feb2026.md` (160 lines) — Distributed JOIN architecture
- `system-reliability-feb2026.md` (164 lines) — Unbounded resource failure taxonomy

**11 guides missing from master.** Notable:
- `worker-patterns.md` — Worker detection, protocols, authority
- `recovery-playbooks.md` — Concise failure recovery steps
- `decision-index.md` — Decision discoverability index

### What Was Correctly Removed (Entropy Noise)

- `PHASE3_REVIEW.md`, `PHASE4_REVIEW.md` — Point-in-time review artifacts
- `_TEMPLATE.md` — Duplicate template (already in `.orch/templates/`)
- `multi-model-evaluation-feb2026.md` — Time-stamped snapshot, concepts covered by model-access-spawn-paths
- Models with "feb2026" suffix — Point-in-time snapshots of things that evolved:
  - `agent-state-architecture-feb2026.md` — Useful insights but the distributed JOIN is documented in guides
  - `system-reliability-feb2026.md` — The unbounded resource pattern is documented in resilient-infrastructure-patterns guide

---

## Model Impact

- [x] **Extends** model with: The orchestrator-session-lifecycle model describes the orchestrator's engagement patterns but lacks the formal framework for WHY certain edges require certain authority. The decidability-graph model provides this structural explanation — the hierarchy is about context-scoping, not capability. This is a significant extension: it explains not just that the orchestrator decides differently than daemon/Dylan, but the mechanism by which decisions get routed.

**Specific extensions the decidability model would add:**
1. **Node taxonomy** gives the orchestrator a classification tool for incoming work (is this Work, Question, or Gate?)
2. **Edge authority** formalizes what the orchestrator-session-lifecycle model treats as implicit role boundaries
3. **Graph dynamics** explains why planning past unresolved questions is unreliable (the subgraph is provisional)
4. **Context-scoping insight** reframes authority as "who decides what context to load" rather than "who is smarter"

---

## Notes

### Recovery Recommendation

**Tier 1 — Restore immediately (unique framework, high quality, actively relevant):**
1. `.kb/models/decidability-graph.md` + probes directory
2. `.kb/decisions/2026-01-19-worker-authority-boundaries.md`
3. `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md`
4. `.kb/decisions/2026-01-30-recommendation-authority-classification.md`

**Tier 2 — Restore after review (valuable but may need updating):**
5. `.kb/decisions/2026-01-30-decidability-graph-substrate-options.md`
6. `.kb/decisions/2026-01-22-orchestrator-constitutional-responsibility.md`
7. `.kb/decisions/2026-01-22-skill-constitutional-constraints.md`
8. `.kb/guides/recovery-playbooks.md`
9. `.kb/guides/worker-patterns.md` (much now in worker-base skill, but has additional detail)

**Tier 3 — Don't restore (superseded or noise):**
- Models with `feb2026` suffix — point-in-time snapshots
- `PHASE3_REVIEW.md`, `PHASE4_REVIEW.md`
- `_TEMPLATE.md`
- Most of the 58 decisions are either superseded by newer work or operational decisions that don't need persistence
- `decision-index.md` — would be immediately stale, better to regenerate

### Recovery Method

```bash
# For each file to restore:
git show entropy-spiral-feb2026:<path> > <path>
git add <path>
git commit -m "recover: restore <artifact> from entropy-spiral branch"
```

### The Existing Recovery Audit

The investigation at `.kb/investigations/archived/2026-02-13-inv-audit-entropy-spiral-recoverable-features.md` is an **empty template** — it was created but never completed. It focused on CODE features (spawn system, completion pipeline, etc.), not knowledge artifacts. This probe covers the knowledge gap that audit missed.
