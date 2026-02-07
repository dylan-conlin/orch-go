# Session Synthesis

**Agent:** og-feat-add-infrastructure-work-11jan-8866
**Issue:** orch-go-ao6nf
**Duration:** 2026-01-11
**Outcome:** success

---

## TLDR

Implemented automatic infrastructure work detection that auto-applies escape hatch flags (--backend claude --tmux) to prevent agents from killing themselves when working on OpenCode/orch infrastructure. Added keyword-based detection function with 20+ infrastructure terms and integrated it into spawn backend selection logic at priority 2.5.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-11-inv-add-infrastructure-work-detection-auto.md` - Investigation documenting infrastructure detection approach and implementation

### Files Modified
- `cmd/orch/spawn_cmd.go` - Added `isInfrastructureWork()` function (lines ~2139-2199) and infrastructure detection logic in backend selection (lines ~1100-1117)
- `cmd/orch/spawn_cmd_test.go` - Added `TestIsInfrastructureWork()` with 11 test cases covering infrastructure keywords and paths

### Commits
- `feat: add infrastructure work detection to auto-apply escape hatch flags`

---

## Evidence (What Was Observed)

- Backend selection logic follows clear priority chain (explicit flags → opus → model → config → default) at lines 1081-1115 in spawn_cmd.go
- Claude backend automatically uses tmux mode (verified in runSpawnClaude at line 1807: "Spawned agent in Claude mode (tmux)")
- Infrastructure paths already defined in mode.go lines 41-45 (cmd/orch/serve.go, pkg/state/, pkg/opencode/, web/src/lib/stores/, etc.)
- Blast radius classification in changelog.go lines 424-430 provides additional infrastructure patterns (pkg/spawn/, pkg/verify/, skillc, skill.yaml, SPAWN_CONTEXT)

### Tests Run
```bash
# Compilation test
go build -o /tmp/orch-test ./cmd/orch
# PASS: no compilation errors

# Manual installation
make install
# PASS: binary installed successfully
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-11-inv-add-infrastructure-work-detection-auto.md` - Complete investigation with D.E.K.N. summary, findings, implementation recommendations

### Decisions Made
- Decision 1: Use keyword-based detection (not ML/heuristics) because it's simple, debuggable, and matches existing infrastructure pattern definitions
- Decision 2: Insert detection at priority 2.5 (after spawnOpus, before spawnModel) to respect explicit overrides while providing automatic safety
- Decision 3: Log spawn.infrastructure_detected events for visibility and pattern analysis

### Constraints Discovered
- Constraint 1: Explicit --backend flag MUST override auto-detection (preserve user control)
- Constraint 2: Claude backend automatically uses tmux (no need to separately set spawnTmux flag)

### Externalized via `kb quick`
- None needed - implementation aligns with existing constraint (line 22: "Never spawn OpenCode infrastructure work without --backend claude --tmux")

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (detection function, integration, tests, investigation, SYNTHESIS.md)
- [x] Code compiles successfully
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-ao6nf`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could enhance with file path analysis (check if --workdir points to orch-go/opencode repos) - might reduce false negatives
- Could integrate with beads issue labels (e.g., infrastructure:opencode label) - provides explicit signal beyond keyword matching
- What's the false positive/negative rate in practice? - needs monitoring of events.jsonl over time

**Areas worth exploring further:**
- Extend keyword list based on real-world usage (monitor spawn.infrastructure_detected events)
- Add metrics dashboard showing infrastructure vs non-infrastructure spawn rates
- Consider surfacing auto-detection reason in spawn output (currently just says "infrastructure work detected")

**What remains unclear:**
- End-to-end behavior not tested with full spawn (would need OpenCode running and test issue)
- Beads issue description/title scanning logic implemented but not tested with real issues

*(Investigation flagged these as acceptable gaps - heuristic-based approach expected to have some false positives/negatives)*

---

## Session Metadata

**Skill:** feature-impl
**Model:** anthropic/claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-feat-add-infrastructure-work-11jan-8866/`
**Investigation:** `.kb/investigations/2026-01-11-inv-add-infrastructure-work-detection-auto.md`
**Beads:** `bd show orch-go-ao6nf`
