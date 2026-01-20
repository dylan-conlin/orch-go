# Session Synthesis

**Agent:** og-arch-investigate-orch-status-20jan-cfe7
**Issue:** untracked (ad-hoc spawn)
**Duration:** 2026-01-20 13:00 → 2026-01-20 14:30
**Outcome:** success

---

## TLDR

Identified root cause of `orch status` performance degradation: unbounded registry growth (534 agents, all "active") causing O(n) processing. Registry designed as "spawn-time cache" but status command treats it as authoritative source. Proven 20x speedup (26.9s → 1.3s) by skipping registry loading.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-20-inv-investigate-orch-status-command-performance.md` - Complete investigation with findings and recommendations

### Files Modified
- `.kb/investigations/2026-01-20-inv-investigate-orch-status-command-performance.md` - Updated with investigation results

### Commits
- Will be created after this synthesis

---

## Evidence (What Was Observed)

- **Registry size**: 534 agents in `~/.orch/agent-registry.json` (256KB), all marked "active"
- **Performance impact**: `orch status` took 26.954 seconds total (65.79s user, 31.74s system)
- **Without registry**: `orch status` took 1.372 seconds (20x speedup) when registry file moved
- **Registry design**: Decision document states registry is "spawn-time metadata cache" with state staleness acceptable
- **Lifecycle commands**: `complete_cmd.go` and `clean_cmd.go` don't interact with registry, only `spawn_cmd.go` writes to it
- **Previous fix**: Jan 16 optimization improved from 15s to 1.8s but registry grew from 106 to 534 agents (5x) in 4 days

### Tests Run
```bash
# Baseline performance with registry
time orch status
# 65.79s user 31.74s system 361% cpu 26.954 total

# Performance without registry
mv ~/.orch/agent-registry.json ~/.orch/agent-registry.json.backup
time orch status  
# 2.11s user 0.99s system 225% cpu 1.372 total

# Registry analysis
jq '.agents | length' ~/.orch/agent-registry.json
# 534
jq '[.agents[] | select(.status == "active")] | length' ~/.orch/agent-registry.json
# 534
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-20-inv-investigate-orch-status-command-performance.md` - Root cause analysis of status performance degradation

### Decisions Made
- **Recommended approach**: Registry-optional status with time-based pruning
- **Why**: Addresses performance issue while respecting existing "registry as cache" design decision
- **Alternative rejected**: Full registry lifecycle management (high cost, fights proven pattern)
- **Alternative rejected**: Remove registry entirely (breaking change, loses useful metadata)

### Constraints Discovered
- **Registry growth without cleanup**: Agents registered as "active" on spawn, never transition to other states
- **Status command O(n) on registry**: Processes all 534 agents each run with multiple API calls
- **Design-reality mismatch**: Registry designed as "cache" but status treats it as authoritative source
- **Boom-bust cycle**: Performance fixes address symptoms (processing), not root cause (registry growth)

### Externalized via `kb`
- `kb quick constrain "orch status performance degrades with registry growth" --reason "Registry entries never marked completed, causing O(n) processing where n = total spawned agents"`
- `kb quick tried "Optimize status processing without addressing registry growth" --failed "Registry grows 5x in 4 days, performance regresses after temporary improvement"`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Optimize orch status to work without registry dependency
**Skill:** feature-impl
**Context:**
Modify `status_cmd.go` to derive agents from primary sources (OpenCode sessions + tmux windows + beads) instead of registry. Registry loading should be optional or limited to recent agents. This implements the 20x speedup proven in testing while preserving registry for other use cases (abandon command, session ID lookups).

### Additional Follow-up (optional)
**Issue:** Add registry pruning to orch clean command
**Skill:** feature-impl  
**Context:**
Add `--registry` flag to `orch clean` that removes registry entries older than 7 days. This provides optional cleanup for users who want to maintain smaller registry size without changing the core "cache-only" design.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the registry design decision be revisited given the performance impact?
- Are there other commands besides `status` that suffer from registry growth?
- Could registry use a more efficient data structure (e.g., SQLite) for large datasets?

**Areas worth exploring further:**
- Registry usage patterns in production - which commands actually need registry vs can derive from primary sources
- Performance profiling of status command to identify other bottlenecks beyond registry
- Automated registry cleanup based on beads issue status (closed = completed)

**What remains unclear:**
- Exact performance characteristics at different registry sizes (100 vs 1000 vs 10000 agents)
- Impact of registry on memory usage during status command execution
- Whether other commands (abandon, wait) would break without registry

---

## Session Metadata

**Skill:** architect
**Model:** deepseek/deepseek-chat
**Workspace:** `.orch/workspace/og-arch-investigate-orch-status-20jan-cfe7/`
**Investigation:** `.kb/investigations/2026-01-20-inv-investigate-orch-status-command-performance.md`
**Beads:** untracked (ad-hoc spawn)