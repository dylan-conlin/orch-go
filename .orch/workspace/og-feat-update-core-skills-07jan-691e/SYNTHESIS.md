# Session Synthesis

**Agent:** og-feat-update-core-skills-07jan-691e
**Issue:** orch-go-y0vvg
**Duration:** 2026-01-07 → 2026-01-07
**Outcome:** success

---

## TLDR

Updated 8 files across 3 core skills (feature-impl, architect, design-session) to use OpenCode's `question` tool instead of the non-existent `AskUserQuestion` tool, addressing the recurring "ask command inline" friction (8x gap).

---

## Delta (What Changed)

### Files Modified

**feature-impl skill:**
- `skills/src/worker/feature-impl/.skillc/phases/clarifying-questions.md` - Replaced AskUserQuestion with question tool, added JSON interface docs
- `skills/src/worker/feature-impl/reference/phase-clarifying-questions.md` - Same updates (deployed version)

**architect skill:**
- `skills/src/worker/architect/.skillc/SKILL.md.template` - Updated tool references and added interface docs
- `skills/src/worker/architect/.skillc/skill.yaml` - Changed `AskUserQuestion` to `question` in allowed-tools
- `skills/src/worker/architect/SKILL.md` - Updated deployed version with correct tool name

**design-session skill:**
- `skills/src/worker/design-session/.skillc/SKILL.md.template` - Updated tool references and added interface docs
- `skills/src/worker/design-session/.skillc/skill.yaml` - Changed `AskUserQuestion` to `question` in allowed-tools
- `skills/src/worker/design-session/SKILL.md` - Updated deployed version with correct tool name

### Commits
- (pending) - feat: update core skills to use opencode question tool

---

## Evidence (What Was Observed)

- **Tool name mismatch:** Skills referenced `AskUserQuestion` but OpenCode provides `question` (source: `opencode/src/tool/question.ts:6`)
- **Tool interface:** JSON-based with questions array, each containing question/header/options (source: `opencode/src/question/index.ts:21-30`)
- **Tool availability:** Question tool is enabled by default for all sessions including spawned agents (source: `registry.ts:96` checks `OPENCODE_CLIENT === "cli"` which defaults to `"cli"`)
- **7 files found with AskUserQuestion references** - All updated and verified clean

### Tests Run
```bash
# Verify all references replaced
grep -rn "AskUserQuestion" /Users/dylanconlin/orch-knowledge/skills/src/
# (empty - no matches)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-update-core-skills-opencode-ask.md` - Full investigation documenting findings and changes

### Decisions Made
- **Use JSON interface examples:** The question tool uses JSON, not YAML-like format. Updated all examples to show correct JSON structure.
- **Preserve directive-guidance pattern:** Kept recommendation about putting recommended option first with "(Recommended)" suffix.
- **Add interface documentation inline:** Rather than just renaming, added full interface docs in each skill for clarity.

### Constraints Discovered
- **Header max 12 chars:** The question tool's `header` field has a 12 character limit
- **Users can always select "Other":** Built-in escape hatch for custom responses

### Externalized via `kn`
- `kn decide "Skills use question tool for inline asking" --reason "AskUserQuestion doesn't exist, opencode provides question tool with JSON interface"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (8 files updated)
- [x] All AskUserQuestion references removed (verified via grep)
- [x] Investigation file has Complete status
- [ ] Commit changes
- [ ] Ready for `orch complete orch-go-y0vvg`

### Follow-up for Orchestrator
After closing, run `skillc deploy` in orch-knowledge to compile and deploy the updated skills.

---

## Unexplored Questions

**Questions that emerged during this session:**
- Does the question tool work correctly in headless/spawned mode? (Would need integration test)
- Should there be a centralized "tool interface reference" document for all available tools?

**Areas worth exploring further:**
- The 8x gap was identified by `orch learn` - verifying the fix reduces this metric

**What remains unclear:**
- Whether `OPENCODE_CLIENT` environment variable is set differently in spawn environment

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude Opus
**Workspace:** `.orch/workspace/og-feat-update-core-skills-07jan-691e/`
**Investigation:** `.kb/investigations/2026-01-07-inv-update-core-skills-opencode-ask.md`
**Beads:** `bd show orch-go-y0vvg`
