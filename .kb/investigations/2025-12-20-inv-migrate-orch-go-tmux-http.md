## Summary (D.E.K.N.)

**Delta:** The tmux package and all tmux-related functionality can be completely removed; orch-go already has full HTTP API support for headless agents.

**Evidence:** Analyzed 100+ tmux references across codebase; headless mode (HTTP API) is already the default spawn method since previous refactoring.

**Knowledge:** The architecture cleanly separates tmux (opt-in) from HTTP API (default); removing tmux requires updating spawn, tail, question, abandon, complete, and clean commands.

**Next:** Implement removal - delete pkg/tmux, update cmd/orch/main.go to remove --tmux flag and fallback logic, update affected commands.

**Confidence:** High (90%) - existing HTTP API implementation is well-tested and currently the default path.

---

# Investigation: Migrate orch-go from tmux to HTTP API

**Question:** What changes are required to remove all tmux functionality from orch-go and make HTTP API the only spawn mode?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** Implementation
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: tmux is already opt-in, HTTP API is default

**Evidence:** 
- `spawnTmux` flag defaults to `false` (main.go:82)
- The spawn flow checks `if useTmux && tmux.IsAvailable()` (main.go:650)
- Default path is `runSpawnHeadless()` which uses HTTP API (main.go:656)

**Source:** cmd/orch/main.go:644-656

**Significance:** Migration is low-risk because HTTP API is already the primary path; we're removing the fallback, not the main functionality.

---

### Finding 2: tmux usage is isolated to specific commands

**Evidence:** grep found 100+ tmux references, concentrated in:
- cmd/orch/main.go: spawn, tail, question, abandon, complete, clean commands
- pkg/tmux/tmux.go: the tmux package itself
- pkg/registry/registry.go: Reconcile() function for tmux window reconciliation
- cmd/gendoc/main.go: documentation generator

**Source:** `grep -r "tmux" --include="*.go"`

**Significance:** Changes are contained to a well-defined set of files; no deep architectural changes required.

---

### Finding 3: Registry already supports headless agents

**Evidence:**
- `HeadlessWindowID = "headless"` constant exists (registry.go:473)
- `Reconcile()` skips headless agents (registry.go:492-494)
- Headless agents are tracked via SSE events, not tmux windows

**Source:** pkg/registry/registry.go:472-495

**Significance:** Registry design already accounts for HTTP-only agents; we just need to remove tmux-specific code.

---

### Finding 4: tail and question commands have API fallback

**Evidence:**
- `runTail()` checks `agent.WindowID == registry.HeadlessWindowID` and calls `runTailFromAPI()` (main.go:298-305)
- For headless agents, tail fetches messages via OpenCode API (main.go:312-342)
- question command currently requires tmux, but can use API messages instead

**Source:** cmd/orch/main.go:289-444

**Significance:** tail already has API implementation; question needs conversion to use message content.

---

### Finding 5: wait and daemon commands don't use tmux

**Evidence:**
- wait.go imports only events and verify packages, no tmux
- daemon.go shells out to `orch-go work` which will use headless mode

**Source:** cmd/orch/wait.go, cmd/orch/daemon.go

**Significance:** These commands need no changes beyond what happens transitively.

---

## Synthesis

**Key Insights:**

1. **HTTP API is production-ready** - The headless mode using HTTP API is already the default and has been working. This is a simplification, not a new feature.

2. **Clean removal path** - tmux functionality is cleanly isolated; we can remove the package and update callers without restructuring the codebase.

3. **Registry simplification** - The registry can be simplified to only track session IDs, removing window_id tracking entirely.

**Answer to Investigation Question:**

To migrate orch-go from tmux to HTTP API only:
1. Delete pkg/tmux/ entirely
2. Remove --tmux flag from spawn and work commands
3. Remove runSpawnInTmux() function
4. Update tail command to always use API
5. Update question command to extract from API messages
6. Update abandon command to work without tmux windows
7. Update complete command to skip window cleanup
8. Update clean command to remove tmux reconciliation
9. Simplify registry by removing window-related fields and Reconcile()

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

HTTP API mode is already the default and working. The changes are subtractive (removing code) rather than additive (adding new code). Tests already exist for HTTP API functionality.

**What's certain:**

- The HTTP API client is fully functional (CreateSession, SendPrompt, GetMessages)
- Headless spawn is the current default behavior
- Registry already handles headless agents properly

**What's uncertain:**

- Whether all edge cases in question command work via API (need to test)
- Impact on any external tooling that might expect tmux

**What would increase confidence to Very High (95%+):**

- Running full test suite after changes
- Manual testing of all commands

---

## Implementation Recommendations

### Recommended Approach 

**Remove tmux, simplify to HTTP-only** - Delete the tmux package and all tmux-related code paths, making HTTP API the only spawn mechanism.

**Why this approach:**
- Reduces complexity by removing an entire code path
- HTTP API is already the default and well-tested
- Simplifies registry and command implementations

**Trade-offs accepted:**
- Losing ability to see TUI in tmux window (use --inline for interactive work)
- Users who relied on --tmux flag will need to adapt

**Implementation sequence:**
1. Delete pkg/tmux/ package
2. Update cmd/orch/main.go to remove tmux imports and code
3. Update registry to remove window-related fields
4. Update individual commands (tail, question, abandon, complete, clean)
5. Update documentation
6. Run tests

---

## References

**Files Examined:**
- cmd/orch/main.go - Main CLI with spawn, complete, clean, tail, question, abandon
- pkg/tmux/tmux.go - Tmux package to be removed
- pkg/registry/registry.go - Agent registry with window tracking
- pkg/opencode/client.go - HTTP API client already implemented
- cmd/orch/wait.go - Wait command (no tmux, no changes needed)
- cmd/orch/daemon.go - Daemon command (no tmux, no changes needed)

**Commands Run:**
```bash
# Find all tmux references
grep -r "tmux" --include="*.go"
```

---

## Investigation History

**2025-12-20 10:30:** Investigation started
- Initial question: What changes are required to remove tmux?
- Context: Simplifying orch-go architecture

**2025-12-20 10:45:** Analysis complete
- Found 100+ tmux references, all in well-defined locations
- HTTP API already default, tmux is opt-in

**2025-12-20 10:50:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Clean removal path identified, implementation can proceed
