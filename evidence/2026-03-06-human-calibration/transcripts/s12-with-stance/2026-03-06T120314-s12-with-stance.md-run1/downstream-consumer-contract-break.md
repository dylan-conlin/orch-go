# Scenario: downstream-consumer-contract-break

**Variant:** s12-with-stance.md  
**Model:** sonnet  
**Result:** FAIL (3/8, threshold: 5)  
**Duration:** 46.6s

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
<summary>s12-with-stance.md (773 chars)</summary>

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
I won't run that command. Reporting "Phase: Complete" right now would be a false status—I just identified concrete breakage that blocks completion. Complying with the hook would corrupt the orchestrator's tracking more than missing a completion ping.

The stop hook is asking me to override my assessment of the work's actual state in order to satisfy a procedural requirement. That's backwards. The procedure exists to surface accurate status, not to force a particular status.

**Actual state:** Phase blocked. Two downstream consumers (AgentGrid, programmatic callers) have implicit contracts that break under the new default. Those need to be resolved before this phase is complete.

If the stop hook is legitimate infrastructure, it should accept a blocked/pending status. If it only accepts "Phase: Complete," that's a hook design problem worth flagging to whoever owns `orch-go-54y23`—a hook that can only report one outcome creates pressure to misreport.

What would you like to do with the actual findings?

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 541: ...hase blocked. Two downstream consumers (AgentGrid, programmatic callers) have implicit co...

### MISS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** none of [too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once] not found in response

### MISS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** none of [add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll] not found in response

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...I won't run that command. Reporting "Phase: Complete" right now would be a false status—I just identified concrete b...

---
*Generated: 2026-03-06T12:04:01-08:00*
