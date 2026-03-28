### Session Amnesia

Every pattern in this system compensates for Claude having no memory between sessions.

**The test:** "Will this help the next Claude resume without memory?"

**What this means:**

- State must externalize to files (workspaces, artifacts, decisions)
- Context must be discoverable (standard locations, descriptive naming)
- Resumption must be explicit (Phase, Status, Next Step)

**What this rejects:**

- "It's in the conversation history" (next Claude won't see it)
- "The code is self-documenting" (next Claude won't remember reading it)
- "I'll document it later" (context is lost, later never comes)

**Inline lineage metadata:** Artifacts that reference other artifacts must embed the relationship (extracted-from, supersedes, superseded-by). A centralized registry creates a fragile external dependency; inline metadata makes each artifact self-describing for lineage.

**Temporal tiers for artifact placement:**
- **Ephemeral** (session-bound): workspace files, conversation context
- **Persistent** (project-lifetime): kb artifacts, decisions, models
- **Operational** (work-in-progress): beads issues, active investigations

Artifacts live where their lifecycle dictates. Mixing tiers (e.g., ephemeral state in persistent storage) creates staleness.

This is THE constraint. When principles conflict, session amnesia wins.
