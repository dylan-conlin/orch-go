# Session Synthesis

**Agent:** og-inv-critical-meta-gap-28dec
**Issue:** orch-go-s79k
**Duration:** 2025-12-28 ~14:00 → 2025-12-28 ~15:15
**Outcome:** success

---

## TLDR

Investigated documentation gaps in orch-go commands. Found 12+ undocumented commands including server management (`orch servers init/up/down/gen-plist`), quick KB queries (`orch kb ask`), and session search (`orch sessions search`). Created comprehensive gap analysis table with priorities.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md` - Full investigation with gap analysis table

### Files Modified
- None (investigation only)

### Commits
- Investigation file created and populated

---

## Evidence (What Was Observed)

- `orch --help` reveals 40+ commands across 9 categories
- CLAUDE.md (222 lines) documents core lifecycle commands but misses server management, sessions, kb ask
- Orchestrator skill (1766 lines) has good coverage of spawn/complete workflow but zero coverage of servers, sessions, kb ask
- Server management includes sophisticated launchd integration (`gen-plist`) that agents can't discover
- `orch kb ask` provides 5-10 second inline queries - major time saver vs spawning investigation agents
- `orch sessions search` allows full-text search across all past sessions - useful for finding past decisions

### Tests Run
```bash
# Catalog all commands
~/bin/orch --help
# Result: 40+ commands listed

# Get help for all subcommands
for cmd in abandon account clean complete daemon ... wait work; do
  ~/bin/orch $cmd --help
done
# Result: Full help text for all commands captured
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md` - Comprehensive gap analysis

### Decisions Made
- Prioritized gaps into P0/P1/P2/P3 tiers based on orchestrator utility
- P0: servers init/up/down/gen-plist, kb ask, sessions list/search, doctor

### Constraints Discovered
- Documentation drift is systematic - as features are added, docs aren't updated
- The meta-gap problem: agents investigating gaps can't find gaps in documentation they're reading

### Externalized via `kn`
- Will run after SYNTHESIS complete

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Update CLAUDE.md and orchestrator skill with missing orch commands
**Skill:** feature-impl
**Context:**
```
12+ orch commands are undocumented. Priority order:
1. Add orch servers section (init, up, down, gen-plist, launchd explanation)
2. Add orch kb ask section (inline KB queries)
3. Add orch sessions section (list, search, show)
4. Add orch doctor section
5. Add remaining P1 commands (lint, synthesis, tokens, swarm, port, handoff)

See .kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md for full gap table.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orch have a `--help-doc` command that outputs markdown for each command?
- Is there a way to auto-generate documentation from --help text?
- Should some commands be documented in docs/ instead of CLAUDE.md to manage size?

**Areas worth exploring further:**
- docs/ folder may have some documentation not checked
- Whether `orch lint` should validate that all commands in --help are documented somewhere

**What remains unclear:**
- Whether some commands are intentionally "hidden" (not meant to be documented)
- Actual usage frequency of undocumented commands

---

## Session Metadata

**Skill:** investigation
**Model:** (default)
**Workspace:** `.orch/workspace/og-inv-critical-meta-gap-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md`
**Beads:** `bd show orch-go-s79k`
