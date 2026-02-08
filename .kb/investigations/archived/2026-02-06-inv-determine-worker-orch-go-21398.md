<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch-go-21398` abandonment resolved an old workspace because `abandon` trusted a stale state.db row (old workspace) and then paired it with a newer live OpenCode session, creating a mixed identity; the "idle+critical died" incident is a manual lifecycle decision, not a proven crash signal.

**Evidence:** State DB contains a single `orch-go-21398` row for `og-audit-architecture-audit-orch-06feb-297f` while five workspace directories have `.beads_id=orch-go-21398`; event log for the abandon command shows `Found agent in state DB: og-audit-...` and `Found OpenCode session: ses_3ca1887d`, then exports transcript into the old workspace path and deletes session `ses_3ca1887d` (whose title/workspace is `og-feat-extract-verify-git-06feb-575b`).

**Knowledge:** `state.db` currently behaves as single-row cache per beads ID (`beads_id UNIQUE`) with insert-only spawn writes and no respawn-safe update path, and session/tmux linkage fields are never populated by runtime calls, so identity drift accumulates and abandon resolution becomes unsafe.

**Next:** Implement respawn-safe state semantics (historical attempts + current pointer), wire `RecordSessionID`/`RecordTmuxWindow`, and harden `abandon` to prefer live workspace/session coherence checks over stale cache rows.

**Authority:** architectural - Fix spans state schema, spawn write lifecycle, and abandon resolution policy across commands/services.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Determine Worker Orch Go 21398

**Question:** Why did worker `orch-go-21398` die as `idle+critical`, and why did abandon resolve a stale workspace mapping before respawn instead of the current workspace?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** OpenCode Worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-01-30-inv-worker-observability-friction.md` | extends | pending | - |
| `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` | deepens | pending | - |
| `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md` | deepens | pending | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Investigation initialized with stale mapping hypothesis

**Evidence:** Spawn context defines the reproduction as state DB drift keyed by `beads_id` with non-respawn-safe semantics and includes a concrete incident where abandon resolved `orch-go-21398` to older workspace `og-audit-architecture-audit-orch-06feb-297f` before respawn.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-determine-worker-orch-06feb-951d/SPAWN_CONTEXT.md:107`; `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-determine-worker-orch-06feb-951d/SPAWN_CONTEXT.md:125`.

**Significance:** Established two test tracks that were both validated: (1) state DB/index drift in beads-to-workspace mapping, and (2) lifecycle decision path causing manual abandon of a session that was then force-deleted.

---

### Finding 2: State DB keeps a stale single row for `orch-go-21398` while multiple respawned workspaces exist

**Evidence:** Runtime query shows exactly one state row for `orch-go-21398` and it points to the old workspace `og-audit-architecture-audit-orch-06feb-297f` with `is_abandoned=1` and empty `session_id`; separate workspace scan shows five directories with `.beads_id` equal to `orch-go-21398` (four newer `og-feat-extract-verify-git-*` + old `og-audit-*`). Schema enforces `beads_id TEXT UNIQUE` and spawn writes use plain `INSERT` (no upsert), and spawn treats state write failures as non-fatal warning.

**Source:** `~/.orch/state.db` query (`SELECT workspace_name, beads_id, session_id, is_completed, is_abandoned, spawn_time, updated_at FROM agents WHERE beads_id='orch-go-21398';`) → `og-audit-architecture-audit-orch-06feb-297f|orch-go-21398||0|1|1770423041838|1770431681452`; grep on `.orch/workspace/**/.beads_id` shows 5 matches for `orch-go-21398`; `/Users/dylanconlin/Documents/personal/orch-go/pkg/state/db.go:142`; `/Users/dylanconlin/Documents/personal/orch-go/pkg/state/agent.go:85`; `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:907`.

**Significance:** Confirms drift mode from reproduction: respawns do not safely replace/advance the beads mapping in cache, so lookup by beads ID can resolve to an obsolete workspace identity.

---

### Finding 3: `orch abandon` composes stale workspace identity with current live session identity

**Evidence:** During the exact abandon action, logs show: `Found agent in state DB: og-audit-architecture-audit-orch-06feb-297f`, then `Found OpenCode session: ses_3ca1887d`, then transcript export path under old `og-audit-*` workspace, then `session.deleted` for `ses_3ca1887d0ffeqldMgZlX1ai7RU` whose title is `og-feat-extract-verify-git-06feb-575b [orch-go-21398]`. Code path explains this: abandon first trusts state-db workspace fields, then independently discovers session ID by beads ID when `session_id` is empty.

**Source:** `/Users/dylanconlin/.orch/event-test.jsonl:18731`; `/Users/dylanconlin/.orch/event-test.jsonl:18732`; `/Users/dylanconlin/.orch/event-test.jsonl:18733`; `/Users/dylanconlin/.orch/event-test.jsonl:18734`; `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/abandon_cmd.go:132`; `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/abandon_cmd.go:174`; `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/abandon_cmd.go:239`.

**Significance:** This is the direct mechanism for the unexpected mapping incident: one command mixed two different agent generations (old workspace + new session) because identity sources were not coherently reconciled.

---

### Finding 4: "idle+critical" reflects manual lifecycle triage, not a verified process crash signature

**Evidence:** The abandoned session (`ses_3ca1887d...`) has an exported transcript with successful tool activity through testing and summary output; no `session.error` event appears for `orch-go-21398`; session termination is explicitly recorded as `session.deleted` during the abandon command. Action log confirms orchestrator initiated `orch abandon ... "Dead session: idle+critical ..."` manually.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-architecture-audit-orch-06feb-297f/SESSION_LOG.md:3`; `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-architecture-audit-orch-06feb-297f/SESSION_LOG.md:330`; grep `session\.error.*orch-go-21398|orch-go-21398.*session\.error` on `~/.orch/events.jsonl` returns no matches; `/Users/dylanconlin/.orch/event-test.jsonl:18734`; `/Users/dylanconlin/.orch/action-log.jsonl:59265`.

**Significance:** The evidence supports "manual abandon after risk triage" rather than "worker crashed on its own". Lifecycle failure here is decision-policy/observability ambiguity, distinct from the state-db drift bug.

---

## Synthesis

**Key Insights:**

1. **State cache drift is real and reproducible** - Single-row `beads_id` cache + insert-only spawn writes + non-fatal write errors yields stale mapping persistence across respawns.

2. **Abandon resolution is currently identity-unsafe** - It can bind stale workspace identity from cache to a different live session identity discovered later from OpenCode.

3. **Lifecycle and cache failures are coupled but distinct** - The session was manually terminated due "idle+critical" triage, while the wrong workspace resolution came from cache drift and abandon lookup order.

**Answer to Investigation Question:**

`orch-go-21398` appeared to "die" because it was manually abandoned after idle/risk triage, not because a crash artifact was detected. The unexpected old workspace mapping happened because `abandon` read stale `state.db` data (`og-audit-architecture-audit-orch-06feb-297f`) for beads ID `orch-go-21398`, then independently discovered and deleted the newer session (`ses_3ca1887d...`) that belonged to workspace `og-feat-extract-verify-git-06feb-575b`. This is a verified cache-drift + identity-composition bug in abandon flow, separate from lifecycle triage policy ambiguity.

---

## Structured Uncertainty

**What's tested:**

- ✅ State DB currently maps `orch-go-21398` to old workspace only (verified via sqlite query on `~/.orch/state.db`).
- ✅ Multiple workspace directories share `.beads_id=orch-go-21398` (verified via workspace grep).
- ✅ Abandon command used old workspace + newer session simultaneously (verified via `event-test.jsonl` tool-output + `session.deleted` record).
- ✅ No explicit `session.error` event was recorded for this issue during incident window (verified via grep on `~/.orch/events.jsonl`).

**What's untested:**

- ⚠️ Whether missing `bd` phase comments during the 18:21-18:34 attempt were lost due beads transport/state behavior vs operator-view mismatch.
- ⚠️ Whether OpenCode `/session` ordering can also misbind session selection for duplicate beads IDs in other commands.
- ⚠️ Whether daemon recovery paths can produce the same stale-workspace/session cross-binding under high retry churn.

**What would change this:**

- If replay with instrumented abandon showed workspace resolution always derived from live session metadata first, this root cause would be wrong.
- If state.db were shown to contain per-attempt history and current-pointer semantics (not single stale row), drift hypothesis would weaken.
- If `session.error`/process-crash telemetry exists for `ses_3ca1887d...` in another authoritative store, lifecycle conclusion would shift toward true runtime failure.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Introduce attempt-aware state model (`agent_attempts` + `current_attempt_by_beads`) and make abandon resolve through current attempt with live validation fallback | architectural | Requires schema + spawn + abandon + status pipeline alignment across components |
| Wire runtime identity updates (`RecordSessionID`, `RecordTmuxWindow`) immediately after session/window creation | implementation | Code exists but is unused; bounded change in spawn flows |
| Add abandon safety checks: if db row is abandoned/completed or workspace/session mismatch is detected, re-resolve by latest workspace `.session_id`/spawn_time and warn loudly | implementation | Command-local hardening with no cross-product policy change |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Attempt-Aware State + Coherent Abandon Resolution** - Track every spawn attempt and explicitly resolve the current attempt before destructive actions.

**Why this approach:**
- Eliminates stale single-row cache ambiguity by design.
- Prevents destructive commands from mixing identities across attempts.
- Directly addresses observed drift and mismatched workspace/session composition.

**Trade-offs accepted:**
- Slight schema and migration complexity for state DB.
- Additional runtime checks in abandon path before delete/kill actions.

**Implementation sequence:**
1. Add attempt history schema and current-attempt pointer, with migration from existing `agents` row.
2. Update spawn to insert new attempt and set current pointer atomically (no insert-only drift).
3. Update abandon/complete/status to resolve current attempt first, validate workspace↔session coherence, and only then perform destructive actions.

### Alternative Approaches Considered

**Option B: Minimal patch (upsert existing row by beads_id)**
- **Pros:** Quick reduction in stale mapping incidents.
- **Cons:** Loses attempt history and can still mask cross-attempt lifecycle data needed for diagnostics.
- **When to use instead:** Emergency hotfix when full schema work cannot ship yet.

**Option C: Keep schema, harden abandon only**
- **Pros:** Lowest-risk command-local change.
- **Cons:** Leaves state drift unresolved; other commands can still mis-resolve mappings.
- **When to use instead:** Temporary mitigation while architectural fix is in flight.

**Rationale for recommendation:** Option A fixes both root classes (drift and destructive-resolution safety) whereas B/C only reduce symptoms in part of the flow.

---

### Implementation Details

**What to implement first:**
- Add/enable `RecordSessionID` and `RecordTmuxWindow` calls in headless/tmux spawn paths.
- In `runAbandon`, reject stale db rows (`is_abandoned`/`is_completed`) as primary identity source.
- Add invariant test: abandoning beads ID with multiple attempts must target newest attempt workspace/session pair.

**Things to watch out for:**
- ⚠️ Multiple sessions with same beads ID in OpenCode list may still require deterministic latest-session selection.
- ⚠️ Migration must preserve existing status display behavior for historic rows.
- ⚠️ Avoid any destructive fallback that can kill orchestrator sessions by loose title matching.

**Areas needing further investigation:**
- Why successful `bd comment` commands in action-log did not all appear in current issue comment timeline.
- Whether daemon recovery (`ResumeAgentByBeadsID`) has parallel stale-resolution behavior under duplicate beads attempts.
- Whether status/coaching UI should explicitly label "manual abandon candidate" vs "crash evidence".

**Success criteria:**
- ✅ Reproducing this scenario resolves to current workspace/session pair, never an older attempt.
- ✅ State DB shows per-attempt continuity for respawns of same beads ID.
- ✅ Abandon logs include explicit coherence check results (workspace/session/beads triad).

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/abandon_cmd.go` - primary abandon identity-resolution flow.
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/state/db.go` - schema constraints on `agents` table.
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/state/agent.go` - insert/update semantics and lookup behavior.
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/state/integration.go` - spawn/complete/abandon integration points.
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go` - non-fatal state write behavior.
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/shared.go` - workspace lookup fallback behavior.
- `/Users/dylanconlin/.orch/events.jsonl` - spawn/abandon timeline for `orch-go-21398`.
- `/Users/dylanconlin/.orch/event-test.jsonl` - exact abandon command output and session deletion evidence.
- `/Users/dylanconlin/.orch/action-log.jsonl` - command-level chronology including manual abandon trigger.
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-architecture-audit-orch-06feb-297f/SESSION_LOG.md` - exported transcript showing session identity mismatch.

**Commands Run:**
```bash
# Inspect state mapping for issue under investigation
sqlite3 "$HOME/.orch/state.db" "SELECT workspace_name, beads_id, session_id, is_completed, is_abandoned, spawn_time, updated_at FROM agents WHERE beads_id='orch-go-21398';"

# Verify multiple workspaces carry same beads ID
grep pattern="orch-go-21398" path=".orch/workspace" include=".beads_id"

# Verify schema constraint driving one-row-per-beads behavior
sqlite3 "$HOME/.orch/state.db" "SELECT sql FROM sqlite_master WHERE type='table' AND name='agents';"

# Extract all spawn attempts for this beads ID
python3 - <<'PY'
import json
for line in open('/Users/dylanconlin/.orch/events.jsonl'):
    ev=json.loads(line)
    if ev.get('type')=='session.spawned' and ev.get('data',{}).get('beads_id')=='orch-go-21398':
        print(ev['timestamp'], ev['data'].get('workspace'), ev.get('session_id') or ev['data'].get('session_id'))
PY

# Verify session/tmux state writers are currently unused callsites
grep pattern="RecordSessionID\\(" path="/Users/dylanconlin/Documents/personal/orch-go" include="*.go"
grep pattern="RecordTmuxWindow\\(" path="/Users/dylanconlin/Documents/personal/orch-go" include="*.go"

# Verify no explicit session.error was emitted for this issue
grep pattern="session\\.error.*orch-go-21398|orch-go-21398.*session\\.error" path="/Users/dylanconlin/.orch" include="events.jsonl"
```

**External Documentation:**
- N/A - Investigation relied on primary local code and runtime artifacts.

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-30-inv-worker-observability-friction.md` - prior context on worker visibility failure patterns.
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-architecture-audit-orch-06feb-297f` - stale workspace that abandon resolved.
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-extract-verify-git-06feb-575b` - actual session workspace tied to deleted session ID.

---

## Investigation History

**[2026-02-06 18:35]:** Investigation started
- Initial question: Why did `orch-go-21398` appear dead and why did abandon resolve old workspace mapping?
- Context: Incident log documented idle+critical abandon with stale workspace resolution.

**[2026-02-06 18:40]:** Reproduced state drift and stale mapping
- Confirmed one stale state row for `orch-go-21398` vs five workspace attempts with same `.beads_id`.

**[2026-02-06 18:45]:** Verified abandon mixed stale workspace with live session
- Event stream showed `Found agent in state DB: og-audit-...` followed by `Found OpenCode session: ses_3ca1887d...` and transcript export to old workspace path.

**[2026-02-06 18:50]:** Investigation completed
- Status: Complete
- Key outcome: Root cause is state.db drift + abandon identity composition bug; lifecycle "death" is manual triage path, not proven crash telemetry.
