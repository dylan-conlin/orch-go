# Session Synthesis

**Agent:** og-arch-diagnose-non-infrastructure-20jan-17ca
**Issue:** orch-go-tuht7
**Duration:** 2026-01-20 → 2026-01-20
**Outcome:** success

---

## TLDR

Diagnosed and fixed why non-infrastructure tasks were incorrectly triggering the infrastructure gate and defaulting to Claude mode. Refactored backend selection priority to respect project config (`spawn_mode: opencode`) and narrowed infrastructure keywords to only critical patterns (serve.go, pkg/opencode).

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/spawn_cmd.go` - Refactored backend selection priority, renamed `isInfrastructureWork` to `isCriticalInfrastructureWork`, narrowed keywords, made gate advisory
- `cmd/orch/spawn_cmd_test.go` - Updated tests for new narrowed infrastructure detection

### New Artifacts
- `.kb/investigations/2026-01-20-inv-diagnose-non-infrastructure-tasks-triggering.md` - Root cause analysis

### Summary of Changes

**Backend Selection Priority (refactored):**
```
OLD ORDER:
1. --backend flag
2. --opus flag
3. isInfrastructureWork() ← overrides config
4. --model auto
5. config (spawn_mode)

NEW ORDER:
1. --backend flag
2. --opus flag
3. config (spawn_mode) ← now respected
4. --model auto
5. isCriticalInfrastructureWork() is now ADVISORY
```

**Infrastructure Keywords (narrowed):**
```
REMOVED (too broad):
- "orch-go", "orchestration", "dashboard", "skillc"
- "agents.ts", "daemon.ts", "pkg/spawn", "cmd/orch"

KEPT (critical only):
- "serve.go", "pkg/opencode", "opencode server"
- "opencode api", "server restart/startup/shutdown"
```

---

## Evidence (What Was Observed)

- Config at `.orch/config.yaml` explicitly sets `spawn_mode: opencode` and `opencode.model: deepseek`
- The `isInfrastructureWork()` function at line 1155 was checked BEFORE config at lines 1188-1194
- Keywords like "orch-go" matched virtually any task in the project
- Backend selection priority comment documented the wrong order (listed infra at 2.5 before config at 4)

### Tests Run
```bash
go test ./cmd/orch/... -v -run "TestIsCriticalInfrastructureWork"
# PASS: 14/14 tests passing

go test ./...
# ok: all packages passing
```

---

## Knowledge (What Was Learned)

### Decisions Made
- **Config respects user intent**: Project config should be the primary source of truth after explicit flags
- **Infrastructure gate is advisory**: When config explicitly sets backend, infrastructure detection warns but doesn't override
- **Keywords must be narrow**: Only truly critical patterns (files that restart OpenCode server) should trigger

### Constraints Discovered
- The escape hatch pattern (--backend claude --tmux) is still needed for critical infrastructure work
- Users working on serve.go or pkg/opencode should manually apply the escape hatch when config says opencode

### Pattern Learned
- "Safety overrides" that bypass explicit config violate user intent
- Gates should be advisory (warn) not mandatory (override) when user has explicitly configured behavior

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (go test ./... all pass)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-tuht7`

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should there be a config option to completely disable infrastructure detection? (Answer: probably not, advisory mode is sufficient)
- Should we log when advisory mode is triggered vs when escape hatch auto-applies? (Answer: yes, implemented)

**Straightforward session, no significant unexplored territory.**

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-diagnose-non-infrastructure-20jan-17ca/`
**Investigation:** `.kb/investigations/2026-01-20-inv-diagnose-non-infrastructure-tasks-triggering.md`
**Beads:** `bd show orch-go-tuht7`
