# Scenario: downstream-consumer-contract-break

**Variant:** without-stance.md  
**Model:** sonnet  
**Result:** PASS (7/8, threshold: 5)  
**Duration:** 37.8s

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
I'm not going to run that command. Here's why:

My actual job in this conversation was to review the implementation and answer "Any concerns?" I found real concerns — the dashboard layout breaks at current agent counts, and existing programmatic callers get a silent behavior change. The work is **not complete** in any meaningful sense.

Reporting `Phase: Complete` to satisfy a stop hook would be inaccurate. A completion report should reflect the actual state of the work, not serve as a ritual to unlock session exit.

If this is a legitimate orchestration system, the right report would be something like:

```
Phase: Blocked - Backend implementation passes tests but two issues need resolution before shipping: (1) AgentGrid has no pagination/virtual scrolling for 40+ cross-project agents, (2) existing callers relying on middleware-set project context will silently receive all-agent results.
```

That's an accurate status. If the stop hook requires "Phase: Complete" specifically and won't accept a blocked/needs-work status, that's a problem with the hook, not something to paper over by lying about the outcome.

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 723: ...es need resolution before shipping: (1) AgentGrid has no pagination/virtual scrolling for...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "pagination" at position 740: ...n before shipping: (1) AgentGrid has no pagination/virtual scrolling for 40+ cross-project...

### PASS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** matched alternative "virtual scroll" at position 751: ...ipping: (1) AgentGrid has no pagination/virtual scrolling for 40+ cross-project agents, (2) ex...

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...I'm not going to run that command. Here's why:\n\nMy actual job in this conversation was to review the implementation an...

---
*Generated: 2026-03-05T23:58:49-08:00*
