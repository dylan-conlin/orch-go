# Session Synthesis

**Agent:** og-inv-why-do-investigation-25dec
**Issue:** orch-go-ya22
**Duration:** 2025-12-25 12:10 → 2025-12-25 12:40
**Outcome:** success

---

## TLDR

Investigated why investigation files get poorly named like `inv-epic-question-how-do-we.md`. Root cause: filename slug is generated at spawn time from the task description by extracting first 5 non-stop-words, before the agent knows what they'll discover. Fix: add naming guidance to skills and SPAWN_CONTEXT.md.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-inv-investigation-file-naming-quality.md` - Complete investigation with root cause analysis

### Files Modified
- None

### Commits
- (pending) Investigation file documenting root cause and fix recommendations

---

## Evidence (What Was Observed)

- `generateSlug(cfg.Task, 5)` at `pkg/spawn/context.go:224` extracts first 5 non-stop-words from task
- Slug is embedded in SPAWN_CONTEXT.md at line 86 as the default `kb create investigation` argument
- `kb create` uses the provided slug directly without transformation (`kb-cli/cmd/kb/create.go:635`)
- Bad example `inv-epic-question-how-do-we.md` from task "Epic question: how do we evolve skills..."
- Good example `inv-investigate-orchestration-lifecycle-end-end.md` had descriptive task string
- Neither investigation nor design-session skills provide naming guidance

### Tests Run
```bash
# Code path traced from spawn to file creation
# Verified slug generation algorithm in pkg/spawn/config.go:147-175
# Confirmed kb create usage in kb-cli/cmd/kb/create.go:635
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-investigation-file-naming-quality.md` - Documents root cause and fix

### Decisions Made
- Filename quality depends on task description, not agent judgment - guidance needed
- Minimal fix is documentation change (add naming guidance to skills/SPAWN_CONTEXT.md)

### Constraints Discovered
- Slug is generated BEFORE agent investigates - can't capture findings in initial filename
- No post-creation rename mechanism exists in the spawn workflow

### Externalized via `kn`
- `kn constrain "Investigation filenames are generated from task description at spawn time" --reason "generateSlug() runs before agent starts, cannot reflect findings"`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Update SPAWN_CONTEXT.md template with naming guidance
**Skill:** feature-impl
**Context:**
```
Update pkg/spawn/context.go lines 86-92 to include guidance for agents to choose descriptive slugs based on FINDINGS not task description. Add examples: Good: "completion-loop-five-breakpoints", Bad: "investigate-thing".
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch complete` validate investigation filename quality?
- Should files be auto-renamed based on D.E.K.N. summary?
- Should the investigation skill self-review include filename check?

**Areas worth exploring further:**
- Git history implications of renaming investigation files
- Pattern recognition to detect truncated mid-sentence filenames

**What remains unclear:**
- How often orchestrators currently craft tasks with naming in mind
- Whether agents would follow naming guidance if provided

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-inv-why-do-investigation-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-investigation-file-naming-quality.md`
**Beads:** `bd show orch-go-ya22`
