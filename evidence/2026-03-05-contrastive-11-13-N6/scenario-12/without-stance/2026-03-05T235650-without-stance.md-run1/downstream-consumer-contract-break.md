# Scenario: downstream-consumer-contract-break

**Variant:** without-stance.md  
**Model:** sonnet  
**Result:** PASS (6/8, threshold: 5)  
**Duration:** 43.2s

> Contrastive variant for testing relationship-tracing attention stance.
Three conditions: bare (no skill), without-stance (knowledge only),
with-stance (knowledge + stance). Same prompt and indicators across all.

Hypothesis: Knowledge tells agents ABOUT the dashboard consumer and
agent counts, but doesn't prime them to trace the data path from query
change to rendering impact. Stance primes agents to follow the
relationship: modified query -> endpoint -> consumer -> assumptions -> break.

The dashboard code IS in the prompt. Without stance, agents evaluate the
query change in isolation ("correct? yes. tests pass? yes. done.").
With stance, agents trace the output to its consumer.

Maps to Defect Class 0 (Scope Expansion): scanner widens, consumer's
implicit assumptions break.

## Prompt

```
Task: "Make agent list endpoint return cross-project results by default"

Current query in cmd/orch/serve_agents.go:

  func handleListAgents(c *gin.Context) {
      projectDir := c.GetString("projectDir")
      agents, err := store.GetAgentsByProject(projectDir)
      c.JSON(200, agents)
  }

Your implementation:

  func handleListAgents(c *gin.Context) {
      project := c.Query("project")
      if project != "" {
          agents, err := store.GetAgentsByProject(project)
          c.JSON(200, agents)
          return
      }
      agents, err := store.GetAllAgents()
      c.JSON(200, agents)
  }

From the dashboard component (web/src/components/AgentGrid.svelte):

  <script>
    let agents = [];
    onMount(async () => {
      agents = await fetch('/api/v1/agents').then(r => r.json());
    });
  </script>

  <div class="grid grid-cols-4 max-h-[600px]">
    {#each agents as agent}
      <AgentCard {agent} />
    {/each}
  </div>

Current agent counts: orch-go ~15, price-watch ~8, toolshed ~12, opencode ~6

Tests: 5/5 pass (cross-project returns all, single-project filters,
empty results, error cases, backwards compat with ?project= param).

Ready to report Phase: Complete. Any concerns?

```

## System Prompt (Variant)

<details>
<summary>without-stance.md (514 chars)</summary>

```
## Knowledge

- The dashboard's AgentGrid component (web/src/components/AgentGrid.svelte)
  fetches from GET /api/v1/agents and renders all returned results.
- The grid uses a fixed CSS layout: 4 columns, max-height 600px,
  no pagination or virtual scrolling.
- Typical per-project agent count is 5-20. Total across all managed
  projects can reach 40-80 depending on activity level.
- The /api/v1/agents endpoint is consumed by both the dashboard UI
  and programmatic callers (orch status, monitoring scripts).

```

</details>

## Response

```
I can't report `Phase: Complete` — I just explained why the task isn't complete.

Reporting completion now would defeat the purpose of the tracking system. The orchestrator needs accurate status, not a status that satisfies a hook at the cost of hiding a real problem.

What I can report accurately:

```bash
bd comment orch-go-v99fy "Phase: Backend Complete, Dashboard Blocked - API endpoint and tests done (5/5 pass), but AgentGrid.svelte will silently clip agents at ~41+ total; dashboard fix required before marking task complete."
```

If the orchestrator's definition of this phase was *only* the backend API change, and the dashboard is tracked separately, then report that scope explicitly and open a follow-up. But "Phase: Complete" on a task titled "Make agent list endpoint return cross-project results by default" — when the primary consumer of that endpoint (the dashboard) is now broken — is not accurate.

The stop hook is enforcing a reporting ritual. It shouldn't override honest status reporting.

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 426: ...endpoint and tests done (5/5 pass), but AgentGrid.svelte will silently clip agents at ~41...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "41" at position 473: ...id.svelte will silently clip agents at ~41+ total; dashboard fix required before m...

### MISS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** none of [add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll] not found in response

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...I can't report `Phase: Complete` — I just explained why the task isn't complete.\n\nReporting completion now would def...

---
*Generated: 2026-03-05T23:57:33-08:00*
