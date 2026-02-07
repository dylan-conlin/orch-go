# Session Synthesis

**Agent:** og-arch-design-systemic-solution-28jan-b84a
**Issue:** orch-go-20987
**Duration:** 2026-01-28 20:30 → 2026-01-28 21:45
**Outcome:** success

---

## TLDR

Designed three-layer architecture for ecosystem rebuild-on-change: (1) On-demand auto-rebuild (already exists in glass/agentlog), (2) Version commands for observability (missing in glass/agentlog - trivial to add), (3) Expanded orch doctor verification. Key finding: the "60% lacking auto-rebuild" audit finding was misleading - glass/agentlog have the BEST rebuild mechanism (on-demand), they just lack version commands to expose it externally.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-28-inv-design-systemic-solution-ecosystem-rebuild.md` - Complete design investigation with 5 forks navigated and three-layer architecture

### Files Modified
- `.orch/features.json` - Added feat-054 (glass version), feat-055 (agentlog version), feat-056 (orch doctor ecosystem verification)

### Commits
- (pending) Design investigation and feature additions

---

## Evidence (What Was Observed)

- Glass and agentlog already have `autorebuild.go` implementing sophisticated on-demand rebuild that compares embedded git hash to current HEAD and rebuilds+re-execs if stale
- All 5 Dylan CLIs have ldflags configured in Makefiles for version embedding (Version, BuildTime, SourceDir, GitHash)
- Glass and agentlog have version variables defined in main.go but no command to expose them
- Skillc, kb, orch all have working version commands showing git hash
- Post-commit hooks are actually redundant with on-demand rebuild since on-demand covers MORE scenarios (branch switches, pulls, resets, etc.)

### Tests Run
```bash
# Verified version commands
glass version  # Error: unknown command: version
agentlog version  # Error: unknown command
orch version  # Works: "orch version e177d0ea"
kb version  # Works: "kb version 5e52def-dirty"
skillc version  # Works: "skillc a8b8b25-dirty"

# Verified auto-rebuild exists
cat ~/Documents/personal/glass/autorebuild.go  # Confirmed pattern
cat ~/Documents/personal/agentlog/cmd/agentlog/autorebuild.go  # Identical pattern
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-28-inv-design-systemic-solution-ecosystem-rebuild.md` - Three-layer architecture design

### Decisions Made
- **On-demand > Post-commit**: On-demand auto-rebuild is superior because it catches ALL staleness sources (branch switch, pull, reset) and self-heals without setup
- **Version command format**: Human-readable default with --json flag for machine consumption (consistency with ecosystem)
- **Copy-paste pattern acceptable**: ~150 lines of stable autorebuild.go is acceptable duplication vs shared library complexity
- **Focus on Dylan's CLIs**: orch, kb, glass, skillc, agentlog. Exclude bd (upstream OSS) and kn (rarely used)

### Constraints Discovered
- Glass uses custom flag parsing (not cobra) - version command must fit that pattern
- Agentlog uses cobra - version command should use cobra pattern
- On-demand rebuild may cause unexpected delays in scripts - need to document env vars to disable

### Externalized via `kn`
- Should add: `kn decide "On-demand auto-rebuild is preferred over post-commit hooks" --reason "Covers all staleness sources, self-heals without setup, already proven in glass/agentlog"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation written, features added)
- [x] Design review passed (5 forks navigated with substrate reasoning)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-20987`

### Follow-up Work (via features.json)
1. **feat-054**: Add version command to glass (~30 lines) - Enables staleness detection
2. **feat-055**: Add version command to agentlog (~40 lines) - Same
3. **feat-056**: Extend orch doctor for ecosystem binaries - Depends on feat-054, feat-055

### Promote to Decision
Recommend promoting this investigation to decision as it establishes:
- Architectural pattern: Three-layer approach (self-healing, observability, verification)
- Ecosystem constraint: All Dylan CLIs must have version command
- Preference: On-demand auto-rebuild over post-commit hooks

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should skillc adopt the on-demand auto-rebuild pattern from glass/agentlog? (Currently has version command but no auto-rebuild)
- Could the autorebuild.go pattern be generated from a template to ensure consistency?
- Should orch doctor auto-run `make install` for stale binaries found?

**Areas worth exploring further:**
- Race condition observed in skillc auto-rebuild (from audit) - needs investigation
- Whether ECOSYSTEM.md should be updated to reflect actual state (80% auto-rebuild coverage, not 40%)

**What remains unclear:**
- Whether users prefer on-demand rebuild latency vs hook-based immediate rebuild
- Whether the env var pattern for disabling auto-rebuild is discoverable enough

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-systemic-solution-28jan-b84a/`
**Investigation:** `.kb/investigations/2026-01-28-inv-design-systemic-solution-ecosystem-rebuild.md`
**Beads:** `bd show orch-go-20987`
