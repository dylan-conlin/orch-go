# Session Synthesis

**Agent:** og-arch-deep-analysis-post-17jan-7444
**Issue:** orch-go-8ykkb
**Duration:** 2026-01-17 ~14:30 → ~16:30
**Outcome:** success

---

## TLDR

Deep analysis of orchestrator session ses_4325 identified 5 platform-level bugs that caused failure: Task tool confusion (agent used Task instead of orch spawn), unhelpful CLI error messages, orchestrator skill verbosity (1,193 lines with critical rules buried), model-skill mismatch (Flash for Opus-designed skill), and stale session counting. Recommends 3-phase fix: improve CLI errors, restructure skill template with critical rules in first 50 lines, add model-skill compatibility warnings.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-deep-analysis-post-mortem-orchestrator.md` - Complete post-mortem with 5 findings, synthesis, and implementation recommendations

### Files Modified
- None (investigation-only session)

### Commits
- (To be committed after session complete)

---

## Evidence (What Was Observed)

- **Session log analysis:** 15+ user corrections, explicit frustration ("WHAT ARE YOU DOING"), repeated Task tool usage despite correction
- **Orchestrator skill size:** 1,193 lines (verified via file read)
- **Delegation rule location:** Line 373 - buried too deep for smaller models
- **CLI error source:** `cmd/orch/spawn_cmd.go:160` uses `cobra.MinimumNArgs(2)` producing generic error
- **Concurrency threshold:** 10-minute threshold in `spawn_cmd.go:449` for active vs idle

### Tests Run
```bash
# Verified file locations and content
grep -n "ABSOLUTE DELEGATION" ~/.claude/skills/meta/orchestrator/SKILL.md
# Line 373

# Verified CLI argument validation
grep -n "MinimumNArgs" cmd/orch/spawn_cmd.go
# Line 160
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-deep-analysis-post-mortem-orchestrator.md` - Complete post-mortem

### Decisions Made
- Decision 1: Critical skill rules must appear in first 50 lines because smaller models fail to maintain attention to buried guidance
- Decision 2: CLI errors should include example usage because agents (and humans) need actionable help when struggling with syntax

### Constraints Discovered
- **Model-skill coupling:** Skills designed for Opus fail on Flash - the platform doesn't degrade gracefully
- **Guidance placement matters:** Rules at line 373 may as well not exist for smaller models
- **Task tool confusion is recurring:** This is not the first time orchestrators have used Task tool instead of orch spawn

### Externalized via `kn`
- (Recommend orchestrator externalize: `kb quick constrain "Critical skill rules in first 50 lines" --reason "Smaller models fail on buried guidance"`)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Improve CLI argument error messages
**Skill:** feature-impl
**Context:**
```
Add custom Args function to spawn_cmd.go that provides helpful error messages with example usage when arguments are wrong. Current generic "requires at least 2 arg(s)" provides no help.
```

**Issue:** Restructure orchestrator skill with critical rules header
**Skill:** feature-impl
**Context:**
```
Add "CRITICAL RULES" section to orchestrator skill template (orch-knowledge) with top 10 rules in first 50 lines. Include: use orch spawn NOT Task tool, delegation boundary, role detection. Note: requires skillc changes.
```

**Issue:** Add model-skill compatibility warning
**Skill:** feature-impl
**Context:**
```
Add skill metadata field `min-capability: opus|sonnet|any` and warn (not block) when spawning incompatible model-skill combination. Start with orchestrator skill requiring opus.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why did price-watch session use Flash instead of Opus for orchestration? (configuration or default issue)
- Are there other skills with similar verbosity problems? (skill audit needed)
- Is the 265 idle agents issue a separate bug or transient OpenCode state? (needs reproduction)

**Areas worth exploring further:**
- Could we add runtime detection of "Task tool for spawning" in orchestrator context and inject a warning?
- Should there be a "skill complexity score" that gates model selection?

**What remains unclear:**
- Whether Flash can ever work for orchestration even with restructured skill
- Whether the concurrency counting bug is edge case or common

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-deep-analysis-post-17jan-7444/`
**Investigation:** `.kb/investigations/2026-01-17-inv-deep-analysis-post-mortem-orchestrator.md`
**Beads:** `bd show orch-go-8ykkb`
