## Summary (D.E.K.N.)

**Delta:** The daemon should keep Opus as the default route, promote GPT-5.4 only for bounded `feature-impl` overflow, and make every non-default route observable and recoverable.

**Evidence:** Current code only hard-pins Opus for reasoning skills and otherwise defers to defaults, while Mar 26 benchmark data shows GPT-5.4 is strong on `feature-impl` (4/4 first-attempt) but still under-tested and less disciplined on reasoning-heavy work.

**Knowledge:** The routing problem is no longer "Anthropic or not" but "which capability class is safe for this issue," so the router needs skill + complexity + failure-history inputs instead of skill-only inference.

**Next:** Implement the new router through issues `orch-go-ckddz`, `orch-go-r7avo`, `orch-go-kdyh6`, and behavioral verification issue `orch-go-xi8tc` under plan `.kb/plans/2026-03-26-daemon-gpt54-routing.md`.

**Authority:** architectural - The decision crosses daemon routing, config, observability, and retry behavior, but stays inside orch-go's system design rather than Dylan's product direction.

---

# Investigation: Design Daemon Route Tasks Gpt

**Question:** How should the daemon decide between GPT-5.4 and Opus for spawned work now that GPT-5.4 has some positive benchmark data but weaker reasoning and scope-control evidence than Opus?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-3k1yo
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/daemon-autonomous-operation/model.md` | extends | yes | Model still describes skill-only routing and needed a GPT-5.4-era refinement. |
| `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md` | deepens | yes | Historical GPT-5.2 failures are still relevant, but prompt-size framing is stale for GPT-5.4. |
| `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` | confirms | yes | Confirms GPT-5.4 is viable for `feature-impl` overflow but not yet default-safe for reasoning work. |
| `.kb/guides/model-selection.md` | confirms | yes | Guide already positions GPT-5.4 as highest-context OpenAI option, but daemon policy is not yet complexity-aware. |

---

## Findings

### Finding 1: The current daemon router is still skill-first and mostly default-preserving

**Evidence:** `InferModelFromSkill()` only returns explicit overrides for `systematic-debugging`, `investigation`, `architect`, `codebase-audit`, and `research`, all mapped to `opus`; `feature-impl` deliberately returns empty string so the resolve pipeline falls back to defaults. `SpawnWork()` only passes `--model` when a non-empty alias is returned, so current daemon routing has no way to distinguish small vs large implementation work or safe vs risky GPT use.

**Source:** `pkg/daemon/skill_inference.go:259`, `pkg/daemon/skill_inference.go:288`, `pkg/daemon/issue_adapter.go:426`, `pkg/daemon/daemon.go:376`

**Significance:** The code already has an escape hatch for non-pinned skills, but it cannot express "use GPT-5.4 only for this subset of implementation work." Any GPT policy needs a richer route object than the current skill-to-model lookup.

---

### Finding 2: GPT-5.4 is validated for bounded implementation overflow, not reasoning-default work

**Evidence:** The Mar 26 benchmark shows GPT-5.4 completed 4/4 `feature-impl` tasks on first attempt and reached 80% first-attempt / 100% with retry across N=5 total tasks, but the only investigation task had a first-attempt silent death and GPT-5.4 also showed weaker scope control on at least one implementation task. The thread update codifies the resulting interim policy: Opus remains default, GPT-5.4 is promoted only to `feature-impl` overflow until reasoning-heavy skills have a larger benchmark.

**Source:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:136`, `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:168`, `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:220`, `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md:72`

**Significance:** The question is no longer whether GPT-5.4 can ever work; it is where its evidence is strong enough to trust. That pushes the router toward capability classes instead of a single provider-wide rule.

---

### Finding 3: The daemon already has the right structural seams for a complexity-aware router

**Evidence:** `RouteIssueForSpawn()` already rewrites route decisions after hotspot extraction and architect escalation, and `ComplianceConfig` in `pkg/daemonconfig` already implements combo-first resolution (`combo > skill > model > default`) for daemon policy. The daemon also already carries `effort:*` labels and uses them in completion routing, giving the model router an existing complexity signal without inventing a new labeling scheme.

**Source:** `pkg/daemon/coordination.go:16`, `pkg/daemon/coordination.go:37`, `pkg/daemonconfig/compliance.go:53`, `pkg/daemon/auto_complete.go:12`, `pkg/daemon/coordination.go:159`

**Significance:** The recommended design can stay coherent with existing daemon architecture: introduce a first-class route policy object, reuse effort labels as the initial complexity signal, and mirror compliance-style override semantics instead of building a bespoke rules engine.

---

### Finding 4: GPT routing needs bounded promotion, not repeated retries on the same path

**Evidence:** Historical GPT-5.2 daemon failures included hallucinated constraints, silent termination, and successful session creation with no useful work; the updated DAO-13 wording explicitly says modern GPT routing should focus on protocol compliance, silent-death frequency, and scope control more than context pressure. Current spawn execution already distinguishes spawn failure rollback from successful-but-bad outcomes, and the separate Mar 26 retry plan proposes classifying `empty_execution` so the next action can be an informed retry rather than a blind repeat.

**Source:** `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md:91`, `.kb/models/daemon-autonomous-operation/claims.yaml:230`, `pkg/daemon/spawn_execution.go:115`, `.kb/plans/2026-03-26-gpt54-empty-execution-retry.md:48`

**Significance:** A GPT route without a promotion policy would reintroduce defect classes 6 and 7 (duplicate action, premature destruction). The safe design is "try GPT where evidence is good, then promote to Opus once the failure pattern says the cheaper route is no longer paying off."

---

## Synthesis

**Key Insights:**

1. **Capability class beats provider class** - The router should decide between "reasoning-default" and "bounded overflow" lanes, not between Anthropic and OpenAI as global camps. Finding 1 shows the current daemon is too coarse, and Finding 2 shows GPT-5.4 has mixed evidence by work type.

2. **Complexity can start from existing labels** - We do not need a new complexity detector to ship v1. Finding 3 shows `effort:*` labels already exist and can gate GPT eligibility immediately, which keeps the design legible and testable.

3. **Routing and recovery are one design** - Finding 4 shows that model choice cannot be separated from what happens after failure. GPT-5.4 becomes production-safe only when the daemon can explain the choice and promote failed GPT work to Opus without looping.

**Answer to Investigation Question:**

The daemon should use a two-lane routing policy. Lane 1 is the default: Opus for all reasoning-heavy skills (`architect`, `investigation`, `systematic-debugging`, `research`, `codebase-audit`) and for any implementation issue marked `effort:large`, escalated by hotspot logic, or otherwise missing complexity evidence. Lane 2 is bounded overflow: GPT-5.4 only for `feature-impl` work that is explicitly `effort:small` or `effort:medium`, not escalated, and not already showing GPT-specific failure signals. This recommendation follows the current code seam in Findings 1 and 3 while respecting the benchmark boundary in Findings 2 and 4. The remaining gap is empirical: GPT-5.4 still needs larger reasoning-skill benchmarks before it can move beyond overflow use.

---

## Structured Uncertainty

**What's tested:**

- ✅ The current daemon only hard-pins Opus for a sparse set of reasoning skills and leaves `feature-impl` model selection to downstream defaults (verified by reading `pkg/daemon/skill_inference.go` and `pkg/daemon/issue_adapter.go`).
- ✅ GPT-5.4 has benchmark evidence strong enough for `feature-impl` overflow but not for reasoning-default routing (verified by reading `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md`).
- ✅ Existing daemon architecture already has reusable seams for override resolution and complexity labels (verified by reading `pkg/daemon/coordination.go` and `pkg/daemonconfig/compliance.go`).

**What's untested:**

- ⚠️ GPT-5.4 may become safe for investigation or debugging after a larger benchmark, but current evidence is too small.
- ⚠️ `effort:*` labels may be incomplete in real queues, so some otherwise-safe `feature-impl` work will remain on Opus until labeling discipline improves.
- ⚠️ The exact promotion trigger set (`empty_execution`, early silent death, repeat failure) still needs implementation and behavioral validation.

**What would change this:**

- If GPT-5.4 reaches >=90% first-attempt completion on reasoning-heavy skills at useful sample size, the default lane should be revisited.
- If `effort:*` labels prove too sparse or noisy to approximate complexity, the router will need a richer issue classifier.
- If promotion-to-Opus creates duplicate spawns or hides GPT regressions, the retry design should be narrowed or made manual-only.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Introduce a capability-aware daemon route object and move model choice out of `InferModelFromSkill()` | architectural | This changes how daemon coordination represents routing and affects multiple packages. |
| Keep Opus as the default lane for reasoning skills and unknown/high-complexity implementation work | architectural | This is a system policy choice that sets the baseline behavior across daemon spawns. |
| Allow GPT-5.4 only for bounded `feature-impl` overflow and add promotion-to-Opus on failure | architectural | It crosses routing, observability, and retry boundaries and must stay coherent across them. |

### Recommended Approach ⭐

**Two-lane capability-aware router** - Replace skill-only model inference with a route policy that picks between an Opus default lane and a GPT-5.4 overflow lane using skill, effort labels, and prior failure state.

**Why this approach:**
- It matches the actual evidence boundary: GPT-5.4 is good enough for bounded implementation overflow, not for reasoning-default work.
- It reuses existing daemon structures (`effort:*`, combo-style config, route rewriting) instead of adding a second ad hoc routing system.
- It directly mitigates defect classes 2, 5, 6, and 7 by making backend/model capability explicit, keeping one canonical route derivation, and bounding retries.

**Trade-offs accepted:**
- We defer GPT-5.4 default routing for investigation/architect/debugging until a larger benchmark exists.
- We accept that some small implementation work will stay on Opus when effort labeling is missing, because false negatives are safer than false positives at this stage.

**Implementation sequence:**
1. Add a first-class `ModelRoute` decision path in daemon coordination so routing can consider issue metadata, explicit overrides, and route reason strings.
2. Add daemon config + observability for route policy so humans can see why GPT-5.4 or Opus was chosen and override defaults when benchmarking changes.
3. Add bounded GPT failure promotion to Opus so the overflow lane has a safe recovery path instead of repeated blind retries.

### Alternative Approaches Considered

**Option B: Route all `feature-impl` work to GPT-5.4**
- **Pros:** Maximizes ChatGPT Pro value and frees Anthropic capacity aggressively.
- **Cons:** Overreaches the benchmark, ignores observed scope-control weakness, and removes a safety distinction between `effort:small` and `effort:large` work.
- **When to use instead:** If GPT-5.4 later reaches >=90% first-attempt completion with acceptable scope control across larger implementation samples.

**Option C: Keep GPT-5.4 manual-only and leave daemon routing unchanged**
- **Pros:** Zero implementation work and zero new daemon risk.
- **Cons:** Wastes the newly validated overflow route, keeps provider monoculture as the operational default, and fails to capture route reasons in the system.
- **When to use instead:** If Dylan decides multi-model daemon routing is not worth operational complexity yet.

**Rationale for recommendation:** Option A captures the real state of evidence: GPT-5.4 is neither "not viable" nor "default-safe." A bounded overflow lane is the coherent middle ground.

---

### Implementation Details

**What to implement first:**
- Replace `InferModelFromSkill()` call sites with a route policy that has access to the whole issue, not just the inferred skill.
- Reuse `effort:*` labels as the initial complexity gate and keep explicit `skill:*` / manual `--model` overrides above daemon heuristics.
- Add route reason + promotion metadata to preview/status/events before enabling GPT routing so behavior stays legible.

**Things to watch out for:**
- ⚠️ Do not let GPT promotion create repeated work on issues already retried once; this is a duplicate-action hazard.
- ⚠️ Keep backend/model selection canonical in one place or the system will reintroduce contradictory authority signals between daemon and resolve layers.
- ⚠️ Do not infer "small" from issue type alone; missing complexity evidence should fall back to Opus, not optimism.

**Areas needing further investigation:**
- A focused N>=10 benchmark for GPT-5.4 on investigation and debugging work.
- Whether Sonnet should become a same-backend middle lane or remain untested until after GPT routing lands.
- Whether route policy should eventually consume empirical success-rate data, not just static defaults and labels.

**Success criteria:**
- ✅ Reasoning-heavy skills and high-complexity work still route to Opus by default.
- ✅ Eligible `feature-impl` issues can route to GPT-5.4 and expose a route reason in preview/status output.
- ✅ GPT-routed empty-execution or repeat failures promote once to Opus and surface that promotion in events/review artifacts.

---

## References

**Files Examined:**
- `pkg/daemon/skill_inference.go` - Current skill and model inference seam.
- `pkg/daemon/coordination.go` - Existing route-rewrite seam used by extraction and architect escalation.
- `pkg/daemon/issue_adapter.go` - How inferred model aliases become `orch work --model` flags.
- `pkg/daemonconfig/compliance.go` - Existing combo-first policy resolution pattern worth reusing.
- `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Current GPT-5.4 reliability evidence.
- `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md` - Historical GPT failure modes and stale prompt-size framing.

**Commands Run:**
```bash
# Verify repository root
pwd

# Create investigation artifact
kb create investigation design-daemon-route-tasks-gpt --orphan

# Pull relevant KB context
kb context "daemon model routing"

# Create implementation plan
orch plan create daemon-gpt54-routing
```

**External Documentation:**
- None.

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Supplies the current GPT-5.4 benchmark that makes bounded overflow routing viable.
- **Plan:** `.kb/plans/2026-03-26-daemon-gpt54-routing.md` - Decomposes this design into implementation phases and issue handoff.
- **Workspace:** `.orch/workspace/og-arch-design-daemon-route-26mar-c0b3/` - Session artifacts for orchestrator completion.

---

## Investigation History

**[2026-03-26 14:32]:** Investigation started
- Initial question: How should daemon routing change now that GPT-5.4 has fresh benchmark evidence?
- Context: Dylan asked for a design that routes work between GPT-5.4 and Opus using skill complexity and model capability.

**[2026-03-26 14:49]:** Current routing seam verified
- Confirmed the daemon still does sparse skill-only model inference and has no issue-aware complexity router.

**[2026-03-26 15:06]:** Benchmark evidence integrated
- Folded Mar 26 GPT-5.4 benchmark and DAO-13 refinement into a two-lane routing recommendation.

**[2026-03-26 15:18]:** Investigation completed
- Status: Complete
- Key outcome: Recommend a capability-aware router that keeps Opus default, uses GPT-5.4 only for bounded `feature-impl` overflow, and promotes failed GPT runs to Opus.
