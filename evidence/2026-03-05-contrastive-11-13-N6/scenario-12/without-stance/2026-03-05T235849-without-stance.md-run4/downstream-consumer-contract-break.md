# Scenario: downstream-consumer-contract-break

**Variant:** without-stance.md  
**Model:** sonnet  
**Result:** PASS (7/8, threshold: 5)  
**Duration:** 29.1s

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
Two concerns before marking complete.

**1. The dashboard will overflow and clip silently.**

Current totals: 15 + 8 + 12 + 6 = 41 agents. In a `grid-cols-4 max-h-[600px]` with no scroll or pagination that's ~10+ rows against a hard pixel cap. Agents past the fold are invisible with no affordance that anything is hidden. The grid change makes programmatic callers better and the dashboard silently worse — that's a regression that the tests won't catch because they test the endpoint, not the rendered output.

**2. AgentGrid wasn't updated, so it implicitly opted into cross-project by accident.**

The component still calls `/api/v1/agents` with no params. Before your change it got the current project's agents via server-side `projectDir` context. Now it gets everything. The behavior changed without the component knowing it changed. If the dashboard is supposed to show cross-project, it should do so intentionally — with grouping headers, a filter, or explicit UX. If it's supposed to stay project-scoped, it needs `?project=<currentProject>`.

**The decision you need to make first:**

Should the dashboard show cross-project agents, or remain project-scoped while the endpoint enables cross-project for programmatic callers (orch status, monitoring scripts)?

- If **project-scoped**: update AgentGrid to pass `?project=` explicitly. The endpoint change is right, the dashboard just needs to opt back in to scoped behavior.
- If **cross-project**: AgentGrid needs a real fix — virtual scroll or pagination, project grouping headers, and a filter. The current fixed layout doesn't survive 40+ agents.

Either way, the backend is solid and the 5/5 tests are correct. But "Phase: Complete" isn't accurate until the UI side of the behavior change is resolved. Which direction do you want to go?

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 520: ...ndpoint, not the rendered output.\n\n**2. AgentGrid wasn't updated, so it implicitly opted ...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "overflow" at position 63: ...king complete.\n\n**1. The dashboard will overflow and clip silently.**\n\nCurrent totals: 1...

### PASS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** matched alternative "virtual scroll" at position 1494: ...oject**: AgentGrid needs a real fix — virtual scroll or pagination, project grouping headers...

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...Two concerns before marking complete.\n\n**1. The dashboard will overflow and clip silently.**\n\nCurrent totals: 15 + 8 +...

---
*Generated: 2026-03-05T23:59:18-08:00*
