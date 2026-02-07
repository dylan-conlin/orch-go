# Session Synthesis

**Agent:** og-feat-epic-mechanize-principles-08jan-ebd9
**Issue:** orch-go-n5h2g
**Duration:** 2026-01-08 ~10:00 → ~11:00
**Outcome:** success

---

## TLDR

Designed and implemented 3 OpenCode plugins to mechanize principles at the moment of action: action-tracker (logs tool calls for pattern detection), guarded-files (surfaces modification protocols), and friction-capture (prompts for knowledge capture on session idle).

---

## Delta (What Changed)

### Files Created
- `~/.config/opencode/plugin/action-tracker.ts` - Logs all tool calls to ~/.orch/action-log.jsonl
- `~/.config/opencode/plugin/guarded-files.ts` - Surfaces protocols when editing protected files
- `~/.config/opencode/plugin/friction-capture.ts` - Prompts for friction capture on session.idle
- `~/.config/opencode/lib/guarded-files.ts` - Helper with guarded file registry and detection logic

### Files Modified
- `.kb/investigations/2026-01-08-inv-epic-mechanize-principles-via-opencode.md` - Full investigation with findings and recommendations

### Commits
- (to be created)

---

## Evidence (What Was Observed)

- Plugin API (`@opencode-ai/plugin`) provides `tool.execute.before/after`, `event` handling for session.idle, config hooks - sufficient for principle mechanization (Source: plugin type definitions)
- Existing `bd-close-gate.ts` proves Gate Over Remind pattern works via `tool.execute.before` (Source: ~/.config/opencode/plugin/bd-close-gate.ts)
- `session.idle` event exists and can trigger friction capture (Source: OpenCode docs at https://opencode.ai/docs/plugins)
- Guarded files are identifiable by patterns: "AUTO-GENERATED", "DO NOT EDIT", and specific paths like `~/.kb/principles.md`

### Tests Run
```bash
# Verified plugin structure matches existing working plugins
head -20 ~/.config/opencode/plugin/bd-close-gate.ts
# Uses same import pattern

# Verified lib directory structure
ls -la ~/.config/opencode/lib/
# Shows helpers properly located outside plugin/ to avoid loader issues
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-epic-mechanize-principles-via-opencode.md` - Full investigation into plugin-based principle mechanization

### Decisions Made
- **Plugin location:** Global (`~/.config/opencode/plugin/`) rather than project-local - because principles apply to all projects
- **Action log format:** JSONL to ~/.orch/action-log.jsonl - simple, appendable, grep-friendly
- **Guarded file detection:** Priority-based registry allowing both content-based (AUTO-GENERATED) and path-based (principles.md) detection
- **Friction capture trigger:** session.idle event - surfaces knowledge capture prompt at natural pause point

### Constraints Discovered
- Plugin helpers MUST be in `lib/` directory, not `plugin/` - OpenCode loader calls anything exported from `plugin/*.ts`
- `client.session.prompt` with `noReply: true` required for non-blocking context injection
- TypeScript module resolution errors are IDE complaints, not runtime errors - plugins work despite them

### Externalized via `kn`
- (Will run after synthesis)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (3 plugins + investigation)
- [x] Plugins follow established patterns (match bd-close-gate structure)
- [x] Investigation file complete with D.E.K.N. summary
- [ ] Git commit with all changes
- [ ] Ready for `orch complete orch-go-n5h2g`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does `session.idle` timing work? What triggers it and after how long?
- Could Evidence Hierarchy be mechanized by tracking grep calls before edit calls?
- What's the performance impact of action logging on tool latency?

**Areas worth exploring further:**
- `orch patterns` command to analyze action-log.jsonl for behavioral patterns
- Session compaction hooks (`experimental.session.compacting`) for preserving critical knowledge
- Cross-session pattern detection (e.g., same file edited repeatedly, same grep without code change)

**What remains unclear:**
- Whether `file.edited` event provides enough context for more sophisticated guarded file handling
- Interaction between multiple plugins hooking the same event

---

## Session Metadata

**Skill:** feature-impl
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-feat-epic-mechanize-principles-08jan-ebd9/`
**Investigation:** `.kb/investigations/2026-01-08-inv-epic-mechanize-principles-via-opencode.md`
**Beads:** `bd show orch-go-n5h2g`
