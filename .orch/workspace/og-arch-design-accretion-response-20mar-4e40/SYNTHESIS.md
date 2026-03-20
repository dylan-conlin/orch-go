# Session Synthesis

**Agent:** og-arch-design-accretion-response-20mar-4e40
**Issue:** orch-go-347uy
**Duration:** 2026-03-20
**Outcome:** success

---

## Plain-Language Summary

Designed three mechanisms that wire existing accretion measurement signals to automated structural responses, based on exploration findings that only 4 of 31 accretion interventions work (all structural or signaling). The three mechanisms are: (1) `orch reject` — a 1-step command that closes the broken feedback loop (currently 0 negative signals in 1,113 completions), (2) event-driven daemon response to `accretion.delta` events — replacing the periodic file scan with reactive detection when agents actually grow files, and (3) CLAUDE.md decomposition from 753 to ~250 lines plus an artifact-sync budget gate that prevents the daemon from endlessly adding reference material. All three follow the proven effectiveness hierarchy (structural > signaling) and consolidate or replace existing infrastructure rather than adding governance code.

---

## Verification Contract

See investigation: `.kb/investigations/2026-03-20-inv-design-accretion-response-layer-wire.md`

Key outcomes:
- Three mechanisms designed with implementation details
- Each follows proven pattern (event → daemon → issue creation)
- Net governance code: flat or decreasing (consolidation)
- Implementation issues created for independent deployment

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-20-inv-design-accretion-response-layer-wire.md` — Full architect design with 3 mechanisms, fork analysis, implementation details

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Added intervention effectiveness hierarchy, response layer section

### Commits
- (pending)

---

## Evidence (What Was Observed)

- Only 4/31 accretion interventions have measured evidence of reducing their target (probe-2 scorecard)
- Effectiveness hierarchy: structural attractors (model/probe dirs) > signaling (daemon cascades) > blocking (100% bypass) > advisory (ignored) > metrics-only
- 35% of codebase is governance/measurement, growing at 77% rate in March (probe-3)
- `accretion.delta` events are emitted at completion time (`complete_postlifecycle.go:471`) but nothing reads them for automated response
- CLAUDE.md: 753 lines, 92% reference, 8% directive, 62% agent-committed, 35% auto-synced via artifact drift agents
- 0 reworks, 0 quality abandons in 1,113 completions (probe-4)
- Proactive extraction pattern (daemon → scan → issue) works: 12→3 CRITICAL files
- 7 daemon periodic tasks are measurement-only (emit events but never create issues or take action)

---

## Architectural Choices

### Choice 1: `orch reject` as separate command, not `--reject` flag on complete
- **What I chose:** New `orch reject <id> "reason"` command
- **What I rejected:** Adding `--reject` flag to `orch complete`
- **Why:** Complete and reject are semantically opposite operations. Complete closes an issue as done; reject reopens it for reassignment. Combining them into one command confuses the mental model. Separate verbs make the action unambiguous.
- **Risk accepted:** Two commands to learn vs one.

### Choice 2: Replace proactive extraction with event-driven accretion response
- **What I chose:** Wire `accretion.delta` events to daemon response, replacing periodic file scan
- **What I rejected:** Keeping both mechanisms (periodic scan + event-driven)
- **Why:** Proactive extraction scans ALL files every cycle regardless of whether anyone touched them. Event-driven response only reacts to files that agents actually grew. More precise, less waste, consolidation (not addition).
- **Risk accepted:** Cold-start problem on first run (no events yet). Mitigated: fall back to proactive extraction scan if no events exist.

### Choice 3: CLAUDE.md decomposition over size gate
- **What I chose:** Physically decompose CLAUDE.md into smaller files, keep only essential content
- **What I rejected:** Adding a pre-commit size gate on CLAUDE.md
- **Why:** Pre-commit gates have 100% bypass rate in this system (decision 2026-03-17). The structural approach (fewer lines in the file) can't be bypassed. Moving content to .kb/guides/ doesn't lose it — it just stops injecting it into every agent's context.
- **Risk accepted:** Agents may miss reference material that was in CLAUDE.md. Mitigated: Key References table in CLAUDE.md points to extracted guides. SPAWN_CONTEXT can inject relevant sections.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-20-inv-design-accretion-response-layer-wire.md` — Full architect investigation with design

### Constraints Discovered
- CLAUDE.md is loaded by Claude Code before SPAWN_CONTEXT — it cannot be split into multiple files that Claude Code auto-loads (only CLAUDE.md and ~/.claude/CLAUDE.md are auto-loaded)
- Artifact-sync spawns agents to update CLAUDE.md — the sync is semi-automated (daemon detects drift, spawns agent, agent edits file)
- `orch reject` must NOT require `--bypass-triage` — that's the friction that killed rework (0 usage)

---

## Next (What Should Happen)

**Recommendation:** close — implementation issues created for each mechanism

### Implementation Issues Created

See investigation for full implementation details. Each mechanism is independently deployable.

---

## Unexplored Questions

- **Does smaller CLAUDE.md actually improve agent performance?** This is the key missing experiment from the judge's coverage gaps. The decomposition is justified by blast-radius reduction regardless, but measuring quality improvement would strengthen the case.
- **Will `orch reject` get used?** If it shows 0 usage after 30 days, the problem is review culture, not UX friction.
- **What's the right accretion.delta threshold?** 200 lines across 3 completions is a starting guess. Production data needed for calibration.

---

## Friction

No friction — smooth session. All probe data was accessible, codebase was navigable.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-accretion-response-20mar-4e40/`
**Investigation:** `.kb/investigations/2026-03-20-inv-design-accretion-response-layer-wire.md`
**Beads:** `bd show orch-go-347uy`
