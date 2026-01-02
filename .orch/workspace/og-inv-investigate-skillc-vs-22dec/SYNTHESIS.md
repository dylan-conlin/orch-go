# Session Synthesis

**Agent:** og-inv-investigate-skillc-vs-22dec
**Issue:** orch-go-iv25
**Duration:** ~30 minutes
**Outcome:** success

---

## TLDR

Investigated the relationship between `orch build skills` (Python orch-cli) and `skillc` (Go standalone). Concluded they are complementary tools for different purposes: orch build skills compiles templated procedural skills to ~/.claude/skills/, while skillc compiles project-local .skillc/ directories to CLAUDE.md. No migration or replacement needed.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md` - Full investigation with findings, synthesis, and recommendations

### Files Modified
- None

### Commits
- (pending) Investigation file creation

---

## Evidence (What Was Observed)

- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py:102-271` - Python skill builder with template expansion (SKILL-TEMPLATE markers)
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:1-240` - Go compiler with dependency graph resolution
- `~/.claude/skills/` contains 32 items (symlinks + category directories) - deployed by orch build skills
- `~/.config/opencode/agent/` contains 17 .md files with transformed frontmatter - also deployed by orch build skills
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go` - orch-go is a CONSUMER of skills, not a builder

### Tests Run
```bash
# Verified deployed artifacts exist
ls -la ~/.claude/skills/  # 32 items
ls -la ~/.config/opencode/agent/  # 17 .md files

# Verified help output
uv run orch build skills --help  # Shows dual-target deployment
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md` - Documents the distinction between the two build systems

### Decisions Made
- **Keep systems separate**: orch build skills for procedural skills, skillc for project context. No migration needed.
- **Rationale**: Different artifact types, different deployment targets, different use cases

### Constraints Discovered
- orch-go's pkg/skills/loader.go is a consumer only - it reads deployed skills but doesn't build them
- If Python orch-cli is deprecated, skill building would need a new home

### Externalized via `kn`
- `kn decide "skillc and orch build skills are complementary, not competing" --reason "skillc compiles project-local .skillc/ to CLAUDE.md; orch build skills compiles templated skills to ~/.claude/skills/. Different purposes, both needed."`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests performed (artifact existence verified)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-iv25`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orch-go have any `orch build` command group? Currently it has none.
- If Python orch-cli is deprecated, where should skill building live?

**Areas worth exploring further:**
- Whether skillc could have a `--template` mode for SKILL-TEMPLATE expansion (probably not needed)
- Whether to consolidate all markdown compilation tools into one

**What remains unclear:**
- Dylan's long-term plan for Python orch-cli (deprecate vs maintain alongside orch-go)

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-investigate-skillc-vs-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md`
**Beads:** `bd show orch-go-iv25`
