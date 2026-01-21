# Session Synthesis

**Agent:** og-arch-analyze-gpt-orchestrator-21jan-1d5a
**Issue:** ad-hoc (--no-track)
**Duration:** 2026-01-21 → 2026-01-21
**Outcome:** success

---

## TLDR

Analyzed GPT-5.2 orchestrator session (ses_4207) comparing behavioral patterns to Claude Opus 4.5. Found five critical anti-patterns making GPT unsuitable for orchestration: reactive gate handling, role boundary confusion, excessive deliberation, poor timeout recovery, and literal instruction interpretation without synthesis.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md` - Comprehensive GPT vs Opus behavioral analysis
- `.orch/workspace/og-arch-analyze-gpt-orchestrator-21jan-1d5a/SYNTHESIS.md` - This file

### Files Modified
- None

### Commits
- Pending commit with investigation and synthesis

---

## Evidence (What Was Observed)

- **3 spawn attempts required** for multi-gate scenario (lines 312-680): GPT hit `--bypass-triage` gate, fixed, then hit strategic-first gate, fixed by changing skill
- **Role boundary collapse** (lines 692-1357): After spawning architect agent, GPT immediately started debugging docker itself instead of delegating
- **6+ timeout failures** without strategy adaptation (lines 962-1200): Repeated identical docker commands despite all timing out at 120000ms
- **200+ second thinking blocks** revealing uncertainty (lines 789-823): "policy suggests we should delegate... I'll stick with using the spawned agent" followed immediately by direct debugging
- **Literal instruction processing** (lines 257-680): GPT processes error messages sequentially rather than synthesizing all requirements upfront

### Behavioral Pattern Summary Table

| Pattern | GPT-5.2 Behavior | Expected Opus Behavior |
|---------|-----------------|----------------------|
| Gate handling | Reactive (hit → fix → repeat) | Anticipatory (synthesize all flags) |
| Role boundaries | Collapses to worker mode | Maintains supervision boundary |
| Deliberation | Excessive, uncertainty-revealing | Confident, decision-focused |
| Failure recovery | Repeats same pattern | Adapts strategy |
| Instruction synthesis | Literal, sequential | Contextual, synthesized |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md` - Model selection analysis for orchestration

### Decisions Made
- Decision 1: GPT-5.2 unsuitable for orchestration role because behavioral patterns show structural deficits in gate anticipation, role modeling, and failure adaptation
- Decision 2: Recommend Claude Opus 4.5 as exclusive orchestrator model; GPT may be suitable for constrained worker tasks

### Constraints Discovered
- **Orchestrator requires gate anticipation** - Model must synthesize compound gating requirements from documentation, not learn them by hitting them
- **Orchestrator requires role boundary maintenance** - Spawning agent then doing its work defeats delegation architecture
- **Orchestrator requires failure adaptation** - Repeated identical failures without strategy change is unacceptable

### Externalized via `kn`
- Not applicable (investigation-only session)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] SYNTHESIS.md created
- [ ] Ready for orchestrator review

### Follow-up Recommendations

1. **Create decision document**: Formalize model selection constraint for orchestration
2. **Consider GPT for workers**: Investigate whether GPT performs acceptably in constrained worker roles with clear boundaries
3. **Track metrics**: If GPT is ever used for orchestration, track spawn retry rates and delegation boundary violations

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Would different GPT prompting strategies improve gate anticipation?
- Is GPT-5.2's behavior consistent across sessions or was this an outlier?
- Do other models (Gemini, etc.) show similar patterns?

**Areas worth exploring further:**
- GPT worker role suitability (constrained tasks with clear boundaries)
- Quantitative success rate comparison at scale (N>1)
- Whether explicit role-boundary reinforcement in prompts improves GPT behavior

**What remains unclear:**
- Root cause of GPT's role boundary collapse (training artifact? instruction weighting?)
- Whether OpenAI's future models will address these patterns
- Cost-quality tradeoff quantification for orchestration role

---

## Session Metadata

**Skill:** architect (analysis task)
**Model:** Claude Opus 4.5
**Workspace:** `.orch/workspace/og-arch-analyze-gpt-orchestrator-21jan-1d5a/`
**Investigation:** `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md`
**Beads:** N/A (ad-hoc)
