## Summary (D.E.K.N.)

**Delta:** Domain-aware spawn routing should be a new Layer 2b gate in the daemon triage pipeline, between hotspot escalation and spawn execution, using the existing kb reflect synthesis cache to detect investigation-dense domains missing models.

**Evidence:** PW comparison-view had 17 investigations over 3 months treating symptoms of 4 domain gaps; a model created after investigation 3 would have prevented 14 blind starts. The daemon's existing architect escalation (Layer 2) provides the proven integration pattern. The kb reflect cache already contains clustered investigation counts by topic.

**Knowledge:** The key design insight is that "same domain" detection doesn't need new clustering logic — kb reflect already clusters investigations by topic with semantic normalization. The gate consumes this cache (already refreshed hourly) and cross-references against `.kb/models/` directory existence. The gate is structurally parallel to hotspot enforcement: both escalate tactical skills to architect when a domain-level condition is unmet.

**Next:** Implement `pkg/daemon/domain_gate.go` with `CheckDomainModelGate()` following the pattern of `CheckArchitectEscalation()`. Add `ModelGateThreshold` to DaemonConfig. Wire into `OnceExcluding()` at line 599 (after hotspot escalation, before spawn).

**Authority:** architectural - Cross-component (daemon, reflect, models, spawn context), multiple valid approaches evaluated, establishes new enforcement pattern

---

# Investigation: Design Domain-Aware Spawn Routing

**Question:** When the daemon sees a 3rd+ investigation in the same domain with no model, how should it gate further tactical work until an architect creates a domain model?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Implementation via feature-impl
**Status:** Complete

**Patches-Decision:** N/A (new enforcement layer)
**Extracted-From:** price-watch/.kb/investigations/2026-03-05-meta-reactive-refinement-vs-domain-modeling.md

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| PW: 2026-03-05-meta-reactive-refinement-vs-domain-modeling.md | motivates | Yes — read primary source | None |
| orch-go: 2026-02-24-design-architect-gate-hotspot-enforcement.md | extends | Yes — reuses escalation pattern | None — complementary, different trigger |
| orch-go: 2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md | confirms | Yes — establishes enforcement layering pattern | None |
| .kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md | extends | Yes — adds Layer 2b to existing 3-layer system | None — additive |

---

## Findings

### Finding 1: The Daemon Already Has the Exact Integration Pattern

**Evidence:** `CheckArchitectEscalation()` in `pkg/daemon/architect_escalation.go:59-123` implements a gate that:
1. Only triggers for implementation skills (feature-impl, systematic-debugging)
2. Checks a condition (hotspot match)
3. Cross-references a bypass (prior architect review)
4. Escalates skill to "architect" when condition is met

The domain model gate has the identical shape:
1. Only trigger for tactical skills (same set)
2. Check a condition (investigation cluster >= threshold AND no model)
3. Cross-reference a bypass (model exists in `.kb/models/`)
4. Escalate skill to "architect" when condition is met

**Source:** `pkg/daemon/architect_escalation.go:59-123`, `pkg/daemon/daemon.go:582-599`

**Significance:** No new integration pattern needed. The domain gate is structurally identical to hotspot escalation. It slots into the same location in `OnceExcluding()`, using the same escalation mechanics (change skill, change model, set flag on result).

---

### Finding 2: KB Reflect Cache Already Contains the Signal

**Evidence:** The `reflect-suggestions.json` cache at `~/.orch/reflect-suggestions.json` contains synthesis clusters with topic names and investigation counts:

```json
{
  "topic": "context",
  "count": 7,
  "investigations": ["2025-12-23-inv-add-stale-flag-kb-context.md", ...]
}
```

The daemon already loads this cache during completion (`complete_synthesis.go:108`). The `SynthesisSuggestion` struct has `Topic string` and `Count int` — exactly what the domain gate needs.

The cache is refreshed hourly by `RunAndSaveReflection()` called from the daemon's periodic tasks. Freshness is already tracked via `Timestamp`.

**Source:** `pkg/daemon/reflect.go:13-46` (types), `~/.orch/reflect-suggestions.json` (live cache), `cmd/orch/complete_synthesis.go:108` (existing consumption pattern)

**Significance:** No new data source needed. The domain gate reads the same cache that completion synthesis already reads. The signal (topic + count) is already computed and available.

---

### Finding 3: Model Existence Is a Simple Directory Check

**Evidence:** Domain models live at `.kb/models/{name}/model.md`. The existing models in orch-go include: `daemon-autonomous-operation`, `spawn-architecture`, `completion-verification`, `agent-lifecycle-state-model`, etc.

The mapping from synthesis topic to model directory is not 1:1 (synthesis topic "daemon" → model directory "daemon-autonomous-operation"). This requires fuzzy matching: does any model directory name contain the synthesis topic as a substring?

Listing `.kb/models/` shows ~25 model directories. A substring check across these is O(topics × models) — trivially fast.

**Source:** `ls .kb/models/` (25 directories), synthesis cache topics (7 current clusters)

**Significance:** Model existence check is cheap and doesn't require shelling out. It's a directory scan + substring match, doable in-process. The fuzzy matching handles the naming mismatch between synthesis topics ("daemon") and model directories ("daemon-autonomous-operation").

---

### Finding 4: Cross-Project Dimension Requires Project-Scoped Checks

**Evidence:** The daemon processes issues from multiple projects via `ProjectRegistry`. The synthesis cache from `kb reflect --global` aggregates across all projects. But models are per-project (`.kb/models/` within each project).

When the daemon processes a PW issue, it needs to check PW's `.kb/models/`, not orch-go's. The `ProjectDir` field on `Issue` provides the project path.

The hotspot checker already handles this — `checker.CheckHotspots("")` is called per-issue. The domain gate should similarly accept a project directory parameter.

**Source:** `pkg/daemon/daemon.go:288-398` (NextIssueExcluding with ProjectDir), `architect_escalation.go:99` (checker.CheckHotspots("") per issue)

**Significance:** The gate must be project-scoped. When the daemon sees a PW issue targeting a domain with 7 investigations and no PW model, it should escalate — even if orch-go has a model for that topic. The synthesis cache is global, but model existence is local to the project.

---

### Finding 5: Investigation Skill Should Get Advisory, Not Block

**Evidence:** The existing pattern from hotspot enforcement: `isImplementationSkill()` returns true only for `feature-impl` and `systematic-debugging`. Investigation and architect skills are explicitly exempt from blocking. This is because investigations are *how you learn* — blocking them is counterproductive.

However, the evidence shows the 4th+ investigation in the same domain starts with less context than it should. The right treatment for investigations is: inject advisory into spawn context ("NOTE: 6 prior investigations in this domain exist. Consider requesting a model before proceeding.") rather than blocking.

**Source:** `architect_escalation.go:24-31` (isImplementationSkill), hotspot enforcement design in `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md`

**Significance:** Two-tier response: **block** tactical skills (escalate to architect) but **advise** investigation skills (context injection). This mirrors the hotspot pattern where Layer 1/2 block implementation skills and Layer 3 provides advisory context for all skills.

---

## Synthesis

**Key Insights:**

1. **Structural parallel to hotspot enforcement** — The domain model gate is a 4th enforcement layer that fits naturally into the existing 3-layer hotspot system. It uses the same escalation pattern, the same skill exemptions, and the same integration point in the daemon. The implementation is additive, not architectural.

2. **Proactive use of retrospective signal** — KB reflect's synthesis detection runs retrospectively (analyzing what already happened). The domain gate makes it proactive by checking the synthesis cache *before* spawning, not *after* completing. Same signal, different timing, different impact.

3. **The "same domain" problem is already solved** — KB reflect's topic normalization (semantic keyword extraction, action verb filtering, subclustering) already defines what "same domain" means. The gate doesn't need its own clustering algorithm — it consumes the existing one.

4. **Cross-project is the hard part** — The synthesis cache is global but model existence is per-project. This is the only part that requires new logic: looking up `.kb/models/` in the *issue's* project directory, not the daemon's working directory.

**Answer to Investigation Question:**

The domain model gate should be implemented as Layer 2b in the daemon triage pipeline — a new function `CheckDomainModelGate()` that runs after hotspot escalation (Layer 2) and before spawn execution. It reads the cached reflect synthesis suggestions, matches issue keywords against synthesis topics with count >= threshold (default 3), and checks whether the issue's project has a model for that topic. If no model exists and the skill is tactical (feature-impl, systematic-debugging), the skill is escalated to architect with a task to create the domain model. Investigations get a spawn context advisory instead of a block.

---

## Structured Uncertainty

**What's tested:**

- ✅ KB reflect cache structure contains topic + count + investigation list (verified: read ~/.orch/reflect-suggestions.json)
- ✅ Daemon loads reflect cache in existing code paths (verified: complete_synthesis.go:108, reflect.go:203-225)
- ✅ Model directories follow `.kb/models/{name}/` pattern (verified: ls .kb/models/ shows 25 directories)
- ✅ Architect escalation pattern is proven and testable (verified: architect_escalation_test.go has comprehensive test suite)
- ✅ PW evidence: 17 investigations in same domain, model at investigation 3 would have prevented 14 blind starts (verified: read primary investigation)

**What's untested:**

- ⚠️ Topic-to-model fuzzy matching accuracy (substring match may produce false positives: topic "config" matches model "daemon-config-extraction")
- ⚠️ Performance impact of loading reflect cache + scanning models at every spawn decision (expected negligible but not benchmarked)
- ⚠️ Whether threshold of 3 is optimal for cross-project domains (PW evidence suggests 3 is right but only one case study)
- ⚠️ Whether keyword extraction from issue titles is sufficient to match synthesis topics (may need description scanning too)

**What would change this:**

- If kb reflect topics don't align with meaningful domain boundaries (false clustering → false gates), the gate would block valid work
- If model creation becomes a bottleneck (architect spawn takes too long, tactical work piles up), the gate creates worse throughput than no gate
- If the synthesis cache is stale >24h in practice, the gate fires on outdated cluster data

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add domain model gate as Layer 2b | architectural | Cross-component (daemon, reflect, models, spawn), establishes new enforcement pattern |
| Default threshold of 3 | implementation | Configurable, reversible, evidence-based default |
| Cross-project model lookup | implementation | Follows existing ProjectDir pattern |
| Advisory for investigations | implementation | Follows existing Layer 3 pattern |

### Recommended Approach ⭐

**Layer 2b Domain Model Gate** — Add `CheckDomainModelGate()` to the daemon triage pipeline that escalates tactical skills to architect when the issue's domain has >= 3 investigations with no domain model.

**Why this approach:**
- Reuses proven architect escalation pattern (identical structure to `CheckArchitectEscalation`)
- Consumes existing signal source (kb reflect cache) — no new data collection
- Prevents investigation-dense domains from accumulating blind tactical fixes
- Configurable threshold via DaemonConfig (same pattern as other daemon settings)

**Trade-offs accepted:**
- Depends on kb reflect cache freshness (24h max staleness) — acceptable because investigation density changes slowly
- Fuzzy topic-to-model matching may have false positives — mitigated by substring matching being conservative (short topic must be contained in longer model name)
- Architect spawn adds latency before tactical work — this is the intended behavior (model creation enables better tactical work)

**Implementation sequence:**

1. **`pkg/daemon/domain_gate.go`** — Core gate logic
   - `DomainModelGateResult` struct (Domain, InvestigationCount, ProjectDir)
   - `CheckDomainModelGate(issue, reflectSuggestions, threshold, projectDir)` function
   - `findMatchingSynthesisTopicForIssue(issue, suggestions, threshold)` — keyword extraction + topic matching
   - `checkModelExistsForTopic(topic, projectDir)` — scan `.kb/models/` for substring match
   - ~100 lines

2. **`pkg/daemon/domain_gate_test.go`** — Tests
   - Escalates feature-impl when domain has 5 investigations, no model
   - Does NOT escalate when model exists
   - Does NOT escalate investigation/architect skills
   - Does NOT escalate below threshold
   - Respects project directory for model lookup
   - ~150 lines

3. **`pkg/daemon/daemon.go`** changes — Wire gate into OnceExcluding
   - Load reflect cache in daemon setup (or lazy-load on first check)
   - Add gate check after line 599 (after hotspot escalation, before spawnIssue)
   - Set `domainGateTriggered` flag on `OnceResult`
   - ~20 lines of integration

4. **`pkg/userconfig/userconfig.go`** changes — Add config
   - `ModelGateThreshold *int` in DaemonConfig (default: 3)
   - `ModelGateEnabled *bool` in DaemonConfig (default: true)
   - ~10 lines

5. **Spawn context advisory** — For investigation skills
   - When domain gate condition is met but skill is investigation, inject advisory into spawn context
   - ~15 lines in `pkg/spawn/context.go`

### Alternative Approaches Considered

**Option B: Real-time investigation scanning (bypass kb reflect cache)**
- **Pros:** Always fresh, no cache staleness concern
- **Cons:** Requires scanning `.kb/investigations/` and clustering at every spawn decision. Duplicates kb reflect's logic in Go. Violates template ownership constraint (kb-cli owns investigation analysis).
- **When to use instead:** If kb reflect cache proves unreliable or stale in practice

**Option C: Beads label-based domain tracking**
- **Pros:** Explicit — issues get `domain:comparison-view` labels. No clustering ambiguity.
- **Cons:** Requires manual labeling or new inference logic. Changes beads workflow. Most issues don't have domain labels today.
- **When to use instead:** If automatic topic clustering proves too noisy (>30% false positive rate)

**Option D: Orchestrator-prompted gate (not daemon-automatic)**
- **Pros:** Better judgment — orchestrator can evaluate whether domain model is actually needed
- **Cons:** Doesn't scale — requires orchestrator attention for every spawn. The PW case happened precisely because the orchestrator (Dylan) didn't notice the pattern.
- **When to use instead:** If false positive rate of automatic gating is unacceptable

**Rationale for recommendation:** Option A (Layer 2b) is the clear winner because it reuses existing infrastructure (reflect cache, escalation pattern, config system), requires minimal new code (~300 lines), and addresses the core problem (blind tactical work in investigation-dense domains). Options B-D either duplicate logic, require manual work, or don't scale.

---

### Implementation Details

**What to implement first:**
- `domain_gate.go` + tests (self-contained, testable in isolation)
- Then wire into daemon.go (small integration change)
- Then config (DaemonConfig field)
- Finally spawn context advisory (enhancement)

**Things to watch out for:**
- ⚠️ **Defect Class 0 (Scope Expansion):** The gate scans synthesis topics — ensure it's scoped to the issue's project, not all projects. Use ProjectDir threading.
- ⚠️ **Defect Class 1 (Filter Amnesia):** The gate must check `isImplementationSkill()` — don't gate investigation/architect. Reuse the existing function.
- ⚠️ **Defect Class 4 (Cross-Project Boundary Bleed):** Model existence check must use issue's ProjectDir, not daemon's cwd. This is the most likely implementation bug.
- ⚠️ **False positive: topic "agent"** — A 3-investigation cluster on "agent" would match model "agent-lifecycle-state-model" via substring, correctly bypassing the gate. But broad topics like "config" (3 investigations) might match "daemon-config-extraction" model — technically a different domain. Consider requiring minimum topic length of 4+ characters.
- ⚠️ **Ordering with hotspot escalation:** If both hotspot AND domain gate trigger, hotspot escalation should take precedence (it's a stricter condition). The implementation handles this naturally: hotspot runs first (line 587), and if it escalates to architect, the domain gate doesn't need to run.

**Areas needing further investigation:**
- How to architect the domain model creation task that the escalated architect receives (what should the spawn context say?)
- Whether to track domain gate firings in events.jsonl (probably yes, for threshold tuning)
- Whether the 3-investigation threshold should differ per project (PW might need 2, orch-go might need 5)

**Success criteria:**
- ✅ Daemon escalates feature-impl to architect when domain has >= 3 investigations and no model
- ✅ Daemon does NOT escalate when model exists (fuzzy match against `.kb/models/`)
- ✅ Investigation skills get advisory in spawn context, not block
- ✅ Config allows adjusting threshold and enabling/disabling
- ✅ Gate respects ProjectDir for cross-project model lookup
- ✅ All existing daemon tests continue to pass (no regression)

---

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when promoting:**

This decision should block spawns that would add tactical fixes to investigation-dense domains.

**Suggested blocks keywords:**
- "domain model gate"
- "investigation cluster"
- "synthesis threshold"
- "domain-aware routing"

---

## Blocking Questions

### Q1: Should the architect task include explicit guidance to create a domain model, or just route as generic architect?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If explicit, the architect spawn context needs a "CREATE DOMAIN MODEL" instruction template. If generic, the architect investigates the domain and decides whether a model is the right artifact (might produce a guide or decision instead).

**Recommendation:** Explicit. The gate fires specifically because investigations are accumulating without a model. The architect should know: "You are here because {topic} has {count} investigations and no model. Your deliverable is a domain model at `.kb/models/{topic}/model.md`."

### Q2: Should the gate also fire for the orchestrator (orch spawn) or only for daemon autonomous spawns?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If orchestrator too, need to add domain gate to spawn preflight (like hotspot spawn gate in Layer 1). If daemon only, implementation is smaller but manual spawns bypass the gate.

**Recommendation:** Daemon first, orchestrator later. The daemon is where autonomous blind spawning happens. Manual `orch spawn` implies an orchestrator has already assessed the context. Add orchestrator gate as a follow-up if daemon-only proves insufficient.

### Q3: When the gate escalates to architect, should it create a new blocking issue (like extraction does) or just change the skill on the existing issue?

- **Authority:** implementation
- **Subtype:** judgment
- **What changes based on answer:** If new issue: more complex (create issue, add dependency, modify original). If skill change: simpler but the original issue's intent (fix bug X) gets replaced with "create domain model" — confusing.

**Recommendation:** Change skill on existing issue (like hotspot escalation does, not like extraction). The architect reviews the domain AND fixes the original issue with the model as context. Extraction needs a separate issue because extraction is independent work. Domain modeling is contextual to the original issue.

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go:478-617` — OnceExcluding triage pipeline
- `pkg/daemon/architect_escalation.go` — Hotspot Layer 2 escalation (integration pattern)
- `pkg/daemon/reflect.go` — Reflect cache types and loading
- `cmd/orch/complete_synthesis.go` — Existing synthesis cache consumption
- `pkg/userconfig/userconfig.go:72-117` — DaemonConfig structure
- `pkg/daemon/skill_inference.go` — Skill inference pipeline
- `~/.orch/reflect-suggestions.json` — Live reflect cache
- `.kb/models/` — Existing model directories

**Related Artifacts:**
- **Evidence:** `price-watch/.kb/investigations/2026-03-05-meta-reactive-refinement-vs-domain-modeling.md` — Primary motivation
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Existing enforcement pattern
- **Investigation:** `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` — Hotspot escalation design

---

## File Targets

| File | Action | Lines |
|------|--------|-------|
| `pkg/daemon/domain_gate.go` | Create | ~100 |
| `pkg/daemon/domain_gate_test.go` | Create | ~150 |
| `pkg/daemon/daemon.go` | Modify (line ~599) | ~20 added |
| `pkg/userconfig/userconfig.go` | Modify (DaemonConfig) | ~10 added |
| `pkg/spawn/context.go` | Modify (advisory injection) | ~15 added |

**Total estimated:** ~295 lines new/modified

---

## Investigation History

**2026-03-05 19:35:** Investigation started
- Initial question: How should the daemon gate tactical work when a domain has 3+ investigations and no model?
- Context: PW comparison-view evidence — 17 investigations over 3 months treating symptoms of 4 domain gaps

**2026-03-05 20:15:** Exploration complete
- Mapped daemon triage pipeline (10 gates, 5 dedup layers)
- Confirmed hotspot escalation pattern is reusable
- Confirmed kb reflect cache contains needed signal
- Identified cross-project as the key complexity

**2026-03-05 20:45:** Investigation complete
- Status: Complete
- Key outcome: Domain model gate is Layer 2b — structurally identical to hotspot escalation, consuming existing reflect cache, ~295 lines implementation
