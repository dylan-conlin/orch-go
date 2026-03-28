# Probe: Pulse Signal — Liveness Assurance Is a Third Information Category

**Model:** dashboard-architecture
**Date:** 2026-03-27
**Status:** Complete
**claim:** DA-01
**verdict:** extends

---

## Question

The dashboard-architecture model (invariant #1) states "Home page is comprehension-only — execution residue subtracted." The execution subtraction probe (2026-03-26) confirmed this was a clean cut with no shared state between comprehension and execution layers.

But Dylan still reaches for tmux to check if agents are alive. Sketchybar gives counts (1/5 C:0) but when counts don't match expectations, the fallback is tmux, not the dashboard. The dashboard has "flashes of usefulness but nothing that really clicks."

**Claim under test:** The comprehension/execution binary is complete — all information needs fall into one or the other. If this is true, then either the comprehension surface or the work-graph surface should satisfy the "is the system alive?" question. If neither does, there's a third category the model doesn't recognize.

---

## What I Tested

### Test 1: What SSE data reaches the home page browser?

Connected to the live SSE endpoint to observe event flow:

```bash
curl -s -N --max-time 5 http://localhost:3348/api/events 2>/dev/null | head -30
# Output:
# event: connected
# data: {"source": "http://127.0.0.1:4096/event"}
# data: {"type":"server.connected","properties":{}}
```

Checked what the `/api/agents` endpoint returns for active agents:

```bash
curl -s http://localhost:3348/api/agents | python3 -c "..."
# Output:
#   orch-go-s1r5m | session_id=@1053 | processing=None | activity=none
#   orch-go-mlqje | session_id=@1021 | processing=True | activity=none
# Total: 15 agents, 2 active
```

### Test 2: Traced the activity data flow through the code

1. **Backend** (`serve_agents_types.go`): `AgentAPIResponse` has `CurrentActivity string` and `LastActivityAt string` — but API returns empty for both.

2. **Frontend SSE handler** (`agents.ts:706-731`): On `message.part` events, updates `current_activity` in the Svelte store:
   ```typescript
   // agents.ts:720-727
   return {
       ...agent,
       is_processing: true,
       current_activity: {
           type: part.type,
           text: part.text || extractActivityText(part),
           timestamp: Date.now()
       }
   };
   ```

3. **Home page SSE connection** (`+page.svelte:195`): `connectSSE()` IS called — the home page receives SSE events and the agent store IS updated with `current_activity`.

4. **Home page rendering** (`+page.svelte:475-486`): Only uses `$trulyActiveAgents.length`:
   ```svelte
   <span class="text-xs text-muted-foreground">
       {$trulyActiveAgents.length} agent{$trulyActiveAgents.length === 1 ? '' : 's'} active
       · {$readyIssues?.count ?? 0} ready
       · {$needsReviewAgents.length} need review
   </span>
   ```

5. **Agent card on work-graph** (`agent-card.svelte:211-249`): Has `getExpressiveStatus()` that renders "Hatching... (thought for 8s)", "Running Bash...", "Reading files..." — this IS pulse rendering, but it's on the work-graph page only.

### Test 3: Checked liveness data availability per backend type

- **OpenCode agents** (session_id `@NNNN`): Get real-time `message.part` SSE events → `current_activity` populated in frontend store
- **Claude agents** (tmux-based): NO SSE events from OpenCode. Only liveness signal is phase comments in beads (periodic, not real-time)

### Test 4: Measured refresh intervals on home page

```
// +page.svelte:218 - refreshInterval
const refreshInterval = setInterval(() => { ... }, 60000);  // 60 seconds

// agents.ts:193 - SSE debounce
const FETCH_DEBOUNCE_MS = 500;  // 500ms debounce on SSE-triggered refetch

// agents.ts:203 - Processing clear delay
const PROCESSING_CLEAR_DELAY_MS = 5000;  // 5s before clearing is_processing
```

SSE events update agent store within 500ms. But the home page only reads `.length` from the store — no visual change occurs unless an agent count changes.

---

## What I Observed

### Observation 1: Pulse Data Arrives But Isn't Rendered

The home page connects to SSE, receives `message.part` events, and updates `current_activity` on agent objects in the Svelte store. This data includes activity type (text/tool/reasoning), descriptive text ("Using Bash", "Reading files"), and millisecond-precision timestamps.

The home page then discards all of this by rendering only `$trulyActiveAgents.length` — a count. The full Agent objects with their real-time activity are available in the store but never rendered.

The `getExpressiveStatus()` function in `agent-card.svelte` already converts raw activity into human-readable pulse strings ("Hatching... (thought for 12s)", "Running Bash...", "Searching code..."). This function exists and works. It just isn't called from the home page.

### Observation 2: The Comprehension/Execution Binary Has a Gap

When Dylan checks tmux, he's not seeking comprehension (what agents learned) or execution detail (which files they're editing). He's seeking **liveness assurance** — visual proof the system is alive. This is a distinct information need:

| Category | Question | Signal type | Example | Location |
|----------|----------|-------------|---------|----------|
| Comprehension | What did agents learn? | Content (prose) | Thread entries, briefs | Home page |
| Execution | What are agents doing in detail? | State + stream | Tool calls, file diffs, event log | Work graph |
| **Liveness** | **Is the system alive?** | **Activity + recency** | **"Running Bash... 3s ago"** | **Nowhere** |

The execution subtraction correctly removed detailed execution monitoring from the home page. But it also removed liveness assurance because the model didn't distinguish between "execution detail" (agent grids, event streams — requires attention) and "liveness proof" (ambient activity indicator — requires only peripheral vision).

Tmux provides liveness through continuous visual change — text scrolling by. The dashboard provides static counts that require trust ("it says 3 active, but are they really doing anything?"). The missing element isn't data — it's **recency proof**: a timestamp that visibly ticks forward.

### Observation 3: Two-Tier Liveness Creates a Growing Gap

- **Claude-backend agents** have tmux windows → visual pulse is available via terminal
- **OpenCode-backend agents** have SSE events → pulse data exists but isn't rendered anywhere except 2+ clicks deep on work-graph

As more agents run on OpenCode (GPT-5.4 agents already have no tmux window), the population of agents with NO visible pulse grows. The dashboard is the ONLY monitoring surface for OpenCode agents, and it renders their activity as a count.

### Observation 4: The Condensed Summary Is the Right Location, Wrong Content

The condensed operational summary (`+page.svelte:475-486`) is already positioned at the bottom of the home page as the bridge between comprehension and execution. It already imports `$trulyActiveAgents` which contains full Agent objects with `current_activity`.

The minimum transformation: render 1-2 lines of activity from `$trulyActiveAgents` using the existing `getExpressiveStatus()` function and `formatActivityAge()` timestamp formatter. Both functions already exist in `agent-card.svelte`.

**Before (static count):**
> 3 agents active · 2 ready · 1 need review — View Work →

**After (live pulse):**
> 3 agents active · 2 ready · 1 need review — View Work →
> s1r5m: Reading files... 3s · mlqje: Running Bash... 12s

The timestamp ticking forward IS the pulse. If "3s" stays "3s" for 30 seconds, something is wrong. This is what tmux provides and the dashboard doesn't: visual proof of flow, not just state.

---

## Model Impact

- [x] **Extends** model with: The comprehension/execution binary is incomplete. There's a third information category — **liveness assurance** — that is neither comprehension (what was learned) nor execution (what's being done in detail). Liveness assurance is ambient proof that the system is alive, analogous to a typing indicator in chat or the hum of a server room. The execution subtraction correctly removed execution detail from the home page but accidentally removed liveness assurance because it wasn't recognized as a separate category.

- [x] **Extends** invariant #1 with refinement: "Home page is comprehension-only" should become "Home page is comprehension-first with ambient liveness." The distinction: execution monitoring requires attention (scanning grids, reading event streams), while liveness assurance requires only peripheral vision (glancing at a ticking timestamp).

- [x] **Confirms** invariant #10 (content vs metadata mode): The condensed summary's counts ("3 active") are metadata mode. Activity descriptions ("Reading files... 3s") are a third mode — **presence mode** — that provides proof of ongoing activity without the cognitive overhead of full execution content.

---

## Notes

- The `getExpressiveStatus()` and `formatActivityAge()` functions in `agent-card.svelte` would need to be extracted to a shared utility for home page use. This is a ~10-line extraction, not a new capability.
- Claude-backend agents can't provide real-time pulse through SSE. Their liveness would remain phase-based ("Planning — 4m ago"). This is still better than a bare count, and honestly more accurate than tmux window existence checks (which the Feb 24 investigation showed are unreliable).
- The 5s `PROCESSING_CLEAR_DELAY_MS` debounce means the home page pulse would show activity for ~5s after each tool call, then go quiet until the next one. For agents actively working, this means near-continuous pulse. For agents thinking/reasoning, there'd be gaps — which is accurate signal.
- **Critical routing note:** This probe identifies a rendering gap and recommends a specific UI change to the home page. Per investigation skill guidance, UI implementation recommendations must route through architect before implementation.
