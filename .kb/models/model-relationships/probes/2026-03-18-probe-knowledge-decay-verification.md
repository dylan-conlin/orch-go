# Probe: Knowledge Decay Verification — Model Relationships

**Date:** 2026-03-18
**Probed by:** orch-go-p7cxf
**Trigger:** 999d since last probe (model created 2026-03-01, never probed)

---

## Claims Tested

### Claim 1: Three-function framework (structural/mechanistic/taxonomic)
**Verdict: CONFIRMED — still accurate and referenced**

The framework remains conceptually sound. The decidability-graph model explicitly uses "structural" as a descriptor (line 98, 101, 404, 590 of its model.md). The orchestrator-session-lifecycle model references the framework. The signal-to-design-loop global model also references mechanistic/structural distinctions.

However, adoption is limited — only 6 files across the entire .kb/ reference these terms, and most are the model itself or archived investigations. The framework hasn't been widely adopted as shared vocabulary by other models or probes.

### Claim 2: Model inventory table (7 models listed)
**Verdict: STALE — severely out of date**

The inventory table (lines 137-145) lists 7 models:
- `decidability-graph` — still exists, still active (updated 2026-03-06)
- `orchestrator-session-lifecycle` — still exists, still active (updated recently)
- `agent-lifecycle-state-model` — still exists
- `drift-taxonomy` — still exists, still active (updated 2026-03-06)
- `spawn-architecture` — still exists, still active (updated 2026-03-06)
- `kb-reflect-cluster-hygiene` — still exists
- `beads-database-corruption` — still exists (described as "stale — 25d without update"; now has a 2026-03-18 probe)

**Missing from inventory:** 25+ models now exist that aren't listed:
- `knowledge-accretion`, `harness-engineering`, `daemon-autonomous-operation`, `architectural-enforcement`, `hotspot-acceleration`, `coordination`, `context-injection`, `escape-hatch-visibility-architecture`, `defect-class-taxonomy`, `architect`, `architectural-defect-intelligence`, `orchestration-cost-economics`, `claude-code-agent-configuration`, `system-learning-loop`, `model-access-spawn-paths`, `entropy-spiral`, `orchestrator-skill`, `workspace-lifecycle-model`, `completion-verification`, `session-deletion-vectors`, `opencode-session-lifecycle`, `follow-orchestrator-mechanism`, `extract-patterns`, `beads-integration-architecture`, `opencode-fork`, `skill-content-transfer`, `dashboard-architecture`, `coaching-plugin`, `macos-click-freeze`

Plus 8 global models in `.kb/global/models/`.

### Claim 3: Dependency pattern (structural → mechanistic → taxonomic)
**Verdict: CONFIRMED — pattern holds but oversimplified**

The general flow holds. The decidability-graph (structural) still grounds authority premises used by session-lifecycle and spawn-architecture (mechanistic). Drift-taxonomy (taxonomic + structural) still provides vocabulary used by reliability-related models.

However, the actual dependency graph is messier than the clean three-layer diagram suggests. Many newer models don't fit neatly: `hotspot-acceleration` is mechanistic but doesn't depend on any structural model. `coaching-plugin` is mechanistic and independent. The framework describes a pattern, not a rule.

### Claim 4: Multi-function models have outsized impact when lost
**Verdict: CONFIRMED — still true by design**

The decidability-graph remains the canonical example. Its 2026-03-06 update and active probe history confirm it's treated as high-value. No multi-function models have been lost since the original entropy-spiral cleanup that motivated this model.

### Claim 5: Model recovery prioritization order
**Verdict: NOT TESTED — no recent model loss events**

The recommended recovery priority (multi-function > structural > taxonomic > mechanistic) hasn't been tested because no significant model loss has occurred since the original decidability-graph recovery. This is an untested recommendation.

---

## Overall Verdict

**Core framework: CURRENT. Inventory section: SEVERELY STALE.**

The three-function framework (structural/mechanistic/taxonomic) and the dependency pattern remain accurate conceptual tools. However:

1. **Model inventory is frozen at 7 models** — the project now has 30+ models. The table should either be removed (it will always be stale) or replaced with a reference to `ls .kb/models/`.
2. **Low adoption** — the framework vocabulary (structural/mechanistic/taxonomic) is used in only ~4 models outside this one. It's a valid lens but not widely applied.
3. **No new examples** — the model still uses the same examples from March 1, 2026. Newer models could illustrate the framework better (e.g., `coaching-plugin` is purely mechanistic, `defect-class-taxonomy` is purely taxonomic).
4. **`beads-database-corruption` staleness note** on line 145 ("stale — 25d without update") is itself stale.

## Recommended Updates

1. **Remove or generalize the inventory table** — it's a snapshot that ages immediately
2. **Add newer examples** to the multi-function table to show the framework scales
3. **Remove the staleness annotation** on beads-database-corruption (it's been probed today)
4. **Consider noting low adoption** — the framework is sound but hasn't become shared vocabulary

---

## References

- `.kb/models/decidability-graph/model.md` — confirmed still uses structural/mechanistic framing
- `.kb/models/drift-taxonomy/model.md` — confirmed structural + taxonomic dual function
- `.kb/models/spawn-architecture/model.md` — confirmed mechanistic, updated 2026-03-06
