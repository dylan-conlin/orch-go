## Summary (D.E.K.N.)

**Delta:** Consolidate orch-go around a content-first comprehension product by making the reading surface unmistakable, subtracting execution residue from the home surface, and defining a minimum open release that teaches the method.

**Evidence:** Product-boundary and openness decisions from 2026-03-26; daemon-boundary investigation; minimum comprehension surface probe; ranking/attention boundary probe; dashboard residue investigation; minimum open release probe.

**Knowledge:** The product center is clearer than the surface expression. The home page already has the right topology, but it still renders comprehension as metadata and leaves too much execution residue visible. The next move is not broad UI invention; it is making the comprehension loop readable and then teaching that method in the first open release.

**Next:** Implement the content-first comprehension surface: render thread entries, brief content, and tension text inline on the home surface, then add thread-grouped, tension-aware ordering.

---

# Plan: Thread Comprehension Product Consolidation

**Date:** 2026-03-26
**Status:** Active
**Owner:** Dylan

<!-- Lineage (fill only when applicable) -->
**Extracted-From:**
- `.kb/threads/2026-03-24-threads-as-primary-artifact-thinking.md`
- `.kb/investigations/2026-03-26-inv-daemon-behaviors-substrate-machinery-vs.md`
- `.kb/investigations/2026-03-26-design-current-dashboard-surfaces-comprehension-product.md`
- `.kb/models/dashboard-architecture/probes/2026-03-26-probe-minimum-comprehension-surface-product-identity.md`
- `.kb/models/dashboard-architecture/probes/2026-03-26-probe-ranking-attention-layer-boundary.md`
- `.kb/models/knowledge-accretion/probes/2026-03-26-probe-minimum-open-release-bundle-definition.md`
**Supersedes:** `.kb/plans/2026-03-26-thread-comprehension-consolidation.md`
**Superseded-By:** 

---

## Objective

Make orch-go unmistakably feel like a thread/comprehension product rather than an orchestration dashboard by (1) rendering the comprehension loop as readable content, (2) demoting execution residue from the primary surface, and (3) packaging the smallest open release that teaches the method on first contact. Success means a new user can encounter threads, briefs, and tensions as a coherent reading surface, while the daemon and other execution machinery remain present but clearly subordinate.

---

## Substrate Consulted

> What existing knowledge informed this plan?

- **Models:** `dashboard-architecture` now distinguishes metadata mode vs content mode and decomposes ranking into substrate ordering, method-expressing ordering, and future learned ranking. `knowledge-accretion` now records that artifact formats alone do not teach the method.
- **Decisions:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`; `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md`.
- **Guides:** `.kb/guides/architecture-overview.md`.
- **Constraints:** Thread/comprehension layer is primary; openness belongs at the boundary; the daemon is mostly substrate; release story must not collapse back into “orchestration CLI.”

---

## Decision Points

> For each fork ahead, what are the options and which is recommended?

### Decision 1: Surface Mode

**Context:** The current home surface has the correct thread-first topology, but still renders comprehension mostly as badges, counts, titles, and hidden content.

**Options:**
- **A: Metadata-first dashboard** - Keep counts, badges, and expandable lists as the dominant mode. Pros: cheap, familiar dashboard pattern. Cons: reads like monitoring software, not the product.
- **B: Content-first reading surface** - Render thread entries, brief text, and tensions inline as readable prose. Pros: matches the product thesis; makes the surface feel like comprehension. Cons: requires curation and subtraction of existing operational clutter.

**Recommendation:** Option B because the minimum comprehension surface probe found that the product-feel comes from readable content, not topology alone.

**Status:** Decided

---

### Decision 2: Ranking Boundary

**Context:** “Ranking intelligence” was treated as a future held-back surface, but the read-next probe found that some ordering logic is actually method core.

**Options:**
- **A: Treat all ranking as future intelligence** - Keep ordering minimal until a more advanced layer exists. Pros: defers complexity. Cons: contradicts thread-first method now.
- **B: Split ranking into layers** - Ship thread-grouping and tension surfacing as method-core ordering now; reserve learned/integrated ranking for later. Pros: makes “read next” coherent now without inventing a moat. Cons: adds near-term product work to the dashboard/briefs surface.

**Recommendation:** Option B because the ranking probe showed that thread grouping and tension surfacing are expressions of the method, not optional future intelligence.

**Status:** Decided

---

### Decision 3: Release Bundle

**Context:** The openness matrix implied that artifact formats could lead the first open wave, but the minimum release probe contradicted that assumption.

**Options:**
- **A: Formats-first release** - Publish schemas and artifact examples alone. Pros: simple, low-cost. Cons: does not teach the method; artifacts are not self-documenting enough.
- **B: Guided method bundle** - Ship artifact formats plus composition guide, thread CLI, and curated examples. Pros: teaches the method on first contact; avoids execution-first framing. Cons: requires documentation and example curation before release.

**Recommendation:** Option B because the minimum open release probe found formats alone score only 2.6/5 for standalone comprehensibility.

**Status:** Decided

---

## Phases

> Execution phases with clear deliverables and exit criteria.

### Phase 1: Make The Reading Surface Real

**Goal:** Transform the home surface from metadata-first dashboard into content-first comprehension surface.
**Deliverables:**
- Inline rendering of the product triangle: thread entry content, brief content, tension text
- Thread-grouped, tension-aware ordering for “read next”
- Fixes needed to make thread reading reliable and legible
**Exit criteria:** A user opening `/` spends time reading prose, not scanning counts; threads, briefs, and tensions appear as one comprehension loop.
**Depends on:** Existing thread-first home surface; current thread/brief APIs; ordering probe findings.

### Phase 2: Subtract Execution Residue

**Goal:** Make the primary surface feel like the product by demoting or deleting execution-first scaffolding.
**Deliverables:**
- Home-page demotion of execution-heavy sections into Work route
- Condensed operational summary on `/`
- Deletion of dead `/thinking` route
- Explicit UI boundary between comprehension core and operational subviews
**Exit criteria:** The home page is visibly comprehension-first all the way down; execution monitoring is available but no longer claims the center.
**Depends on:** Phase 1; dashboard residue classification.

### Phase 3: Define The Minimum Open Release

**Goal:** Package the smallest release that teaches the method without rebranding the product as orchestration infra.
**Deliverables:**
- Composition guide documenting how artifacts fit together
- Thread CLI/init flow that leads with comprehension rather than execution
- Curated examples showing the method in action
- Release boundary doc: must-ship, defer, background substrate
**Exit criteria:** A new user can understand and try the method without needing the full repo ontology or a daemon-centric workflow.
**Depends on:** Phase 1 for credible product surface examples.

### Phase 4: Reframe Future Investment Filters

**Goal:** Use the new boundary to govern future work across daemon, dashboard, and release decisions.
**Deliverables:**
- Future daemon work filter: substrate vs bridge vs method-core
- UI deletion/demotion criteria
- Product proposal filter based on surface mode and release-boundary findings
**Exit criteria:** New work is judged against an explicit comprehension-product filter rather than drifting by inertia.
**Depends on:** Phases 1-3.

---

## Readiness Assessment

> Can we navigate each decision point ahead?

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Surface mode | Minimum comprehension surface probe; current thread-first home surface | Yes |
| Ranking boundary | Ranking/attention probe; existing thread/brief infrastructure | Yes |
| Release bundle | Minimum open release probe; openness matrix; thread-comprehension decision | Yes |
| Daemon/UI boundary | Daemon-boundary investigation; dashboard residue investigation | Yes |

**Overall readiness:** Ready to execute

---

## Structured Uncertainty

**What's tested:**
- ✅ The daemon is mostly substrate, with only a thin bridge/method-core seam.
- ✅ The current home surface already has the correct topology; the remaining issue is rendering mode and execution residue.
- ✅ Thread grouping and tension surfacing belong to method-core ordering, not a speculative future moat.
- ✅ Artifact formats alone do not teach the method; a composition guide and examples are required.

**What's untested:**
- ⚠️ Whether content-first rendering on `/` will immediately feel like “the product” without a dedicated thread-reader route.
- ⚠️ Whether subtracting operational sections from the home page will generate backlash from current usage patterns.
- ⚠️ Whether the minimum open release bundle is enough without a more polished hosted/local comprehension UX.

**What would change this plan:**
- If real users prefer operational monitoring over reading-first orientation even after the content-first surface ships
- If the release examples and composition guide still fail to teach the method in first-contact testing
- If the ranking layer proves too dependent on future integrated signals to ship a useful method-core Layer 2 now

---

## Success Criteria

- [ ] The home page renders threads, briefs, and tensions as readable content rather than mostly counts and badges.
- [ ] Execution-heavy home page sections are demoted, condensed, or removed without losing access to operational visibility.
- [ ] The first open release bundle is explicitly defined as formats + composition guide + thread CLI + curated examples.
- [ ] New daemon and dashboard work can be judged with explicit substrate/bridge/method-core and product/residue filters.
