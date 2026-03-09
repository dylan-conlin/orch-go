## Summary (D.E.K.N.)

**Delta:** Take the harness engineering model from internal tooling to published framework with portable tooling.

**Evidence:** 13/13 harness plan issues shipped (5 layers), 3 entropy spirals documented, daemon.go +892/spawn_cmd.go -840 as natural experiment, 265 contrastive trials, 1,625 lost commits, MVH checklist produced, `orch harness init` automated.

**Knowledge:** Agent failure is harness failure. Soft instructions dilute under pressure; hard gates don't. Nobody else has pain + evidence + working system + vocabulary — all four.

**Next:** Release governance health metric and escape hatch tracking (Phase 1 parallelizable), begin 30-day trajectory measurement.

---

# Plan: Harness Publication

**Date:** 2026-03-08
**Status:** Active
**Owner:** Dylan

**Extracted-From:** Harness engineering plan (2026-03-08, 13 issues, completed), thread "harness-engineering-as-strategic-position"
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Objective

Publish the harness engineering framework — moving from "internal tooling that works for us" to "published framework with evidence that others can adopt." Success = a publication with unassailable evidence, portable tooling that works outside orch-go, and cross-language validation.

---

## Substrate Consulted

- **Models:** `harness-engineering` (5 layers, hard/soft distinction), `architectural-enforcement` (multi-layer gates)
- **Decisions:** Three-layer hotspot enforcement (2026-02-26), harness plan phases 1-6 (complete)
- **Guides:** `minimum-viable-harness.md` (3-tier MVH checklist, just produced)
- **Constraints:** Cross-language evidence needed before publication claims generality. 30-day trajectory data needed before claiming gates work.
- **Threads:** "Harness engineering as strategic position" (build -> publish -> tooling), "Open questions in harness-as-governance" (9 questions, 3 active beads)

---

## Phases

### Phase 1: Deepen the Model (parallelizable)

**Goal:** Answer the open questions that strengthen the framework before publishing.
**Deliverables:**
- Governance health metric designed and implemented (`orch-go-ycdbr`)
- Escape hatch usage tracking in events.jsonl (`orch-go-v892h`)
- 30-day accretion trajectory measurement started (`orch-go-1ittt`)
**Exit criteria:** `orch entropy` reports a single governance health score; escape hatches are tracked; first weekly snapshot taken.
**Depends on:** Nothing — can start immediately.
**Issues:** `orch-go-ycdbr`, `orch-go-v892h`, `orch-go-1ittt`

### Phase 2: Cross-Language Evidence

**Goal:** Validate that harness patterns translate beyond Go.
**Deliverables:**
- `orch harness init` run on a Python or TypeScript project
- 1-2 weeks of agent operation under governance
- Comparison of accretion trajectories (governed vs ungoverned)
**Exit criteria:** Evidence for or against cross-language applicability documented.
**Depends on:** Phase 1 (need measurement tooling working first)
**Issues:** `orch-go-xi1tk`

### Phase 3: Publication Draft

**Goal:** Write the framework paper/post with full evidence chain.
**Deliverables:**
- Publication draft: problem, evidence, framework, open questions
- Structure: (1) accretion in multi-agent codebases, (2) 3 entropy spirals + natural experiment data, (3) hard/soft harness 5-layer framework, (4) open questions as honest gaps
**Exit criteria:** Draft reviewed, evidence claims traceable to sources.
**Depends on:** Phase 1 (trajectory data), Phase 2 (cross-language evidence), `orch-go-ycdbr`, `orch-go-v892h`
**Issues:** `orch-go-ap2jw`

### Phase 4: Portable Tooling

**Goal:** Extract harness tooling into standalone package.
**Deliverables:**
- Standalone CLI or library for harness governance (init, lock, verify, entropy)
- Works outside orch-go — any team with Claude Code agents
- 30-minute setup target
**Exit criteria:** Non-orch project can bootstrap governance using the tool.
**Depends on:** Phase 3 (publication validates framework), Phase 2 (cross-language evidence)
**Issues:** `orch-go-f85z0`

---

## Decision Points

### Decision 1: Publication format

**Context:** Blog post vs paper vs both. Audience determines depth.

**Options:**
- **A: Long-form blog post** — Accessible, shareable, iteratable. Pros: fast, wide reach. Cons: less credibility.
- **B: Technical paper** — Formal, cited, archived. Pros: durability, academic reach. Cons: slow, narrow audience.
- **C: Blog post first, paper later** — Blog validates interest, paper follows with more data.

**Recommendation:** C — blog post first to capture the position, paper later with 30-day data and cross-language evidence.

**Status:** Open

### Decision 2: Portable tooling scope

**Context:** Full CLI vs library vs Claude Code extension vs `orch harness` subcommands.

**Options:**
- **A: Standalone CLI** — Independent binary, no orch dependency. Pros: widest adoption. Cons: duplication.
- **B: Library + thin CLI** — Core logic as importable package. Pros: embeddable. Cons: Go-only initially.
- **C: Claude Code MCP server** — Governance as a service. Pros: language-agnostic. Cons: coupling to Claude Code.

**Recommendation:** Deferred until cross-language evidence clarifies what's portable.

**Status:** Deferred

---

## Structured Uncertainty

**What's tested:**
- Hard gates prevent accretion (spawn_cmd.go -840 lines after pkg/spawn/backends/)
- Soft instructions dilute under pressure (265 contrastive trials)
- Structural attractors break re-accretion cycle (pkg/spawn/ stayed small)

**What's untested:**
- Whether 5-layer harness bends 30-day line count trajectory (measurement starting)
- Whether patterns translate to non-Go languages
- Whether governance health can be captured as a single metric
- Soft harness budget curve — how many behavioral slots before dilution?

**What would change this plan:**
- If 30-day trajectory shows no improvement, harness claim weakens significantly
- If cross-language test fails, scope publication to "Go + Claude Code" specifically
- If someone publishes a competing framework first, accelerate blog post (Phase 3)

---

## Success Criteria

- [ ] Governance health metric implemented and reporting (`orch-go-ycdbr`)
- [ ] Escape hatch tracking live for 30 days (`orch-go-v892h`)
- [ ] 30-day accretion trajectory shows measurable improvement (`orch-go-1ittt`)
- [ ] Cross-language evidence collected (`orch-go-xi1tk`)
- [ ] Publication draft complete with traceable evidence (`orch-go-ap2jw`)
- [ ] Portable tooling works outside orch-go (`orch-go-f85z0`)
