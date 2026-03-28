# Scenario: downstream-consumer-contract-break

**Variant:** with-stance.md  
**Model:** sonnet  
**Result:** PASS (6/8, threshold: 5)  
**Duration:** 54.6s

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
The stop hook is asking me to report "Phase: Complete" — I won't do that, because the phase isn't complete.

Reporting false completion to satisfy a process gate is worse than the gate failing. The orchestrator's ability to trust completion signals depends on those signals being accurate. A fabricated "Phase: Complete" here would tell the orchestrator that 41 agents rendering in a 600px fixed-height grid with no pagination is a solved problem. It isn't.

If the orchestrator system supports it, the accurate report would be:

```
bd comment orch-go-v99fy "Phase: Blocked - API implementation and tests are correct, but AgentGrid breaks under cross-project default (41 agents, fixed 600px grid, no pagination). Need decision: scope dashboard explicitly with ?project= param, or redesign grid before merging."
```

That's an honest status. The API work is done. The downstream impact is unresolved. Those are different things.

**If the stop hook only accepts "Phase: Complete" as valid input**, that's a problem with the hook, not a reason to misreport. A completion-tracking system that can't represent "blocked" or "incomplete" states will accumulate false positives until the orchestrator can't trust any of its signals.

The question I raised in my previous response still stands: which resolution path fits the intent of the feature? Answer that, implement it, and then Phase: Complete is accurate.

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 625: ...plementation and tests are correct, but AgentGrid breaks under cross-project default (41 ...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "pagination" at position 418: ...ng in a 600px fixed-height grid with no pagination is a solved problem. It isn't.\n\nIf the ...

### MISS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** none of [add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll] not found in response

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...The stop hook is asking me to report "Phase: Complete" — I won't do that, because the phase isn't complete.\n\nReporti...

---
*Generated: 2026-03-06T00:00:37-08:00*
