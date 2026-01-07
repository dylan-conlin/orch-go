## Summary (D.E.K.N.)

**Delta:** Enhanced `orch doctor --sessions` to cross-reference workspaces, OpenCode sessions, AND orchestrator registry to detect orphaned and zombie sessions.

**Evidence:** Tested on orch-go with 302 workspaces, 171 sessions - correctly identified 271 orphaned workspaces, 140 orphaned sessions, 0 zombies; tests pass.

**Knowledge:** Orchestrator sessions use `~/.orch/sessions.json` registry which lacks session IDs (they're empty) for most entries, making registry-based zombie detection ineffective without session ID population.

**Next:** Close - implementation complete. Follow-up: Consider populating session_id in registry when spawning orchestrators.

---

# Investigation: Add Orch Doctor Sessions Workspace

**Question:** How to cross-reference workspaces, OpenCode sessions, and registry to detect orphaned/zombie sessions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing implementation only covered 2 of 3 layers

**Evidence:** The original `runSessionsCrossReference()` only cross-referenced workspaces ↔ OpenCode sessions, missing the orchestrator registry entirely.

**Source:** `cmd/orch/doctor.go:544-670` (original implementation)

**Significance:** Registry entries can become stale if sessions are garbage-collected, leading to registry mismatches that weren't detected.

---

### Finding 2: Registry session IDs are mostly empty

**Evidence:** Examining `~/.orch/sessions.json`, most orchestrator sessions have empty `session_id` fields. Only newer sessions have IDs populated.

```json
{
  "workspace_name": "meta-orch-continue-meta-orch-06jan-2c9a",
  "session_id": "",  // Empty!
  "project_dir": "/Users/dylanconlin/Documents/personal/orch-go",
  "status": "active"
}
```

**Source:** `cat ~/.orch/sessions.json`

**Significance:** Registry-based zombie detection relies on session IDs to match against OpenCode sessions. Empty IDs mean we can't detect zombies for older orchestrator sessions.

---

### Finding 3: Zombie detection requires both session freshness and registry state

**Evidence:** A zombie session is one that:
1. Has a workspace (was spawned by orch)
2. Hasn't been updated in >30 minutes
3. Is still marked as "active" in registry

**Source:** Implementation in `runSessionsCrossReference()` step 6

**Significance:** The three-way check (workspace + OpenCode API + registry) catches sessions that appear active in registry but have gone silent.

---

## Synthesis

**Key Insights:**

1. **Three-layer architecture** - The orch system has three independent state stores (workspace files, OpenCode sessions, registry JSON) that can drift out of sync.

2. **Registry population gap** - Session IDs aren't being populated for orchestrator sessions in the registry, limiting the effectiveness of registry-based checks.

3. **Clean output format** - The new summary format matches the spec and provides actionable recommendations.

**Answer to Investigation Question:**

Cross-referencing requires:
1. Reading `.orch/workspace/*/.*` files for workspace → session mapping
2. Calling `client.ListDiskSessions()` for OpenCode session state
3. Reading `~/.orch/sessions.json` for orchestrator registry
4. Comparing all three to find: orphaned workspaces (session deleted), orphaned sessions (no workspace), zombies (active in registry but idle), registry mismatches (session ID not found)

---

## Structured Uncertainty

**What's tested:**

- ✅ Cross-reference logic correctly identifies orphaned workspaces (verified: 271 found on orch-go)
- ✅ Cross-reference logic correctly identifies orphaned sessions (verified: 140 found)
- ✅ Empty session IDs handled gracefully (verified: test passes)
- ✅ Output format matches spec (verified: visual inspection)

**What's untested:**

- ⚠️ Zombie detection with populated registry session IDs (no test data available)
- ⚠️ Performance with very large workspace counts (>1000)

**What would change this:**

- If registry started populating session IDs, zombie detection would become more accurate
- If OpenCode changed its session API, ListDiskSessions might need updates

---

## Implementation Recommendations

### Recommended Approach (Implemented)

Enhanced `runSessionsCrossReference()` with:
- Registry loading from `~/.orch/sessions.json`
- Zombie detection (idle >30min but active in registry)
- Registry mismatch detection (session ID not found in OpenCode)
- Clean summary output format

**Implementation sequence:**
1. Build workspace → session maps (existing)
2. Load OpenCode sessions via API (existing)
3. Load registry from JSON file (new)
4. Cross-reference all three (enhanced)
5. Detect zombies using idle time + registry status (new)
6. Print summary in spec format (new)

---

## References

**Files Examined:**
- `cmd/orch/doctor.go` - Main implementation
- `pkg/session/registry.go` - Registry types (used for reference)
- `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Architecture context

**Commands Run:**
```bash
# Test command
go run ./cmd/orch doctor --sessions

# Check registry structure
cat ~/.orch/sessions.json | head -50
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Full architecture context

---

## Investigation History

**2026-01-06 17:30:** Investigation started
- Initial question: How to add registry and zombie detection to orch doctor --sessions?
- Context: Spawned from beads issue orch-go-0l2f9

**2026-01-06 17:45:** Implementation completed
- Enhanced `runSessionsCrossReference()` with all three layers
- Added tests for new functionality
- All tests passing

**2026-01-06 17:50:** Investigation completed
- Status: Complete
- Key outcome: `orch doctor --sessions` now cross-references workspaces, sessions, and registry with zombie detection
