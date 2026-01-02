# Session Synthesis

**Agent:** og-arch-design-multi-project-26dec
**Issue:** (ad-hoc spawn, no tracking)
**Duration:** ~45 min
**Outcome:** success

---

## TLDR

Designed multi-project orchestration architecture: global visibility + project-scoped operations. The current hybrid approach (OpenCode/dashboard global, beads per-project) is correct by design. Main improvements needed are better error messages when cross-project operations fail, guiding users to the correct project directory.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-design-multi-project-orchestration-architecture.md` - Full design investigation with recommendation

### Files Modified
- None (design-only session)

### Commits
- None yet (will commit after this file)

---

## Evidence (What Was Observed)

- `serve.go:338-442` already has `extractUniqueProjectDirs()` and `buildMultiProjectWorkspaceCache()` for cross-project agent visibility
- `main.go:673-748` `runAbandon()` calls `verify.GetIssue()` which uses cwd for beads socket discovery
- `beads/client.go:75-106` `FindSocketPath()` walks up directory tree from cwd looking for `.beads/bd.sock`
- Prior investigation `2025-12-26-inv-design-proper-cross-project-agent.md` already solved dashboard aggregation
- `extractProjectFromBeadsID()` exists and can derive project name from beads ID

### Tests Run
```bash
# Code analysis only - no tests run (architecture investigation)
# Verified existing code patterns through file reading
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-design-multi-project-orchestration-architecture.md` - Design investigation with 4 approaches evaluated

### Decisions Made
- **Approach 3 (Global Visibility, Project-Scoped Operations) is recommended** because:
  1. Matches existing direction (dashboard already aggregates)
  2. Minimal disruption to workflows
  3. Respects beads per-project architecture
  4. Clear mental model: "I see everything; I act on my project"

### Constraints Discovered
- Beads is fundamentally per-project - each project has its own `.beads/bd.sock`
- OpenCode is fundamentally global - single server on port 4096
- These constraints are features, not bugs - they define the hybrid model

### Externalized via `kn`
- None (findings documented in investigation artifact)

---

## Next (What Should Happen)

**Recommendation:** close

This is a design investigation producing a recommendation. Implementation is follow-up work.

### If Close
- [x] All deliverables complete (investigation file produced)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for orchestrator review

### Follow-up Issues (For Implementation)

**Issue 1:** Improve `orch abandon` cross-project error messages
**Skill:** feature-impl
**Context:**
```
When orch abandon fails due to cross-project beads ID, provide helpful error message:
"Issue X not found in current project (Y). This appears to be from project 'Z'.
To abandon, run: cd /path/to/Z && orch abandon X"
```

**Issue 2:** Improve `orch complete` cross-project error messages
**Skill:** feature-impl
**Context:**
```
Same pattern as abandon - detect project mismatch, provide actionable guidance
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch spawn --workdir` create beads issue in target or current project?
- Should focus support project-aware goals for multi-project orchestration?
- How does daemon handle cross-project issue spawning?
- Should session titles include project name for quick identification?

**Areas worth exploring further:**
- Project selector UI in dashboard (filter by project in web UI)
- Visual project grouping in agent cards
- Persistent project filter preference

**What remains unclear:**
- Whether cross-project beads routing would ever be worth implementing
- How common cross-project operations actually are in practice

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-multi-project-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-design-multi-project-orchestration-architecture.md`
**Beads:** N/A (ad-hoc spawn)
