## Summary (D.E.K.N.)

**Delta:** Publish knowledge accretion theory + kb-cli as a coherent Path A+B narrative — agent coordination problems (blog audience) explained by general theory (knowledge accretion), with tooling to try it.

**Evidence:** 5 existing blog posts (all Path A), falsifiability probe (15+ counterexamples, none clean), kb-cli v0.1.0 tagged, knowledge-accretion model resolved, 5 active threads synthesized.

**Knowledge:** Path A and Path B are not competing — Path A (harness engineering) is the empirical evidence, Path B (knowledge accretion) is the theory that explains it. The blog audience arrives via Path A. Knowledge accretion is the payoff.

**Next:** Resolve Ostrom framing thread (orch-go-3hdyt) — this shapes the essay's claims and tone.

---

# Plan: Knowledge Accretion Publication

**Date:** 2026-03-10
**Status:** Active
**Owner:** Dylan

**Extracted-From:** Threads: knowledge-accretion, abiogenesis, distribution-as-substrate, validation-gap, what-kind-of-theory, publication-sequencing
**Supersedes:** Previous Path B-only publication plan

---

## Objective

Publish the knowledge accretion essay to dylanconlin.com, announce kb-cli in the same cycle, and follow with the coordination failure demo. Each piece creates demand for the next. Success = the essay names a problem readers recognize, kb-cli gives them a "try it" button, and the demo provides the strongest empirical hook for sharing.

---

## Substrate Consulted

- **Models:** knowledge-accretion (five conditions, substrate-independent dynamics), harness-engineering (compliance vs coordination)
- **Decisions:** Path A+B (both, not either), Ostrom-scale framing (diagnostic framework, not natural law)
- **Guides:** Publication sequencing thread — sequential demand creation vs parallel launches
- **Constraints:** Validation gap — one person, one system. Essay must be honest about this.

---

## Decision Points

### Decision 1: Essay framing — Ostrom or physics?

**Context:** "Knowledge accretion" is a catchy name but overclaims. Ostrom-scale institutional analysis is more accurate.

**Options:**
- **A: Keep "knowledge accretion" as title, Ostrom as framing** — Name is memorable, body is honest about scope. Pros: SEO, shareability. Cons: sets expectations high.
- **B: Lead with Ostrom framing, "knowledge accretion" as shorthand** — More academically honest. Pros: defensible claims. Cons: less catchy.

**Recommendation:** A — the name is a hook, the body earns it. The falsifiability probe gives credibility to bold framing.

**Status:** Resolved (orch-go-3hdyt) — Keep "knowledge accretion" as title, Ostrom as framing. Theory Type section added to model with analogues (Ostrom, Conway, Brooks). Tone: "here's a predictive pattern I found and tried to break" not "here's a new field of science."

### Decision 2: kb-cli timing — same post or separate?

**Context:** kb-cli is already public. Could announce in the essay or as a follow-up post.

**Options:**
- **A: Mention in essay, separate announcement post** — Essay stays theoretical, announcement is practical. Pros: each piece focused. Cons: two launches to coordinate.
- **B: Essay includes "try it" section at the end** — Single post, longer. Pros: immediate conversion. Cons: dilutes the theory.

**Recommendation:** A — separate posts, same week. The essay creates the "I have this problem" moment, the announcement post says "here's what I built."

**Status:** Open

---

## Phases

### Phase 1: External validation (validate-then-publish)
**Goal:** Get independent data points before theorizing in public
**Deliverables:**
- Coordination failure demo post — reproducible experiment, no theory required, include exact steps so anyone can run it (orch-go-orlcp)
- One external kb-cli user — tool feedback, not theory validation. Their observations are independent data (orch-go-3occh)
**Exit criteria:** At least one person reproduces the coordination demo OR reports kb-cli observations
**Depends on:** Nothing — this is the root

**Why this goes first:** The theory was built by one person with AI agents that reinforce coherent framing. Internal consistency is not external validation. The formula-shaped claims (accretion_risk = f(...)) emerged from the closed loop, not from measurement. Ship the reproducible observation first, see if it holds.

### Phase 2: Essay (only after external data)
**Goal:** Publish knowledge accretion essay grounded in externally-confirmed observations
**Deliverables:**
- Ostrom framing resolved (orch-go-3hdyt) — DONE
- Knowledge accretion essay draft on blog (orch-go-lnve9)
- Distribution-as-substrate paragraph folded in (orch-go-ightw)
- Abiogenesis as Future Directions section (orch-go-dm79z)
**Exit criteria:** Essay published to dylanconlin.com, grounded in at least one external data point
**Depends on:** Phase 1 (external validation)

### Phase 3: Tooling announcement
**Goal:** Give readers a "try it" button
**Deliverables:**
- kb-cli announcement post (orch-go-knmq8)
**Exit criteria:** Post published, links to repo, shows investigation/probe/model cycle
**Depends on:** Phase 2 (essay creates demand)

### Phase 4: Ongoing validation
**Goal:** External confirmation at scale
**Deliverables:**
- Additional external kb-cli users (orch-go-ymd06)
- Conference talk or practitioner pushback
**Exit criteria:** Multiple independent confirmations or a clean counterexample
**Depends on:** Phases 1-3

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Essay framing | knowledge-accretion model, falsifiability probe, Ostrom thread | Yes |
| kb-cli timing | v0.1.0 tagged, repo public | Yes |
| Coordination demo scope | N=10 data, compliance/coordination distinction | Yes |
| Validation path | No external users yet | Blocked on publication |

**Overall readiness:** Ready to execute Phase 1 (external validation).

---

## Structured Uncertainty

**What's tested:**
- Theory survives 15+ counterexamples (falsifiability probe)
- Five conditions predict accretion in code and knowledge substrates
- kb-cli works end-to-end (investigation/probe/model cycle)
- Blog audience exists for Path A content (5 published posts)

**What's untested:**
- Whether Path A audience converts to Path B interest
- Whether "knowledge accretion" framing resonates or alienates
- Whether kb-cli is usable by someone who didn't build it
- Whether the theory holds for non-code, non-knowledge substrates

**What would change this plan:**
- Early feedback on essay says framing is wrong → revisit Ostrom vs physics
- kb-cli first external user hits major friction → pause Phase 3, fix tooling
- Someone finds a clean counterexample → update theory before publishing

---

## Success Criteria

- [ ] Knowledge accretion essay published to dylanconlin.com
- [ ] kb-cli announcement post published within same week
- [ ] Coordination failure demo post published within 2 weeks
- [ ] At least one external person tries kb-cli
- [ ] No clean counterexamples surface post-publication
