## Summary (D.E.K.N.)

**Delta:** Convert the newly accepted product boundary into a staged consolidation plan so orch-go can reorganize around the thread/comprehension layer without destabilizing the substrate that still makes the system usable.

**Evidence:** Product-boundary decision (`2026-03-26-thread-comprehension-layer-is-primary-product.md`), threads on threads-as-primary and OpenClaw migration, current repo/docs/UI still weighted toward orchestration/execution identity.

**Knowledge:** The project no longer needs broad identity search. It needs consolidation. The right move is not immediate deletion of execution infrastructure, but demotion of execution from product identity to replaceable substrate while promoting thread-centric comprehension surfaces to the center.

**Next:** Execute Phase 1 by rewriting the top-level story and mapping current systems into core / substrate / adjacent categories before starting major new feature work.

---

# Plan: Thread/Comprehension Consolidation

**Date:** 2026-03-26
**Status:** Active
**Owner:** Dylan

**Extracted-From:**
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`
- `.kb/threads/2026-03-24-threads-as-primary-artifact-thinking.md`

## Objective

Reorganize orch-go around the thread/comprehension layer as the primary product, while preserving execution infrastructure as portable substrate rather than continuing to let it define the system's identity, roadmap, and UI center of gravity.

Success means:

- the top-level story matches the strongest actual idea in the repo
- the product surface becomes thread-first rather than work-graph-first
- execution plumbing remains usable but no longer drives strategic scope by default
- adjacent assets (research, benchmarking, migration work) are clearly separated from core product identity

## Context

The project discovered its center through building:

- orchestration, spawn, daemon, verification, and dashboard work created real capability
- recent threads clarified that the differentiated value is not generic agent execution
- the real value is the system that makes agent output accumulate into durable understanding

This creates a familiar but important transition:

- **Past phase:** search broadly, ship aggressively, learn what matters
- **Current phase:** consolidate around the strongest layer and use it to prune future scope

The main risk is inertia. Existing code, docs, and UI still reinforce the older orchestration-centric identity. Without an explicit consolidation plan, the repo will keep funding the old story even after the new one is understood.

## Boundaries

### Core

These should increasingly define both product identity and roadmap:

- threads and thread graph
- synthesis / briefs / comprehension surfaces
- claims, models, decisions, and structured knowledge composition
- review / verification / routing where they improve trust and legibility
- human-facing surfaces for understanding what changed, what was learned, and what remains unresolved

### Substrate

These remain necessary but should be treated as enabling layers:

- spawn and daemon plumbing
- backend/client integrations
- session transport
- execution-specific observability
- provider/platform migration mechanisms

### Adjacent

These are valuable but should not keep bending the main product:

- coordination research and experimental harnesses
- model benchmarking
- migration studies and platform probes

## Phases

### Phase 1: Rewrite the top-level story

**Goal:** Make the repo and product legible in terms of the new boundary.

**Deliverables:**
- README rewritten around coordination + comprehension, not orchestration CLI first
- architecture overview updated to reflect core/substrate distinction
- one concise positioning artifact for future writing and demos

**Exit criteria:** A new reader can understand the product as a thread/comprehension layer without reading the entire KB.

**Why first:** Until the story changes, the old identity keeps reproducing itself in future work.

### Phase 2: Map the current system into core / substrate / adjacent

**Goal:** Turn the strategic boundary into a concrete inventory.

**Deliverables:**
- written classification of major subsystems and docs
- list of what remains first-class, what becomes supporting, and what should be extracted or de-emphasized
- explicit "do not invest by default" list for execution-centric work

**Exit criteria:** New work can be judged against a visible map instead of memory or vibes.

**Why second:** The decision is too abstract unless it becomes a classification tool.

### Phase 3: Promote thread-first product surfaces

**Goal:** Make the product behave like the decision is true.

**Deliverables:**
- design for a thread-centric primary UI surface
- path for briefs/synthesis/evidence to render inside thread context
- work graph repositioned as subordinate view rather than the conceptual home screen

**Exit criteria:** There is a credible path to starting the day from threads/questions/learning rather than active agents/issues.

**Why third:** This is the clearest product expression of the new center.

### Phase 4: Demote execution plumbing from identity to substrate

**Goal:** Preserve execution capability without over-investing in ownership of every layer.

**Deliverables:**
- explicit substrate strategy for OpenClaw / Claude Code / OpenAI / other clients
- criteria for when execution work is strategically necessary vs maintenance-only
- reduction of roadmap pressure from backend-specific features unless they unlock core value

**Exit criteria:** Execution work is governed by product needs, not by accidental code ownership.

**Why fourth:** This phase prevents the older infrastructure identity from reasserting itself.

### Phase 5: Separate adjacent research assets

**Goal:** Keep coordination research as an asset without letting it define the main product.

**Deliverables:**
- recommendation on what should remain inside orch-go vs move toward `coord-bench` or similar
- explicit relationship between product methodology and publishable/portable research outputs

**Exit criteria:** Research increases credibility and insight without bloating core scope.

## Readiness Assessment

| Area | Current State | Ready? |
|------|---------------|--------|
| Strategic boundary | Decision accepted | Yes |
| Thinking lineage | Existing thread already active | Yes |
| Product narrative | Drifted toward orchestration | Needs rewrite |
| UI center | Work/execution-centric | Needs redesign |
| Substrate portability | Emerging via OpenClaw migration work | Partially |
| Scope control | No explicit keep/de-emphasize map yet | No |

**Overall readiness:** Ready for consolidation planning and initial execution.

## Structured Uncertainty

**What is clear:**

- the strongest differentiated idea is above the execution layer
- the current repo presentation lags the current understanding
- thread-first product surfaces are strategically aligned with the new identity

**What is not yet resolved:**

- how quickly to de-emphasize work-graph-centric UX
- how much execution infrastructure should remain owned vs delegated to external platforms
- whether the thread-first surface should replace or merely sit above current views
- which research assets should be extracted first, if any

**What would change this plan:**

- evidence that users actually prefer execution-centric surfaces even after thread-first alternatives exist
- evidence that owning more of the execution layer is strategically necessary to protect the comprehension layer
- discovery that thread-centric workflows are powerful for Dylan but not portable to other users

## Initial Work Queue

These are the first candidate execution slices implied by the plan:

1. Rewrite README and top-level framing
2. Update architecture overview to reflect the new boundary
3. Create a subsystem inventory: core / substrate / adjacent
4. Design the thread-first primary UI surface
5. Reposition the work graph in docs and product language

## Success Criteria

- [ ] Top-level docs describe orch-go primarily as a thread/comprehension layer
- [ ] A visible subsystem map exists and informs prioritization
- [ ] Thread-first UI direction is explicit, not implicit
- [ ] Execution work is treated as substrate by default
- [ ] Adjacent research work has a defined relationship to the product
