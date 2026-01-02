# Session Synthesis

**Agent:** og-work-session-end-workflow-26dec
**Issue:** (ad-hoc spawn, no tracking)
**Duration:** 2025-12-26
**Outcome:** success

---

## TLDR

Investigated the gap in orchestrator session-end workflow: workers have "Leave it Better" as a gated phase, but orchestrators have no equivalent. Recommend adding "Session Reflection" section to orchestrator skill with three checkpoints (friction audit, gap capture, system reaction check) before the existing session-transition/cleanup.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md` - Full investigation with findings, synthesis, and implementation recommendation

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- Worker skills (feature-impl, investigation, codebase-audit) have mandatory "Leave it Better" phase at `skills/src/worker/*/SKILL.md`
- Session-transition skill (`skills/shared/session-transition/SKILL.md`) focuses on git status, context capture, cleanup - not reflection
- Orchestrator skill (`skills/meta/orchestrator/SKILL.md`) has no session-end reflection section despite having extensive completion guidance
- Global CLAUDE.md has "Reflection checkpoint" at line 45-51 but it's not gated and not orchestrator-specific
- `orch learn` infrastructure exists for gap tracking but no trigger at session-end

### Tests Run
```bash
# Knowledge context search
kb context "session-end landing plane reflection"
# Result: No context found

# Grep for "Leave it Better" pattern
grep -r "Leave it Better" ~/.claude/skills
# Result: Found in worker skills, not in orchestrator skill
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md` - Investigation with implementation recommendation

### Decisions Made
- Investigation output (not epic): Scope is clear but needs Dylan's decision on whether to accept recommendation
- Two-phase model: Session Reflection (new, reflection) → Landing the Plane (existing, cleanup)

### Constraints Discovered
- Session-transition is shared between workers and orchestrators - adding orchestrator-specific reflection there would pollute worker context
- orch learn requires explicit invocation - no automatic session-end trigger exists

### Externalized via `kn`
- (none yet - recommend orchestrator runs `kn decide` on whether to implement)

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** Should we add "Session Reflection" section to orchestrator skill?

**Options:**
1. **Add section to orchestrator skill** - Mirrors "Leave it Better" for workers, uses existing tools (orch learn, kn), minimal new code
   - Pros: Aligns with existing patterns, progressive (can add hook enforcement later)
   - Cons: Relies on orchestrator following skill, not automatically enforced

2. **Create SessionEnd hook for enforcement** - Automatically runs reflection checklist at session end
   - Pros: Can't be skipped
   - Cons: May be too heavy, hooks run unconditionally, requires new code

3. **Expand session-transition skill** - Add reflection before cleanup phase
   - Pros: Keeps all session-end logic together
   - Cons: session-transition is shared (worker/orchestrator); would pollute worker context

**Recommendation:** Option 1 (add section to orchestrator skill). Start with guidance, add hook enforcement later if needed. This mirrors how "Leave it Better" works for workers.

**Proposed Section Content:**
```markdown
## Session Reflection (Before Ending)

**When to use:** Before ending an orchestrator session.

**The Three Checkpoints:**

1. **Friction Audit:** What was harder than it should have been?
   - Run `orch learn` to see recurring gaps

2. **Gap Capture:** What knowledge should have been surfaced but wasn't?
   - Use `kn decide/tried/constrain/question` to externalize

3. **System Reaction Check:** Does this session suggest system improvements?
   - New skill/hook/CLAUDE.md update needed?

**Gate:** Run at least one of: `orch learn`, any `kn` command, or explicit skip.

Then proceed to session-transition for git/cleanup.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is Dylan's actual manual reflection prompt? (Would validate the three checkpoints)
- Should `orch learn` have a `--session-end` mode that prompts for context?

**Areas worth exploring further:**
- SessionEnd hook capabilities in OpenCode (could enforce reflection later)
- Whether orch learn gap tracking is the right tool or something simpler is needed

**What remains unclear:**
- Whether hard gating (must run command) is appropriate for orchestrator flow
- What Dylan's actual workflow is when he prompts "same reflection every session"

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-session-end-workflow-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md`
**Beads:** (ad-hoc, no tracking)
