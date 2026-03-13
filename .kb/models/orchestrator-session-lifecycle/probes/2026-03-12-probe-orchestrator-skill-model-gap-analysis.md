# Probe: Orchestrator Skill Model — Gap Analysis Between Session-Lifecycle and New Skill Model

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-12
**Status:** Complete

---

## Question

The orchestrator-session-lifecycle model (48KB) covers orchestrator sessions broadly, including extensive sections about the orchestrator skill's injection paths, failure modes, and evolution. A new orchestrator-skill model has been created at `.kb/models/orchestrator-skill/model.md`. What content should migrate vs stay? What gaps exist in NEITHER model? Where is the correct boundary?

---

## What I Tested

### Test 1: Section-by-section audit of skill-specific content in session-lifecycle model

Read the full orchestrator-session-lifecycle model (642 lines) and tagged every section by whether it's about sessions, the skill, or both:

**Skill-specific sections (should migrate or be referenced):**
- Lines 46-70: Skill Injection Paths (5 paths, 4 bugs, fix path) — 25 lines
- Lines 88-102: Orchestrator Detection via skill metadata — 15 lines
- Lines 156-174: Orientation Preservation Dimension (skill's orientation gap) — 19 lines
- Lines 179-362: "Why This Fails" section — 13 failure modes, 11 of which are directly about the skill document, its injection, or its behavioral effects — 184 lines
- Lines 553-565: Phase 8 (Skill Injection Audit) — 13 lines
- Lines 619-642: Merged Probes table — 21 of 22 probes are skill-related — 24 lines

**Total skill-specific content in session-lifecycle model:** ~280 of 642 lines (~44%)

**Session-specific sections (should stay):**
- Lines 9-17: Summary (strategic comprehender, SESSION_HANDOFF, state derivation) — 9 lines
- Lines 19-44: Core Mechanism (strategic comprehender pattern, hierarchy diagram) — 26 lines
- Lines 72-87: Session Types and Boundaries — 16 lines
- Lines 104-154: State Derivation, Checkpoint Discipline, Frame Shift — 51 lines
- Lines 365-464: Constraints section (beads tracking, tmux default, phase reporting, checkpoint thresholds, knowledge surfacing, decidability graph, lifecycle ownership, session identity) — 100 lines
- Lines 466-551: Evolution Phases 1-7 — 86 lines
- Lines 567-588: Resume Protocol — 22 lines
- Lines 591-617: References — 27 lines

### Test 2: Probe migration analysis

Read all 6 probes specified in the task plus the merged probes table. Categorized by primary domain:

| Probe | Primary Domain | Skill-Specific? | Should Migrate? |
|-------|---------------|-----------------|-----------------|
| 2026-02-24 behavioral-compliance | Skill signal ratio, identity vs action gap | YES | YES — core skill design finding |
| 2026-03-11 failure-mode-taxonomy | Skill failure catalog from 6 investigations | YES | YES — skill failure taxonomy |
| 2026-03-11 current-state-audit | Skill implementation status audit | YES | YES — skill operational status |
| 2026-03-02 emphasis-language | Constraint expression style | DUAL — general principle, tested on skill | COPY to orchestrator-skill, keep in behavioral-grammars |
| 2026-02-25 cross-project-injection | Skill injection infrastructure bug | YES | YES — skill infrastructure |
| 2026-03-12 architect-design-bypass | Spawn pipeline knowledge feedback | PARTIAL — about spawn pipeline, not the skill itself | STAY — more about session/pipeline than skill |

### Test 3: Gap identification — content in probes not in either model

Cross-referenced all 6 probes against both the orchestrator-session-lifecycle model.md and the new orchestrator-skill model.md:

**Gap 1: Failure mode #13 (Architect Design Bypass) not in orchestrator-skill model**
- The session-lifecycle model was updated 2026-03-12 with failure mode #13
- The orchestrator-skill model lists only 12 failure modes (A1-A4, B1-B3, C1-C3, D1-D2)
- The 5-layer failure chain (issue framing > kb pointer > no injection mechanism > no skill checkpoint > rushed planning) has no home in the skill model

**Gap 2: Knowledge-feedback inversion pattern**
- The 68gcy probe identifies a new failure CLASS: when issue description framing contradicts prior architect design
- Neither model names this as "knowledge-feedback inversion" or captures the general pattern
- The orchestrator-skill model discusses the spawn pipeline superficially but doesn't model how knowledge signals compete for attention in SPAWN_CONTEXT

**Gap 3: Stance as distinct content type**
- The behavioral-grammars model (Mar 6 refinement) distinguishes knowledge/stance/behavioral
- The orchestrator-skill model uses knowledge/behavioral (two types), missing stance
- "Test before concluding" and "evidence hierarchy" are stance in the skill, but not identified as such

**Gap 4: Measurement validity propagation**
- Behavioral-grammars has a detailed "Measurement Validity" section with 3 compounding issues
- The orchestrator-skill model mentions "N=3 unreplicated" and "directional hypotheses" but doesn't adopt the full 3-layer validity concern (wrong injection level, intent ≠ action, replication failure)
- This matters because orchestrator-skill's budget numbers (4 behavioral, 50 knowledge) inherit all three validity issues

**Gap 5: Concrete fix designs from probes**
- Cross-project probe: 3-change fix design (ORCH_SPAWNED, is_spawned_agent, main restructure)
- Behavioral compliance probe: specific prevention vectors (graduated hooks, prompt-level vs infrastructure separation)
- These exist in probes but aren't captured in actionable form in either model

**Gap 6: The CLAUDECODE env var testing blocker**
- Current-state audit identifies that `skillc test` bare-parity regression is blocked because spawned agents can't run it (CLAUDECODE env var)
- This is the critical reason why the behavioral validation gate was NEVER completed
- Neither model captures this as a specific technical blocker vs general "testing pending"

### Test 4: Boundary analysis between the two models

Identified the natural boundary by examining what each model's invariants depend on:

**Session-lifecycle model depends on:**
- How sessions are created, managed, and completed
- How state is derived from 4 sources (no local state)
- Checkpoint thresholds as context exhaustion proxy
- Hierarchical completion model (completed by level above)
- Frame shift patterns and detection

**Orchestrator-skill model depends on:**
- How the skill document shapes behavior (probability, not grammar)
- How skill content is injected and cached
- How constraints dilute under attention competition
- How infrastructure (hooks) enforces what prompts can't
- How the skill accumulates and simplifies cyclically

**Boundary rule:** If a finding is about "what orchestrators do in sessions" → session-lifecycle. If it's about "how the skill document shapes what orchestrators do" → orchestrator-skill.

**Overlapping areas that need clear attribution:**
- Failure modes: Many are about the skill operating within a session. The skill model should own the canonical descriptions. The session-lifecycle model should reference them.
- Skill injection paths: Infrastructure that makes sessions work but is fundamentally about the skill. Should live in orchestrator-skill.
- Evolution Phase 8: Entirely about the skill audit. Should be referenced from session-lifecycle, owned by orchestrator-skill.

### Test 5: Behavioral-grammars relationship analysis

Read the full behavioral-grammars model.md (196 lines). Identified what's general vs specific:

**General (belongs in behavioral-grammars):**
- 7 core claims (probabilistic constraints, redundancy, situational pull, fused artifacts, self-detection failure, intent degradation, grammar coupling)
- 9 design dimensions (prior strength, phase coverage, etc.)
- Revert spiral anti-pattern
- Three content types refinement (knowledge/stance/behavioral)
- Measurement validity framework
- Open questions about cross-model generalization

**Specific to orchestrator skill (belongs in orchestrator-skill, can REFERENCE behavioral-grammars):**
- 17:1 competing signal ratio with Claude Code system prompt
- Specific hook implementations and their tests
- The routing table structure and intent classification
- Dylan's 4 orientation moments
- The 5 injection paths and their bugs
- The specific accretion-crisis token trajectory
- The 4 behavioral norms (delegation, filter, act-by-default, answer-the-question)

**Dual-homed findings (evidence from skill, principle is general):**
- Identity vs action compliance gap (Feb 24 probe) — already dual-homed correctly
- Emphasis vs neutral language (Mar 2 probe) — already dual-homed correctly
- Constraint dilution thresholds — correctly general (in BG) with application (in OS)

---

## What I Observed

### The session-lifecycle model is 44% about the skill

This is the core structural finding. Nearly half the content in the "session lifecycle" model is actually about the skill document itself — its injection, its failure modes, its evolution. This isn't drift; it happened because the skill IS the primary mechanism by which sessions are shaped. But it conflates two distinct models:

1. **How sessions work** (state, boundaries, checkpoint discipline, completion)
2. **How the skill shapes sessions** (injection, dilution, failure modes, design tensions)

### The new orchestrator-skill model is already comprehensive

The 161-line orchestrator-skill model.md captures the majority of what the probes found. It has the right structure (core nature, two-layer architecture, critical invariants, 12 failure modes, 9 design tensions, accretion cycle, current state, open questions). The primary gaps are:

1. Missing failure mode #13 (architect design bypass — just added to session-lifecycle 2026-03-12)
2. Missing stance content type (from behavioral-grammars refinement)
3. Missing measurement validity propagation from behavioral-grammars
4. Missing concrete fix designs from individual probes
5. Missing CLAUDECODE env var as named technical blocker

### Probe migration is mostly clean

5 of 6 probes are clearly skill-specific and should migrate. The 6th (architect design bypass) is at the boundary — it's about the spawn pipeline's knowledge feedback, which affects how the skill's spawn context works but is equally about session lifecycle.

### The behavioral-grammars relationship is correctly structured

The general principles live in behavioral-grammars. The orchestrator skill is the most-studied instance of a behavioral grammar. The dual-homing of the Feb 24 and Mar 2 probes is correct — they provide evidence for general principles while analyzing a specific skill.

---

## Model Impact

- [x] **Confirms** invariant: The session-lifecycle model's assertion that "Framing cues override skill instructions" (line 183) remains valid. The skill model's framing (probability-shaper, not grammar) is consistent with this.

- [x] **Extends** model with: The session-lifecycle model should slim down by migrating ~280 lines of skill-specific content to the orchestrator-skill model. Sections to migrate: Skill Injection Paths (46-70), failure modes (#1-#4, #6-#11, #13) whose canonical descriptions should live in orchestrator-skill, and Phase 8 evolution. The session-lifecycle model should retain references/pointers.

- [x] **Extends** model with: 5 probes (Feb 24 behavioral-compliance, Mar 11 failure-taxonomy, Mar 11 current-state, Mar 2 emphasis-language, Feb 25 cross-project-injection) should be COPIED to `.kb/models/orchestrator-skill/probes/` since they are primarily about the skill, not about sessions. They can remain in session-lifecycle probes as historical artifacts.

- [x] **Extends** model with: The orchestrator-skill model has 5 concrete gaps that should be filled: (1) failure mode #13 from the 68gcy probe, (2) stance as a third content type from behavioral-grammars, (3) full measurement validity framework adoption, (4) CLAUDECODE env var as named testing blocker, (5) concrete fix designs from cross-project and behavioral compliance probes.

---

## Notes

- This probe is the gap analysis that informs construction of the orchestrator-skill model. It does NOT recommend changes to the session-lifecycle model — only identifies what would move if the models were separated.
- The boundary rule ("what orchestrators do" vs "how the skill shapes what they do") is clean in theory but fuzzy in practice. Failure modes like Frame Collapse (#1) are about sessions (the orchestrator drops levels) caused by the skill (framing > instructions). The recommendation: canonical description in orchestrator-skill, reference in session-lifecycle.
- The new orchestrator-skill model at 161 lines is ~30% the size of the session-lifecycle model but covers ~80% of the skill-specific findings. This is efficient — it synthesized well.
