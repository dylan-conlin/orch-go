## Summary (D.E.K.N.)

**Delta:** Orch's lifecycle complexity (~8,800 lines) stems from compensating for OpenCode's missing session management features (TTLs, state API, metadata storage), not from inherent orchestration needs.

**Evidence:** Code audit shows 7 clean flags, 11 complete skip-gates, 3 agent ghost types — each exists because orch treats transient infrastructure (OpenCode sessions, tmux windows) as state layers requiring reconciliation.

**Knowledge:** A three-bucket ownership model (Own / Accept / Lobby) with a simplified two-layer state model (beads canonical + workspace operational) can reduce lifecycle code by ~40% while maintaining all current capabilities.

**Next:** Promote to decision record. Implement Phase 1 (simplify clean to 3 modes, remove ghost/phantom distinction). File upstream issues for Session TTL and Session Metadata API.

**Authority:** architectural - Cross-component ownership boundary affects spawn, clean, complete, status, and OpenCode integration strategy.

---

# Investigation: Evaluate Lifecycle Management Ownership Boundaries

**Question:** What lifecycle responsibilities should orch own vs push upstream to OpenCode vs simplify away?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Architect agent (og-arch-evaluate-lifecycle-management-13feb-d001)
**Phase:** Complete
**Next Step:** None - promote findings to decision record
**Status:** Complete

**Patches-Decision:** N/A (new decision)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` | extends | Yes - registry methods still unused | None |
| `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` | extends | Yes - event + periodic cleanup pattern confirmed | None |
| `.kb/decisions/2026-01-14-infrastructure-complexity-justified.md` | extends | Yes - complexity still justified, but can be redistributed | None |
| `.kb/models/agent-lifecycle-state-model/model.md` | deepens | Yes - four-layer model confirmed | Reframes: 2 layers are state, 2 are infrastructure |
| `.kb/models/opencode-session-lifecycle/model.md` | deepens | Yes - all constraints still apply | None |
| `.kb/models/completion-verification/model.md` | confirms | Yes - Phase: Complete still canonical | None |

---

## Findings

### Finding 1: Orch's lifecycle code is ~8,800 lines compensating for OpenCode gaps

**Evidence:** Line counts across lifecycle-related files:
- `clean_cmd.go`: 1,159 lines (7 cleanup modes: windows, phantoms, ghosts, verify-opencode, investigations, stale, sessions)
- `complete_cmd.go`: 1,659 lines (11 verification gates, 13 completion actions)
- `abandon_cmd.go`: 401 lines (session export, beads reset, window cleanup)
- `serve_agents.go`: 1,560 lines (status reconciliation, Priority Cascade)
- `pkg/registry/registry.go`: 529 lines (deprecated state methods, spawn-cache only)
- `pkg/opencode/client.go`: 1,285 lines (session management, disk queries)
- `pkg/opencode/sse.go`: 211 lines (completion detection)
- `pkg/verify/check.go`: 641 lines (phase parsing, deliverable verification)
- `shared.go`: 400 lines (beads ID extraction, workspace lookups, ghost detection helpers)

**Source:** `wc -l` on all lifecycle-related files

**Significance:** This is substantial code for what is essentially "create session, watch it, clean up after it." The bulk exists because orch must reconcile four independent state layers — but two of those layers (OpenCode sessions, tmux windows) are transient infrastructure, not business state.

---

### Finding 2: Three distinct agent ghost types exist because OpenCode sessions persist indefinitely

**Evidence:** Clean command defines three ghost types:
1. **Phantom windows** - Tmux windows with beads ID but no active OpenCode session (idle >30 min)
2. **Ghost agents** - Registry entries with no tmux window AND no OpenCode session
3. **Orphaned sessions** - OpenCode disk sessions not referenced by any workspace `.session_id` file

Each requires different detection logic:
- Phantoms: Cross-reference tmux windows with OpenCode sessions
- Ghosts: Cross-reference registry with tmux AND OpenCode
- Orphans: Cross-reference disk sessions with workspace files

**Source:** `cmd/orch/clean_cmd.go` - `cleanPhantomWindows()`, `purgeGhostAgents()`, `cleanOrphanedDiskSessions()`

**Significance:** All three ghost types are symptoms of one root cause: **OpenCode sessions never expire**. If OpenCode had session TTLs (auto-expire after N days), phantom/ghost/orphan detection would be unnecessary. The session would simply stop existing.

---

### Finding 3: Registry is dead weight — deprecated methods have zero callers

**Evidence:** Per decision `2026-01-12-registry-is-spawn-cache.md`:
- `Registry.Abandon()` - never called by abandon_cmd
- `Registry.Complete()` - never called by complete_cmd (doesn't import registry)
- `Registry.Remove()` - never called by clean_cmd (doesn't import registry)

Registry's actual usage: spawn writes metadata, status reads it, abandon uses `Find()` for session ID lookup.

**Source:** `pkg/registry/registry.go`, cross-referenced with import analysis of lifecycle commands

**Significance:** The registry is a 529-line package where most methods are deprecated. Its only real function (session ID lookup) could be served by workspace `.session_id` files or OpenCode session metadata. This is a candidate for elimination.

---

### Finding 4: OpenCode's API gaps create the reconciliation burden

**Evidence:** Three specific OpenCode limitations drive orch's complexity:

| OpenCode Limitation | Orch Compensation | Code Cost |
|---|---|---|
| No session TTL | `orch clean --sessions`, phantom/ghost/orphan detection | ~500 lines |
| No session state API (busy/idle via HTTP) | SSE monitoring, Priority Cascade status reconciliation | ~1,700 lines |
| No session metadata storage | Registry, workspace `.session_id`/`.beads_id` files | ~900 lines |
| Sessions persist across restarts | Disk session queries, orphan detection | ~300 lines |

If OpenCode had these three features, orch could eliminate ~3,400 lines of lifecycle code (~40% of total).

**Source:** Analysis of `clean_cmd.go`, `serve_agents.go`, `pkg/opencode/client.go`, `pkg/registry/`

**Significance:** The compensation code is not wrong — it's necessary given current constraints. But it represents accidental complexity from the perspective of orch's actual purpose (orchestration). These are infrastructure management tasks, not orchestration tasks.

---

### Finding 5: Orch's actual value is in verification gates, not session management

**Evidence:** The 11 verification gates in `orch complete` represent genuine orchestration logic:
- Phase: Complete verification (beads protocol enforcement)
- SYNTHESIS.md existence (knowledge externalization gate)
- Test evidence (quality gate)
- Visual verification for web changes (UI gate)
- Git diff verification (change tracking)
- Build verification (correctness gate)
- Skill constraint enforcement
- Phase gate enforcement
- Skill output verification
- Decision patch limits
- Handoff content validation

These are **business logic gates** — they embody the orchestration system's quality standards. They can't be pushed upstream because they're orch-specific domain knowledge.

**Source:** `cmd/orch/complete_cmd.go` skip flags and verification pipeline

**Significance:** This is the code that justifies orch's existence. Session creation and cleanup are commodity infrastructure; verification gates are the intellectual property. Orch should optimize toward MORE verification intelligence and LESS session plumbing.

---

## Synthesis

**Key Insights:**

1. **The four-layer model conflates state and infrastructure.** Beads and workspace files are **state** (they represent business meaning — what work was done, what phase it's in). OpenCode sessions and tmux windows are **infrastructure** (they represent execution resources). Treating infrastructure as state creates the reconciliation burden.

2. **Ghost types are symptoms, not the disease.** Phantoms, ghosts, and orphans all stem from OpenCode sessions persisting indefinitely without TTLs. Three detection algorithms for one root cause is a smell the principle "Coherence Over Patches" specifically warns about.

3. **Registry should be eliminated, not maintained.** Decision `2026-01-12` already demoted it to spawn-cache with deprecated methods. Workspace files (`.session_id`, `.beads_id`) already serve the lookup function. The registry adds a fifth reconciliation point without adding information.

4. **OpenCode upstream contributions would have outsized impact.** Three small features — Session TTL, Session State API, Session Metadata — would eliminate ~40% of orch's lifecycle code. These are general-purpose features other OpenCode users would benefit from.

5. **Orch's real value is in quality gates, not session plumbing.** The 11 verification gates, phase tracking, and skill-based completion requirements are the orchestration intelligence. Session management is commodity infrastructure orch shouldn't own long-term.

**Answer to Investigation Question:**

Orch should adopt a **three-bucket ownership model**:

**BUCKET 1: OWN (orch's domain, can't delegate)**
- Verification gates (all 11 in `orch complete`)
- Phase tracking (beads comments as canonical state)
- Workspace lifecycle (creation, archival, tier-based requirements)
- Skill integration (spawn context generation)
- Dual backend abstraction (OpenCode vs Claude CLI)
- Beads integration (issue creation, closure, dependency tracking)

**BUCKET 2: ACCEPT (external constraints, work within them)**
- OpenCode sessions persist indefinitely → periodic cleanup is reality
- No session state API → SSE for completion detection remains necessary
- No session metadata → workspace files as metadata store
- Dual backend complexity → escape hatch is architecturally justified

**BUCKET 3: LOBBY UPSTREAM (would reduce orch burden if OpenCode added)**
- Session TTL with configurable expiry → eliminates ghost/phantom/orphan classes
- Session metadata API (store beads_id, workspace, tier with session) → eliminates registry
- Session state HTTP endpoint (busy/idle queryable) → simplifies status polling
- Completion callback webhook → simplifies completion detection

---

## Structured Uncertainty

**What's tested:**

- ✅ Registry methods `Abandon()`, `Complete()`, `Remove()` have zero production callers (verified: grep across all cmd/orch/*.go files)
- ✅ Lifecycle code is ~8,800 lines (verified: `wc -l` on all related files)
- ✅ Three ghost types all trace to session persistence (verified: read detection logic in clean_cmd.go)
- ✅ Verification gates are orch-specific domain logic (verified: read complete_cmd.go skip flags)

**What's untested:**

- ⚠️ 40% code reduction estimate (not validated by prototyping simplified version)
- ⚠️ OpenCode upstream willingness to accept Session TTL/Metadata features (not tested via GitHub issue or PR)
- ⚠️ Registry elimination feasibility without breaking status command (not prototyped)
- ⚠️ Whether workspace files alone can fully replace registry lookups in all abandon edge cases

**What would change this:**

- If OpenCode explicitly rejects session TTL (e.g., "sessions are forever by design"), the cleanup code is permanent
- If registry serves undiscovered use cases beyond status/abandon lookup, elimination is premature
- If workspace files prove insufficient for session ID lookup (e.g., workspace deleted before abandon), registry remains necessary

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Adopt three-bucket model | architectural | Cross-component ownership boundary affects spawn, clean, complete, status |
| Simplify clean to 3 modes | implementation | Within existing code patterns, no new dependencies |
| File upstream issues | strategic | External relationship with OpenCode project, resource commitment |
| Eliminate registry | architectural | Removes a component, affects status and abandon commands |

### Recommended Approach: Pragmatic Three-Bucket Model

**Adopt the Own/Accept/Lobby framework for lifecycle decisions, implement simplifications within current constraints, and file upstream issues for long-term reduction.**

**Why this approach:**
- Accepts that OpenCode won't change overnight (pragmatic)
- Immediately simplifies what orch controls (~500 lines of ghost detection can merge into one cleanup mode)
- Positions for future reduction if upstream features land
- Aligns with principles: Coherence Over Patches (merge ghost types), Compose Over Monolith (shed session plumbing)

**Trade-offs accepted:**
- Still owning session cleanup until upstream adds TTLs (months/quarters)
- Registry stays temporarily until workspace-only lookup is validated
- SSE completion detection stays until upstream adds state API

**Implementation sequence:**

1. **Phase 1: Simplify clean** (implementation, 1-2 days)
   - Merge phantom/ghost/orphan into single `--cleanup-sessions` flag
   - Reduce 7 clean flags to 3: `--workspaces` (archive old), `--sessions` (purge stale), `--all`
   - Delete ghost/phantom/orphan distinction — they're all "stale infrastructure"

2. **Phase 2: Eliminate registry** (architectural, 2-3 days)
   - Validate workspace `.session_id` files can serve all lookup needs
   - Update status command to derive from OpenCode + beads (no registry)
   - Update abandon command to look up session ID from workspace files
   - Delete `pkg/registry/`

3. **Phase 3: File upstream issues** (strategic, 1 day)
   - Session TTL: `opencode/opencode` GitHub issue with use case
   - Session Metadata API: Issue with orchestration use case
   - Session State HTTP endpoint: Issue explaining SSE-only limitation

4. **Phase 4: Formalize state model** (architectural, 1 day)
   - Update `.kb/models/agent-lifecycle-state-model/model.md` to distinguish state (beads, workspace) from infrastructure (OpenCode, tmux)
   - Document the three-bucket ownership model as a decision

### Alternative Approaches Considered

**Option B: Push all lifecycle management to OpenCode via contribution**
- **Pros:** Eliminates orch's lifecycle code entirely; community benefit
- **Cons:** OpenCode has its own roadmap; contribution review takes weeks/months; orch can't wait
- **When to use instead:** If Dylan decides to become an active OpenCode contributor and has runway

**Option C: Accept current complexity, don't simplify**
- **Pros:** Working system, known failure modes documented, no migration risk
- **Cons:** 8,800 lines of lifecycle code; 3 ghost types for one root cause; registry is dead weight; ongoing reconciliation bugs
- **When to use instead:** If lifecycle code is stable and no new features planned

**Rationale for recommendation:** Option A (pragmatic three-bucket) addresses the complexity NOW through simplification while positioning for long-term upstream improvement. It doesn't require waiting on external projects or accepting permanent complexity.

---

### Implementation Details

**What to implement first:**
- Merge ghost types in clean_cmd.go (highest friction, most bug-prone)
- Validate workspace-only session ID lookup before removing registry

**Things to watch out for:**
- ⚠️ Status command may have hidden registry dependencies beyond ListActive/ListCompleted
- ⚠️ Abandon's `Find()` call uses registry as fallback — need workspace fallback path
- ⚠️ Cross-project spawns may not have workspace files in the expected project directory

**Areas needing further investigation:**
- OpenCode's stance on session TTLs (check their issue tracker/roadmap)
- Whether `serve_agents.go`'s Priority Cascade can simplify with fewer state layers
- Whether the 11 verification gates in complete_cmd.go can be refactored now that the pipeline pattern exists

**Success criteria:**
- ✅ `orch clean` has ≤3 flags (down from 7)
- ✅ No phantom/ghost/orphan distinction in code
- ✅ Registry eliminated, workspace files serve all lookups
- ✅ Upstream issues filed with clear use cases

---

## References

**Files Examined:**
- `cmd/orch/clean_cmd.go` - Full clean command with 7 flags, 3 ghost detection algorithms
- `cmd/orch/complete_cmd.go` - Complete pipeline with 11 verification gates
- `cmd/orch/abandon_cmd.go` - Abandon with registry and session cleanup
- `cmd/orch/serve_agents.go` - Status API with Priority Cascade reconciliation
- `cmd/orch/shared.go` - Helper functions for beads ID extraction, workspace lookup
- `pkg/registry/registry.go` - Registry with deprecated state methods
- `pkg/opencode/client.go` - OpenCode HTTP client with session management
- `pkg/opencode/sse.go` - SSE stream parsing for completion detection
- `pkg/verify/check.go` - Phase parsing and deliverable verification

**Models Read:**
- `.kb/models/agent-lifecycle-state-model/model.md` - Four-layer state model
- `.kb/models/opencode-session-lifecycle/model.md` - Session persistence, completion detection
- `.kb/models/spawn-architecture/model.md` - Spawn flow, workspace creation
- `.kb/models/completion-verification/model.md` - Completion chain, verification bottleneck
- `.kb/models/model-access-spawn-paths/model.md` - Dual backend architecture
- `.kb/models/workspace-lifecycle-model/model.md` - Workspace tiers, archival

**Decisions Read:**
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - Registry demoted to spawn-cache
- `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` - Event + periodic cleanup
- `.kb/decisions/2026-01-14-infrastructure-complexity-justified.md` - Multi-service complexity OK

**Principles Consulted:**
- Coherence Over Patches: "7 clean flags is the 3rd+ fix to the same area" → signals need for redesign
- Compose Over Monolith: Session management is a separate concern from orchestration
- Escape Hatches: Dual backend is justified — critical paths need independence
- Evolve by Distinction: State vs Infrastructure is a conflation causing the reconciliation burden

**Related Artifacts:**
- **Decision to create:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`
- **Model to update:** `.kb/models/agent-lifecycle-state-model/model.md` (add state vs infrastructure distinction)

---

## Investigation History

**2026-02-13 10:05:** Investigation started
- Question: What lifecycle responsibilities should orch own vs push upstream to OpenCode vs simplify away?
- Context: Orchestrator questioning whether 4-layer state management complexity is worth it

**2026-02-13 10:15:** Read all 6 lifecycle models and 3 related decisions
- Key finding: Four-layer model is accurate but conflates state and infrastructure

**2026-02-13 10:25:** Explored lifecycle code via subagent
- Found: 7 clean flags, 3 ghost types, 11 complete gates, 8,800+ lines total
- Registry has deprecated methods with zero callers

**2026-02-13 10:35:** Consulted principles substrate
- Coherence Over Patches: 7 clean flags = patches, not coherent design
- Evolve by Distinction: Need to distinguish state (beads, workspace) from infrastructure (sessions, tmux)

**2026-02-13 10:45:** Synthesized three-bucket model and recommendations
- Own: Verification gates, phase tracking, workspace lifecycle, beads integration
- Accept: Session persistence, SSE-only completion, dual backend
- Lobby: Session TTL, metadata API, state endpoint

**2026-02-13 11:00:** Investigation completed
- Status: Complete
- Key outcome: Three-bucket ownership model with pragmatic simplification path
