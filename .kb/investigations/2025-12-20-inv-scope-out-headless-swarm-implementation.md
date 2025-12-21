<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless Swarm = batch execution with rate-limit management. Created epic orch-go-bdd with 6 child tasks.

**Evidence:** User clarified scope: focus on (A) batch execution and (C) rate-limit awareness across accounts.

**Knowledge:** Implementation requires: usage tracking → capacity manager → concurrent daemon → swarm command. Two tasks ready immediately (usage tracking, status enhancement).

**Next:** Start with orch-go-bdd.1 (usage tracking) and orch-go-bdd.5 (status enhancement) - both marked triage:ready.

**Confidence:** High (90%) - Clear scope, decomposed into implementable tasks.

---

# Investigation: Scope Out Headless Swarm Implementation

**Question:** What does "Headless Swarm" mean for orch-go and what implementation work is needed?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Design Session Agent
**Phase:** Complete
**Next Step:** None - epic created (orch-go-bdd)
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Headless spawn already works for single agents

**Evidence:** 
- `runSpawnHeadless()` in `cmd/orch/main.go:834-911` creates sessions via HTTP API
- Uses `client.CreateSession()` + `client.SendPrompt()` 
- Registers with `WindowID: registry.HeadlessWindowID` special marker
- No TUI required, returns immediately

**Source:** `cmd/orch/main.go:834-911`, `pkg/opencode/client.go:226-261`

**Significance:** The foundation for headless agents exists. A "swarm" would need to extend this to manage multiple concurrent agents.

---

### Finding 2: Daemon currently processes issues sequentially

**Evidence:**
- `Daemon.Once()` processes a single issue at a time
- `Daemon.Run()` loops through issues one by one
- No parallelism or concurrency control
- Calls `orch-go work` which spawns headlessly by default

**Source:** `pkg/daemon/daemon.go:186-243`

**Significance:** Current daemon is single-threaded. A "swarm" would need concurrent spawning with limits.

---

### Finding 3: Registry supports tracking multiple agents but lacks concurrency awareness

**Evidence:**
- Registry tracks agents with `HeadlessWindowID` marker for headless spawns
- No limit on how many active agents can exist
- `Reconcile()` skips headless agents (tracked via SSE, not tmux)
- File locking prevents write conflicts but no read coordination

**Source:** `pkg/registry/registry.go:471-507`

**Significance:** Registry can track many agents but doesn't enforce concurrency limits or provide aggregate status.

---

### Finding 4: Rate-limiting is an active pain point

**Evidence:**
- DYLANS_THOUGHTS.org mentions: "workers spawned a ton of concurrent agents while testing the spawn command. this resulted in maxing out both of my claude max accounts session limits"
- Account management exists but no automatic rate-limit detection
- No coordination between multiple concurrent requests

**Source:** `DYLANS_THOUGHTS.org:4-10`

**Significance:** Any "swarm" implementation MUST include rate-limit awareness to avoid the exact problem Dylan experienced.

---

## Synthesis

**Key Insights:**

1. **Infrastructure is partially ready** - Headless spawns work, registry can track multiple agents, SSE monitors completion. The pieces exist but aren't orchestrated.

2. **Missing coordination layer** - No concurrency limits, no progress aggregation, no rate-limit awareness across agents.

3. **"Swarm" could mean different things** - Could be concurrent daemon, parallel task processing, or even distributed architecture.

**Answer to Investigation Question:**

The term "Headless Swarm" isn't defined in the codebase. Based on context, it likely means:
- Running multiple headless agents concurrently
- With coordination to prevent rate-limiting
- With aggregate progress visibility

However, this needs user clarification to properly scope.

---

## Confidence Assessment

**Current Confidence:** Medium (65%)

**Why this level?**

Architecture analysis is solid. Intent behind "Headless Swarm" is ambiguous.

**What's certain:**

- ✅ Headless spawns work via HTTP API
- ✅ Registry can track multiple agents
- ✅ Rate-limiting has been a real problem
- ✅ Daemon is currently single-threaded

**What's uncertain:**

- ⚠️ What "swarm" means in this context
- ⚠️ Target concurrency level (2? 5? unlimited with rate-limit awareness?)
- ⚠️ Whether this is about daemon enhancement or new capability

**What would increase confidence to High:**

- Clarification on what "swarm" means
- Defined concurrency limits or goals
- Whether rate-limit management is in scope

---

## Possible Interpretations to Clarify

### A. Concurrent Daemon Mode

Make daemon spawn multiple issues in parallel instead of sequentially:
- Add `--concurrency N` flag
- Track active agent count
- Wait for slot before spawning next

### B. Swarm Orchestration Command

New `orch swarm` command for batch operations:
- Spawn multiple agents from list of issues
- Monitor collective progress
- Aggregate completion status

### C. Rate-Limit-Aware Spawning

Focus on account/rate-limit coordination:
- Detect when approaching limits
- Auto-switch accounts
- Queue spawns across available capacity

### D. Full Multi-Model Swarm

Distribute work across multiple models:
- Route work to flash/opus based on complexity
- Balance load across API keys
- Maximize throughput within limits

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Spawn command implementation
- `pkg/daemon/daemon.go` - Daemon issue processing
- `pkg/registry/registry.go` - Agent state management
- `pkg/opencode/client.go` - HTTP API client
- `DYLANS_THOUGHTS.org` - User context on pain points

**Commands Run:**
```bash
# Search for headless/swarm references
rg -l "headless|swarm|parallel" /Users/dylanconlin/Documents/personal/orch-go

# Check for relevant decisions
ls -la /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/

# Find related investigations
rg -l "swarm|concurrent|parallel" .kb/investigations/
```

---

## Investigation History

**2025-12-20 19:18:** Investigation started
- Initial question: What does "Headless Swarm" mean and what implementation is needed?
- Context: Design session to scope feature work

**2025-12-20 19:25:** Context gathered
- Reviewed spawn, daemon, registry implementations
- Found rate-limiting pain point in DYLANS_THOUGHTS.org
- Determined "swarm" term isn't defined in codebase

**2025-12-20 19:30:** Findings synthesized
- Presented four possible interpretations
- Need user clarification to proceed with scoping
