## Summary (D.E.K.N.)

**Delta:** `~/.claude/skills/` has 3 conflicting deployment formats from 3 time periods; 1 source skill (ux-audit) is completely unfindable by the loader, 14 broken symlinks and 13 flat .md files are debris from prior formats, and the loader's 2-level search depth cannot find any of them.

**Evidence:** Simulated `FindSkillPath()` for all 30+ skill names — ux-audit returns ErrSkillNotFound; 14 symlinks resolve to non-existent `src/worker/*` paths; 13 flat `.md` files are invisible to directory-based loader; 15 files in spurious `skills/src/` nested path also invisible.

**Knowledge:** The deployment directory accumulated 3 format layers (symlinks Jan 29, flat .md Mar 1, directories Mar 4) without cleanup. The loader only supports `skillName/SKILL.md` and `*/skillName/SKILL.md` patterns. Re-running `skillc deploy --target ~/.claude/skills/ --cleanup skills/src` from the project root would fix ux-audit and clean orphans, but worktrees containing `.skillc` dirs break `detectSkillSourceRoot()`.

**Next:** Two actions — (1) immediate: re-deploy with `--cleanup` from clean working tree to fix ux-audit; (2) architectural: fix `detectSkillSourceRoot()` to exclude `.claude/worktrees/` paths, then clean all debris.

**Authority:** architectural - Cross-component (skillc deploy + loader + worktree interaction), multiple valid approaches

---

# Investigation: Skill System Deployment Health After orch-go Merge

**Question:** After merging skill sources into orch-go, is every source skill findable by the loader? What deployment debris exists and what's broken?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** Worker agent (orch-go-9zq3)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

---

## Findings

### Finding 1: Three Conflicting Deployment Formats Coexist

**Evidence:** `~/.claude/skills/` contains three distinct deployment formats from three time periods:

| Format | Date | Count | Example | Loader-Findable |
|--------|------|-------|---------|-----------------|
| **Symlinks** | Jan 29 / Nov 2025 | 27 (14 broken, 13 working) | `architect -> src/worker/architect` | Only if target exists |
| **Flat .md files** | Mar 1 | 13 | `ux-audit.md`, `architect.md` | **NO** — loader checks `SKILL.md` in dirs only |
| **Directory entries** | Mar 4 | Correct set | `worker/architect/SKILL.md` | **YES** |
| **Spurious nested** | Feb 28 | 15 | `skills/src/worker/architect/SKILL.md` | **NO** — 3+ levels deep |

**Source:** `ls -la ~/.claude/skills/`, `find ~/.claude/skills -type f -o -type l`, broken symlink resolution test

**Significance:** The directory has never been cleaned between deployment format changes. Each `skillc deploy` run added a new layer without removing the previous format. Total debris: ~55 files/symlinks that serve no purpose.

---

### Finding 2: ux-audit is the Only Source Skill Not Findable by Loader

**Evidence:** Simulated `FindSkillPath()` for all 15 source skills:
- 14/15 source skills → **FOUND** (via `worker/*/SKILL.md` or `meta/*/SKILL.md`)
- **ux-audit → NOT FOUND**

ux-audit exists in three locations, all unfindable:
1. `~/.claude/skills/ux-audit.md` — flat file, loader ignores `.md` at root
2. `~/.claude/skills/skills/src/worker/ux-audit/SKILL.md` — too deep (3+ levels)
3. Does NOT exist at `~/.claude/skills/worker/ux-audit/SKILL.md` — the correct path

All other worker skills were deployed to `worker/*/SKILL.md` on Mar 4 12:27:31. ux-audit was NOT included in that deploy.

**Source:** Findability simulation script, `ls ~/.claude/skills/worker/`, file timestamps

**Significance:** This is the confirmed loader bug that triggered this investigation. The spawn system's `LoadSkillContent("ux-audit")` returns `ErrSkillNotFound`, preventing ux-audit spawns.

---

### Finding 3: Worktrees Break `detectSkillSourceRoot()` in skillc deploy

**Evidence:** Running `skillc deploy --target <dir>` from the orch-go project root:
- Found 30 `.skillc` directories (15 from `skills/src/` + 15 from `.claude/worktrees/agent-a7d640a3/skills/src/`)
- `detectSkillSourceRoot()` goes 3 levels up from each `.skillc` and requires all candidates to agree
- Two different roots: `skills/src/` vs `.claude/worktrees/agent-a7d640a3/skills/src/`
- **Disagreement → fallback to project root → all skills deploy to `skills/src/category/skillName/` instead of `category/skillName/`**

This is likely how the Feb 28 `skills/src/` nested path was created, and why ux-audit was missing from the Mar 4 deploy (if worktrees existed at that time).

**Source:** `skillc deploy --target /tmp/test` output showing 30 dirs and `skills/src/` output paths; `deploy.go:515-545` (`detectSkillSourceRoot()`)

**Significance:** Any `skillc deploy` from project root while worktrees exist produces incorrect paths. This is a recurring bug — every agent session creates a worktree copy of `skills/src/`.

---

### Finding 4: 14 Broken Symlinks Point to Non-Existent `src/worker/*` Paths

**Evidence:** 14 symlinks at `~/.claude/skills/` root point to `src/worker/*` paths that don't exist:
- `architect -> src/worker/architect`
- `codebase-audit -> src/worker/codebase-audit`
- `design-session -> src/worker/design-session`
- `feature-impl -> src/worker/feature-impl`
- `investigation -> src/worker/investigation`
- `research -> src/worker/research`
- `systematic-debugging -> src/worker/systematic-debugging`
- `delegating-to-team -> shared/delegating-to-team`
- `issue-creation -> src/worker/issue-creation`
- `issue-quality -> shared/issue-quality`
- `kb-reflect -> src/worker/kb-reflect`
- `reliability-testing -> src/worker/reliability-testing`
- `ui-design-session -> src/worker/ui-design-session`
- `writing-skills -> meta/writing-skills`

The first 7 are harmless (skills are findable via `worker/*/SKILL.md`). The last 7 are for legacy skills NOT in current source and NOT deployed elsewhere — they are completely unfindable.

**Source:** Symlink resolution test script

**Significance:** 7 legacy skills (delegating-to-team, issue-creation, issue-quality, kb-reflect, reliability-testing, ui-design-session, writing-skills) exist only as broken symlinks. If any spawn or reference attempts to load them, it silently fails.

---

### Finding 5: Flat .md Files Are From `--agent-dir` Deploy (OpenCode Format)

**Evidence:** The 13 flat `.md` files at `~/.claude/skills/` root (e.g., `architect.md`, `ux-audit.md`) have YAML frontmatter with `mode: subagent` — this is the OpenCode agent file format produced by `skillc deploy --agent-dir`.

These were either:
1. Deployed to `~/.claude/skills/` as both `--target` and `--agent-dir` simultaneously
2. Or deployed as `--agent-dir ~/.claude/skills/` (treating the skills dir as the agent dir)

The loader never checks for flat `.md` files — it only looks for `SKILL.md` inside directories.

**Source:** `head -5` on each flat file showing YAML frontmatter; `deploy.go:673-724` (`deployAgentFile()`)

**Significance:** These files are dead weight. They take space but are never loaded by the orch-go loader. They could confuse `skillc deploy --cleanup` if the orphan detector doesn't account for them.

---

## Synthesis

**Key Insights:**

1. **Format evolution without cleanup** — The skill deployment went through 3 format generations (symlinks → flat files → directories) but no cleanup was run between transitions. Each new deploy added files without removing the old format. `skillc deploy --cleanup` exists but was never used.

2. **ux-audit is the canary** — Only 1 of 15 source skills is broken, suggesting the Mar 4 directory deploy worked correctly for 14 skills. ux-audit's absence is likely due to the worktree interference bug (`detectSkillSourceRoot()` disagreement) during that specific deploy run.

3. **Worktrees are a deployment hazard** — Every spawned agent creates a `.claude/worktrees/` directory containing a full copy of `skills/src/`. If `skillc deploy` runs from project root while any worktree exists, `detectSkillSourceRoot()` falls back to the wrong root, producing nested `skills/src/` paths.

**Answer to Investigation Question:**

The skill system has one critical breakage (ux-audit unfindable) and significant deployment debris (14 broken symlinks, 13 orphan flat files, 15 nested-path duplicates). The root cause is format evolution without cleanup plus a worktree interaction bug in skillc. A clean re-deploy with `--cleanup` from the correct source path would fix the immediate issue, but the worktree bug needs a fix in skillc.

---

## Structured Uncertainty

**What's tested:**

- ✅ Loader findability for all 30+ skill names (simulated FindSkillPath algorithm)
- ✅ ux-audit compiles successfully with `skillc build` (24,538 tokens, 70.1% budget)
- ✅ ux-audit deploys successfully with `skillc deploy` when given skill-specific path
- ✅ Worktrees cause `detectSkillSourceRoot()` fallback (reproduced with current deploy)
- ✅ Broken symlink count: 14 (verified each target)
- ✅ All 15 source skills have valid `skill.yaml` in `.skillc`

**What's untested:**

- ⚠️ Whether the Mar 4 deploy specifically was affected by worktrees (didn't verify worktree state at that time)
- ⚠️ Whether `skillc deploy --cleanup` correctly removes all debris formats (flat .md files may not be detected as orphans since they lack the AUTO-GENERATED marker used by orphan scanner)
- ⚠️ Whether legacy skills (issue-creation, kb-reflect, etc.) are actually referenced by any spawn config or daemon

**What would change this:**

- Finding would be wrong if ux-audit was intentionally excluded from deployment (no evidence of this)
- Worktree finding would be invalidated if `.claude/worktrees/` is always cleaned before deploy (not currently enforced)

---

## Source Skills Inventory

| # | Skill | Category | Spawnable | Dependencies | Deployed (dir) | Deployed (flat) | Findable |
|---|-------|----------|-----------|--------------|----------------|-----------------|----------|
| 1 | diagnostic | meta | No | orchestrator | meta/diagnostic/ ✓ | diagnostic.md | ✓ |
| 2 | meta-orchestrator | meta | Yes | — | meta/meta-orchestrator/ ✓ | meta-orchestrator.md | ✓ |
| 3 | orchestrator | meta | No | — | meta/orchestrator/ ✓ | — | ✓ |
| 4 | worker-base | shared | No | — | shared/worker-base/ ✓ | worker-base.md | ✓ |
| 5 | architect | worker | Yes | worker-base | worker/architect/ ✓ | architect.md | ✓ |
| 6 | codebase-audit | worker | Yes | worker-base | worker/codebase-audit/ ✓ | codebase-audit.md | ✓ |
| 7 | design-session | worker | Yes | worker-base | worker/design-session/ ✓ | design-session.md | ✓ |
| 8 | experiential-eval | worker | Yes | worker-base | worker/experiential-eval/ ✓ | experiential-eval.md | ✓ |
| 9 | experiment | worker | Yes | worker-base | worker/experiment/ ✓ | — | ✓ |
| 10 | feature-impl | worker | Yes | worker-base | worker/feature-impl/ ✓ | feature-impl.md | ✓ |
| 11 | head-to-head | worker | Yes | worker-base | worker/head-to-head/ ✓ | head-to-head.md | ✓ |
| 12 | investigation | worker | Yes | worker-base | worker/investigation/ ✓ | investigation.md | ✓ |
| 13 | research | worker | Yes | worker-base | worker/research/ ✓ | research.md | ✓ |
| 14 | systematic-debugging | worker | Yes | worker-base | worker/systematic-debugging/ ✓ | systematic-debugging.md | ✓ |
| 15 | **ux-audit** | **worker** | **Yes** | worker-base | **MISSING ✗** | ux-audit.md | **✗ BROKEN** |

## Legacy/External Skills (deployed but not in orch-go source)

| Skill | Origin | Deployed Format | Findable | Notes |
|-------|--------|-----------------|----------|-------|
| analyze-skill-usage | meta/ (pre-merge) | Working symlink + directory | ✓ | |
| audit-claude-md | meta/ (pre-merge) | Working symlink + directory | ✓ | |
| brainstorming | worker/ (manual) | Directory only | ✓ | |
| capture-knowledge | shared/ (pre-merge) | Working symlink + directory | ✓ | |
| code-review | shared/ (pre-merge) | Working symlink + directory | ✓ | |
| hello | worker/ (test) | Working symlink + directory | ✓ | |
| playwright-cli | standalone | Directory only | ✓ | |
| record-decision | shared/ (pre-merge) | Working symlink + directory | ✓ | |
| testing-anti-patterns | utilities/ | Working symlink + directory | ✓ | |
| testing-skills-with-subagents | meta/ | Working symlink + directory | ✓ | |
| tmux-workspace-sync | utilities/ | Working symlink + directory | ✓ | |
| ui-mockup-generation | utilities/ | Working symlink + directory | ✓ | |
| workspace-isolation | utilities/ | Working symlink + directory | ✓ | |
| **delegating-to-team** | **shared/ (gone)** | **Broken symlink only** | **✗** | Source removed |
| **issue-creation** | **worker/ (gone)** | **Broken symlink only** | **✗** | Source removed |
| **issue-quality** | **shared/ (gone)** | **Broken symlink only** | **✗** | Source removed |
| **kb-reflect** | **worker/ (gone)** | **Broken symlink only** | **✗** | Source removed |
| **reliability-testing** | **worker/ (gone)** | **Broken symlink only** | **✗** | Source removed |
| **ui-design-session** | **worker/ (gone)** | **Broken symlink only** | **✗** | Source removed |
| **writing-skills** | **meta/ (gone)** | **Broken symlink only** | **✗** | Source removed |

## Loader Analysis

**File:** `pkg/skills/loader.go:55-88`

**Search paths (in order):**
1. `~/.claude/skills/{skillName}/SKILL.md` — direct path (handles symlinks that resolve to dirs with SKILL.md)
2. `~/.claude/skills/*/{skillName}/SKILL.md` — one-level subdirectory scan (handles `worker/`, `meta/`, `shared/`, etc.)

**Formats the loader CANNOT find:**
- Flat `.md` files at root (`skillName.md`) — never checked
- Paths 2+ levels deep (`skills/src/worker/skillName/SKILL.md`) — only 1-level subdirs scanned
- Broken symlinks (`skillName -> src/worker/skillName`) — stat fails, no SKILL.md found

## skillc Deploy Analysis

**Source:** `~/Documents/personal/skillc/cmd/skillc/deploy.go`

**How it deploys:**
1. Walks source tree for `.skillc` dirs containing `skill.yaml`
2. `detectSkillSourceRoot()` auto-detects source root (3 levels up from each .skillc)
3. Calculates `relPath = Rel(sourceRoot, skillBaseDir)`
4. Deploys to `targetDir/relPath/SKILL.md`
5. `--agent-dir` separately creates flat `{name}.md` files with agent YAML frontmatter
6. `--cleanup` scans for orphaned skillc-generated files (checks `AUTO-GENERATED` header)

**Root cause of three formats:**
- **Symlinks (Jan 29):** Manual setup before skillc deploy existed. `src/worker/` paths were valid when skills source was at `~/.claude/skills/src/worker/`
- **Flat .md (Mar 1):** `--agent-dir ~/.claude/skills/` used to deploy OpenCode agent files alongside Claude skills
- **Directories (Mar 4):** Correct `skillc deploy --target ~/.claude/skills/` from source
- **Nested skills/src/ (Feb 28):** Deploy with worktree interference or before `detectSkillSourceRoot()` worked correctly

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Re-deploy ux-audit to fix immediate breakage | implementation | Single command, reversible |
| Clean all deployment debris | implementation | Reversible, known-safe with --cleanup |
| Fix worktree interference in skillc | architectural | Cross-component (skillc + orch-go worktree pattern) |
| Remove broken legacy symlinks | implementation | Dead references, no downstream impact |

### Recommended Approach: Two-Phase Cleanup

**Phase 1 — Immediate fix (5 min):**
```bash
# Deploy from skills/src/ directly (avoids worktree interference)
cd /path/to/orch-go
skillc deploy --target ~/.claude/skills/ --cleanup skills/src
```
This will:
- Deploy ux-audit to `worker/ux-audit/SKILL.md` (fixes the breakage)
- Clean orphaned skillc-generated files (removes nested `skills/src/` tree)
- NOT clean flat .md files or manual symlinks (they lack AUTO-GENERATED headers)

**Phase 2 — Full debris removal (10 min):**
```bash
# Remove all 14 broken symlinks
for link in architect codebase-audit delegating-to-team design-session \
  feature-impl investigation issue-creation issue-quality kb-reflect \
  reliability-testing research systematic-debugging ui-design-session \
  writing-skills; do
  rm ~/.claude/skills/$link
done

# Remove all 13 flat .md files (agent format, not used by Claude Code)
rm ~/.claude/skills/*.md
```

**Phase 3 — Prevent recurrence (architectural):**
Fix `detectSkillSourceRoot()` in skillc to exclude `.claude/worktrees/` paths from candidate detection. This prevents future deploys from producing incorrect paths.

### Alternative: Extend Loader to Support Flat .md Files

**Pros:** Flat files would become findable; backward compatibility with agent-dir deploys
**Cons:** Maintains two formats indefinitely; agent .md files have different frontmatter than SKILL.md; increases loader complexity
**When to use:** Only if OpenCode and Claude Code need to share the same skills directory

---

## References

**Files Examined:**
- `pkg/skills/loader.go` — Loader search logic (FindSkillPath lines 55-88)
- `~/Documents/personal/skillc/cmd/skillc/deploy.go` — Full deploy logic including detectSkillSourceRoot
- All `.skillc/skill.yaml` files in `skills/src/` — Metadata for source inventory

**Commands Run:**
```bash
# Find all skill source files
find skills/src -type f -name "*.md"

# Find all deployed files
find ~/.claude/skills -type f -o -type l

# Check broken symlinks
for entry in ~/.claude/skills/*; do [ -L "$entry" ] && [ ! -e "$entry" ] && echo "BROKEN: $(basename $entry)"; done

# Simulate loader findability
# (custom script checking direct path then */skillName/SKILL.md)

# Test ux-audit compilation
cd skills/src/worker/ux-audit && skillc build .skillc

# Test full deploy from project root
skillc deploy --target /tmp/test

# Extract skill metadata
grep '^name:\|^type:\|^spawnable:' skills/src/*/*/.skillc/skill.yaml
```

**Related Artifacts:**
- **Constraint (kb quick):** "skillc cannot compile SKILL.md templates without template expansion feature"
- **Guide:** `.kb/guides/skill-system.md`
