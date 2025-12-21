<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Concurrency limit implementation requires: 1) new flag/env var for spawn cmd, 2) ActiveCount() in registry, 3) pre-spawn check in runSpawnWithSkill.

**Evidence:** Code review of cmd/orch/main.go:672-752 (runSpawnWithSkill) and pkg/registry/registry.go:405-416 (ListActive).

**Knowledge:** Registry already has ListActive() which returns active agents slice; need ActiveCount() for efficient count without allocating full slice.

**Next:** Implement concurrency limit with --max-agents flag (default 5), ORCH_MAX_AGENTS env var, and pre-spawn check.

**Confidence:** High (90%) - straightforward feature addition to existing patterns.

---

# Investigation: Add Concurrency Limit to orch spawn

**Question:** How should we implement concurrency limiting to prevent runaway agent spawning?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None - proceeding to implementation
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Registry already tracks active agents

**Evidence:** `ListActive()` method at registry.go:405-416 returns all agents with `StateActive` status. This provides the foundation for counting active agents.

**Source:** pkg/registry/registry.go:405-416

**Significance:** We can add a simple `ActiveCount()` method that returns len(active agents) without needing to change the data model.

---

### Finding 2: Spawn command has clear entry point for check

**Evidence:** `runSpawnWithSkill()` at main.go:672 is the unified entry point for all spawn modes (inline, headless, tmux). The check should happen early, before workspace creation.

**Source:** cmd/orch/main.go:672-752

**Significance:** Single place to add concurrency check ensures all spawn modes respect the limit.

---

### Finding 3: Flag/env var pattern already established

**Evidence:** Spawn command already has numerous flags (lines 147-161) following consistent patterns. Environment variable support can use `os.Getenv()` with flag as override.

**Source:** cmd/orch/main.go:147-161, 197-209

**Significance:** Implementation should follow established patterns: flag defined in var block, registered in init(), env var checked in run function.

---

## Synthesis

**Key Insights:**

1. **Minimal code changes needed** - Registry has ListActive(), just need to add count method and use it before spawning.

2. **Single enforcement point** - runSpawnWithSkill is the right place because all spawn modes go through it.

3. **Default should be conservative** - Default of 5 prevents runaway spawning while allowing reasonable parallelism.

**Answer to Investigation Question:**

Implement by: 1) Adding `--max-agents` flag with default 5, 2) Adding `ORCH_MAX_AGENTS` env var support, 3) Adding `ActiveCount()` to registry, 4) Checking count < limit before spawning in runSpawnWithSkill, erroring if at limit.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Code is well-structured, patterns are clear, implementation is straightforward.

**What's certain:**

- ✅ Registry.ListActive() provides the active agent list
- ✅ runSpawnWithSkill is the unified spawn entry point
- ✅ Flag/env var patterns are established

**What's uncertain:**

- ⚠️ Whether default of 5 is the right number (can be adjusted)

---

## References

**Files Examined:**
- cmd/orch/main.go - Spawn command implementation
- pkg/registry/registry.go - Agent tracking

