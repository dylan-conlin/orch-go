# Session Synthesis

**Agent:** og-inv-pre-spawn-kb-22dec
**Issue:** orch-go-gcf8
**Duration:** 2025-12-22 ~08:30 → ~09:45
**Outcome:** success

---

## TLDR

Investigated why pre-spawn `kb context` surfaces 1,200+ entries with 33% noise from irrelevant repos (price-watch, dotfiles). Root causes: (1) `--global` searches all 17 projects, (2) generic keyword extraction. Recommended tiered filtering: current project + orch ecosystem allowlist + per-category limits.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` - Full investigation with findings, evidence, and recommendations

### Files Modified
- None (investigation-only task)

### Commits
- Investigation file to be committed

---

## Evidence (What Was Observed)

- Query "pre" returned 1,210 entries from 17 repos (source: `kb context "pre" --global | wc -l`)
- 33% (400 entries) from irrelevant repos: price-watch, dotfiles, scs-slack, .doom.d
- `kb search` has `--project` flag but `kb context` doesn't (gap in kb-cli)
- `--limit 20` reduces to 65 entries but still includes noise from all repos
- More specific queries like "spawn kb context" return only 2 highly relevant results

### Tests Run
```bash
# Measure noise ratio
kb context "pre" --global 2>&1 | grep -E '^\- \[' | wc -l
# 1210

# Count relevant vs irrelevant
kb context "pre" --global 2>&1 | grep -E '^\- \[(orch-go|orch-cli|kb-cli|orch-knowledge)\]' | wc -l
# 792 (relevant)

kb context "pre" --global 2>&1 | grep -E '^\- \[(price-watch|dotfiles)\]' | wc -l  
# 315 (noise subset)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` - Full analysis of noise problem with implementation recommendations

### Decisions Made
- Decision: Filter to orch ecosystem repos because 33% of global results are noise from unrelated projects
- Decision: Use tiered approach (current project → orch ecosystem → global) rather than binary (local vs global)

### Constraints Discovered
- kb-cli `kb context` lacks `--project` flag that `kb search` has
- Keyword extraction in orch-go creates too-generic queries from task descriptions

### Externalized via `kn`
- `kn decide "Pre-spawn kb context should filter to orch ecosystem repos" --reason "33% of global results are noise from unrelated repos (price-watch, dotfiles). Filtering preserves cross-repo signal while eliminating noise."`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement tiered kb context filtering in orch-go
**Skill:** feature-impl
**Context:**
```
Investigation complete. Implement tiered filtering in runPreSpawnKBCheck():
1. Query current project first (no --global)
2. If sparse, add orch ecosystem repos: orch-go, orch-cli, kb-cli, orch-knowledge, beads, kn
3. Apply --limit 20 per category
4. Post-filter results to allowlist if using --global

See .kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md for full analysis.
```

**Follow-up Issue for kb-cli:**
**Issue:** Add --project flag to kb context command
**Skill:** feature-impl
**Context:**
```
The --project flag exists in kb search but not kb context. Port the implementation:
1. Add projectFilter string flag to context command
2. Pass to ContextOptions.Project field (already exists in struct)
3. GetContextGlobalWithProjects already filters by opts.Project

See kb-cli/cmd/kb/search.go lines 191-194 for the pattern.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the orch ecosystem allowlist be configurable via `.orch/config.yaml`?
- What's the right balance between precision (too narrow) and recall (too broad)?

**Areas worth exploring further:**
- Better keyword extraction: use multiple keywords, phrase detection, or domain-specific terms
- Relevance scoring: rank results by match quality, not just filter by project

**What remains unclear:**
- Whether kb-cli team will accept the --project flag addition
- Optimal allowlist composition (should beads-ui-svelte be included?)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus (via OpenCode)
**Workspace:** `.orch/workspace/og-inv-pre-spawn-kb-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md`
**Beads:** `bd show orch-go-gcf8`
