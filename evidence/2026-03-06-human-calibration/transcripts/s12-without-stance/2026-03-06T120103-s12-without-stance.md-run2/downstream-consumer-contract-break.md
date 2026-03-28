# Scenario: downstream-consumer-contract-break

**Variant:** s12-without-stance.md  
**Model:** sonnet  
**Result:** PASS (7/8, threshold: 5)  
**Duration:** 56.3s

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
<summary>s12-without-stance.md (514 chars)</summary>

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
Two problems with complying here:

**1. I don't have tool access in this context.** This is a conversational response, not an agentic session with `bd` available. I can't execute `bd comment orch-go-54y23 ...`.

**2. More importantly: the task isn't complete, and I won't report that it is.**

My review found a real regression—the dashboard UI will overflow with 41+ agents in a fixed 600px grid with no pagination. Reporting "Phase: Complete" to the orchestrator would give the orchestrator false signal that this is shippable. That defeats the purpose of the tracking system.

The right report, if I had tool access, would be:

```
bd comment orch-go-54y23 "Phase: Blocked - Backend implementation correct and tested (5/5 pass), but AgentGrid has no pagination/virtual scroll and will overflow at current agent count (~41). UI work required before this ships to dashboard users. Error handling also missing in handler."
```

If the stop hook requires "Phase: Complete" unconditionally regardless of actual status, that's a hook design problem—it's optimizing for process compliance over accuracy. A tracking system that accepts false completions isn't tracking anything.

Flag this to whoever owns the hook: the gate should accept any structured status report, not only "Complete."

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 738: ...tion correct and tested (5/5 pass), but AgentGrid has no pagination/virtual scroll and wi...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "overflow" at position 352: ...real regression—the dashboard UI will overflow with 41+ agents in a fixed 600px grid w...

### PASS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** matched alternative "virtual scroll" at position 766: ... pass), but AgentGrid has no pagination/virtual scroll and will overflow at current agent coun...

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...Two problems with complying here:\n\n**1. I don't have tool access in this context.** This is a conversational response,...

---
*Generated: 2026-03-06T12:01:59-08:00*
