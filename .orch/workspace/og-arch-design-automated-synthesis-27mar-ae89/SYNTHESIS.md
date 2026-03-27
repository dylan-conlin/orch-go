# Session Synthesis

**Agent:** og-arch-design-automated-synthesis-27mar-ae89
**Issue:** orch-go-rhwly
**Duration:** 2026-03-27T07:33 → 2026-03-27T08:15
**Outcome:** success

---

## TLDR

Designed a synthesis quality signal system that ranks agent-produced briefs by composition quality (evidence, model connection, insight, open questions) using 6 mechanically-detectable signals computed at completion time and embedded in brief YAML frontmatter. This enables comprehension queue ordering by quality without adding new infrastructure or requiring LLM judgment.

---

## Plain-Language Summary

When multiple agents answer the same question, Dylan currently reads all their summaries and picks the useful one. This design automates the filtering step: at completion time, each agent's SYNTHESIS.md gets scored on 6 signals — does it cite evidence? does it connect to existing models? does it have causal reasoning? does it surface open questions? — and the scores get baked into the brief as metadata. The comprehension queue then sorts higher-scored briefs first. Dylan still makes the final call, but sees the best work first instead of reading chronologically.

The key design choice: boolean signals with a simple count, not weighted scores. The knowledge accretion model's own warning about "formula-shaped sentences" applies — numeric scores without calibrated weights pretend precision that doesn't exist. The HyperAgents research confirmed this: their self-modified selection mechanism didn't beat the handcrafted one.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root for implementation acceptance criteria.

**Key outcomes:**
- Investigation: `.kb/investigations/2026-03-27-design-automated-synthesis-ranking.md`
- Probe: `.kb/models/knowledge-accretion/probes/2026-03-27-probe-selection-pressure-via-quality-signals.md`
- Model updated: knowledge-accretion effectiveness hierarchy extended with "attention routing"
- 1 follow-up implementation issue recommended

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-27-design-automated-synthesis-ranking.md` — Full architect investigation with 5 forks navigated, implementation plan, defect class analysis
- `.kb/models/knowledge-accretion/probes/2026-03-27-probe-selection-pressure-via-quality-signals.md` — Probe testing whether quality signals fit the effectiveness hierarchy

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Merged probe findings: "attention routing" added to effectiveness hierarchy, handcrafted-beats-adaptive principle, probe evidence reference

---

## Evidence (What Was Observed)

- `verify.ParseSynthesis()` already extracts all fields needed for quality signal computation (TLDR, Delta, Evidence, Knowledge, Next, UnexploredQuestions)
- `debrief.CheckQuality()` already implements connective language detection (17 phrases) and action-verb detection (20 prefixes) — reusable for synthesis quality
- `complete_brief.go:buildBriefFromSynthesis()` is the natural injection point for quality metadata
- `comprehension_queue.go` sorts by mod-time only — signal-aware ordering is additive
- Brief feedback mechanism (shallow/good) exists in comprehension_queue.go but has zero consumers — future calibration data source
- Ranking/attention layer boundary probe (2026-03-26) already identified Layer 2 (method-expressing ordering) as missing and correctly classified it as "should be open" — this design implements Layer 2

---

## Architectural Choices

### Signals, Not Scores
- **Chose:** Boolean/count signal list with `signal_count` as sort key
- **Rejected:** Weighted numeric score (0-100)
- **Why:** Knowledge accretion model warns about "formula-shaped sentences." HyperAgents' handcrafted selection outperformed self-modified.
- **Risk:** Equal-weight signals may underweight evidence_specificity vs structural_completeness

### Completion-Time Scoring
- **Chose:** Compute at `orch complete`, store in brief YAML frontmatter
- **Rejected:** Dynamic scoring at serve time
- **Why:** SYNTHESIS.md is immutable post-completion. No-local-state constraint.
- **Risk:** Stale scores if signal definitions improve. Mitigation: batch rescore command.

### Extend Existing, No New Package
- **Chose:** ~150 lines across 3 existing files (verify, complete_brief, comprehension_queue)
- **Rejected:** New `pkg/ranking/` package
- **Why:** Too little code for a separate package. Easy to extract later if it grows.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-27-design-automated-synthesis-ranking.md` — Full design investigation
- `.kb/models/knowledge-accretion/probes/2026-03-27-probe-selection-pressure-via-quality-signals.md` — Probe extending effectiveness hierarchy

### Decisions Made
- Quality signals should be handcrafted (6 enumerated), not adaptive, because HyperAgents evidence shows self-modified selection doesn't outperform handcrafted
- Score representation should be signal list + count, not weighted numeric, because calibration data doesn't exist yet
- Implementation is 3 extensions to existing code, not a new subsystem

### Constraints Discovered
- Ranking/attention probe already designed Layer 2 ordering and identified brief→thread reverse-lookup gap — this design enables that work but doesn't duplicate it
- "Synthesis is strategic orchestrator work" (decision 2026-01-07) constrains this to ranking/filtering, never auto-selection

---

## Next (What Should Happen)

**Recommendation:** close

### Follow-up Work
- Create feature-impl issue: "Implement synthesis quality signals with brief metadata injection" covering pkg/verify/synthesis_quality.go, complete_brief.go metadata injection, and comprehension_queue.go signal-aware ordering

### Blocking Question (for orchestrator)
- Should signal-aware ordering apply only to comprehension queue, or also to `orch review` batch listing? Recommend starting with comprehension queue only.

```
MIGRATION_STATUS:
  designed: synthesis quality signal taxonomy (6 signals), brief metadata injection, signal-aware comprehension ordering
  implemented: none
  deployed: none
  remaining: feature-impl issue for 3 components
```

---

## Unexplored Questions

**Questions that emerged during this session:**
- Once brief feedback (shallow/good) accumulates, can signal→feedback correlation calibrate weights? The HyperAgents finding predicts adaptive won't beat handcrafted, but this is testable.
- Should agents be aware of their quality signals? If agents could see "your last brief scored 2/6 on quality signals," would that improve composition quality? Or would it create Goodhart optimization?
- The every-spawn-composes-knowledge thread identified "Tension sections are orphaned knowledge seeds." Quality signals detect tension quality but don't route tension content to threads. This is a separate knowledge-routing feature.

**What remains unclear:**
- Whether 6 signals is the right granularity. Could be too many (most fire for all good briefs) or too few (missing important signals like "addressed original question").
- How signal_count distribution looks across actual briefs. If 80% score 5-6/6, the sort key has no discriminating power. Need empirical measurement.

---

## Friction

Friction: none — existing infrastructure (synthesis parser, quality checks, brief generation) was well-documented and easy to understand. The ranking/attention probe had already done much of the analysis needed.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-automated-synthesis-27mar-ae89/`
**Investigation:** `.kb/investigations/2026-03-27-design-automated-synthesis-ranking.md`
**Beads:** `bd show orch-go-rhwly`
