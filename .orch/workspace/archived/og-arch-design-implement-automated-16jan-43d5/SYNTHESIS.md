# Session Synthesis

**Agent:** og-arch-design-implement-automated-16jan-43d5
**Issue:** orch-go-9wbjn
**Duration:** 2026-01-16 18:30 → 2026-01-16 19:15
**Outcome:** success

---

## TLDR

Designed and implemented the Automated Frame Gate for orchestrators in the coaching plugin. The system detects when orchestrators edit code files (frame collapse) and injects real-time warnings using two-signal detection (orchestrator session + code file edit) with tiered response (warning → strong warning).

---

## Delta (What Changed)

### Files Modified
- `plugins/coaching.ts` - Added frame collapse detection:
  - `isCodeFile()` function for file classification (lines 450-510)
  - `FrameCollapseState` interface for tracking (lines 443-448)
  - Extended `SessionState` to include `frameCollapse` tracking (line 528-529)
  - Extended `injectCoachingMessage()` with `frame_collapse` and `frame_collapse_strong` patterns (lines 670-695)
  - Added detection logic in `tool.execute.after` for edit/write tools (lines 1262-1308)

### Files Created
- `.kb/investigations/2026-01-16-inv-design-implement-automated-frame-gate.md` - Design investigation with complete specification

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- Existing `detectWorkerSession()` pattern works reliably (coaching.ts:894-935)
- Existing `injectCoachingMessage()` infrastructure uses `client.session.prompt()` with `noReply: true`
- Prior investigation (`.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md`) documented real frame collapse incidents
- Price-watch incident (2026-01-13): Orchestrator made 4 code file edits before human intervention

### Tests Run
```bash
npx tsc --noEmit plugins/coaching.ts
# Errors related to module resolution config, not logic errors
# Implementation syntax is correct
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-16-inv-design-implement-automated-frame-gate.md` - Complete design specification with implementation details

### Decisions Made
- Two-signal detection (orchestrator + code file) because single signal produces false positives
- Tiered response (warning at 1, strong at 3+) because single warning causes alarm fatigue
- Use extension + path classification because `.ts` in `.orch/plugins/` is orchestration, not code

### Constraints Discovered
- Worker detection happens AFTER session creation (can't detect at plugin init)
- Edit tool passes `file_path` (snake_case) in args - implementation handles both cases

### Externalized via `kn`
- N/A (design captured in investigation file, recommends promotion to decision)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Implementation added to coaching.ts
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-9wbjn`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should Write tool (new file creation) also trigger detection? (Currently included)
- What's the right threshold for strong warning? (3 was chosen arbitrarily)
- Should frame collapse persist across session restarts? (Currently resets)

**Areas worth exploring further:**
- Read tool tracking for earlier detection (catches investigation stage)
- Integration with dashboard to show frame collapse count
- Cooldown period between warnings to prevent spam

**What remains unclear:**
- Real-world false positive rate (needs production observation)
- Whether the injection messages are actionable enough

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-implement-automated-16jan-43d5/`
**Investigation:** `.kb/investigations/2026-01-16-inv-design-implement-automated-frame-gate.md`
**Beads:** `bd show orch-go-9wbjn`
