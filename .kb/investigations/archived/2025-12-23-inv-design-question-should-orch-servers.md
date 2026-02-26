<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux session abstraction is correct for dev servers; `orch serve` API should be managed separately as infrastructure, not conflated with project servers.

**Evidence:** `orch serve` runs as separate process (verified via lsof), not in tmuxinator; code shows distinct operational patterns (ephemeral dev vs persistent infrastructure); tmux check is in cmd/orch/servers.go:172-186.

**Knowledge:** The problem isn't the abstraction - it's mixing two categories under one command. Principle "Evolve by distinction" applies: separate `orch servers` (project dev) from `orch api` (orchestrator infrastructure).

**Next:** Remove port 3348 from project allocations, implement `orch api status` command, update tmuxinator template to remove API comment.

**Confidence:** High (85%) - Validated via code and runtime, but haven't confirmed user expectations or measured performance impact at scale.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Design Question Should Orch Servers

**Question:** Should 'orch servers status' check actual port listening, or is tmux session existence the right abstraction?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current implementation checks tmux session existence, not port listening

**Evidence:** 
- `cmd/orch/servers.go:172-186` - `runServersList()` calls `tmux.ListWorkersSessions()` to determine running status
- `cmd/orch/servers.go:217-220` - Status is set to "running" if tmux session exists, "stopped" otherwise
- No port listening checks are performed in current implementation

**Source:** 
- `cmd/orch/servers.go:146-229` (runServersList function)
- `pkg/tmux/tmux.go:434-451` (ListWorkersSessions function)

**Significance:** The abstraction is "session existence = running". This means a project shows as "running" even if the actual server processes aren't listening on their ports. This creates a discrepancy between reported status and actual network availability.

---

### Finding 2: `orch serve` API server is NOT managed by tmuxinator

**Evidence:**
- `~/.tmuxinator/workers-orch-go.yml` contains: `# api server on port 3348` (just a comment)
- Actual `orch serve` process runs separately (PID 23280, verified via `lsof -i :3348`)
- Tmuxinator config only runs `bun run dev --port 5188` (web server)
- `cmd/orch/serve.go:1-103` - `orch serve` is a standalone command, not integrated with tmuxinator

**Source:**
- `~/.tmuxinator/workers-orch-go.yml` (tmuxinator config)
- `lsof -i :3348` (verified port 3348 listening with separate orch process)
- `pkg/tmux/tmuxinator.go:86-98` (buildServerCommand function - only generates commands for PurposeVite and PurposeAPI with comments)

**Significance:** There's a fundamental mismatch: tmuxinator manages dev servers (web), but the API server (`orch serve`) is a separate long-running process. The current "session existence = running" abstraction cannot detect whether `orch serve` is running because it's not part of the tmux session.

---

### Finding 3: Two distinct server categories with different operational patterns

**Evidence:**
- **Dev servers** (web): Started via tmuxinator, run in tmux panes, project-specific (e.g., `bun run dev --port 5188`)
- **API server** (`orch serve`): Standalone process, not in tmux, serves dashboard UI, runs on fixed port 3348
- Port registry tracks both: `Purpose: "vite"` for dev servers, `Purpose: "api"` for API servers
- Dev servers restart with project work; API server is long-lived infrastructure

**Source:**
- `pkg/port/port.go:36-39` (PurposeVite and PurposeAPI constants)
- `cmd/orch/serve.go:27-42` (serve command documentation)
- `pkg/tmux/tmuxinator.go:86-98` (different command generation per purpose)

**Significance:** These are fundamentally different operational concerns. Dev servers are tied to project development lifecycle (start/stop with work sessions). The API server is infrastructure for monitoring (should be always-on). Conflating them under "servers status" may be mixing abstractions.

---

## Synthesis

**Key Insights:**

1. **Tmux session abstraction is correct for dev servers, wrong for API servers** - Dev servers (web) are tightly coupled to tmux lifecycle - when you start/stop project work, you start/stop dev servers. Tmux session existence accurately reflects dev server intent. But `orch serve` is infrastructure - it should run independently and persist across project sessions. (Findings 2, 3)

2. **The root issue is conflating two distinct operational patterns** - We're trying to use one status command for two categories: ephemeral dev infrastructure (tmux-managed) and persistent monitoring infrastructure (standalone). The current implementation optimizes for the former, making the latter invisible. (Finding 3)

3. **Port listening checks would add accuracy but at the cost of speed and complexity** - Checking actual port listening (Option C) would detect the `orch serve` gap, but adds syscall overhead on every status check. For 10+ projects, this becomes noticeable. The real question is whether "orch servers" should even be responsible for monitoring `orch serve`. (Finding 1)

**Answer to Investigation Question:**

**Tmux session existence is the right abstraction for `orch servers status` - but `orch serve` shouldn't be part of "servers".**

The current design is correct: `orch servers` manages per-project development servers (web, API endpoints specific to that project). The command is fast because it's a simple tmux query. The issue isn't the abstraction - it's that we're treating `orch serve` (global monitoring API) as if it were a project server.

**Recommendation:** Keep tmux session abstraction. Remove `orch serve` from project port allocations and manage it separately via a different command (e.g., `orch daemon status` or `orch api status`). This respects separation of concerns: `orch servers` = project dev infrastructure, `orch api` = orchestrator infrastructure.

Limitations: This recommendation assumes `orch serve` is truly global infrastructure. If it becomes per-project in the future, we'd need to revisit.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

High confidence because the code, runtime behavior, and architectural patterns all point to the same conclusion: `orch serve` is infrastructure, not a project server. The evidence is direct (reading code, checking process state, verifying port listening). The 15% uncertainty comes from not having interviewed Dylan about the intended design.

**What's certain:**

- ✅ **Current implementation uses tmux session existence** - Verified in `cmd/orch/servers.go:172-186` and tested with live tmux sessions
- ✅ **`orch serve` runs outside tmuxinator** - Confirmed via `lsof -i :3348` showing separate orch process (PID 23280), not in tmux session
- ✅ **Two distinct operational patterns exist** - Dev servers (tmuxinator, ephemeral) vs API server (standalone, persistent) - verified in code and runtime

**What's uncertain:**

- ⚠️ **User expectations** - Haven't validated whether users expect `orch servers` to show `orch serve` status. Recommendation assumes they care about dev servers primarily.
- ⚠️ **Future architecture** - If `orch serve` becomes per-project (unlikely but possible), the recommendation changes
- ⚠️ **Performance impact** - Haven't benchmarked port listening checks at scale. Stated "noticeable" impact is estimated, not measured.

**What would increase confidence to Very High (95%+):**

- Interview Dylan about original design intent for `orch servers` vs `orch serve`
- Observe real user workflows - do they start/stop `orch serve` with projects or run it continuously?
- Benchmark port listening checks for 10, 50, 100 projects to quantify performance concern

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Separate Infrastructure: Keep tmux abstraction, move `orch serve` to separate command** - Remove API server (port 3348) from project port allocations and manage `orch serve` status via a dedicated infrastructure command.

**Why this approach:**
- Respects separation of concerns: dev servers vs monitoring infrastructure (Finding 3)
- Maintains fast tmux-based status checks (no syscall overhead per project)
- Aligns abstraction with operational reality: tmux lifecycle matches dev server lifecycle
- Makes `orch serve` explicitly infrastructure, not project-scoped

**Trade-offs accepted:**
- Users must check two commands (`orch servers status` + `orch api status`) to see full picture
- More commands to learn, but more semantically clear

**Implementation sequence:**
1. **Remove `orch serve` from port allocations** - Stop allocating port 3348 via `orch port allocate orch-go api api`. This is foundational - it breaks the false coupling.
2. **Add `orch api status` command** - New command that checks if `orch serve` is listening on port 3348 (or reads PID file, or checks process list). Single purpose = single port check is fast.
3. **Update tmuxinator template** - Remove the `# api server on port 3348` comment from generated configs to eliminate confusion
4. **Document the distinction** - Update CLAUDE.md and README to clarify: `orch servers` = project dev servers, `orch api` = orchestrator monitoring infrastructure

### Alternative Approaches Considered

**Option A (from task): Keep current design, document separation**
- **Pros:** Zero code changes, fastest to implement
- **Cons:** Status will always be misleading (shows "running" when API down). Documentation doesn't fix broken abstraction. Users will continue to be confused when `orch servers list` says "running" but API isn't accessible.
- **When to use instead:** If `orch serve` truly belongs to orch-go project lifecycle (start/stop with project work), then this makes sense. But evidence suggests it's persistent infrastructure.

**Option C (from task): Change status to check actual port listening**
- **Pros:** Accurate status - would detect when `orch serve` is down
- **Cons:** Performance degradation (syscalls scale with project count). Conflates two operational patterns under one abstraction. Doesn't solve the semantic problem - still treating infrastructure as project server.
- **When to use instead:** If there's only one "servers" concept and accuracy matters more than speed. Could be useful for production health checks, but overkill for dev workflow.

**Option B (from task): Add `orch serve` to tmuxinator config**
- **Pros:** Makes API server part of tmux lifecycle, status checks become accurate
- **Cons:** Forces infrastructure to restart with project sessions (wrong lifecycle). User would need to spawn `orch serve` in every project's tmuxinator, causing port conflicts. API server is global (serves all projects), not per-project.
- **When to use instead:** If `orch serve` became per-project (e.g., `orch-go` has its own API on 3348, `price-watch` has its own on 3349). Not the current architecture.

**Rationale for recommendation:** 

The recommended approach (Separate Infrastructure) is the only option that fixes the semantic mismatch. Options A/B/C all try to force `orch serve` into the "project servers" abstraction, but the evidence shows it's not a project server - it's orchestrator infrastructure. 

Principle alignment: **"Evolve by distinction"** - The problem recurs because we're conflating dev servers and infrastructure. Making the distinction explicit (separate commands) eliminates the ambiguity.

---

### Implementation Details

**What to implement first:**
- **Add `orch api status` command** - Quick win, immediately useful, demonstrates the separation. Implementation: check if port 3348 is listening via `net.Dial("tcp", "127.0.0.1:3348")` with short timeout.
- **Remove port 3348 allocation for orch-go** - Run `orch port release orch-go api` to break the false coupling immediately
- **Clean up tmuxinator template** - Remove the `# api server on port 3348` comment from `pkg/tmux/tmuxinator.go:92-94`

**Things to watch out for:**
- ⚠️ **Other projects may have API allocations** - Check if any other projects allocated "api" purpose ports for `orch serve`. If so, those need migration too.
- ⚠️ **Existing scripts/docs referencing port registry** - Search for references to "3348" or "api server" in docs/scripts that assume it's in port registry
- ⚠️ **Web UI dashboard dependencies** - If beads-ui dashboard hardcodes port 3348, ensure it still works after removing from registry. Likely fine since it probably uses env var or hardcoded constant, not registry lookup.

**Areas needing further investigation:**
- **Should `orch api` be part of `orch daemon`?** - The API server serves the monitoring UI. Is it conceptually part of daemon operations? Could be `orch daemon api status` instead of top-level `orch api status`.
- **PID file vs port checking** - Should `orch serve` write a PID file for more reliable status checking? Port checking can give false positives (different process on that port).
- **Lifecycle management** - Should there be `orch api start` and `orch api stop`? Or keep it manual (`orch serve` in background)?

**Success criteria:**
- ✅ **Semantic clarity** - `orch servers list` only shows project dev servers (web, project-specific APIs), never global infrastructure
- ✅ **Accurate status** - `orch api status` correctly reports whether `orch serve` is reachable on port 3348
- ✅ **No performance regression** - `orch servers status` remains fast (no port checks for dev servers)
- ✅ **Documentation updated** - CLAUDE.md and README explain the distinction clearly

---

## References

**Files Examined:**
- `cmd/orch/servers.go:146-229` - runServersList implementation (how status is determined)
- `cmd/orch/serve.go:1-103` - orch serve command definition and HTTP server
- `pkg/tmux/tmux.go:434-451` - ListWorkersSessions (tmux session enumeration)
- `pkg/tmux/tmuxinator.go:86-98` - buildServerCommand (how tmuxinator config is generated)
- `pkg/port/port.go:36-39` - Purpose constants (PurposeVite, PurposeAPI)
- `~/.tmuxinator/workers-orch-go.yml` - Generated tmuxinator config for orch-go project

**Commands Run:**
```bash
# Check tmux sessions
tmux list-sessions

# Verify port 3348 listening (orch serve)
lsof -i :3348

# Verify port 5188 listening (web dev server)
lsof -i :5188

# Read tmuxinator config
cat ~/.tmuxinator/workers-orch-go.yml
```

**External Documentation:**
- None referenced

**Related Artifacts:**
- **Principle:** `.kb/principles.md` - "Evolve by distinction" guides the recommendation to separate infrastructure from project servers
- **Decision:** `2025-12-21-beads-oss-relationship-clean-slate.md` - Prior decision on separation of concerns patterns

---

## Investigation History

**2025-12-23 10:30:** Investigation started
- Initial question: Should 'orch servers status' check actual port listening, or is tmux session existence the right abstraction?
- Context: User noticed `orch servers list` shows orch-go as "running" when workers-orch-go tmux session exists, but API server (:3348) isn't actually listening (in their scenario - in current testing it is listening but as separate process)

**2025-12-23 10:45:** Key finding - `orch serve` runs outside tmuxinator
- Verified via `lsof -i :3348` that `orch serve` is a separate process (PID 23280)
- Tmuxinator config has comment about API server but doesn't run it
- Web server (5188) runs via tmuxinator as expected

**2025-12-23 11:00:** Synthesis complete
- Identified root cause: conflating two operational patterns (dev servers vs infrastructure)
- Recommendation: Separate infrastructure - keep tmux abstraction, move `orch serve` to dedicated command

**2025-12-23 11:15:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Tmux abstraction is correct; `orch serve` should be managed separately as infrastructure, not project server
