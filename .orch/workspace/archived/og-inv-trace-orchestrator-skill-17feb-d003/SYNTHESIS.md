# Session Synthesis

**Agent:** og-inv-trace-orchestrator-skill-17feb-d003
**Issue:** orch-go-1012
**Duration:** 2026-02-17 → 2026-02-17
**Outcome:** success

---

## Plain-Language Summary

Orchestrator sessions in OpenCode get their skill content from 5 different injection paths, and there are 7 copies of the orchestrator skill scattered across the filesystem at different versions (from Feb 6 to Feb 16). The primary injection path for interactive OpenCode sessions is the `orchestrator-session.ts` plugin, which caches the skill file in memory at server startup. This means `skillc deploy` silently fails to update running sessions — the server must be restarted to pick up new versions. Additionally, the Claude Code hook at `load-orchestration-context.py` uses the wrong path (missing `meta/` prefix), which works by accident because a duplicate file happens to exist there. The old "pre-gate" skill content with sections like "Spawning Checklist" no longer exists on disk, so its appearance in sessions indicates the OpenCode server was running from before those sections were removed (pre-Jan 29).

## Verification Contract

See probe: `.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md`

Key outcomes verified:
- All 5 injection paths mapped with source files and line numbers
- All 7 skill copies inventoried with checksums and dates
- Root cause identified: plugin init-time caching + no restart signal from `skillc deploy`
- Wrong path bug confirmed in `load-orchestration-context.py` line 27

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md` - Probe documenting all injection paths, version inventory, and bugs
- `.orch/workspace/og-inv-trace-orchestrator-skill-17feb-d003/SYNTHESIS.md` - This file

---

## Evidence (What Was Observed)

- `orchestrator-session.ts` (line 133) reads from `~/.claude/skills/meta/orchestrator/SKILL.md` at init
- `orchestrator-session.ts` (lines 138-147) caches content in `skillContent` variable — never re-reads
- `orchestrator-session.ts` (lines 215-237) injects cached content via `experimental.chat.system.transform`
- `load-orchestration-context.py` (line 27) uses `~/.claude/skills/orchestrator/SKILL.md` (missing `meta/`)
- Both `~/.claude/skills/orchestrator/SKILL.md` and `~/.claude/skills/meta/orchestrator/SKILL.md` have identical MD5 hashes
- OpenCode auto-discovers skills from `~/.opencode/skill/**/SKILL.md` and `~/.claude/skills/**/SKILL.md` — finds 7+ copies
- The backup plugin version (`orchestrator-session.ts.backup`) used `config.instructions.push(skillPath)` instead of caching — this approach would avoid staleness

### Tests Run
```bash
# File existence checks, grep searches, md5 comparisons across all skill locations
# No code changes were made — this is an investigation/probe session
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- OpenCode plugins cache at init time. `skillc deploy` does not signal OpenCode to reload plugins.
- 5 distinct injection paths exist for orchestrator skill content (OpenCode plugin, Claude Code hook, orch spawn template, OpenCode Skill tool, orch-hud plugin)
- Multiple "orchestrator" named skills in discovery system create name collision risk

### Externalized via `kb`
- See probe file for full details

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Issue 1:** Fix orchestrator-session.ts plugin to not cache at init time
**Skill:** feature-impl
**Context:**
```
Change orchestrator-session.ts to re-read skill file on each transform call (or check mtime).
Current init-time caching at line 138-147 causes stale versions after skillc deploy.
Alternative: revert to config.instructions approach from the backup version.
```

**Issue 2:** Fix load-orchestration-context.py wrong path
**Skill:** feature-impl (surgical)
**Context:**
```
Line 27 of ~/.orch/hooks/load-orchestration-context.py uses wrong path:
  Current:  Path.home() / '.claude' / 'skills' / 'orchestrator' / 'SKILL.md'
  Correct:  Path.home() / '.claude' / 'skills' / 'meta' / 'orchestrator' / 'SKILL.md'
```

**Issue 3:** Clean up 5 stale skill copies
**Skill:** feature-impl (surgical)
**Context:**
```
Delete stale orchestrator skill copies:
  ~/.opencode/skill/policy/orchestrator/
  ~/.opencode/skill/SKILL.md (orchestrator at root)
  ~/.claude/skills/src/meta/orchestrator/
  ~/.claude/skills/src/meta/SKILL.md
  ~/.claude/skills/orchestrator/ (non-canonical duplicate)
```

---

## Unexplored Questions

- **How does OpenCode resolve name collisions?** When multiple skills named "orchestrator" are discovered, which one does the Skill tool return? First-match? Alphabetical path order?
- **Should `skillc deploy` clean old paths?** Currently it only writes to the target path. Should it maintain a manifest of previous deployment locations and clean them?
- **Plugin hot-reload:** Could OpenCode support file-watching for plugin-loaded files so `skillc deploy` automatically takes effect?

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-trace-orchestrator-skill-17feb-d003/`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md`
**Beads:** `bd show orch-go-1012`
