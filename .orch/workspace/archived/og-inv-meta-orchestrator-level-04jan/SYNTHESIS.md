# Session Synthesis

**Agent:** og-inv-meta-orchestrator-level-04jan
**Issue:** orch-go-xdr7
**Duration:** 2026-01-04 17:45 → 2026-01-04 18:45
**Outcome:** success

---

## TLDR

Investigated why spawned meta-orchestrators collapse to worker behavior. Root cause: ORCHESTRATOR_CONTEXT.md template uses task-completion framing ("work toward goal") that overrides skill guidance, causing agents to do work instead of managing sessions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md` - Full investigation with D.E.K.N. summary, 5 findings, and implementation recommendations

### Files Modified
- None (investigation-only session)

### Commits
- `944c593f` - investigation: meta-orchestrator level collapse analysis - framing root cause

---

## Evidence (What Was Observed)

- **Template framing is task-oriented**: `pkg/spawn/orchestrator_context.go:19-116` uses "Session Goal:", "Begin working toward your session goal", and "When you've accomplished your session goal" - all task-completion language
- **Skill content was comprehensive but ignored**: Session transcript (`session-ses_4743.md`) shows agent received full meta-orchestrator skill but immediately started reading files and writing synthesis
- **Agent required external prompting to recognize level violation**: Lines 355-425 of transcript show Dylan had to ask "what is your role?" before agent self-diagnosed
- **No template differentiation**: Both orchestrator and meta-orchestrator use identical `ORCHESTRATOR_CONTEXT.md` template
- **Frame shift decision is documented**: `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` confirms meta-orchestrators should be interactive, not task-completing

### Tests Run
```bash
# Verified template source
grep -r "ORCHESTRATOR_CONTEXT" /Users/dylanconlin/Documents/personal/orch-go
# Found: single template used for all policy-type skills

# Verified skill content
cat ~/.claude/skills/meta/meta-orchestrator/SKILL.md | head -20
# Confirmed: skill-type: policy, comprehensive guardrails present
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md` - Root cause analysis with implementation recommendations

### Decisions Made
- Framing is root cause, not skill content - Template framing sets behavioral mode before skill processing
- Need tiered context templates - META_ORCHESTRATOR_CONTEXT.md separate from ORCHESTRATOR_CONTEXT.md
- Interactive framing for meta-orchestrators - "Manage sessions" not "work toward goal"

### Constraints Discovered
- Framing trumps skill content - Even comprehensive skill guidance is overridden by opening context framing
- Level detection needs skill-name check - Current logic only checks skill-type, not skill-name

### Externalized via `kn`
- (Pending: should externalize the framing-trumps-content insight)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (implementation needed)

### If Spawn Follow-up
**Issue:** Create tiered context template for meta-orchestrators
**Skill:** feature-impl
**Context:**
```
Create META_ORCHESTRATOR_CONTEXT.md template with interactive framing. Key changes:
1. "You are managing orchestrator sessions" not "work toward goal"
2. First action: "Check orch status for sessions to complete or review"
3. No SESSION_HANDOFF.md requirement (stay interactive)
Add skill-name detection in spawn logic to use this template for "meta-orchestrator".
Reference: .kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should meta-orchestrators be spawnable at all, or always human-in-the-loop?
- What's the right completion pattern for meta-orchestrators if they stay interactive?
- Does the meta-orchestrator skill need updating beyond the template?

**Areas worth exploring further:**
- Testing the fix with an actual spawned meta-orchestrator session
- Whether there's a deeper architectural pattern here about context framing vs skill content

**What remains unclear:**
- Whether a template change alone is sufficient or if stronger guardrails are needed
- How to validate success (no automated test for "agent stayed at right level")

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-inv-meta-orchestrator-level-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md`
**Beads:** `bd show orch-go-xdr7`
