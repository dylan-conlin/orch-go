## Summary (D.E.K.N.)

**Delta:** 11 tmux investigations (Dec 20-23, 2025) document a complete architectural transition from tmux-default to headless-default spawning, with tmux becoming an opt-in mode via `--tmux` flag.

**Evidence:** Investigations trace the arc: concurrent spawn validation (6 agents) → migration planning → tmux flag implementation → attach mode for API access → orch send session resolution fixes → tmux fallback for status/tail.

**Knowledge:** Tmux mode provides visual monitoring + API access (via opencode attach), while headless is now the default for automation; the two modes complement rather than compete.

**Next:** Archive 8 of 11 investigations as superseded by this synthesis; keep 3 as canonical reference (spawn-agent-tmux, debug-orch-send Dec 22, tmux-fallback-orch-status).

---

# Investigation: Synthesis of 11 Tmux Investigations

**Question:** What patterns emerge across 11 tmux investigations, and which can be consolidated or archived?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Synthesis agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:**
- .kb/investigations/2025-12-20-inv-migrate-orch-go-tmux-http.md (migration planning, partially implemented)
- .kb/investigations/2025-12-20-inv-tmux-concurrent-delta.md (concurrent validation)
- .kb/investigations/2025-12-20-inv-tmux-concurrent-epsilon.md (concurrent validation)
- .kb/investigations/2025-12-20-inv-tmux-concurrent-zeta.md (concurrent validation)
- .kb/investigations/2025-12-21-inv-add-tmux-flag-orch-spawn.md (implemented)
- .kb/investigations/2025-12-21-inv-implement-attach-mode-tmux-spawn.md (implemented)
- .kb/investigations/2025-12-21-inv-tmux-spawn-killed.md (bug fix, resolved)
- .kb/investigations/2025-12-21-debug-orch-send-fails-silently-tmux.md (superseded by Dec 22 version)

---

## Findings

### Finding 1: Investigations Document a Complete Architectural Transition

**Evidence:** The 11 investigations span Dec 20-23, 2025 and document:
1. **Dec 20:** Concurrent spawn validation (delta, epsilon, zeta) confirming tmux scales to 6+ agents
2. **Dec 20:** Migration planning (migrate-orch-go-tmux-http) proposing HTTP-only architecture
3. **Dec 21:** Tmux flag implementation (add-tmux-flag-orch-spawn) making tmux opt-in
4. **Dec 21:** Attach mode (implement-attach-mode-tmux-spawn) enabling dual TUI+API access
5. **Dec 21-22:** Bug fixes (orch send session resolution, tmux spawn SIGKILL)
6. **Dec 21:** Tmux fallback (add-tmux-fallback-orch-status) for status/tail commands

**Source:** 11 investigation files in .kb/investigations/

**Significance:** These investigations collectively document orch-go's spawn mode evolution. The current state (headless default, tmux opt-in) is the result of this development arc.

---

### Finding 2: Three Investigations Remain Canonical Reference

**Evidence:** Three investigations contain unique, non-duplicated knowledge:

1. **2025-12-22-inv-spawn-agent-tmux.md** - Comprehensive reference for how tmux spawn works (3 modes, attach mode, TUI detection, window naming)
2. **2025-12-22-debug-orch-send-fails-silently-tmux.md** - Documents the final fix for session ID validation (supersedes Dec 21 version)
3. **2025-12-21-inv-add-tmux-fallback-orch-status.md** - Documents tmux fallback implementation for status/tail/question

**Source:** Code review of investigation content vs implementation state

**Significance:** These 3 investigations provide the authoritative reference for tmux-related functionality. The other 8 are historical artifacts.

---

### Finding 3: Eight Investigations Are Historical or Superseded

**Evidence:**

| Investigation | Status | Reason |
|---------------|--------|--------|
| concurrent-delta/epsilon/zeta | Historical | Validation tests, not reference docs |
| migrate-orch-go-tmux-http | Partially obsolete | Migration to HTTP-default happened, but tmux remains as opt-in mode |
| add-tmux-flag-orch-spawn | Implemented | Feature complete, code is the reference |
| implement-attach-mode-tmux-spawn | Implemented | Feature complete, code is the reference |
| tmux-spawn-killed | Resolved | Bug fixed, prevention measures documented |
| debug-orch-send Dec 21 | Superseded | Dec 22 version is more complete |

**Source:** Cross-reference between investigations and current codebase

**Significance:** These 8 investigations served their purpose during development but no longer provide unique value. They can be archived.

---

### Finding 4: Key Architectural Knowledge Extracted

**Evidence:** Across all 11 investigations, the key architectural decisions are:

1. **Three spawn modes exist:** tmux (visual), inline (blocking TUI), headless (HTTP API)
2. **Headless is now default** - `orch spawn` uses HTTP API; `--tmux` for visual mode
3. **Tmux uses opencode attach** - Connects to shared server for dual TUI+API access
4. **TUI readiness detection** - Polls pane content for visual indicators (prompt box + agent selector)
5. **Session ID resolution** - Supports beads IDs, workspace names, and raw session IDs with validation
6. **Concurrent spawning scales** - Validated to 6+ agents with proper workspace isolation
7. **Per-project sessions** - `workers-{project}` sessions organize agent windows

**Source:** Synthesis across all 11 investigations

**Significance:** This is the knowledge that should be preserved. Much of it is now in the orchestrator skill or code comments.

---

## Synthesis

**Key Insights:**

1. **Development arc is complete** - The investigations document a deliberate transition from tmux-first to headless-first architecture. This transition is now complete, and the investigations served their purpose as development artifacts.

2. **Tmux mode remains valuable** - Despite the shift to headless default, tmux mode with attach provides unique value: visual monitoring + API access. The investigations document this dual-access pattern well.

3. **Bug fixes created durable patterns** - The session resolution and tmux fallback fixes (Dec 21-22) established patterns that are now part of the codebase. These patterns should be referenced, not the investigations.

**Answer to Investigation Question:**

The 11 tmux investigations document a complete architectural transition from tmux-default to headless-default spawning (Dec 20-23, 2025). 

**Consolidation recommendation:**
- **Keep 3 canonical:** spawn-agent-tmux (Dec 22), debug-orch-send (Dec 22), tmux-fallback-orch-status (Dec 21)
- **Archive 8 as superseded:** concurrent tests (3), migration planning (1), feature implementations (2), bug fix (1), earlier debug version (1)

The architectural knowledge from all 11 is now embedded in:
- The orchestrator skill (spawn modes documentation)
- pkg/tmux/tmux.go (implementation)
- This synthesis investigation (historical context)

---

## Structured Uncertainty

**What's tested:**

- ✅ Current spawn modes work correctly (verified: daily production use)
- ✅ Tmux attach mode provides API access (verified: orch status shows tmux agents)
- ✅ Session resolution handles multiple identifier formats (verified: code and tests)

**What's untested:**

- ⚠️ Upper concurrency limit for tmux spawns (tested to 6+, not to failure)
- ⚠️ Long-term tmux window accumulation (cleanup patterns exist but not validated)

**What would change this:**

- Different decision would be needed if opencode drops attach mode support
- Different decision would be needed if headless mode proves unreliable

---

## Implementation Recommendations

### Recommended Approach ⭐

**Archive superseded investigations** - Add `Superseded-By:` header to 8 investigations pointing to this synthesis.

**Why this approach:**
- Reduces .kb/ clutter without losing history
- Makes clear which investigations are canonical
- Provides navigational path to current state

**Trade-offs accepted:**
- Requires updating 8 files with headers
- Some historical context may be less discoverable

**Implementation sequence:**
1. Update this synthesis with final Supersedes list
2. Add `Superseded-By: .kb/investigations/2026-01-01-inv-synthesize-tmux-investigations-11-synthesis.md` to 8 investigations
3. Consider moving archived investigations to `.kb/investigations/archived/`

### Alternative Approaches Considered

**Option B: Delete superseded investigations**
- **Pros:** Cleaner .kb/ directory
- **Cons:** Loses historical context, may break any references
- **When to use instead:** If .kb/ size becomes a problem

**Option C: Do nothing**
- **Pros:** No work required
- **Cons:** Continued duplication, confusion about canonical source
- **When to use instead:** If archive work is lower priority

---

## References

**Investigations Synthesized:**

1. 2025-12-20-inv-migrate-orch-go-tmux-http.md - Migration planning
2. 2025-12-20-inv-tmux-concurrent-delta.md - Concurrent validation (delta)
3. 2025-12-20-inv-tmux-concurrent-epsilon.md - Concurrent validation (epsilon)
4. 2025-12-20-inv-tmux-concurrent-zeta.md - Concurrent validation (zeta)
5. 2025-12-21-debug-orch-send-fails-silently-tmux.md - Session resolution (v1)
6. 2025-12-21-inv-add-tmux-fallback-orch-status.md - Tmux fallback ⭐
7. 2025-12-21-inv-add-tmux-flag-orch-spawn.md - Tmux flag implementation
8. 2025-12-21-inv-implement-attach-mode-tmux-spawn.md - Attach mode
9. 2025-12-21-inv-tmux-spawn-killed.md - SIGKILL bug fix
10. 2025-12-22-debug-orch-send-fails-silently-tmux.md - Session resolution (v2) ⭐
11. 2025-12-22-inv-spawn-agent-tmux.md - Comprehensive spawn reference ⭐

**⭐ = Canonical reference (keep)**

---

## Investigation History

**2026-01-01 10:00:** Investigation started
- Initial question: What patterns emerge across 11 tmux investigations?
- Context: kb suggest identified topic for consolidation

**2026-01-01 10:30:** All 11 investigations read
- Found clear development arc from tmux-default to headless-default
- Identified 3 canonical vs 8 archivable investigations

**2026-01-01 10:45:** Investigation completed
- Status: Complete
- Key outcome: 8 investigations superseded by this synthesis; 3 remain canonical reference
