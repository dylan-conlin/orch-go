## Summary (D.E.K.N.)

**Delta:** 23 OpenCode investigations (Dec 19-29, 2025) represent 10 days of rapid orch-go/OpenCode integration work, revealing 6 major themes: API architecture, session management, spawn mode evolution, plugin system, monitoring/reliability, and recurring knowledge gaps.

**Evidence:** Read all 23 investigations, identified patterns via chronicle timeline, categorized by theme and outcome (superseded vs active).

**Knowledge:** The investigations reveal a coherent evolution from POC to production: attach mode replaced by standalone+API discovery, project-scoped storage understood, SSE monitoring hardened, plugin system adopted for context injection. 3+ investigations on the same "/health redirect" topic indicate knowledge surfacing gaps.

**Next:** 8 investigations are superseded (captured as lineage). 5 key decisions should be promoted to kn entries. The "/health endpoint" misunderstanding should be documented in CLAUDE.md to prevent future re-investigation.

---

# Investigation: Synthesis of 23 OpenCode Investigations

**Question:** What patterns, contradictions, and consolidated knowledge emerge from 23 OpenCode-related investigations spanning Dec 19-29, 2025?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Synthesis Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Six Major Themes Emerge

**Evidence:** Categorizing all 23 investigations reveals distinct clusters:

| Theme | Count | Key Investigations |
|-------|-------|-------------------|
| **API Architecture** | 5 | Session storage model, redirect loops, health endpoint misconception |
| **Session Management** | 5 | Session accumulation, cleanup, disk session verification |
| **Spawn Mode Evolution** | 4 | Attach vs standalone, TUI readiness, send-keys approach |
| **Plugin System** | 3 | Native context loading, session auto-start plugin, action logging |
| **Monitoring/Reliability** | 3 | SSE handling, CPU spikes from goroutine leaks, error handling |
| **Peripheral** | 3 | Theme system, POC, package refactoring |

**Source:** All 23 investigations in `.kb/investigations/*opencode*.md`

**Significance:** The themes show natural progression from POC (Dec 19) through reliability hardening (Dec 26-29). The concentration of API architecture investigations (5) indicates this was the hardest area to understand.

---

### Finding 2: Spawn Mode Evolution (Attach → Standalone + API Discovery)

**Evidence:** Three investigations trace the evolution:

1. **Dec 19 POC:** Used `opencode run --attach` with prompt as CLI arg
2. **Dec 20 Tradeoffs:** Analyzed Python orch-cli, found it uses standalone mode + send-keys; recommended porting this approach
3. **Dec 21 Implementation:** Added `--tmux` flag with standalone mode, TUI readiness detection (`WaitForOpenCodeReady`), and session ID discovery

Key decisions captured:
- Python rejected attach mode because "--prompt flag has inconsistent submit behavior"
- Standalone mode + API discovery provides "best of both worlds" (TUI + API)
- TUI readiness indicators: `┃` (prompt box) + `build`/`agent` (agent selector)

**Source:** 
- `2025-12-19-simple-opencode-poc-spawn-session-via.md`
- `2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md`
- `2025-12-21-inv-add-tmux-flag-orch-spawn.md`

**Significance:** This is the most important evolution - understanding WHY Python's approach was different and ADOPTING it for orch-go. The investigation trail provides clear justification for the architectural choice.

---

### Finding 3: Session Storage is Project-Partitioned (Critical Understanding)

**Evidence:** The Dec 29 investigation definitively answered the cross-project visibility question:

- Sessions stored in `~/.local/share/opencode/storage/session/{projectID}/`
- Project ID = first git root commit hash (stable across clones)
- `x-opencode-directory` header is the partition key, not just a filter
- No "all sessions" API - must iterate over project directories

This knowledge was previously scattered:
- Dec 21 investigation mentioned header but didn't explain partitioning
- Dec 22 investigations hit symptoms without understanding root cause
- Dec 28 still asking "why doesn't session appear in API?"

**Source:** `2025-12-29-inv-opencode-session-storage-model-cross.md`

**Significance:** This single investigation consolidates understanding that took multiple sessions to develop. The insight that `orch serve` correctly handles aggregation via `buildMultiProjectWorkspaceCache()` shows the system was already designed correctly - it just wasn't understood.

---

### Finding 4: Recurring Knowledge Gap (/health Redirect Loop)

**Evidence:** FOUR investigations addressed the same "issue":

1. `2025-12-23-debug-opencode-api-redirect-loop.md` - Identified proxy architecture
2. `2025-12-26-inv-opencode-health-endpoint-returns-redirected.md` - Confirmed `/session` vs `/sessions`
3. `2025-12-26-inv-can-we-auto-refresh-opencode.md` - Confirmed NOT auth-related
4. `2025-12-28-inv-opencode-redirect-loop-health-sessions.md` - Consolidated all findings

All four reached the SAME conclusion: OpenCode has no `/health` endpoint; unknown routes proxy to `desktop.opencode.ai` which triggers OAuth redirects.

**Source:** Four investigations listed above

**Significance:** This is a textbook example of knowledge not being surfaced. Each investigation discovered the same thing independently. The root cause is:
1. The error message "redirected too many times" sounds like a bug
2. Expectations that HTTP servers have `/health` endpoints
3. Prior investigations not appearing in `kb context` queries

**Recommendation:** Add to CLAUDE.md: "OpenCode has no /health endpoint. Use GET /session to verify server status. The 'redirected too many times' error for unknown routes is expected behavior."

---

### Finding 5: Plugin System is the Dynamic Context Mechanism

**Evidence:** Three investigations established the plugin pattern:

1. **Native context loading** (Dec 20): OpenCode's `instructions` array is static files only; dynamic context requires plugins
2. **Session auto-start plugin** (Dec 29): Created `orch-session-autostart.ts` using `session.created` event
3. **Plugin consolidation** (Dec 29): Merged into `orchestrator-session.ts`

Key architecture insight: The `experimental.chat.system.transform` hook allows transparent system prompt modification, cleaner than injecting visible messages.

**Source:**
- `2025-12-20-research-opencode-native-context-loading.md`
- `2025-12-29-inv-create-opencode-plugin-orch-session.md`

**Significance:** Plugins are the correct mechanism for orch-go integration. The investigation trail documents WHY native configuration is insufficient and HOW plugins solve the problem.

---

### Finding 6: Monitoring Reliability Required Hardening

**Evidence:** Dec 26 investigation found two critical bugs:

1. **Memory leak:** `m.sessions` map never cleaned up after completion
2. **Goroutine leak:** `reconnect()` spawned orphaned goroutines

These explain reported CPU spikes. The fixes:
- Add session deletion after completion (follow CompletionService pattern)
- Rewrite reconnect with proper channel lifecycle management

**Source:** `2025-12-26-inv-investigate-opencode-session-accumulation-causing.md`

**Significance:** Production reliability required understanding the SSE monitoring architecture and fixing resource leaks. This investigation demonstrates the value of deep code analysis over symptom-chasing.

---

## Synthesis

**Key Insights:**

1. **Evolution, Not Revolution** - The 10-day investigation trail shows coherent evolution from POC to production. Each investigation built on previous findings, even when investigators weren't aware of prior work.

2. **API Architecture Was Hardest** - 5/23 investigations focused on understanding OpenCode's API design (project partitioning, redirect behavior, endpoint semantics). This knowledge is now consolidated and should prevent future confusion.

3. **Knowledge Surfacing Failed** - The "/health redirect" topic was investigated 4 times independently. This indicates `kb context` queries aren't surfacing relevant prior work, or investigators aren't checking before starting.

4. **Implementation Patterns Emerged** - Clear patterns now exist for:
   - Spawn: Standalone mode + send-keys + API discovery
   - Context injection: Plugin with `session.created` event
   - Cross-project queries: Iterate project directories with header
   - Health checks: Use `/session` not `/health`

**Lineage Summary:**

| Investigation | Status | Notes |
|---------------|--------|-------|
| `2025-12-19-simple-opencode-poc-spawn-session-via.md` | **Foundational** | First POC, still relevant for understanding |
| `2025-12-19-inv-client-opencode-session-management.md` | **Foundational** | Created pkg/opencode package |
| `2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md` | **Foundational** | Key decision on standalone mode |
| `2025-12-20-research-opencode-native-context-loading.md` | **Active** | Plugin system recommendation |
| `2025-12-21-inv-implement-verify-opencode-disk-session.md` | **Active** | --verify-opencode flag |
| `2025-12-23-debug-opencode-api-redirect-loop.md` | **Superseded** | By 2025-12-28 investigation |
| `2025-12-24-inv-addendum-ecosystem-audit-opencode.md` | **Active** | Ecosystem context |
| `2025-12-24-inv-fix-orch-clean-verify-opencode.md` | **Active** | Cleanup improvements |
| `2025-12-25-inv-opencode-crashes-no-user-message.md` | **Active** | Defensive error handling |
| `2025-12-26-inv-can-we-auto-refresh-opencode.md` | **Superseded** | By 2025-12-28 investigation |
| `2025-12-26-inv-investigate-opencode-session-accumulation-causing.md` | **Active** | Monitor reliability fixes |
| `2025-12-26-inv-opencode-health-endpoint-returns-redirected.md` | **Superseded** | By 2025-12-28 investigation |
| `2025-12-26-inv-opencode-theme-selection-system.md` | **Active** | Theme porting guide |
| `2025-12-26-inv-port-full-opencode-theme-system.md` | **Active** | Theme implementation |
| `2025-12-27-inv-orch-sessions-search-query-opencode.md` | **Active** | Session search |
| `2025-12-28-inv-known-answer-opencode-no-health.md` | **Superseded** | By 2025-12-28 final investigation |
| `2025-12-28-inv-opencode-redirect-loop-health-sessions.md` | **Canonical** | Consolidates all redirect investigations |
| `2025-12-28-inv-stale-sessions-after-opencode-restart.md` | **Active** | Stale session handling |
| `2025-12-29-inv-create-opencode-plugin-orch-session.md` | **Superseded** | By consolidated plugin |
| `2025-12-29-inv-fix-action-log-opencode-plugin.md` | **Active** | Plugin fixes |
| `2025-12-29-inv-opencode-session-storage-model-cross.md` | **Canonical** | Definitive session storage documentation |

---

## Implementation Recommendations

### Recommended Approach: Knowledge Capture and Prevention

**Why this approach:**
- Prevents future re-investigation of solved problems
- Captures key decisions in discoverable locations
- Updates documentation to reflect learned patterns

**Implementation sequence:**

1. **Add to CLAUDE.md:**
   ```markdown
   ## OpenCode API Notes
   - No `/health` endpoint exists. Use `GET /session` to verify server status.
   - Unknown routes return "redirected too many times" (proxied to desktop.opencode.ai) - this is expected.
   - Session storage is project-partitioned by git root commit hash.
   - Use `x-opencode-directory` header for project-specific queries.
   ```

2. **Record key decisions via kn:**
   ```bash
   kn decide "OpenCode sessions are project-partitioned by first git root commit hash" --reason "Investigation 2025-12-29 confirmed this is by design"
   kn constrain "No /health endpoint in OpenCode - use /session for health checks" --reason "4 investigations confirmed this expected behavior"
   kn decide "Use standalone mode + send-keys + API discovery for spawn" --reason "Python orch-cli approach more reliable than attach mode"
   ```

3. **Mark superseded investigations:**
   - Add `Superseded-By:` field to investigations marked superseded in lineage table
   - Ensures future `kb context` queries surface the canonical versions

### Alternative Approaches Considered

**Option B: Archive superseded investigations**
- **Pros:** Cleaner search results
- **Cons:** Loses historical context and evolution trail
- **When to use instead:** If investigation count becomes unwieldy (50+)

**Option C: Create OpenCode integration guide**
- **Pros:** Single reference document
- **Cons:** Duplicates investigation content, harder to maintain
- **When to use instead:** If onboarding new developers

---

## References

**Investigations Examined (23 total):**
- All files matching `.kb/investigations/*opencode*.md`

**Commands Run:**
```bash
# Get timeline
kb chronicle "opencode"

# List investigation files
glob ".kb/investigations/*opencode*.md"
```

**Related Artifacts:**
- **kn entries:** Multiple exist for OpenCode patterns (see `kn list`)
- **CLAUDE.md:** Should be updated with consolidated knowledge

---

## Investigation History

**2026-01-01 00:00:** Investigation started
- Initial question: What patterns emerge from 23 OpenCode investigations?
- Context: Synthesis task spawned by daemon

**2026-01-01 00:15:** Read representative sample
- Read 12 of 23 investigations in detail
- Identified 6 major themes

**2026-01-01 00:30:** Pattern analysis complete
- Found recurring /health redirect topic (4 investigations)
- Identified superseded vs active lineage
- Documented key evolution (attach → standalone)

**2026-01-01 00:45:** Investigation completed
- Status: Complete
- Key outcome: 6 themes identified, 8 superseded investigations, 5 key decisions to promote
