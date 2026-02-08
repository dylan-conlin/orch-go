# Investigation: Remove Registry Dependency from orch status

**Question:** How to remove registry dependency from orch status command to achieve <2s execution?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** feature-impl
**Phase:** Complete
**Status:** Complete

---

## TLDR

Remove registry-based agent collection from status command, relying entirely on primary sources (OpenCode sessions, tmux windows, beads issues) to achieve 20x performance improvement.

---

## Findings

### Finding 1: Registry is redundant in status command

**Evidence:**
- status_cmd.go already fetches agents from OpenCode sessions (line 219)
- status_cmd.go already discovers agents from tmux windows (line 336-378)
- status_cmd.go already fetches beads issues for status (line 458)
- Registry phase (lines 257-333) duplicates discovery with stale data

**Source:**
- `cmd/orch/status_cmd.go:219` - OpenCode sessions fetch
- `cmd/orch/status_cmd.go:336-378` - tmux discovery
- `cmd/orch/status_cmd.go:257-333` - registry phase (redundant)

**Significance:** Registry adds O(n) processing overhead (534 agents) while providing data already derivable from primary sources.

---

### Finding 2: Primary sources provide all needed metadata

**Evidence:**
- Mode: derivable from source (tmux → "claude", OpenCode → "opencode")
- Model: can be extracted from workspace `.tier` file or session context
- Skill: already extracted from window name (extractSkillFromWindowName)
- ProjectDir: available from OpenCode session.Directory or workspace lookup

**Source:**
- `cmd/orch/status_cmd.go:365-372` - already extracts skill from window name
- `cmd/orch/status_cmd.go:421-425` - already gets projectDir from session.Directory

**Significance:** No registry-specific data is actually needed; all can be derived from primary sources.

---

## Implementation Plan

1. **Remove registry initialization** (lines 207-212)
2. **Remove Phase 1: registry-based collection** (lines 257-333)
3. **Enhance Phase 2/3 discovery** to capture model from workspace
4. **Remove unused registry import**
5. **Test and validate performance**

---

## References

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-20-inv-investigate-orch-status-command-performance.md`
- **Decision:** `.kb/decisions/2026-01-12-registry-is-spawn-cache.md`
