<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode uses project-specific storage partitioning; sessions are stored under `~/.local/share/opencode/storage/session/{projectID}/` where projectID is derived from the first git root commit hash. Sessions are only accessible when the correct `x-opencode-directory` header is provided.

**Evidence:** API tests confirm: GET `/session/{id}` without header returns NotFoundError (looks in global/); with correct directory header, session is found. Verified by reading OpenCode source: project.ts:44-128 and server.ts:173-181.

**Knowledge:** The directory header doesn't just filter results - it fundamentally determines which project's storage partition is searched. Cross-project visibility requires iterating over all known project directories.

**Next:** Document the pattern; `orch serve` already handles this correctly via `buildMultiProjectWorkspaceCache()`. No code changes needed - the behavior is by design.

---

# Investigation: OpenCode Session Storage Model for Cross-Project Agents

**Question:** Why do cross-project agents not appear in /session API? How does OpenCode decide global vs project-specific storage? Is there an API param to get all sessions?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Session Storage is Partitioned by Project ID

**Evidence:** 
Sessions are stored in directories named by project ID:
- `~/.local/share/opencode/storage/session/global/` - for sessions without git repo
- `~/.local/share/opencode/storage/session/{projectID}/` - for each git project

The session `ses_494dd5506ffe3pGoOteLvelxqv` mentioned in the task exists in:
`~/.local/share/opencode/storage/session/aca13819f57d62c96e5f8c734d7ef8e50377d4fb/`

This corresponds to the price-watch project at `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch`.

**Source:** 
- `ls ~/.local/share/opencode/storage/session/` - showed 20 directories (19 project hashes + global)
- `cat ~/.local/share/opencode/storage/project/aca13819...json` - confirmed worktree mapping

**Significance:** The storage is fundamentally partitioned by project. There is no single "all sessions" location.

---

### Finding 2: Project ID is Derived from Git Root Commit Hash

**Evidence:** 
OpenCode determines project ID in `project.ts:fromDirectory()` (lines 44-93):

```typescript
const roots = await $`git rev-list --max-parents=0 --all`
  .quiet()
  .nothrow()
  .cwd(worktree)
  .text()
  .then((x) =>
    x.split("\n").filter(Boolean).map((x) => x.trim()).toSorted(),
  )
id = roots[0]
```

If no git repo or no root commit exists, `id = "global"`.

**Source:** `~/Documents/personal/opencode/packages/opencode/src/project/project.ts:44-93`

**Significance:** The project ID is stable across clones of the same repo (same root commit hash), but different repos get different IDs. This explains why sessions are isolated by project.

---

### Finding 3: x-opencode-directory Header Determines Storage Partition

**Evidence:**
Server middleware in `server.ts:173-181`:
```typescript
.use(async (c, next) => {
  const directory = c.req.query("directory") ?? c.req.header("x-opencode-directory") ?? process.cwd()
  return Instance.provide({
    directory,
    init: InstanceBootstrap,
    async fn() {
      return next()
    },
  })
})
```

When `Session.get(id)` is called, it uses `Instance.project.id` to locate the session:
```typescript
// session/index.ts:213-216
export const get = fn(Identifier.schema("session"), async (id) => {
  const read = await Storage.read<Info>(["session", Instance.project.id, id])
  return read as Info
})
```

**Source:** 
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts:173-181`
- `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:213-216`

**Significance:** The header is not just a filter - it fundamentally determines which storage partition is searched. Without the correct header, you cannot find sessions from other projects.

---

### Finding 4: No API Parameter for "All Sessions Across Projects"

**Evidence:**
Tested API endpoints:
1. `GET /session` without header → returns 252 sessions (from global/)
2. `GET /session` with `x-opencode-directory: /path/to/price-watch` → returns sessions from that project
3. `GET /project` → returns list of all 20 known projects with their worktrees

There is no `?all=true` or similar parameter to get sessions across all projects in a single call.

**Source:**
```bash
curl -s http://127.0.0.1:4096/session | jq 'length'  # Returns 252 (global only)
curl -s -H "x-opencode-directory: /path/to/orch-go" http://127.0.0.1:4096/session | jq 'length'  # Returns 389
```

**Significance:** Cross-project visibility requires iterating over all project directories.

---

### Finding 5: orch serve Already Handles This Correctly

**Evidence:**
`cmd/orch/serve.go:570-624` implements the correct pattern:

1. Build workspace cache to discover all project directories
2. Query OpenCode sessions for EACH discovered project directory
3. Merge all sessions together with deduplication

```go
// Query OpenCode sessions for each project directory
// This ensures we find sessions created with x-opencode-directory header
var sessions []opencode.Session
seenSessionIDs := make(map[string]bool)

for dir := range projectDirsMap {
    dirSessions, err := client.ListSessions(dir)
    // ... merge with deduplication
}
```

**Source:** `cmd/orch/serve.go:570-624`

**Significance:** The `orch serve` API endpoint already correctly aggregates sessions across projects. The problem in the original observation was likely using a raw OpenCode API call without the aggregation logic.

---

## Synthesis

**Key Insights:**

1. **Storage Partitioning is by Design** - OpenCode partitions session storage by project to enable multi-project workflows without namespace collisions. Each project gets its own storage partition.

2. **The Header is the Partition Key** - The `x-opencode-directory` header (or query param) is not just a filter - it determines which partition is searched. This is similar to a tenant ID in multi-tenant systems.

3. **Cross-Project Visibility Requires Iteration** - There is no "show all" API. Cross-project visibility requires knowing all project directories and querying each one.

**Answer to Investigation Question:**

(1) **How does OpenCode decide global vs project-specific storage?** 
OpenCode uses the first git root commit hash as the project ID. If no git repo exists or has no commits, it uses "global". The project ID determines the storage directory.

(2) **Is there an API param to get all sessions?**
No. You must query each project directory separately using the `x-opencode-directory` header and merge the results.

(3) **Can orch spawn control storage location?**
Yes. When `CreateSession` is called with a `directory` parameter, and the `x-opencode-directory` header is set, the session is stored in that project's partition. The `opencode.NewClientWithDirectory()` function is used to set this header on all API calls.

(4) **What's the right fix?**
The system works as designed. `orch serve` correctly handles cross-project aggregation by:
- Building a workspace cache from known workspaces
- Extracting unique project directories
- Querying each project directory separately
- Merging results with deduplication

For direct API access, callers must either:
- Know the correct project directory for the session they're querying
- Use the `/project` endpoint to list all projects, then query each one

---

## Structured Uncertainty

**What's tested:**

- ✅ GET `/session/{id}` without header returns NotFoundError (tested via curl)
- ✅ GET `/session/{id}` with correct directory header returns session (tested via curl)
- ✅ GET `/session/{id}` with wrong directory header returns NotFoundError (tested via curl)
- ✅ Project ID is derived from git root commit hash (verified in source code)
- ✅ `orch serve` aggregates sessions across all discovered project directories (verified in source code)

**What's untested:**

- ⚠️ Behavior when git repo has multiple root commits (assumed: first sorted one is used)
- ⚠️ Session migration when project ID changes (code exists in project.ts but not tested)
- ⚠️ Performance impact of querying many project directories

**What would change this:**

- Finding would be wrong if OpenCode added a `?scope=all` parameter to the `/session` endpoint
- Finding would be incomplete if there's a caching layer that bypasses the directory check

---

## Implementation Recommendations

**Purpose:** The investigation found this is working as designed. No fix needed.

### Recommended Approach: Documentation

The system is correctly designed and implemented. The "fix" is understanding how it works:

1. **When using raw OpenCode API:** Always include `x-opencode-directory` header for project-specific sessions
2. **When building cross-project tools:** Use `/project` to list all projects, then query each one's sessions
3. **When using orch tools:** They already handle this via `orch serve` aggregation

### Alternative Approaches Considered

**Option B: Force all sessions to global storage**
- **Pros:** Simpler cross-project visibility
- **Cons:** Breaks multi-project isolation, namespace collisions possible
- **When to use instead:** Never - current design is correct

**Option C: Add "all sessions" API to OpenCode**
- **Pros:** Single API call for cross-project visibility
- **Cons:** Requires upstream OpenCode change
- **When to use instead:** If orch becomes widely used and this pattern is needed by others

---

## References

**Files Examined:**
- `~/Documents/personal/opencode/packages/opencode/src/project/project.ts` - Project ID derivation logic
- `~/Documents/personal/opencode/packages/opencode/src/session/index.ts` - Session storage logic
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts` - API middleware
- `~/Documents/personal/opencode/packages/opencode/src/storage/storage.ts` - Storage layer
- `~/Documents/personal/orch-go/cmd/orch/serve.go` - Cross-project session aggregation
- `~/Documents/personal/orch-go/pkg/opencode/client.go` - Client implementation

**Commands Run:**
```bash
# Test session lookup without header
curl -s http://127.0.0.1:4096/session/ses_494dd5506ffe3pGoOteLvelxqv
# Result: NotFoundError

# Test session lookup with correct header
curl -s -H "x-opencode-directory: /path/to/price-watch" http://127.0.0.1:4096/session/ses_494dd5506ffe3pGoOteLvelxqv
# Result: Session found

# List project mappings
cat ~/.local/share/opencode/storage/project/aca13819...json
# Result: {"id": "aca...", "worktree": "/path/to/price-watch", ...}
```

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED
