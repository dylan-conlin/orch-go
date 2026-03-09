# Session Synthesis

**Agent:** og-inv-context-dylan-built-09mar-f84a
**Issue:** orch-go-8m7w9
**Duration:** 2026-03-09
**Outcome:** success

---

## Plain-Language Summary

Knowledge systems exhibit the same physics as code systems — accretion, attractors, gates, entropy. Measured across the full orch-go knowledge corpus (1,166 investigations, 32 models, 187 probes): 85.5% of investigations are orphaned (knowledge bloat equivalent to dead code), models act as attractors that pull investigation density toward them (daemon model attracted 34 probes over 21 days), and every knowledge gate is advisory — zero hard gates exist in the knowledge system. This means the knowledge substrate is governed entirely by soft harness, which contrastive testing has already proven degrades under pressure. The system-learning-loop model's gap→pattern→suggestion→improvement cycle is a specialized instance of knowledge physics, describing attractor formation from entropy without naming it. The physics generalize to any shared mutable substrate where amnesiac agents contribute: databases, config systems, APIs, documentation. Harness engineering is not code-specific — it's substrate governance.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for key outcomes.

---

## TLDR

Investigated whether knowledge exhibits the same physics as code (accretion, attractors, gates, entropy). The hypothesis holds: empirical measurement shows 85.5% orphan rate (accretion), models acting as attractors (34 probes to daemon model), zero hard knowledge gates, and the system-learning-loop already describing this physics without naming it. Recommends creating `.kb/models/knowledge-physics/model.md`.

---

## Delta (What Changed)

### Files Created
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` - Comprehensive probe testing knowledge physics hypothesis

### Files Modified
- `.kb/models/system-learning-loop/model.md` - Added Knowledge Physics Assessment section, updated Merged Probes table, added knowledge physics reframe to Summary

### Commits
- (pending)

---

## Evidence (What Was Observed)

### Accretion
- 85.5% orphan rate (997/1,166 investigations have no traceable model connection)
- Active investigations: 50% orphan rate. Archived: 93.6%.
- Quick entry duplication confirmed (kb-69d5cf / kb-9f3964 duplicate pair)
- 4 synthesis opportunity clusters (17 investigations that should be models)

### Attractor Effect
- daemon-autonomous-operation: 12.5% → 50% reference rate post-model, 34 probes (strongest attractor)
- harness-engineering: 2% → 100% reference rate post-model, 3 probes (launch burst)
- entropy-spiral: 12.5% → 8.8% post-model (capstone behavior — model settled topic)

### Missing Gates
- All knowledge transitions ungated: investigation→model, probe→model update, quick entry→decision, decision→implementation
- Pre-commit hooks only run on *.go files, not .kb/ files
- 4 "contradicts" verdicts sitting unmerged in probe files
- 1 of 56 decisions has an `kb agreements` check (1.8% enforcement rate)
- Prior Work table: 52% adoption rate (soft gate, not enforced)

### System-Learning-Loop Mapping
- gap→pattern→suggestion→improvement maps exactly to entropy→attractor→gate→reduction
- RecurrenceThreshold=3 is an attractor formation criterion
- The model describes knowledge physics in one domain (context gaps) without naming the general pattern

### Tests Run
```bash
kb reflect --type stale     # 66+ stale decisions identified
kb reflect --type synthesis # 4 synthesis clusters, 17 investigations
# Grep/count analysis across 1,166 investigation files
# Git log analysis for 3 models (creation dates + before/after reference rates)
```

---

## Architectural Choices

### Probe vs new model
- **What I chose:** Wrote probe for system-learning-loop model (extending it) rather than creating a new knowledge-physics model
- **What I rejected:** Creating `.kb/models/knowledge-physics/model.md` directly
- **Why:** The investigation skill requires probes when model claims are injected. A knowledge-physics model should be created as follow-up with this probe's evidence as the foundation.
- **Risk accepted:** Findings are currently spread across probe + model update rather than consolidated in a standalone model

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` - Knowledge physics hypothesis tested empirically

### Constraints Discovered
- Knowledge system has zero hard gates (all advisory) — the harness-engineering model's "every convention without a gate will eventually be violated" applies to knowledge itself
- Knowledge attractors work through attention priming (kb context injection), not structural coupling (unlike code attractors which work through imports/compilation)
- Three model behaviors exist: attractor (pulls work), capstone (settles topic), dormant (no engagement). Current system only describes attractors.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Create .kb/models/knowledge-physics/model.md
**Skill:** capture-knowledge or architect
**Context:**
```
This probe confirmed knowledge exhibits code-like physics. The evidence is in the
system-learning-loop probe (2026-03-09). A standalone knowledge-physics model should
synthesize: (1) accretion dynamics (85.5% orphan rate), (2) attractor taxonomy
(attractor/capstone/dormant), (3) gate deficit (zero hard knowledge gates),
(4) six proposed entropy metrics, (5) substrate generalization framework.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does adding hard gates to the knowledge system (e.g., commit-time contradiction checking) actually reduce entropy, or does it just add ceremony?
- Is the 85.5% orphan rate a problem to fix, or a natural property of exploratory investigation systems? (Not all code needs to be in a library.)
- Do capstone models represent healthy lifecycle completion, or is "settling a topic" premature closure?
- What would knowledge-level "pre-commit hooks" look like? (e.g., `kb validate` checking new .kb/ files against model claims before commit)

**Areas worth exploring further:**
- Cross-repo knowledge physics — does the orphan rate differ between orch-go and opencode?
- Knowledge attractor half-life — how long does a model sustain attraction before going dormant?
- Probe-as-gate — could probes be required (not optional) for model claims older than N days?

**What remains unclear:**
- Whether attention-primed attractors (knowledge) are fundamentally weaker than structurally-coupled attractors (code), or just ungated
- The right threshold for "too many claims per model" (the knowledge equivalent of lines-per-file bloat)

---

## Friction

Friction: none — kb data was accessible and measurable, commands worked as expected.

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-context-dylan-built-09mar-f84a/`
**Probe:** `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md`
**Beads:** `bd show orch-go-8m7w9`
