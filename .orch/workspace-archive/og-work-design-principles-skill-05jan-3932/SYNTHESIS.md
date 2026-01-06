# Session Synthesis

**Agent:** og-work-design-principles-skill-05jan-3932
**Issue:** ad-hoc (--no-track)
**Duration:** 2026-01-05 18:30 → 2026-01-05 19:15
**Outcome:** success

---

## TLDR

Design-principles skill should be integrated as a standalone shared/policy skill with project-config-based loading via opencode.json, not merged into feature-impl. This provides conditional loading for UI-heavy projects while keeping context costs low for non-UI work.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-05-inv-design-principles-skill-integration-skill.md` - Full design session investigation answering the 5 integration questions

### Files Modified
- None (design session, no implementation)

### Commits
- None (investigation/design only)

---

## Evidence (What Was Observed)

- Design-principles skill is 238 lines (~1-2K tokens) - smaller than most skills (verified: wc -l)
- Orchestrator skill is 1395 lines (policy type), feature-impl is 434 lines (procedure type) (verified: wc -l)
- Skill structure has no phases/workflow - pure principles/guidance - matches policy skill pattern (verified: read full skill)
- ui-mockup-generation (301 lines) focuses on tooling (Gemini mockups), not design principles - complementary not conflicting (verified: read skill)
- skillc source structure exists at `~/orch-knowledge/skills/src/shared/` with existing shared skills (delegating-to-team, issue-quality, worker-base)

### Context Gathered
```bash
# Skill line counts
wc -l ~/.claude/skills/worker/feature-impl/SKILL.md ~/.claude/skills/meta/orchestrator/SKILL.md
# 434, 1395 lines respectively

# Shared skills structure
ls ~/orch-knowledge/skills/src/shared/
# delegating-to-team, issue-quality, worker-base
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-inv-design-principles-skill-integration-skill.md` - Complete design session analysis

### Decisions Made
1. **Standalone skill (not merge)** because feature-impl is already 434 lines with 7 phases, and design principles apply to non-feature-impl work too (investigations, audits, bug fixes touching UI)
2. **Project-config loading** because it follows "surfacing over browsing" principle - UI-heavy projects declare needs, all spawns get context automatically
3. **Policy skill type** because the skill has no phases/workflow - it's guidance that augments procedural skills
4. **Extension pattern for preferences** because Dylan may have different aesthetic preferences than the upstream skill defaults

### Constraints Discovered
- opencode.json skill injection feature doesn't exist yet - needs implementation
- Skill references Phosphor Icons specifically - may need note about alternatives in preferences
- Multiple skill loading order matters - design-principles should load after feature-impl (augments, doesn't replace)

### Key Integration Questions Answered

| Question | Answer |
|----------|--------|
| Trigger/loading mechanism | Project config via `opencode.json` `"skills": ["design-principles"]` |
| Standalone vs merge | Standalone shared/policy skill (not merge into feature-impl) |
| Context window cost | Acceptable - 238 lines is small, value per line is high |
| Personal preferences | Extension pattern - base skill + `design-preferences.md` override file |
| ui-mockup-generation interaction | Complementary - mockups for exploration, principles for implementation |

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

This design session produced clear recommendations. Implementation requires an epic with 3 tasks:

**Task 1: Create skillc source for design-principles**
- Create `orch-knowledge/skills/src/shared/design-principles/.skillc/`
- Add skill.yaml with `skill-type: policy`
- Copy upstream skill.md content to SKILL.md.template
- Deploy via `skillc build && skillc deploy`

**Task 2: Implement opencode.json skill injection**
- Design: `"skills": ["design-principles"]` in project's opencode.json
- Effect: Listed skills are injected into SPAWN_CONTEXT.md for all spawns in that project
- Implementation location: orch-go spawn.go (reads opencode.json, injects skill content)

**Task 3: Create design-preferences extension pattern**
- Create `~/.claude/design-preferences.md` as example personal preferences file
- Document in skill: "Override specific choices via preferences file"
- Consider: Should this be a second skill load or inline extension?

**Epic:** "Integrate design-principles skill with project-config loading"
**Skills needed:** feature-impl for tasks 1 and 2, architect for task 2 (new opencode.json feature)

### Context for Implementation Agent
```
Prior investigation: .kb/investigations/2026-01-05-design-claude-design-skill-evaluation.md
Design session: .kb/investigations/2026-01-05-inv-design-principles-skill-integration-skill.md
Upstream skill: /Users/dylanconlin/Documents/personal/claude-design-skill/skill/skill.md
Pattern to follow: ~/.claude/skills/meta/orchestrator/SKILL.md (policy skill structure)
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How should skill loading order be controlled when multiple skills are injected? (may matter for override behavior)
- Should opencode.json skill injection be in orch-go or opencode itself? (orch owns spawn context, but opencode owns config)
- Can/should preferences file be loaded as a second skill rather than custom extension mechanism?

**Areas worth exploring further:**
- Testing the actual improvement in agent UI output with design-principles loaded (before/after comparison)
- Whether other policy skills should be loadable via project config (audit-principles, security-guidelines, etc.)

**What remains unclear:**
- Exact implementation of opencode.json skill injection (needs design spike in task 2)

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-design-principles-skill-05jan-3932/`
**Investigation:** `.kb/investigations/2026-01-05-inv-design-principles-skill-integration-skill.md`
**Beads:** ad-hoc (--no-track)
