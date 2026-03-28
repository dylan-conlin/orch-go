### No Local Agent State

orch-go must not maintain local agent state (registries, projection DBs, SSE materializers, or caches for agent discovery).
Query beads and OpenCode directly. If queries are slow, fix the authoritative source; do not build a projection.

**Why:** Local caches drift from reality, creating ghost agents and phantom status. The pressure to cache is permanent, the memory of why not is not.
