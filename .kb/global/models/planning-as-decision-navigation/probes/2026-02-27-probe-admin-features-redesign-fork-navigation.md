# Probe: Admin Features Redesign — Fork Navigation in Practice

**Status:** Complete
**Model:** Planning as Decision Navigation (`~/.kb/models/planning-as-decision-navigation.md`)
**Date:** 2026-02-27
**Context:** Designing a better /admin/features page for Toolshed. Current UI is a flat database join table — no grouping, free-text inputs, no revoke confirmation, doesn't work at narrow widths.

---

## Question

Does the fork-navigation approach (identify decision forks → consult substrate → navigate with recommendations) produce actionable design decisions for a UI redesign task, compared to jumping straight to a task list?

**Model claims being tested:**
1. "A plan is 'ready' not when tasks are listed, but when you have sufficient model to navigate the decisions ahead."
2. Substrate consultation (principles, models, decisions) constrains the option space and prevents context-free recommendations.
3. Unknown forks should be spiked, not guessed.

---

## What I Tested

**Approach:** Applied full fork-navigation protocol to the /admin/features redesign:

### Forks Identified
1. **View structure** — Feature-centric vs user-centric vs both
2. **Grant controls** — Free text vs dropdowns vs combobox
3. **Responsive layout** — Cards vs table vs hybrid
4. **Revoke confirmation** — Browser confirm vs inline vs modal vs two-step
5. **Component structure** — Single file vs multi-component split
6. **Backend API changes** — New endpoints vs client-side grouping

### Substrate Consulted
- **CLAUDE.md breakpoint rules**: 640px for structural shifts, never 768px. This directly constrained Fork 3.
- **CLAUDE.md component limits**: Max 250 lines per Svelte component. This directly constrained Fork 5.
- **Existing UI patterns**: KpiCard, InsightCard, AiPanel patterns from comparison view. Informed card styling decisions.
- **Principles (Premise Before Solution)**: Asked "should we redesign?" before "how?" — answer: yes, the flat table fails at all stated goals.
- **No prior decisions** on admin UI or feature management. Clean slate.

### Key Observations During Fork Navigation

**Fork 1 (View structure):** Substrate had nothing — no prior decision, no model constraint. But domain analysis (3 features, 5 users) made the choice clear: feature-centric, because features are the scarcer dimension and "who has this feature?" is the primary admin question. No spike needed.

**Fork 3 (Responsive layout):** Substrate directly constrained this. CLAUDE.md's breakpoint rules eliminated table-based approaches (tables need min-width hacks). Cards with `sm:` at 640px was the only option consistent with substrate.

**Fork 4 (Revoke confirmation):** No substrate constraint. Chose inline confirmation over modal because existing codebase has no modal component, and building one for a single button interaction is over-engineering (Premise Before Solution principle).

**Fork 6 (Backend changes):** Spawn context explicitly scoped this out. But analysis showed no backend changes are needed anyway — client-side grouping of the flat list is sufficient.

---

## What I Observed

1. **Fork navigation surfaced decisions that task-list planning would have buried.** A task list might say "redesign the admin page with cards" without ever explicitly deciding feature-centric vs user-centric. The fork approach forced that decision to be explicit and justified.

2. **Substrate consultation was unevenly useful.** For 2 of 6 forks (responsive layout, component structure), CLAUDE.md rules directly constrained the answer. For the other 4, substrate returned nothing useful and the decision came from domain analysis or simplicity heuristics.

3. **No forks required spiking.** All 6 were navigable with available context. This may be because UI redesign is a well-understood problem space — there aren't deep technical unknowns.

4. **The readiness test worked.** After navigating all 6 forks, I could explain every design choice with reasoning. Compare to: "just make it look like cards" which leaves 5 decisions implicit and likely to be revisited during implementation.

---

## Model Impact

**Confirms:**
- Claim 1 (readiness = navigable decisions): After fork navigation, the design is implementable without further clarification. A task list without fork navigation would have left ambiguity (which view structure? what kind of responsive? what confirmation pattern?).
- Claim 2 (substrate constrains options): CLAUDE.md breakpoint and component rules directly eliminated options for 2 forks.

**Extends:**
- **Substrate density varies by domain.** For UI/UX design forks, project-level substrate (CLAUDE.md conventions) is more useful than principles/models. The general principles (Premise Before Solution) provided meta-guidance but didn't constrain specific forks.
- **Domain analysis fills substrate gaps.** When substrate returns nothing, analysis of the current data (3 features, 5 users) and usage patterns ("who has this feature?" is the primary question) was sufficient to navigate forks. Not every fork needs formal substrate — some are decidable from the problem structure.
- **UI redesign forks are mostly navigable without spikes.** The model's spike protocol is more relevant for technical unknowns (API behavior, performance characteristics) than UI pattern selection, where the option space is well-known.
