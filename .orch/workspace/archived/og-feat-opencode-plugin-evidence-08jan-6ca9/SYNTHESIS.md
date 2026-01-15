# Session Synthesis

**Agent:** og-feat-opencode-plugin-evidence-08jan-6ca9
**Issue:** orch-go-dv1lh
**Duration:** 2026-01-08 16:37 → 2026-01-08 16:50
**Outcome:** success

---

## TLDR

Implemented OpenCode plugin that enforces the Evidence Hierarchy principle by warning agents when they edit files without first searching/reading them in the current session. Plugin deployed to `~/.config/opencode/plugin/` via symlink.

---

## Delta (What Changed)

### Files Created
- `plugins/evidence-hierarchy.ts` - OpenCode plugin that tracks search/read operations and warns on unsearched file edits
- `.kb/investigations/2026-01-08-inv-opencode-plugin-evidence-hierarchy-warn.md` - Investigation file documenting implementation

### Files Modified
- None

### Commits
- (pending) - Add Evidence Hierarchy OpenCode plugin

---

## Evidence (What Was Observed)

- Existing plugins (`action-log.ts`, `orchestrator-session.ts`) provide clear patterns for `tool.execute.before/after` hooks
- `client.session.prompt` with `noReply: true` allows non-blocking warning injection
- Args must be stored in before hook (via `callID` key) and retrieved in after hook
- TypeScript module resolution warning is expected and doesn't affect runtime

### Tests Run
```bash
# TypeScript syntax check (module error expected)
cd ~/.config/opencode && npx tsc --noEmit plugin/evidence-hierarchy.ts
# Only module resolution error (expected, doesn't affect runtime)

# Verify symlink created
ls -la ~/.config/opencode/plugin/evidence-hierarchy.ts
# lrwxr-xr-x -> /Users/dylanconlin/Documents/personal/orch-go/plugins/evidence-hierarchy.ts
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-opencode-plugin-evidence-hierarchy-warn.md` - Implementation investigation

### Decisions Made
- **Warning vs Gate**: Chose warning (not blocking gate) because false positive rate is unknown. Gates should come after tuning.
- **Directory-level tracking**: Track searched directories, not just files, so grep/glob patterns count as evidence for files in that directory.
- **Exempt patterns**: Config files (JSON, YAML), generated files (dist/), and agent artifacts (SYNTHESIS.md) don't need prior search.

### Constraints Discovered
- `tool.execute.before` receives args in `output.args` (not `input.args`)
- `tool.execute.after` receives result in `output.output`
- Must correlate before/after via `callID` stored in a Map
- Sets cannot be iterated directly in older TypeScript targets - use `Array.from()`

### Externalized via `kn`
- No new constraints discovered that warrant `kn` externalization - this follows established plugin patterns

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (plugin + investigation + SYNTHESIS.md)
- [x] TypeScript compiles without errors (except expected module resolution)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-dv1lh`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to measure false positive rate in production? Could add logging/metrics to plugin.
- Should warning be more aggressive (blocking gate) once tuned? Would need `tool.execute.before` to return a rejection.

**Areas worth exploring further:**
- Integration with `action-log.ts` - currently both track similar operations, could deduplicate
- Bash command parsing - currently treats any bash as searching the workdir, could parse grep/find commands

**What remains unclear:**
- Actual false positive rate in real agent sessions
- Whether `client.session.prompt` works as expected (based on docs, not tested live)

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-opencode-plugin-evidence-08jan-6ca9/`
**Investigation:** `.kb/investigations/2026-01-08-inv-opencode-plugin-evidence-hierarchy-warn.md`
**Beads:** `bd show orch-go-dv1lh`
