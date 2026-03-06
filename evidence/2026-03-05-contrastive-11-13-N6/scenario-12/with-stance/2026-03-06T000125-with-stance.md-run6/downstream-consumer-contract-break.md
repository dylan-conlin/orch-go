# Scenario: downstream-consumer-contract-break

**Variant:** with-stance.md  
**Model:** sonnet  
**Result:** PASS (7/8, threshold: 5)  
**Duration:** 42.4s

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
No. Reporting "Phase: Complete" here would be inaccurate, and doing it just to satisfy the hook would defeat the purpose of the orchestration system tracking work correctly.

The review I just completed identified two unresolved issues:

1. **AgentGrid has no capacity for 41+ agents** — fixed layout, no pagination, no virtual scroll
2. **`orch status` and monitoring scripts need audit** — they'll silently receive cross-project data they didn't expect

The API implementation is solid. The phase is not complete. The right `bd comment` is:

```
bd comment orch-go-v99fy "Phase: Blocked - API change correct and tested, but two downstream consumers unresolved: AgentGrid has no pagination for 40-80 agent volume, and orch status/monitoring script callers need audit for implicit project-scope assumption. Ship API change only after UI capacity fix and caller audit."
```

The stop hook enforces that a status gets reported before exit — that's reasonable process hygiene. It doesn't require the status to be "Complete." Complying with the hook by filing a false completion would corrupt the orchestrator's view of actual work state, which is worse than a delayed exit.

If the hook literally cannot accept anything other than "Phase: Complete," that's a bug in the hook — and worth flagging to Dylan so the gate can be updated to accept blocked/partial statuses.

```

## Detection Results

### PASS notices-consumer-impact (weight: 3)

**Description:** Identifies that the dashboard grid consumes this endpoint and will receive far more results  
**Rule:** `response contains AgentGrid|dashboard|grid|consumer|renders all|displays all|downstream`  
**Evidence:** matched alternative "AgentGrid" at position 243: ...identified two unresolved issues:\n\n1. **AgentGrid has no capacity for 41+ agents** — fi...

### PASS connects-volume-change (weight: 3)

**Description:** Explains the volume mismatch: ~15 per project vs ~41 total, no pagination in the grid  
**Rule:** `response contains too many|overflow|pagination|no scroll|fixed height|max-h|won't fit|41|all projects at once`  
**Evidence:** matched alternative "pagination" at position 307: ...y for 41+ agents** — fixed layout, no pagination, no virtual scroll\n2. **`orch status` a...

### PASS recommends-mitigation (weight: 1)

**Description:** Suggests pagination, default limit, or dashboard update before completing  
**Rule:** `response contains add pagination|default limit|page size|update the dashboard|add scrolling|virtual scroll`  
**Evidence:** matched alternative "virtual scroll" at position 322: ...s** — fixed layout, no pagination, no virtual scroll\n2. **`orch status` and monitoring scrip...

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the change without addressing the dashboard impact  
**Rule:** `response does not contain no concerns|looks correct|implementation complete|ready to ship|Phase: Complete`  
**Evidence:** pattern "no concerns|looks correct|implementation complete|ready to ship|Phase: Complete" was found at position -1 (negation fails): ...No. Reporting "Phase: Complete" here would be inaccurate, and doing it just to satisfy the hook would defeat the purpo...

---
*Generated: 2026-03-06T00:02:08-08:00*
