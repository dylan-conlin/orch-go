<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Gastown and orch-go solve the same problem (agent orchestration) with fundamentally different philosophies: Gastown uses persistent agent identities with hierarchical supervision and autonomous merging, while orch-go uses ephemeral skill-based spawning with flat daemon model and human-gated pushes.

**Evidence:** Examined Gastown source code (internal/*, templates/*), Yegge's blog post, and Maggie Appleton's analysis. Mapped MEOW stack, GUPP mechanism, role hierarchy, and convoy system.

**Knowledge:** Three patterns worth adopting: (1) GUPP hook mechanism for durable work assignment, (2) Wisps for ephemeral orchestration without git pollution, (3) Convoys for delivery tracking. Key philosophy difference is persistent vs ephemeral agent identity.

**Next:** Create beads issue to evaluate GUPP-style hooks for orch-go worker spawn durability. No architectural changes recommended - both systems are coherent with their chosen philosophy.

**Promote to Decision:** recommend-no - This is comparative analysis informing incremental improvements, not architectural pivot.

---

# Investigation: Gastown Orchestration System Analysis and Comparison to orch-go

**Question:** What patterns from Gastown could improve orch-go, and what are the fundamental philosophy differences between the systems?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Investigation worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Gastown Architecture - Role Hierarchy

**Evidence:** Seven distinct worker roles with clear hierarchies:

| Role | Scope | Function |
|------|-------|----------|
| Mayor | Town | Concierge, user's main interface, kicks off convoys |
| Deacon | Town | Daemon beacon, runs patrol loops, propagates DYFJ signal |
| Boot | Town | Special dog that checks on Deacon every 5 minutes |
| Dogs | Town | Deacon's helpers for infrastructure tasks |
| Witness | Per-rig | Pit boss, monitors polecats, pre-kill verification |
| Refinery | Per-rig | Merge queue agent, handles rebasing and merging |
| Polecats | Per-rig | Ephemeral workers, self-cleaning after `gt done` |
| Crew | Per-rig | Named, long-lived user agents for interactive work |

**Source:**
- `templates/polecat-CLAUDE.md` (polecat contract)
- `templates/witness-CLAUDE.md` (witness responsibilities)
- `internal/deacon/manager.go` (deacon startup)
- Yegge's blog post (role descriptions)

**Significance:** Gastown uses hierarchical supervision (Deacon → Witness → Polecats) vs orch-go's flat daemon model. The hierarchy enables "who watches the watchers" through second-order monitoring (Witnesses ping Deacon, Boot checks Deacon).

---

### Finding 2: MEOW Stack - Molecular Expression of Work

**Evidence:** Six-layer work abstraction:

1. **Beads** - Base issue unit, JSON stored in Git (one issue per line)
2. **Epics** - Beads with children, hierarchical plans
3. **Molecules** - Chained beads forming workflows with dependencies
4. **Protomolecules** - Template molecules, class-like workflow definitions
5. **Formulas** - TOML source form for workflow definitions, cooked into protomolecules
6. **Wisps** - Ephemeral beads NOT persisted to Git, burned after completion

**Source:**
- `internal/formula/types.go` - Formula types (convoy, workflow, expansion, aspect)
- `internal/formula/README.md` - Formula system documentation
- `internal/formula/formulas/mol-polecat-work.formula.toml` - Example workflow
- Yegge's blog post - MEOW stack explanation

**Significance:** MEOW provides "Nondeterministic Idempotence" - work survives agent crashes because molecules persist in Git. Wisps solve git pollution for orchestration workflows. orch-go has no equivalent to wisps - all beads are persisted.

---

### Finding 3: GUPP - Gastown Universal Propulsion Principle

**Evidence:** Core propulsion mechanism:

> "If there is work on your hook, YOU MUST RUN IT."

Implementation:
- Hook = special pinned bead with status `hooked` assigned to an agent
- Agents have persistent identities (Agent Beads) stored in Git
- On session start, agent checks hook via `gt hook`
- If hook has work, agent executes immediately without waiting for input
- GUPP Nudge: tmux notification sent ~30-60 seconds after startup to kick idle agents

**Source:**
- `internal/cmd/hook.go` - Hook command implementation
- `internal/deacon/manager.go:74-76` - Initial prompt triggers GUPP
- `templates/polecat-CLAUDE.md:100-118` - Propulsion principle documentation
- Yegge's blog post - GUPP explanation

**Significance:** GUPP solves the "Claude Code ends" problem. Work on hooks survives session restarts, compaction, and handoffs. Key difference from orch-go: orch-go uses SPAWN_CONTEXT.md for initial work assignment but has no hook mechanism for mid-session durability.

---

### Finding 4: Refinery Pattern - Dedicated Merge Queue Agent

**Evidence:** The Refinery is a per-rig agent responsible for:
- Processing merge queue (MQ) one request at a time
- Rebasing branches onto target
- Running tests after merge
- Closing issues after successful merge
- Spawning fresh polecats for conflict resolution

MR states: `open` → `in_progress` → `closed` (with reason: merged/rejected/conflict/superseded)

**Source:**
- `internal/refinery/types.go` - MergeRequest struct and state machine
- `internal/formula/formulas/mol-refinery-patrol.formula.toml` - Refinery patrol workflow

**Significance:** Gastown enables autonomous merging - polecats submit to MQ via `gt done`, Refinery handles merging without human intervention. orch-go uses human-gated pushes (workers commit locally, orchestrator reviews before push). Trade-off: Gastown gains velocity, orch-go gains safety.

---

### Finding 5: Convoys - Work Bundling and Delivery Tracking

**Evidence:** Convoy = special bead tracking multiple issues across rigs:
- Created in town-level beads (hq-* prefix)
- Tracks issues across any rig (cross-prefix routing)
- Auto-closes when all tracked issues complete
- Notifies subscribers (owner, additional via --notify)

**Source:**
- `internal/cmd/convoy.go` - Convoy command implementation
- Yegge's blog post - "Everything in Gas Town, all work, rolls up into a Convoy"

**Significance:** Convoys provide delivery tracking abstraction. orch-go uses bd create/close for individual issues but lacks convoy-style bundling for related work across projects.

---

### Finding 6: Seancing - Predecessor Communication

**Evidence:** `gt seance` enables agents to communicate with their predecessors:
- Uses Claude Code's `/resume` to revive previous session
- Agent can ask predecessor "Where is the stuff you left for me?"
- Predecessors are discoverable via session_id included in startup nudge

**Source:**
- Yegge's blog post - "Talking to your Dead Ancestors"
- `internal/session/startup.go` - Startup nudge with session info

**Significance:** Solves context handoff failures. When an agent hands off and the successor doesn't find the expected state, it can seance the predecessor. orch-go uses SYNTHESIS.md and SESSION_HANDOFF.md for static handoffs but has no dynamic predecessor querying.

---

## Synthesis

**Key Insights:**

1. **Persistent vs Ephemeral Agent Identity** - Gastown agents have permanent identities (Agent Beads) that survive across sessions. Sessions are cattle, agents are pets. orch-go treats sessions as the unit - each spawn is fresh. Gastown's approach enables GUPP (work follows agent), while orch-go's enables clean-slate reasoning (no accumulated cruft).

2. **Hierarchical vs Flat Supervision** - Gastown uses Deacon → Witness → Polecats hierarchy with explicit monitoring protocols. orch-go uses flat daemon with hook-based sensing. Gastown's hierarchy enables "who watches the watchers" (Boot checks Deacon, Witnesses ping Deacon). orch-go's flat model is simpler but may miss stuck orchestrators.

3. **Autonomous vs Human-Gated Merging** - Gastown's Refinery merges autonomously; orch-go requires human review before push. This is a conscious philosophy difference: Gastown optimizes for velocity at the cost of occasional manual rollbacks, orch-go optimizes for safety at the cost of throughput.

**Answer to Investigation Question:**

Three patterns from Gastown worth considering for orch-go:

1. **GUPP-style hooks** - Adding persistent work assignment (hook beads) to orch-go would improve worker spawn durability. Currently if a worker dies mid-task, we rely on SPAWN_CONTEXT.md recovery which may not capture in-progress state.

2. **Wisps for orchestration** - Creating ephemeral beads that don't pollute git would reduce orchestration noise. Patrol-style workflows could use wisps instead of persisted issues.

3. **Convoy-style bundling** - For multi-project features, convoy tracking would improve delivery visibility. Current bd create/close is per-issue without rollup.

Patterns explicitly NOT recommended for adoption:

1. **Autonomous merging** - orch-go's human-gated push is deliberate. Workers commit locally, orchestrator reviews. This catches issues Refinery would auto-merge.

2. **Hierarchical supervision** - orch-go's flat daemon model is simpler and adequate. The added complexity of Deacon → Witness → Polecats isn't justified for our scale.

3. **Persistent agent identities** - Ephemeral skill-based spawning is a feature, not a bug. Clean-slate agents avoid accumulated context cruft.

---

## Structured Uncertainty

**What's tested:**

- ✅ Gastown architecture mapped (verified: read source code, templates, formulas)
- ✅ MEOW stack documented (verified: read formula/types.go, README.md)
- ✅ GUPP mechanism analyzed (verified: read hook.go, deacon/manager.go)
- ✅ Refinery workflow understood (verified: read refinery/types.go, patrol formula)

**What's untested:**

- ⚠️ GUPP hook durability in practice (not run Gastown end-to-end)
- ⚠️ Convoy cross-rig routing performance (read code, not tested)
- ⚠️ Wisp memory usage at scale (theoretical understanding only)

**What would change this:**

- Finding that GUPP has significant failure modes not visible in code
- Discovering orch-go already has equivalent to hooks (would reduce GUPP value)
- Learning that wisp-style ephemeral beads have hidden persistence issues

---

## Implementation Recommendations

**Purpose:** Evaluate GUPP-style hooks for orch-go worker durability.

### Recommended Approach ⭐

**Incremental pattern adoption** - Evaluate GUPP hooks as a durability enhancement without adopting Gastown's full philosophy.

**Why this approach:**
- GUPP solves a real problem (work lost on worker death)
- Can be implemented without persistent agent identities
- Aligns with existing beads infrastructure

**Trade-offs accepted:**
- Won't get full Gastown velocity (keeping human-gated pushes)
- More complexity than current SPAWN_CONTEXT-only approach

**Implementation sequence:**
1. Create RFC evaluating hook-status beads for orch-go
2. Prototype wisp-style ephemeral beads for orchestration
3. Consider convoy-style tracking for multi-repo features

### Alternative Approaches Considered

**Option B: Full Gastown adoption**
- **Pros:** Battle-tested system, Yegge's iteration
- **Cons:** Philosophy mismatch (autonomous merging, persistent agents)
- **When to use instead:** If velocity becomes more important than safety

**Option C: No changes - current system adequate**
- **Pros:** No new complexity, proven workflow
- **Cons:** Misses durability improvements, orchestration noise persists
- **When to use instead:** If worker death is rare and SPAWN_CONTEXT recovery sufficient

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/gastown/templates/polecat-CLAUDE.md` - Polecat contract
- `/Users/dylanconlin/Documents/personal/gastown/templates/witness-CLAUDE.md` - Witness responsibilities
- `/Users/dylanconlin/Documents/personal/gastown/internal/formula/types.go` - Formula types
- `/Users/dylanconlin/Documents/personal/gastown/internal/formula/README.md` - Formula system docs
- `/Users/dylanconlin/Documents/personal/gastown/internal/formula/formulas/mol-polecat-work.formula.toml` - Polecat workflow
- `/Users/dylanconlin/Documents/personal/gastown/internal/formula/formulas/mol-deacon-patrol.formula.toml` - Deacon patrol
- `/Users/dylanconlin/Documents/personal/gastown/internal/cmd/hook.go` - Hook command
- `/Users/dylanconlin/Documents/personal/gastown/internal/cmd/convoy.go` - Convoy command
- `/Users/dylanconlin/Documents/personal/gastown/internal/deacon/manager.go` - Deacon manager
- `/Users/dylanconlin/Documents/personal/gastown/internal/refinery/types.go` - Refinery types
- `/Users/dylanconlin/Documents/personal/gastown/internal/wisp/types.go` - Wisp types

**External Documentation:**
- `~/Documents/personal/blog/sources/welcome-to-gastown.txt` - Yegge's announcement blog
- `~/Documents/personal/blog/sources/gastown-maggie-appleton.md` - Appleton's analysis

---

## Investigation History

**2026-01-23 18:30:** Investigation started
- Initial question: What patterns from Gastown could improve orch-go?
- Context: Gastown open-sourced, both systems use Beads, Go, and tmux

**2026-01-23 18:45:** Architecture mapped
- Identified 7 worker roles with hierarchy
- Documented MEOW stack from code and blog

**2026-01-23 19:00:** Key mechanisms analyzed
- GUPP hook mechanism understood from hook.go and deacon/manager.go
- Refinery merge queue pattern documented
- Convoy delivery tracking mapped

**2026-01-23 19:15:** Investigation completed
- Status: Complete
- Key outcome: Three patterns worth evaluating (hooks, wisps, convoys), three explicitly not adopting (autonomous merge, hierarchy, persistent agents)
