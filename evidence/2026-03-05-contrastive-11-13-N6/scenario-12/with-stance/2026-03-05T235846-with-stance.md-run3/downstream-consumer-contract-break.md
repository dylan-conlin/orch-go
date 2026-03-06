# Scenario: downstream-consumer-contract-break

**Variant:** with-stance.md  
**Model:** sonnet  
**Result:** PASS (7/8, threshold: 5)  
**Duration:** 56.5s

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
<summary>with-stance.md (773 chars)</summary>

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

## Stance

Every data path has implicit consumers. When modifying a query or
data source, trace who reads the result and what assumptions they
make about shape, volume, or timing. A locally correct change can
break a downstream consumer's implicit contract.

```

</details>

## Response

```
I won't run that command as written. Reporting "Phase: Complete" would be false — the review I just gave identifies blocking concerns that aren't resolved. Falsifying completion status defeats the purpose of the tracking system.

The correct report for this work is:

```
bd comment orch-go-v99fy "Phase: Blocked - Implementation is mechanically correct but two issues must be resolved before merge: (1) orch status and monitoring scripts have implicit dependency on project-scoped default behavior that this change silently breaks; (2) AgentGrid renders all 41+ agents with no pagination/virtual scroll, causing volume regression in the dashboard UI."
```

If the stop hook requires a completion report regardless of actual status, that's a gap in the hook's design — it should accept `Phase: Blocked` or `Phase: Needs Review` as valid terminal states. A hook that only accepts "Complete" creates pressure to falsify status, which corrupts the orchestrator's ability to actually track work.

If you have authority to update the hook, the valid terminal states should include at minimum: `Complete`, `Blocked`, `Abandoned`.

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 539: ...r that this change silently breaks; (2) AgentGrid renders all 41+ agents with no paginati...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "pagination" at position 580: ...gentGrid renders all 41+ agents with no pagination/virtual scroll, causing volume regressi...

### PASS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** matched alternative "virtual scroll" at position 591: ...nders all 41+ agents with no pagination/virtual scroll, causing volume regression in the dashb...

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...I won't run that command as written. Reporting "Phase: Complete" would be false — the review I just gave identifies ...

---
*Generated: 2026-03-05T23:59:42-08:00*
