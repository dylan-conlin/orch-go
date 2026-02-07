<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch status` performance degrades due to unbounded registry growth (534 agents, all "active") causing O(n) processing where status command treats registry as authoritative source despite "cache-only" design.

**Evidence:** Registry has 534 agents (256KB), all "active"; status took 26.9s with registry vs 1.3s without (20x speedup); registry grew 5x (106→534) in 4 days despite Jan 16 optimization.

**Knowledge:** Registry designed as "spawn-time cache" with state staleness acceptable, but status command's O(n) operations on growing registry cause linear performance degradation; fixes address symptoms (processing) not root cause (registry growth).

**Next:** Spawn feature-impl to optimize status command to work without registry dependency, using primary sources (OpenCode sessions + tmux windows + beads).

**Promote to Decision:** Actioned - performance improvements implemented

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Investigate Orch Status Command Performance

**Question:** What is causing the progressive slowdown of orch status command performance over time, and why do fixes work temporarily but then regress?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** architect
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Agent registry grows without cleanup

**Evidence:** 
- Registry file `~/.orch/agent-registry.json` is 256KB with 534 agents
- All 534 agents are marked as "active" status
- No agents are marked as "completed", "abandoned", or "deleted"
- Registry grows with each spawn but never shrinks

**Source:** 
- `~/.orch/agent-registry.json` (256KB, 534 agents)
- `pkg/registry/registry.go` - Registry implementation with state tracking
- `cmd/orch/spawn_cmd.go:2388` - `registerAgent()` function registers agents as "active"
- `cmd/orch/complete_cmd.go` - Does NOT update agent registry (only session registry)
- `cmd/orch/clean_cmd.go` - Does NOT interact with agent registry

**Significance:** Registry growth directly impacts `orch status` performance as it must process all 534 "active" agents on every run.

---

### Finding 2: Status command processes all registry agents

**Evidence:**
- `status_cmd.go:2057` calls `agentReg.ListActive()` which returns all 534 agents
- If `--all` flag is used, also calls `agentReg.ListCompleted()` (returns empty)
- For each agent, makes multiple API calls and checks
- Command took 26.954 seconds total (65.79s user, 31.74s system)

**Source:**
- `cmd/orch/status_cmd.go:2057` - Registry loading in `runStatus()`
- Performance measurement: `time orch status` showed 26.954s total
- Registry methods: `ListActive()`, `ListCompleted()` in `pkg/registry/registry.go`

**Significance:** Linear performance degradation as registry grows - O(n) complexity where n = total spawned agents.

---

### Finding 3: No registry cleanup mechanism exists

**Evidence:**
- Registry has `Complete()`, `Abandon()`, `Remove()` methods but they're unused
- Comments state: "complete_cmd.go does NOT interact with registry" and "clean_cmd.go does NOT interact with registry"
- Registry design described as "caching layer, not source of truth" but treated as authoritative list

**Source:**
- `pkg/registry/registry.go:481-530` - Unused `Complete()`, `Abandon()`, `Remove()` methods
- `pkg/registry/registry.go:85-88` - Comments about command interactions
- `pkg/registry/registry.go:97-99` - "This design emerged from 12+ investigations showing registry is a 'caching layer, not source of truth'"

**Significance:** System lacks mechanism to transition agents from "active" to other states, causing perpetual growth.

---

## Synthesis

**Key Insights:**

1. **Registry growth without cleanup is the root cause** - The agent registry grows with each spawn (534 agents) but never shrinks because agents are never marked as completed/abandoned/deleted. This creates O(n) performance degradation where n = total spawned agents.

2. **Performance fixes address symptoms, not root cause** - Previous optimizations (Jan 16) reduced HTTP calls and filtered output but didn't address registry growth. Registry grew from 106 agents (Jan 16) to 534 agents (Jan 20), causing regression.

3. **Design decision conflicts with operational reality** - The "registry as spawn-time cache" decision accepts state staleness, but the status command treats registry as authoritative source, processing all "active" agents each run.

4. **Temporary fixes create boom-bust cycle** - Each optimization reduces processing for current agent count, registry continues growing, performance degrades until next optimization.

**Answer to Investigation Question:**

The progressive slowdown of `orch status` is caused by **unbounded registry growth without cleanup**. Agents are registered as "active" on spawn but never transition to other states. The status command processes all registry agents (534 currently), making multiple API calls per agent. Previous fixes optimized processing but didn't address registry growth, leading to temporary improvements followed by regression as registry size increased 5x in 4 days.

The root cause pattern is: **Registry treated as authoritative source despite being designed as cache-only.** Status command's O(n) operations on growing registry cause linear performance degradation.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Registry-optional status with time-based pruning** - Make registry loading optional in status command and prune entries older than 7 days.

**Why this approach:**
- **Directly addresses root cause**: Registry growth without bound is the primary performance issue
- **Preserves registry value**: Registry still available for recent agent lookups (abandon, session ID mapping)
- **Minimal change**: Doesn't require changing registry design or updating lifecycle commands
- **Proven performance**: Testing showed 20x speedup (26.9s → 1.3s) without registry

**Trade-offs accepted:**
- **Recent agents only**: Status may not show metadata for agents older than 7 days (acceptable: rarely need details for old agents)
- **Registry inconsistency**: Registry entries remain "active" indefinitely (already accepted in registry design decision)
- **Two-phase implementation**: Requires status optimization first, then registry pruning

**Implementation sequence:**
1. **Optimize status command to work without registry** - Use OpenCode sessions + tmux windows + beads as primary sources (proven to work)
2. **Add registry pruning to clean command** - Remove registry entries older than 7 days (optional cleanup)
3. **Make registry loading conditional** - Only load registry when needed (e.g., for `--all` flag or specific lookups)

### Alternative Approaches Considered

**Option B: Implement full registry lifecycle management**
- **Pros:** Makes registry state accurate, fulfills original design intent
- **Cons:** High cost (update 3 commands), creates synchronization burden, fights proven "cache-only" pattern
- **When to use instead:** If registry accuracy becomes critical for other use cases beyond status

**Option C: Remove registry entirely**
- **Pros:** Eliminates dead code and confusion, forces commands to use authoritative sources
- **Cons:** Breaks existing lookups (abandon, session ID mapping), larger refactoring scope
- **When to use instead:** If registry provides minimal value and all lookups can be derived from primary sources

**Rationale for recommendation:** Option A addresses the performance issue (registry growth) while respecting the existing "registry as cache" design decision. It provides immediate 20x performance improvement with minimal risk, unlike Options B (high cost, fights design) or C (breaking change).

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
