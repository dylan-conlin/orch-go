# Skill System Audit: Inventory, Deployment Formats, Loader Compatibility

**Date:** 2026-03-05
**Status:** Complete
**Issue:** orch-go-9zq3

## Summary

The skill system has accumulated 5 deployment formats across 3 eras. The loader (`pkg/skills/loader.go`) handles 2 of them correctly. 13 broken symlinks, 13 orphaned loose `.md` files, 1 nested `skills/src/` directory (stale), and 1 skill (`ux-audit`) completely unreachable by the loader.

---

## Skill Inventory

### Source Skills (skills/src/ — managed by skillc)

All use `.skillc/` format with `skill.yaml` metadata.

| Category | Skill | Type | Audience |
|----------|-------|------|----------|
| meta | diagnostic | policy | orchestrator |
| meta | meta-orchestrator | policy | meta-orchestrator |
| meta | orchestrator | policy | orchestrator |
| shared | worker-base | foundation | worker |
| worker | architect | procedure | worker |
| worker | codebase-audit | procedure | worker |
| worker | design-session | procedure | worker |
| worker | experiential-eval | procedure | worker |
| worker | experiment | procedure | worker |
| worker | feature-impl | procedure | worker |
| worker | head-to-head | procedure | worker |
| worker | investigation | procedure | worker |
| worker | research | procedure | worker |
| worker | systematic-debugging | procedure | worker |
| worker | ux-audit | procedure | worker |

**Total source skills: 15**

### Deployed-Only Skills (no source in skills/src/)

These skills exist only at `~/.claude/skills/{category}/{name}/SKILL.md` with no corresponding `.skillc` source. They use raw YAML frontmatter (no `skillc` build process).

| Category | Skill | Words |
|----------|-------|-------|
| meta | analyze-skill-usage | 1125 |
| meta | audit-claude-md | 1047 |
| meta | testing-skills-with-subagents | 641 |
| playwright-cli | playwright-cli | 778 |
| shared | capture-knowledge | 1403 |
| shared | code-review | 1316 |
| shared | record-decision | 1829 |
| utilities | testing-anti-patterns | 1634 |
| utilities | tmux-workspace-sync | 1469 |
| utilities | ui-mockup-generation | 1577 |
| utilities | workspace-isolation | 798 |
| worker | brainstorming | 279 |
| worker | hello | 105 |

**Total deployed-only: 13**

### Grand Total: 28 unique skills (15 in source + 13 deployed-only)

---

## Deployment Formats Found

### Format 1: `skillc deploy` (Current)
- **Path:** `~/.claude/skills/{category}/{name}/SKILL.md`
- **Source:** `skills/src/{category}/{name}/.skillc/`
- **Loader compatible:** YES (via subdirectory scan)
- **Skills using:** 15 (all source-managed skills)

### Format 2: Direct SKILL.md (Manual)
- **Path:** `~/.claude/skills/{category}/{name}/SKILL.md`
- **Source:** None (hand-written)
- **Loader compatible:** YES (same path structure as skillc deploy)
- **Skills using:** 13 (deployed-only skills)

### Format 3: Root Symlinks (Legacy, Jan 2026)
- **Path:** `~/.claude/skills/{name}` → `{category}/{name}/` or `src/worker/{name}/`
- **Loader compatible:** YES for working symlinks (resolves via direct path)
- **Status:** 13 BROKEN, 13 working (redundant with Format 1/2)

### Format 4: Loose `.md` Files (Orphaned, Mar 1 2026)
- **Path:** `~/.claude/skills/{name}.md` (NOT in a directory)
- **Loader compatible:** NO (loader expects `{name}/SKILL.md`, not `{name}.md`)
- **Skills affected:** 13 files (diagnostic, meta-orchestrator, worker-base, architect, codebase-audit, design-session, experiential-eval, feature-impl, head-to-head, investigation, research, systematic-debugging, ux-audit)

### Format 5: Nested `skills/src/` (Accidental)
- **Path:** `~/.claude/skills/skills/src/{category}/{name}/SKILL.md`
- **Loader compatible:** NO (3 levels deep, loader only scans 1 level)
- **Content:** All 15 source skills duplicated, ALL DIVERGED from correct path copies (except ux-audit which is only here)
- **Origin:** Likely accidental `skillc deploy` targeting or copy

---

## Issues Found

### CRITICAL: ux-audit Unreachable
`ux-audit` exists in source (`skills/src/worker/ux-audit/`) but is only deployed at the nested path `~/.claude/skills/skills/src/worker/ux-audit/SKILL.md`. No copy exists at `~/.claude/skills/worker/ux-audit/SKILL.md`. The loader cannot find it.

**Fix:** `skillc deploy` for ux-audit, or manually copy to correct path.

### HIGH: 13 Broken Symlinks
Old-style symlinks pointing to `src/worker/{name}` or deleted skills:
- `architect → src/worker/architect` (BROKEN — `src/` doesn't exist at deploy target)
- `codebase-audit → src/worker/codebase-audit`
- `design-session → src/worker/design-session`
- `feature-impl → src/worker/feature-impl`
- `investigation → src/worker/investigation`
- `research → src/worker/research`
- `systematic-debugging → src/worker/systematic-debugging`
- `delegating-to-team → shared/delegating-to-team` (skill doesn't exist)
- `issue-creation → src/worker/issue-creation` (skill doesn't exist)
- `issue-quality → shared/issue-quality` (skill doesn't exist)
- `kb-reflect → src/worker/kb-reflect` (skill doesn't exist)
- `reliability-testing → src/worker/reliability-testing` (skill doesn't exist)
- `ui-design-session → src/worker/ui-design-session` (skill doesn't exist)
- `writing-skills → meta/writing-skills` (skill doesn't exist)

**Impact:** Currently harmless — the loader finds skills via subdirectory scan (Format 1), so these broken symlinks just add noise. But they could cause confusion during debugging.

### MEDIUM: 13 Loose `.md` Files
Root-level markdown files dated Mar 1 (same day as source skills):
`~/.claude/skills/{name}.md` — these are NOT loadable by the skill loader since it expects `{name}/SKILL.md` directory structure. They appear to be a stale artifact of a previous build/deploy operation.

### MEDIUM: Nested `skills/src/` Directory
`~/.claude/skills/skills/src/` contains all 15 source skills but with DIVERGED content from the correct deployment path. This is stale data occupying space and causing confusion. The loader never reaches these (3 levels deep).

### LOW: Empty `policy/` Directory
`~/.claude/skills/policy/` exists but is empty.

### LOW: 13 Working Redundant Symlinks
Symlinks like `hello → worker/hello`, `orchestrator → meta/orchestrator` etc. These work but are redundant — the loader finds skills via subdirectory scan anyway. They cause the loader to sometimes find a skill twice (via direct symlink AND subdirectory scan — direct wins).

---

## Loader Analysis (`pkg/skills/loader.go`)

### Resolution Algorithm
1. **Direct:** `~/.claude/skills/{name}/SKILL.md`
2. **Subdirectory:** `~/.claude/skills/*/{name}/SKILL.md` (1 level)

### Features
- YAML frontmatter parsing for metadata
- Dependency resolution (prepends dependency content)
- Section filtering via `<!-- @section: key=value -->` markers (progressive disclosure)

### Gaps
- No recursive scan — skills nested >1 level deep are invisible
- No validation of deployed skill integrity
- No detection of duplicate skills (symlink + subdirectory)
- Silent failure on missing dependencies (no warning logged)

---

## Recommendations

### Immediate Cleanup (No Code Changes)
1. **Deploy ux-audit:** `cd skills/src/worker/ux-audit && skillc deploy`
2. **Remove broken symlinks:** `rm ~/.claude/skills/{architect,codebase-audit,...}` (13 links)
3. **Remove loose `.md` files:** `rm ~/.claude/skills/*.md` (13 files)
4. **Remove nested `skills/src/`:** `rm -rf ~/.claude/skills/skills/`
5. **Remove empty `policy/`:** `rmdir ~/.claude/skills/policy/`
6. **Remove working but redundant symlinks:** All 13 (skills are found via subdirectory scan)

### Follow-up (Architect Recommended)
- **Migrate 13 deployed-only skills to `.skillc` source** — currently unversioned, not managed by skillc
- **Add loader warning** for missing dependencies (currently silent)
- **Add `skillc doctor` check** for broken symlinks and stale artifacts

---

## Reference

- Loader: `pkg/skills/loader.go`
- Source skills: `skills/src/`
- Deploy target: `~/.claude/skills/`
- Skill guide: `.kb/guides/skill-system.md`
