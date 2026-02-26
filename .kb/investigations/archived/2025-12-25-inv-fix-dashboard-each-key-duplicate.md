## Summary (D.E.K.N.)

**Delta:** Dashboard each blocks use workspace name (agent.id) as key, but workspace names can duplicate on daemon/manual respawn, causing Svelte each_key_duplicate errors.

**Evidence:** Agent interface shows `id` is workspace name, `session_id` is the unique identifier. Four {#each} blocks in +page.svelte keyed by `agent.id`.

**Knowledge:** Session IDs are guaranteed unique per OpenCode session. Workspace names can repeat when agents are respawned for the same task.

**Next:** Change all `(agent.id)` keys to `(agent.session_id ?? agent.id)` with fallback for agents without session_id.

**Confidence:** High (95%) - Clear root cause, straightforward fix.

---

# Investigation: Fix Dashboard Each Key Duplicate

**Question:** Why does the dashboard throw each_key_duplicate Svelte errors and how to fix it?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: Agent ID is workspace name, not session ID

**Evidence:** In `web/src/lib/stores/agents.ts` line 15-18:
```typescript
export interface Agent {
  id: string;           // Workspace name (e.g., "og-feat-fix-dashboard-each-25dec")
  session_id?: string;  // Unique OpenCode session ID
  ...
}
```

**Source:** web/src/lib/stores/agents.ts:15-18

**Significance:** The `id` field represents the workspace name, which can be reused when an agent is respawned (daemon respawn, manual respawn). The `session_id` is the unique identifier.

---

### Finding 2: Four each blocks use agent.id as key

**Evidence:** In `web/src/routes/+page.svelte`:
- Line 497: `{#each sortedActiveAgents as agent (agent.id)}`
- Line 520: `{#each sortedActiveAgents as agent (agent.id)}`
- Line 537: `{#each sortedRecentAgents as agent (agent.id)}`
- Line 554: `{#each sortedArchivedAgents as agent (agent.id)}`

**Source:** web/src/routes/+page.svelte:497,520,537,554

**Significance:** These are the locations causing the each_key_duplicate error when multiple agents share the same workspace name.

---

## Fix

Change all `(agent.id)` keys to `(agent.session_id ?? agent.id)`:
- Uses session_id when available (unique per session)
- Falls back to workspace id for agents without session_id
- Guarantees unique keys
