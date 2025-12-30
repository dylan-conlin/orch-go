<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard is coupled to orch-go project via compile-time `sourceDir` variable embedded in binary, plus beads socket discovery that walks up directory tree from `sourceDir`.

**Evidence:** serve.go uses `sourceDir` (lines 161-162, 645-646, 1153-1154, 2757, 2998, 3160) for workspace paths and beads socket discovery; frontend hardcodes `localhost:3348` API base.

**Knowledge:** The coupling is architectural (build-time project embedding) not just configuration. Fixing requires either: (1) multi-project API design where each project runs its own `orch serve`, or (2) a global orchestration server that queries multiple project directories.

**Next:** Create beads issue for "project-agnostic dashboard" feature; recommend Option A (multi-project API design) as least invasive path.

---

# Investigation: Web Dashboard Coupling to orch-go

**Question:** What dashboard features assume orch-go as the project, and what would 'project-agnostic orchestration dashboard' require?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Spawned agent (og-inv-web-dashboard-coupling-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Build-time `sourceDir` embeds project path into binary

**Evidence:** The `sourceDir` variable is set at compile time via ldflags in the Makefile:
```go
// cmd/orch/main.go:44
sourceDir = "unknown" // Absolute path to source directory
```

This is populated during `make install` with the absolute path to the orch-go repository. When `orch serve` runs:

1. **Beads socket discovery** (serve.go:161-167):
   ```go
   if sourceDir != "" && sourceDir != "unknown" {
       beads.DefaultDir = sourceDir
   }
   socketPath, err := beads.FindSocketPath(sourceDir)
   ```

2. **Workspace path construction** (serve.go:2757, 2998, 3160):
   ```go
   workspaceDir := filepath.Join(sourceDir, ".orch", "workspace")
   workspacePath := filepath.Join(sourceDir, ".orch", "workspace", req.WorkspaceID)
   ```

3. **Project context for agents** (serve.go:645-650):
   ```go
   projectDir := sourceDir
   if projectDir == "" || projectDir == "unknown" {
       projectDir, _ = os.Getwd()
   }
   ```

**Source:** 
- `cmd/orch/main.go:44` (variable declaration)
- `cmd/orch/serve.go:161-167, 645-650, 1153-1154, 2757, 2998, 3160` (usage sites)

**Significance:** **CRITICAL BLOCKER** - The dashboard is permanently bound to the project where `orch-go` was compiled. Running `orch serve` from a different project doesn't change the context because `sourceDir` is baked into the binary.

---

### Finding 2: Beads integration uses local .beads/ directory

**Evidence:** The `pkg/beads/client.go` finds the beads socket by walking up the directory tree:

```go
// pkg/beads/client.go:78-106
func FindSocketPath(dir string) (string, error) {
    if dir == "" {
        if DefaultDir != "" {
            dir = DefaultDir
        } else {
            var err error
            dir, err = os.Getwd()
        }
    }
    // Walk up directory tree looking for .beads/bd.sock
    current := dir
    for { ... }
}
```

The `DefaultDir` is set from `sourceDir` in serve.go, meaning:
- Beads stats (`/api/beads`) reflect orch-go's `.beads/` 
- Ready issues (`/api/beads/ready`) show orch-go issues only
- Issue creation (`POST /api/issues`) creates in orch-go's beads database

**Source:** 
- `pkg/beads/client.go:76-106` (FindSocketPath)
- `cmd/orch/serve.go:161-162` (DefaultDir setting)

**Significance:** **CRITICAL BLOCKER** - Dashboard can only show/manage beads issues from orch-go, not from price-watch or other projects.

---

### Finding 3: Multi-project discovery exists but is incomplete

**Evidence:** The serve.go has sophisticated multi-project agent discovery via OpenCode session storage:

```go
// serve.go:367-429
func discoverAllProjectDirs() []string {
    // Scans ~/.local/share/opencode/storage/session/{partition_hash}/
    // Each session JSON has "directory" field with project path
}
```

This enables cross-project agent visibility in `/api/agents`, but:
1. **Beads operations still use single project** - `beads.DefaultDir` is set once at startup
2. **Workspace paths hardcoded to sourceDir** - Can't access workspaces from other projects
3. **SSE multiplexing works** - serve.go:1128-1279 multiplexes SSE from all discovered projects

**Source:**
- `cmd/orch/serve.go:367-429` (discoverAllProjectDirs)
- `cmd/orch/serve.go:664-707` (handleAgents using multi-project)
- `cmd/orch/serve.go:1128-1279` (handleEvents SSE multiplexing)

**Significance:** The infrastructure for multi-project exists for agents but wasn't extended to beads, workspaces, or kb.

---

### Finding 4: Frontend hardcodes API base URL

**Evidence:** All stores in `web/src/lib/stores/*.ts` use:

```typescript
// web/src/lib/stores/agents.ts:105
const API_BASE = 'http://localhost:3348';
```

Same pattern in `beads.ts:4`, `usage.ts`, `focus.ts`, `servers.ts`, `daemon.ts`, `pending-reviews.ts`, `gaps.ts`, `config.ts`, `patterns.ts`.

**Source:** All files in `web/src/lib/stores/`

**Significance:** **COSMETIC** - The hardcoded URL is correct (orch serve runs on 3348). This would only matter if we wanted to support connecting to different orch instances.

---

### Finding 5: kb/knowledge integration points

**Evidence:** The dashboard fetches gap analysis from `~/.orch/gap-tracker.json` (serve.go:2242-2297) and reflect suggestions from `~/.orch/reflect-suggestions.json` (serve.go:2349-2443). These are user-global files, not project-specific.

However, investigation artifacts live in project-specific `.kb/` directories, and the dashboard doesn't surface these.

**Source:**
- `cmd/orch/serve.go:2242-2297` (handleGaps)
- `cmd/orch/serve.go:2349-2443` (handleReflect)

**Significance:** **LOW** - Global files work fine; project-specific kb is not surfaced in dashboard.

---

## Synthesis

**Key Insights:**

1. **Build-time coupling is the root cause** - The `sourceDir` compile-time embedding creates a fundamental assumption that orch serve = orch-go project. This isn't configuration that can be changed at runtime.

2. **Multi-project agents already work** - The OpenCode session discovery and SSE multiplexing patterns show how to support multiple projects. The same pattern could be extended to beads operations.

3. **Three layers of coupling:**
   - **Critical:** Workspace paths (`{sourceDir}/.orch/workspace/`)
   - **Critical:** Beads socket discovery (`{sourceDir}/.beads/bd.sock`)
   - **Cosmetic:** Frontend API base URL (would need environment variable for different hosts)

**Answer to Investigation Question:**

The dashboard assumes orch-go context through the build-time `sourceDir` variable that's embedded in the binary. Specifically:

1. **Features assuming orch-go:**
   - `/api/beads` (stats), `/api/beads/ready` (issues) - queries orch-go's beads only
   - `/api/pending-reviews`, `/api/dismiss-review`, `/api/act-on-review` - workspace paths
   - Workspace discovery for synthesis and gap analysis

2. **Features that ARE project-agnostic:**
   - `/api/agents` - already discovers sessions across all projects via OpenCode storage
   - `/api/events` - already multiplexes SSE from all project directories
   - `/api/usage`, `/api/focus` - user-global, not project-specific
   - `/api/gaps`, `/api/reflect` - read from `~/.orch/` (user-global)

3. **What project-agnostic dashboard requires:**
   - **Option A (Multi-project API):** Accept `project` query param on beads/workspace endpoints; maintain multiple beads client connections
   - **Option B (Global orchestration server):** Run `orch serve` from a neutral location (like `~/.orch/`) and configure it to aggregate all projects
   - **Option C (Per-project serve instances):** Run `orch serve` in each project, dashboard connects to multiple ports

---

## Structured Uncertainty

**What's tested:**

- ✅ sourceDir is used in serve.go for workspace paths (verified: grep found 10 usage sites)
- ✅ beads.DefaultDir is set from sourceDir at startup (verified: serve.go:161-162)
- ✅ Multi-project agent discovery works via discoverAllProjectDirs (verified: code review)

**What's untested:**

- ⚠️ Whether running `orch serve` from price-watch actually fails (not run)
- ⚠️ Performance impact of maintaining multiple beads client connections
- ⚠️ Whether OpenCode session storage reliably contains all active projects

**What would change this:**

- Finding would be wrong if `sourceDir` could be overridden at runtime (but Makefile shows it's ldflags only)
- Multi-project approach might not work if beads daemon only allows one client connection

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Option A - Multi-Project API with Query Parameter

**Why this approach:**
- Minimal changes to existing architecture - single serve instance
- Leverages existing multi-project discovery code for agents
- Beads client already supports `WithCwd(cwd)` option for per-request context
- Dashboard already has project filter UI (could extend to project selector)

**Trade-offs accepted:**
- Beads clients for each project need to be lazily initialized (complexity)
- Workspace paths require project context in API calls (more params)
- Can't aggregate beads stats across projects (one project at a time view)

**Implementation sequence:**
1. Add `--workdir` flag to `orch serve` (override sourceDir at runtime)
2. Add `project` query param to `/api/beads/*` endpoints
3. Lazily create beads clients per project directory
4. Add project selector to dashboard UI

### Alternative Approaches Considered

**Option B: Global orchestration server**
- **Pros:** Single source of truth, true aggregation
- **Cons:** Major architectural change; needs new directory structure; beads can't aggregate across repos
- **When to use instead:** If we want unified orchestration separate from any project

**Option C: Per-project serve instances**
- **Pros:** Each project is fully self-contained
- **Cons:** Multiple ports to manage; dashboard would need multi-instance support
- **When to use instead:** If projects have conflicting configurations

**Rationale for recommendation:** Option A is least invasive, reuses existing patterns (multi-project discovery, beads client options), and matches user mental model (orchestrator in orch-go managing work across projects).

---

### Implementation Details

**What to implement first:**
1. Add `--workdir` runtime flag to override sourceDir (unblocks testing)
2. Refactor handleBeads to accept project context
3. Create BeadsClientPool for lazy per-project client management

**Things to watch out for:**
- ⚠️ Beads socket path is per-project (`.beads/bd.sock`) - client pool needs to track paths
- ⚠️ Cross-project spawns already use `--workdir` flag - leverage same pattern
- ⚠️ Workspace discovery needs to scan multiple `.orch/workspace/` directories

**Areas needing further investigation:**
- How to handle issue creation when no project context specified
- Whether to add project selector to dashboard header vs per-section

**Success criteria:**
- ✅ Dashboard shows beads stats from price-watch when spawned there
- ✅ Ready issues from multiple projects visible with project filter
- ✅ Workspace synthesis accessible regardless of where serve runs

---

## References

**Files Examined:**
- `cmd/orch/serve.go` - API server implementation (3406 lines)
- `cmd/orch/main.go` - CLI entry point with sourceDir declaration
- `pkg/beads/client.go` - Beads RPC client with socket discovery
- `web/src/lib/stores/agents.ts` - Frontend agent store with API calls
- `web/src/lib/stores/beads.ts` - Frontend beads store
- `web/src/routes/+page.svelte` - Main dashboard page

**Commands Run:**
```bash
# Find sourceDir usages
grep "sourceDir" cmd/orch/serve.go

# Find project-specific paths in serve.go
grep "\.orch\|\.beads\|PROJECT_DIR" cmd/orch/serve.go
```

**External Documentation:**
- N/A (internal investigation)

**Related Artifacts:**
- **Decision:** Consider creating decision record for multi-project dashboard architecture
- **Workspace:** This investigation spawned from: `.orch/workspace/og-inv-web-dashboard-coupling-30dec/`

---

## Coupling Points Summary Table

| Coupling Point | Severity | Location | Impact |
|---------------|----------|----------|--------|
| `sourceDir` build-time embedding | CRITICAL | main.go:44, serve.go (10 sites) | Dashboard bound to compile-time project |
| Beads socket discovery | CRITICAL | serve.go:161-167, beads/client.go | Can only query one project's beads |
| Workspace path construction | CRITICAL | serve.go:2757, 2998, 3160 | Can't access cross-project workspaces |
| SSE events | OK | serve.go:1128-1279 | Already multiplexes all projects |
| Agent discovery | OK | serve.go:367-429, 664-707 | Already discovers all projects |
| Frontend API base | COSMETIC | stores/*.ts | Hardcoded but correct |
| Gap/reflect paths | OK | serve.go | Uses ~/.orch (user-global) |

---

## Investigation History

**2025-12-30 [start]:** Investigation started
- Initial question: What dashboard features assume orch-go as the project?
- Context: Dylan discovered dashboard doesn't work when orchestrating from external projects

**2025-12-30 [complete]:** Investigation completed
- Status: Complete
- Key outcome: Build-time sourceDir is the root cause; multi-project API is recommended fix

---

## Self-Review

- [x] Real test performed (grep for sourceDir usages, code review)
- [x] Conclusion from evidence (findings directly support summary)
- [x] Question answered (comprehensive coupling analysis)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (replaced all placeholders)

**Self-Review Status:** PASSED
