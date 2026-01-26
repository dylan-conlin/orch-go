<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The 212 stuck agents are orphaned registry entries from completed agents; the registry is never cleaned when agents complete.

**Evidence:** Registry has 218 active entries but only ~10 have real tmux/OpenCode sessions; sample beads IDs (orch-go-u5ly0, orch-go-21enf) show status=closed in beads but status=active in registry.

**Knowledge:** The registry was designed as a "spawn-time metadata cache" but its status field creates false stuck agent counts in the frontier API.

**Next:** Either (A) remove registry from frontier stuck calculation, or (B) add registry cleanup to `orch complete`/`orch clean --stale`.

**Promote to Decision:** recommend-yes - This affects observability accuracy and warrants an architectural decision on registry's role.

---

# Investigation: 212 Stuck Agents Analysis

**Question:** Why do we have 212 stuck agents? Are they real sessions or orphaned registry entries?

**Started:** 2026-01-25
**Updated:** 2026-01-25
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Related-Decision:** `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - Established registry as spawn-time cache, not lifecycle tracker

---

## Findings

### Finding 1: Registry Contains 218 "Active" Entries

**Evidence:**
- `jq '.agents | length' ~/.orch/agent-registry.json` = 219 total
- `jq '[.agents[] | select(.status == "active")] | length'` = 218 active
- Oldest active entry: 2026-01-20T13:23:58 (5+ days ago)

**Source:** `~/.orch/agent-registry.json`

**Significance:** The registry has far more "active" entries than actual running agents. This is the source of the 212 "stuck" agents in the frontier API.

---

### Finding 2: Registry Status is Never Updated on Completion

**Evidence:**
- `complete_cmd.go` does NOT import `pkg/registry`
- `clean_cmd.go` does NOT import `pkg/registry`
- Registry design contract (lines 76-101 of registry.go): "Agent lifecycle state is derived from authoritative sources: OpenCode API (session state), Beads (issue status)"
- Methods `Abandon()`, `Complete()`, `Remove()` are marked DEPRECATED with "not integrated into lifecycle commands"

**Source:**
- `pkg/registry/registry.go:76-101` (design contract)
- `cmd/orch/complete_cmd.go` (no registry import)
- `cmd/orch/clean_cmd.go` (no registry import)

**Significance:** By design, the registry never transitions agents from "active" to "completed". It's a write-once, read-many cache that accumulates entries forever.

---

### Finding 3: Frontier API Calculates Stuck from Registry

**Evidence:**
- `serve_frontier.go:48`: `activeAgents, stuckAgents := getActiveAndStuckAgents()`
- `frontier.go` `getActiveAndStuckAgents()`: Calls `registry.ListActive()` and filters by `stuckThreshold`
- Any agent in registry with `status == "active"` and spawn time > 2 hours is counted as "stuck"

**Source:**
- `cmd/orch/serve_frontier.go:48`
- `cmd/orch/frontier.go:getActiveAndStuckAgents()`

**Significance:** The frontier API trusts registry status, but registry status is never updated. This creates phantom "stuck" agents.

---

### Finding 4: Sample "Stuck" Agents are Actually Closed in Beads

**Evidence:**
- `orch-go-u5ly0`: Status: closed in beads, status: active in registry
- `orch-go-21enf`: Status: closed in beads, status: active in registry

**Source:** `bd show orch-go-u5ly0`, `bd show orch-go-21enf`

**Significance:** Confirms these are orphaned registry entries, not real stuck agents. The issues completed successfully.

---

### Finding 5: Mode Distribution Shows Many Docker Agents

**Evidence:**
- 65 claude mode (tmux)
- 125 docker mode
- 28 opencode mode

**Source:** `jq -r '.agents[] | select(.status == "active") | .mode' ~/.orch/agent-registry.json | sort | uniq -c`

**Significance:** Docker containers are ephemeral - they're long gone by the time frontier checks. These are definitely orphaned entries.

---

### Finding 6: Duplicates Exist in Stuck List

**Evidence:**
- `ok-k16g`: 4 duplicate entries
- `pw-ww8p`, `orch-go-df2n8`, `orch-go-21enf`, `ok-od0l`: 3 duplicate entries each
- Total entries: 212, Unique: 183

**Source:** `curl -sk https://localhost:3348/api/frontier | jq -r '.stuck[] | .beads_id' | sort | uniq -c | sort -rn`

**Significance:** Some agents were registered multiple times (possibly from respawns or retries), further inflating the stuck count.

---

## Synthesis

**Key Insights:**

1. **Registry is Append-Only in Practice** - The registry was designed as a spawn-time cache, but the "status" field creates an implicit lifecycle tracking obligation that's never fulfilled.

2. **Status Command vs Frontier API Use Different Sources** - `orch status` filters to real tmux windows + recent OpenCode sessions (shows ~10 agents). Frontier API uses registry's `ListActive()` (shows 218 agents). This explains the discrepancy.

3. **The Problem is Architectural, Not Data** - The registry design contract explicitly says lifecycle state comes from beads/OpenCode, but the frontier API violates this by using registry status directly.

**Answer to Investigation Question:**

The 212 stuck agents are **orphaned registry entries**, not real sessions. They are agents that:
- Were spawned over the past 5+ days
- Completed their work (beads issues closed)
- Were never marked as completed in the registry (by design - registry doesn't track lifecycle)
- Now show as "stuck" because frontier API trusts registry status

Only ~10 agents are actually running (as shown by `orch status`). The other 208 are phantom entries from completed work.

---

## Structured Uncertainty

**What's tested:**

- ✅ Registry has 218 "active" entries (verified: `jq` query on registry.json)
- ✅ Sample stuck agents are closed in beads (verified: `bd show orch-go-u5ly0`)
- ✅ `complete_cmd.go` doesn't use registry (verified: no `registry` import)
- ✅ `orch status` shows only ~10 agents (verified: ran `orch status --all`)

**What's untested:**

- ⚠️ Whether adding registry cleanup to `orch complete` would break anything
- ⚠️ Whether removing registry from frontier would miss any real stuck agents
- ⚠️ Performance impact of cleaning registry on every completion

**What would change this:**

- Finding would be wrong if registry status IS updated somewhere (but grep shows it's not)
- Finding would be wrong if these agents have active tmux windows (but they don't)

---

## Implementation Recommendations

**Purpose:** Resolve the phantom stuck agent problem.

### Recommended Approach: Remove Registry from Frontier Calculation

**Why this approach:**
- Registry was explicitly designed NOT to track lifecycle (see design contract)
- Frontier should use authoritative sources: tmux windows + OpenCode sessions (like `orch status` does)
- No changes needed to `orch complete` or `orch clean`

**Trade-offs accepted:**
- Registry status field becomes vestigial
- Agents without tmux/OpenCode presence might not show as stuck (acceptable - they're not real agents)

**Implementation sequence:**
1. Update `getActiveAndStuckAgents()` to use the same sources as `status_cmd.go`
2. Filter to tmux windows + OpenCode sessions updated in last 3 hours
3. Apply stuck threshold (>2h) to this filtered set

### Alternative Approaches Considered

**Option B: Add Registry Cleanup to Completion Flow**
- **Pros:** Keeps registry accurate, allows registry-based stuck detection
- **Cons:** Requires changes to both `orch complete` and beads close hook; adds complexity
- **When to use instead:** If registry status is needed for other features

**Option C: Periodic Registry Reconciliation (`orch clean --registry`)**
- **Pros:** One-time cleanup, doesn't change completion flow
- **Cons:** Stuck agents would accumulate between cleanups; user must remember to run
- **When to use instead:** As a stopgap until A or B is implemented

**Rationale for recommendation:** Option A aligns with the existing design contract (registry as spawn-time cache). The status command already uses the right sources.

---

## References

**Files Examined:**
- `~/.orch/agent-registry.json` - Registry data showing 218 active entries
- `pkg/registry/registry.go` - Design contract explaining registry as spawn-time cache
- `cmd/orch/serve_frontier.go` - Frontier API handler using registry
- `cmd/orch/frontier.go:getActiveAndStuckAgents()` - Stuck calculation logic
- `cmd/orch/complete_cmd.go` - Confirmed no registry integration
- `cmd/orch/clean_cmd.go` - Confirmed no registry integration
- `cmd/orch/status_cmd.go` - Shows correct approach (tmux + OpenCode filtering)

**Commands Run:**
```bash
# Count registry entries
jq '.agents | length' ~/.orch/agent-registry.json

# Count active entries
jq '[.agents[] | select(.status == "active")] | length' ~/.orch/agent-registry.json

# Check beads status of stuck agents
bd show orch-go-u5ly0  # Status: closed

# Mode distribution
jq -r '.agents[] | select(.status == "active") | .mode' ~/.orch/agent-registry.json | sort | uniq -c

# Frontier API stuck count
curl -sk https://localhost:3348/api/frontier | jq '.stuck_total'  # 212
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - Establishes registry design contract

---

## Investigation History

**2026-01-25 21:14:** Investigation started
- Initial question: Why do we have 212 stuck agents?
- Context: Frontier API showing 212 stuck agents, far more than visible in `orch status`

**2026-01-25 21:20:** Found registry has 218 active entries
- Most spawned 5+ days ago, should have completed

**2026-01-25 21:25:** Confirmed registry status never updated
- Design contract explicitly says lifecycle comes from beads/OpenCode
- Complete/clean commands don't use registry

**2026-01-25 21:30:** Verified sample stuck agents are closed in beads
- Confirms these are orphaned entries, not real stuck agents

**2026-01-25 21:35:** Investigation completed
- Status: Complete
- Key outcome: 212 stuck agents are orphaned registry entries; frontier API should use tmux+OpenCode like status command does
