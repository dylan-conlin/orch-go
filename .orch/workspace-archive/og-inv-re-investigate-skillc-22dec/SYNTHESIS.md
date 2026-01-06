# Session Synthesis

**Agent:** og-inv-re-investigate-skillc-22dec
**Issue:** orch-go-vsjv
**Duration:** 2025-12-22
**Outcome:** success

---

## TLDR

Re-investigated whether skillc could replace `orch build skills` given skillc's decision doc lists SKILL.md as in-scope. Found the decision doc over-promised: skillc's concatenation model cannot handle the template expansion (`<!-- SKILL-TEMPLATE: X -->` markers) that orch-knowledge skills require.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-re-investigate-skillc-vs-orch.md` - Investigation documenting the template expansion gap

### Files Modified
- None

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- Skillc decision doc (`skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md:11-17`) explicitly lists SKILL.md as in-scope artifact
- Orch-cli `skills_cli.py:180-194` implements template expansion via regex: `r'<!--\s*SKILL-TEMPLATE:\s*([a-zA-Z0-9_-]+)\s*-->'`
- Feature-impl template has 8 template markers (investigation, clarifying-questions, design, etc.) that get expanded from `src/phases/*.md` files
- Skillc `compiler.go:94-105` only does dependency-ordered concatenation - no template expansion capability
- Prior investigation was correct about functionality but missed the decision doc claim

### Tests Run
```bash
# Verified orch build skills detects templates
uv run python -c "from orch.skills_cli import cli; cli(['build', '--source', '/Users/dylanconlin/orch-knowledge/skills/src', '--check'])"
# Result: Found 2 templated skills needing rebuild

# Verified template markers exist
grep -n "SKILL-TEMPLATE" /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/src/SKILL.md.template
# Result: 8 template markers found
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-re-investigate-skillc-vs-orch.md` - Documents the template expansion gap between skillc and orch build skills

### Decisions Made
- Prior investigation conclusion confirmed: keep systems separate (skillc for project context, orch for skills)
- Recommendation: update skillc decision doc to either remove SKILL.md from scope or mark as "future" pending template expansion feature

### Constraints Discovered
- `kn-291e13`: skillc cannot compile SKILL.md templates without template expansion feature

### Externalized via `kn`
- `kn constrain "skillc cannot compile SKILL.md templates without template expansion feature" --reason "orch-knowledge skills use SKILL-TEMPLATE markers that require regex substitution not concatenation"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (investigation methodology followed)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-vsjv`

### Follow-up Issue (Optional)
**Issue:** Update skillc decision doc to clarify SKILL.md scope
**Skill:** feature-impl
**Context:**
```
The skillc decision doc (2025-12-21-skillc-artifact-scope.md) lists SKILL.md as in-scope but skillc lacks template expansion. Either remove SKILL.md from in-scope table, or add a note that it's aspirational pending template expansion feature.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Would it be valuable to add template expansion to skillc? (would need new manifest syntax, significant work)
- Should orch-go eventually port `orch build skills` functionality? (only if Python orch-cli deprecated)

**Areas worth exploring further:**
- Whether the dual-target deployment (Claude Code + OpenCode) could be simplified

**What remains unclear:**
- Original intent of decision doc - was SKILL.md aspirational or a mistake?

---

## Session Metadata

**Skill:** investigation
**Model:** claude
**Workspace:** `.orch/workspace/og-inv-re-investigate-skillc-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-re-investigate-skillc-vs-orch.md`
**Beads:** `bd show orch-go-vsjv`
