## Summary (D.E.K.N.)

**Delta:** Three structural mechanisms wire existing accretion measurement signals to automated responses, following the proven effectiveness hierarchy (structural > signaling > blocking > advisory).

**Evidence:** 4 probes with judge synthesis confirm: only 4/31 interventions work (all structural/signaling), 35% of codebase is governance (accelerating), feedback loop is broken (0 reworks, 0 quality abandons in 1,113 completions), CLAUDE.md has 8x growth with 92% reference content and no budget.

**Knowledge:** The accretion response layer consists of: (1) `orch reject` closing the feedback loop, (2) accretion.delta event-driven daemon responses replacing periodic scans, (3) CLAUDE.md decomposition with artifact-sync budget. All three consolidate or replace existing infrastructure — zero net governance increase.

**Next:** Create implementation issues for each mechanism. Each is independently deployable.

**Authority:** architectural — Cross-component design (completion pipeline, daemon, CLAUDE.md substrate, event system) requiring synthesis across boundaries.

---

# Investigation: Design Accretion Response Layer

**Question:** How should existing accretion measurement signals be wired to automated structural responses, given that only 4/31 interventions demonstrably work and the governance infrastructure itself is accreting at 35% of the codebase?

**Started:** 2026-03-20
**Updated:** 2026-03-20
**Owner:** architect
**Phase:** Complete
**Next Step:** None — implementation issues created
**Status:** Complete
**Model:** knowledge-accretion

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/knowledge-accretion/probes/2026-03-20-probe-intervention-effectiveness-audit.md` | extends | Yes — all 31 intervention statuses verified against source | None |
| `.kb/models/knowledge-accretion/probes/2026-03-20-probe-governance-infrastructure-self-accretion.md` | extends | Yes — line counts and growth rates confirmed | None |
| `.kb/models/knowledge-accretion/probes/2026-03-20-probe-prompt-context-as-accreting-substrate.md` | extends | Yes — CLAUDE.md at 753 lines, 62% agent-committed | None |
| `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md` | extends | Yes — 0 reworks, 11 operational abandons confirmed | None |
| `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` | builds-on | Yes — advisory decision stands; this design follows its finding that signaling > blocking | None |

---

## Findings

### Finding 1: The Proven Pattern Is Event → Daemon → Issue Creation

**Evidence:** The only code accretion mechanism with measured reduction (12→3 CRITICAL files, 75% reduction) is the daemon extraction cascade: hotspot gate *events* trigger daemon periodic task (`RunPeriodicProactiveExtraction`), which scans files and creates architect issues. The blocking gate had 0% effectiveness. The signaling path produced all the actual structural improvement.

**Source:**
- `pkg/daemon/proactive_extraction.go` — Full implementation (287 lines)
- `cmd/orch/daemon_periodic.go:192` — Integration point
- Intervention effectiveness probe finding #1

**Significance:** This is the template for all three mechanisms. Event emission already exists (`accretion.delta` events in completion pipeline). The gap is that nothing reads those events and reacts. We wire the same pattern: event → daemon reads → threshold → issue creation.

---

### Finding 2: Reject Is Missing, Not Rework

**Evidence:** `orch rework` exists (356 lines, cmd/orch/rework_cmd.go) but has 0 uses because it requires 8 friction points. The real gap is that no "reject" verb exists — a 1-step negative signal equivalent to `orch complete`. All 11 historical abandons are operational (stuck/orphaned agents), not quality judgments. The learning loop has 0 negative signal: skill success rates show 100% because failure paths don't emit distinct events.

**Source:**
- `cmd/orch/rework_cmd.go:69-71` — Mandatory `--bypass-triage` requirement
- `~/.orch/events.jsonl` — 1,113 completed, 0 reworked, 11 abandoned (all operational)
- Human feedback probe recommendations 1-4

**Significance:** `orch reject` is not a replacement for rework — it's a different verb. Reject = "this work failed quality review, reopen issue for reassignment." Rework = "try again with this specific feedback." Reject is the missing feedback channel; rework is an existing but over-complex mechanism.

---

### Finding 3: CLAUDE.md Is an Unbudgeted, Agent-Maintained, Auto-Growing Substrate

**Evidence:** CLAUDE.md is 753 lines (8x growth in 91 days). 92% is reference material (architecture diagrams, package descriptions, event tables, command lists). Only 62 lines (~8%) are directives that shape agent behavior. 62% of modifications are by agents. At least 7 git commits are "artifact drift" updates where the daemon detected new packages/events and spawned an agent to add them to CLAUDE.md — these additions have no size check or relevance filter.

**Source:**
- `git log --oneline --all -- CLAUDE.md` — Shows "artifact drift" update pattern
- `pkg/artifactsync/artifactsync.go` — Drift detection that triggers sync agents
- Prompt context probe content analysis (92% reference, 8% directive)

**Significance:** CLAUDE.md has the highest blast radius of any accreting substrate because every agent in every session reads it. A 10-line addition to CLAUDE.md costs more system-wide than a 100-line addition to daemon_periodic.go (which only affects daemon-related agents). Yet there are multiple gates on code files and zero gates on CLAUDE.md.

---

### Finding 4: Seven Daemon Tasks Are Measurement-Only — Consolidation Candidates

**Evidence:** Of 26 daemon periodic tasks, 7 produce automated actions (create/close issues or spawn agents) and 7 are measurement-only (emit events, write to JSONL, update in-memory state, but never create issues or modify the system). The remaining 12 are core operations (4) or mixed. The measurement-only tasks total ~1,060 lines across 7 files.

**Source:**
- `pkg/daemon/model_drift_reflection.go` (74 lines) — Emits event only
- `pkg/daemon/friction_accumulator.go` (289 lines) — Writes to friction.jsonl only
- `pkg/daemon/plan_staleness.go` (291 lines) — Advisory report only
- `pkg/daemon/investigation_orphan.go` (212 lines) — Advisory report only
- `pkg/daemon/periodic_learning.go` (58 lines) — Updates in-memory state only
- `pkg/daemon/digest.go` (73 lines) — Writes digest files only
- `pkg/daemon/capacity_poll.go` (63 lines) — Caches API data only

**Significance:** These tasks follow the "metrics-only" tier — the lowest effectiveness tier identified in the intervention audit. They collect data but never close the loop to action. Not all should be removed (capacity_poll serves spawn decisions, learning_refresh feeds compliance config), but several could be consolidated or cut.

---

## Synthesis

**Key Insights:**

1. **Wire existing signals, don't build new measurement** — `accretion.delta` events already exist in the completion pipeline (complete_postlifecycle.go:471). `agent.completed` events already exist. The gap is the response side, not the measurement side. Every mechanism in this design consumes existing events rather than creating new measurement infrastructure.

2. **Consolidation offsets additions** — `orch reject` adds ~150 lines, but replacing proactive extraction's periodic file scan with event-driven detection eliminates the periodic scan (~100 lines). CLAUDE.md decomposition removes ~480 lines from the prompt context substrate. Artifact-sync budget adds ~20 lines of size checking. Net: governance code stays flat or shrinks.

3. **Follow the effectiveness hierarchy strictly** — Structural > Signaling > Blocking > Advisory. `orch reject` is structural (it creates a new path in the system). Accretion.delta wiring is signaling (event → daemon → issue). CLAUDE.md budget enforcement should be structural (split the file, not advisory "please keep it small").

**Answer to Investigation Question:**

The accretion response layer consists of three mechanisms, ordered by impact:

**Mechanism 1: `orch reject` (Structural — closes the broken feedback loop)**
- New command: `orch reject <beads-id> "reason"`
- 1-step friction (matching `orch complete`)
- Emits `agent.rejected` event with quality category
- Reopens the beads issue and tags with `rejected` label
- Daemon learning loop gains negative signal (skill failure rates become real)
- No workspace lookup, no mandatory triage bypass, no preconditions beyond beads-id and reason

**Mechanism 2: Accretion.delta → Daemon Response (Signaling — replaces periodic scan)**
- When `accretion.delta` events show a file gained >200 net lines across last 3 completions, daemon creates architect extraction issue
- Replaces `RunPeriodicProactiveExtraction`'s periodic file scan with reactive event-driven detection
- Consolidation: event-driven detection is more precise (reacts to actual agent-caused growth) and eliminates a periodic scan
- Same pattern as existing proactive extraction: scan → dedup → create issue

**Mechanism 3: CLAUDE.md Decomposition + Artifact-Sync Budget (Structural — caps prompt context growth)**
- Decompose CLAUDE.md from 753 → ~250 lines
- Keep: Commands reference (~168 lines), Gotchas (~16 lines), Tab editing (~15 lines), OpenCode Fork (~10 lines), KB Structure (~18 lines), Key References (~16 lines), Related (~7 lines)
- Move to `.kb/guides/`: Architecture overview, Spawn backends + flow, Key packages, Event tracking, Development, Common commands, Dashboard management
- Modify artifact-sync: Before spawning a sync agent, check CLAUDE.md line count against 300-line budget. If over budget, the sync agent should identify and remove lowest-relevance content before adding new content. Transform from "additive only" to "budget-constrained."

---

## Structured Uncertainty

**What's tested:**

- The effectiveness hierarchy (structural > signaling > blocking > advisory) is confirmed across 31 interventions with 2-week measurement (verified: probe-2)
- `accretion.delta` events are already emitted at completion time (verified: `complete_postlifecycle.go:471`, `pkg/events/logger.go:715`)
- The proactive extraction pattern (daemon scan → dedup → create issue) works and has measured results (verified: 12→3 CRITICAL files)
- CLAUDE.md is 753 lines with 92% reference material (verified: `wc -l`, manual section classification)
- 0 reworks and 0 quality abandons in 1,113 completions (verified: events.jsonl grep)

**What's untested:**

- Whether agents with smaller CLAUDE.md context actually perform better (the key missing experiment from the judge's coverage gaps)
- Whether the 200-line accretion.delta threshold is correctly calibrated (needs tuning from production data)
- Whether `orch reject` will actually get used, or whether it follows the same friction pattern as rework

**What would change this:**

- If agents perform *better* with larger CLAUDE.md context (unlikely given 92% reference, but untested), the decomposition recommendation would change
- If `orch reject` shows 0 usage after 30 days, the feedback loop problem is about review culture, not UX friction — would need a different intervention (e.g., mandatory quality assessment in completion flow)
- If accretion.delta threshold produces too many false positives (architect issues for normal file growth), threshold needs raising or context-awareness (e.g., new files expected to grow)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| `orch reject` command | architectural | Cross-component: completion pipeline, event system, daemon learning, beads integration |
| Accretion.delta daemon wiring | architectural | Replaces existing daemon task, changes signal flow |
| CLAUDE.md decomposition | strategic | Affects every agent's context window; irreversible content reorganization |
| Artifact-sync budget gate | architectural | Changes daemon behavior for CLAUDE.md maintenance |
| Daemon task consolidation | architectural | Requires judgment about which measurement tasks to keep |

### Recommended Approach: Three Independent, Sequenced Mechanisms

**Implementation sequence:**

1. **`orch reject` first** — Highest impact, lowest risk. ~150 lines of new code. Independently deployable. Closes the feedback loop immediately. Deploy and measure usage for 2 weeks before proceeding.

2. **CLAUDE.md decomposition second** — Highest blast-radius improvement. No code changes required — pure content reorganization. Creates .kb/guides/ files for extracted content. Can be done in a single session.

3. **Accretion.delta daemon wiring third** — Replaces proactive extraction scan. Requires reading events.jsonl for recent accretion.delta events, aggregating per-file growth, and creating architect issues when thresholds are crossed. Deploy after reject has been running (so accretion.delta events include the reject signal).

### Alternative Approaches Considered

**Option B: Build a comprehensive "accretion dashboard" with visualization**
- **Pros:** Full visibility into accretion trends across substrates
- **Cons:** Measurement-only (lowest effectiveness tier). Adds ~500+ lines of governance code. Violates constraint "must not increase governance percentage."
- **When to use instead:** Never for this problem. Dashboards are for human consumption; this system needs automated responses.

**Option C: Make accretion gates blocking again with better calibration**
- **Pros:** Direct enforcement
- **Cons:** Contradicts decision 2026-03-17 (100% bypass rate measured). Blocking gates fail in systems with capable agents. Previous blocking gates lasted 1 day to 3 weeks before being disabled.
- **When to use instead:** Only for structurally unbypassable gates (like `go build` or model-stub precommit).

### Implementation Details

**`orch reject` design:**
- File: `cmd/orch/reject_cmd.go` (~150 lines)
- Args: `<beads-id> "reason"` (both mandatory)
- Flags: `--category [quality|scope|approach|stale]` (optional, defaults to "quality")
- Actions: (1) Validate beads ID exists and is closed, (2) Reopen issue via `bd reopen`, (3) Add rejection comment with category, (4) Emit `agent.rejected` event, (5) Tag issue with `rejected` and `triage:ready` labels
- Integration: `orch review` should offer reject as an action alongside complete
- Event data: `{beads_id, reason, category, original_skill, original_model}`
- Defect class exposure: Class 6 (Duplicate Action) — need dedup if same issue rejected twice. Mitigation: check for existing `rejected` label.

**CLAUDE.md decomposition plan:**
- Create: `.kb/guides/architecture-overview.md` (~117 lines)
- Create: `.kb/guides/event-tracking.md` (~91 lines)
- Create: `.kb/guides/package-reference.md` (~96 lines)
- Merge into existing: `.kb/guides/spawn.md` (spawn backends + spawn flow, ~57 + ~14 lines)
- Create: `.kb/guides/development.md` (~28 lines)
- Keep in CLAUDE.md: Commands, Gotchas, Tab editing, OpenCode Fork, KB structure, Key references, Related, brief intro
- Update Key References table to include the new guides
- Modify artifact-sync to check CLAUDE.md size before spawning sync agents

**Accretion.delta wiring:**
- New function: `RunPeriodicAccretionResponse()` in `pkg/daemon/`
- Reads recent `accretion.delta` events from events.jsonl (last 7 days)
- Aggregates per-file: sum net_delta across events for same file path
- Threshold: if file net_delta > +200 lines across ≥3 events, create architect issue
- Dedup: check for existing open extraction issues (same as proactive extraction)
- Replaces: `RunPeriodicProactiveExtraction` (consolidation — same purpose, better signal)
- Consolidation notes: proactive extraction scans ALL files periodically regardless of whether anyone touched them. Event-driven response only reacts to files that agents actually grew. More precise, less waste.

**Consolidation targets (recommendations, not blocking):**
- Merge `RunPeriodicPlanStaleness` + `RunPeriodicInvestigationOrphan` into single advisory scan (~300 lines saved)
- Merge `RunPeriodicTriggerExpiry` + `RunPeriodicLightweightCleanup` into single issue-expiry task (~150 lines saved)
- These are independent of the three mechanisms above and can be done separately

**Things to watch out for:**
- `orch reject` must NOT require `--bypass-triage` (that's what killed rework usage)
- CLAUDE.md decomposition must update the Key References table to point to extracted guides
- Accretion.delta wiring must handle the cold-start problem (no events on first run)
- Artifact-sync budget gate should log when it prevents additions (for measurement)

**Success criteria:**
- `agent.rejected` events > 0 within 30 days of deployment
- CLAUDE.md stays under 300 lines for 30 days after decomposition
- At least 1 architect extraction issue created from accretion.delta data (vs periodic scan)
- Governance code percentage does not increase (measure via `orch hotspot` or line count)

---

## References

**Files Examined:**
- `pkg/daemon/proactive_extraction.go` — Template pattern (event → scan → issue)
- `pkg/events/logger.go:715-727` — AccretionDeltaData struct
- `cmd/orch/complete_postlifecycle.go:468-505` — collectAccretionDelta implementation
- `cmd/orch/complete_lifecycle.go:71-73` — accretion.delta emission in completion pipeline
- `cmd/orch/rework_cmd.go` — Existing rework command (356 lines, 0 usage)
- `cmd/orch/daemon_periodic.go` — All 26 periodic tasks (885 lines)
- `pkg/daemon/artifact_sync.go` — Artifact sync daemon task
- `pkg/artifactsync/artifactsync.go` — Drift detection that triggers CLAUDE.md updates
- `CLAUDE.md` — Current state (753 lines)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` — Foundation: signaling > blocking
- **Probe:** `.kb/models/knowledge-accretion/probes/2026-03-20-probe-intervention-effectiveness-audit.md` — 4/31 effectiveness rate
- **Probe:** `.kb/models/knowledge-accretion/probes/2026-03-20-probe-prompt-context-as-accreting-substrate.md` — CLAUDE.md as accreting substrate
- **Probe:** `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md` — Broken feedback loop
- **Probe:** `.kb/models/knowledge-accretion/probes/2026-03-20-probe-governance-infrastructure-self-accretion.md` — 35% governance, accelerating
- **Judge:** `.kb/models/knowledge-accretion/probes/2026-03-20-probe-judge-verdict-accretion-exploration.md` — Cross-probe synthesis
