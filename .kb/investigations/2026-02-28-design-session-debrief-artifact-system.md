---
status: open
type: design
date: 2026-02-28
triggers: ["orchestrator debrief comprehension evaporates between sessions", "orch orient surfaces facts but not comprehension"]
---

**Status:** Active
**Phase:** Complete

# Design: Session Debrief Artifact System

## Problem Statement

The orchestrator produces comprehension at session end — understanding of why work mattered, what changed about constraints, how threads connect. This is the orchestrator's unique output: not raw data, but synthesized understanding. Currently this evaporates when the session closes.

**The gap exists specifically for interactive orchestrator sessions** (Dylan + Claude in Claude Code). Spawned orchestators persist via SESSION_HANDOFF.md. Workers persist via SYNTHESIS.md. But interactive sessions — which are the majority pattern — produce debriefs conversationally that die with the session.

`orch orient` surfaces facts (throughput, ready work, model freshness) but has no comprehension layer. It tells you *what the state is* but not *how you got here* or *why it matters*.

**Success criteria:** The next orchestrator session starts with both facts (what is) and comprehension (what changed, why it matters, what threads are active).

**Constraints:**
- Must not duplicate SESSION_HANDOFF.md (for spawned orchestrators)
- Must not duplicate MEMORY.md (for tactical agent memory)
- Must integrate with existing `orch orient` infrastructure
- Must respect Session Amnesia principle — every session starts from externalized state
- Must respect Capture at Context principle — capture during debrief, not deferred

## Design Context (Decided)

These decisions were made before this investigation:

- **Artifact location:** `.kb/sessions/YYYY-MM-DD-debrief.md` (one per day, not per session)
- **Knowledge candidates:** Captured inline as `kb quick` entries during debrief writing, not deferred
- **Template structure:** Threads, What Changed, In Flight, Next

## Design Forks

### Fork 1: How should orch orient consume debriefs?

**Question:** How many recent debriefs? What gets surfaced vs omitted? What format?

**Options:**

A. **Read last 1 debrief, extract structured summary (2-3 sentences per section)**
   - Orient reads most recent debrief file
   - Extracts: "## Threads" summaries, "## What Changed" items, "## Next" proposals
   - Truncates to fit orient's existing format (100-char summaries)
   - Adds new section: "Previous session:" to orient output

B. **Read last 3 debriefs, produce rolling comprehension window**
   - Richer context but more to process
   - Risk: stale threads from 3 days ago polluting current orientation
   - More parsing complexity

C. **Read last 1 debrief, inject full content as separate section**
   - No summary extraction, just present the whole debrief
   - Simpler implementation but potentially verbose

**Substrate says:**
- Principle: Session Amnesia — externalize everything, but the *next* session needs to quickly orient, not read a novel
- Model: `orch orient` constants — `truncateSummaryLen = 100`, `maxReadyIssues = 3` — orient is designed for concise, scannable output
- Decision: "Designed for orchestrator consumption, not direct human use" — orient is read by the orchestrator agent, who then presents conversationally to Dylan
- Flow-integrated knowledge surfacing investigation: "Surface summaries and pointers, not full files. 2-3 sentences per model"

**RECOMMENDATION:** Option A — last 1 debrief, structured summary extraction.

**Rationale:** Orient is consumed by the orchestrator agent, not Dylan directly. The orchestrator has full context of what it reads and presents conversationally. One debrief is sufficient because:
1. The debrief captures "What's Next" which seeds the new session
2. Multi-day context is already captured by throughput metrics (events.jsonl), beads state, and model freshness
3. If something from 3 days ago matters, it will be in a model, decision, or beads issue — not in the debrief

**Implementation shape:**
```
== SESSION ORIENTATION ==

Last 24h:
   Completions: 2 | Abandonments: 1 | In-progress: 3
   Avg duration: 38 min

Previous session:
   Threads: Shipped debrief artifact design, completed hotspot extraction for spawn_cmd.go
   Changed: New constraint — debriefs are orchestrator-only, not worker artifacts
   In flight: 2 agents working (orch-go-abc1, orch-go-def2), 1 pending question
   Next: Prototype orch debrief command, review hotspot results

Ready to work:
   [P1] Fix spawn crash on empty skill (orch-go-abc1)
   ...
```

**Trade-off accepted:** Losing multi-day comprehension rolling window. Mitigated by: multi-day context lives in models/decisions/beads, which orient already surfaces.

**When this would change:** If Dylan starts having 4+ day gaps between sessions, a 3-day lookback would help recovery. Add `--lookback N` flag then.

---

### Fork 2: Should the debrief be written by a command or manually?

**Question:** `orch debrief` command that auto-populates vs orchestrator writes manually vs hybrid?

**Options:**

A. **`orch debrief` command that auto-populates then orchestrator fills comprehension**
   - Command creates/appends to `.kb/sessions/YYYY-MM-DD-debrief.md`
   - Auto-populates from sources the system already has:
     - Threads: from `bd list --status=in_progress` + recent completions from events.jsonl
     - In flight: from `orch status` (active agents)
     - Throughput: from events.jsonl (session-scoped if possible, else day-scoped)
   - Orchestrator fills in: What Changed (comprehension), Next (strategic framing)
   - Flags: `--threads "summary"`, `--changed "summary"`, `--next "summary"` for orchestrator to pass programmatically

B. **Fully manual — orchestrator writes the file directly**
   - Most flexible, no tooling needed
   - But: friction means it won't happen consistently
   - Anti-pattern: "I'll document later" → later never comes (from CLAUDE.md)

C. **Fully automated — `orch debrief` produces complete artifact**
   - Can auto-populate facts but cannot produce comprehension
   - A debrief without "what changed" or "why it matters" is just `orch status` in markdown
   - Violates the core premise: the debrief's value IS the comprehension layer

**Substrate says:**
- Principle: Gate Over Remind — make the valuable thing the easy path, not something you remember to do
- Principle: Capture at Context — capture during the moment of understanding, not after
- CLAUDE.md: "Anti-pattern: 'I'll document later' → later never comes"
- User interaction model: Dylan interacts conversationally. Orchestrator calls CLI tools programmatically. Orchestrator asks Dylan conversationally, then calls `orch debrief --changed "..." --next "..."`

**RECOMMENDATION:** Option A — hybrid `orch debrief` command with orchestrator-provided comprehension.

**Rationale:** The orchestrator already runs the debrief protocol conversationally (orchestrator skill lines 365-383). Adding a command that:
1. Auto-populates what the system knows (threads from beads/events, in-flight from status)
2. Requires comprehension input via flags (what changed, what's next)
3. Creates/appends to the day's debrief file

This matches the user interaction model: orchestrator asks Dylan "What changed today?", Dylan answers conversationally, orchestrator runs `orch debrief --changed "New constraint: X" --next "Thread 1, Thread 2"`.

**Implementation shape:**
```bash
# Orchestrator runs after conversational debrief with Dylan
orch debrief \
  --threads "Shipped debrief design (orch-go-o8np). Completed hotspot extraction." \
  --changed "New constraint: debriefs are orchestrator-only. Decision: orient reads 1 debrief." \
  --next "Implement orch debrief command. Review hotspot agent results."

# Creates/appends to .kb/sessions/2026-02-28-debrief.md
```

**Trade-off accepted:** Requires orchestrator to call the command with flags. Mitigated by: orchestrator already has the information from the conversational debrief and programmatic CLI usage is the established pattern.

**When this would change:** If debriefs consistently have poor comprehension quality, add interactive mode where the command prompts for each section.

---

### Fork 3: How does this interact with MEMORY.md?

**Question:** Does the debrief replace some of what MEMORY.md currently captures?

**Options:**

A. **Distinct artifacts, no overlap — debrief is comprehension, MEMORY.md is tactical**
   - MEMORY.md: "how to build", "known gotchas", "current refactoring patterns" → agent-scoped
   - Debrief: "what happened", "what changed", "what's next" → human-scoped via orchestrator
   - No migration needed, both serve different audiences

B. **Debrief absorbs MEMORY.md's "session close state" role**
   - MEMORY.md currently captures some session-scoped context (e.g., "Pipeline Refactoring Pattern completed Feb 28")
   - Move this to debrief, keep MEMORY.md purely for stable tactical knowledge
   - Requires editing MEMORY.md guidance

C. **Merge them — one artifact for everything**
   - Conflates two distinct audiences (agent vs orchestrator-for-human)
   - Violates Evolve by Distinction

**Substrate says:**
- Principle: Evolve by Distinction — "when problems recur, ask what are we conflating?"
- CLAUDE.md knowledge placement table: MEMORY.md = "Session-tactical context" → "Where did we leave off?"
- Flow-integrated surfacing investigation: MEMORY.md "serves individual agent orientation" vs models "should surface at completion/start"

**RECOMMENDATION:** Option A — distinct artifacts, no overlap.

**Rationale:** These serve fundamentally different purposes and audiences:

| | MEMORY.md | Debrief |
|---|---|---|
| **Audience** | Next Claude agent | Orchestrator (for Dylan) |
| **Content** | How to work here (build cmds, gotchas) | What happened and why it matters |
| **Lifespan** | Evolves slowly (stable patterns) | Daily (one per day) |
| **Read by** | Auto-loaded into every session | `orch orient` at session start |
| **Written by** | Any agent discovering patterns | Orchestrator at session end |

Conflating these would violate Evolve by Distinction. They look similar (both persist across sessions) but serve different functions.

**Trade-off accepted:** Two artifacts to maintain. Mitigated by: they update at different rates (MEMORY.md rarely, debriefs daily).

**When this would change:** If MEMORY.md starts capturing session-scoped state that belongs in debriefs, migrate those entries.

---

### Fork 4: Multi-session days — append or overwrite?

**Question:** If there are 3 sessions in one day, how does the orchestrator add to the day's file?

**Options:**

A. **Append with session delimiter**
   - Each `orch debrief` call appends a timestamped section to the day's file
   - Orient reads all sections from the most recent day
   - Structure: `## Session 2 (15:30)` headers within the day file

B. **Overwrite — last session wins**
   - Simple but loses earlier sessions' context
   - A morning session's insights would be lost if afternoon session runs

C. **One file per session, not per day**
   - `.kb/sessions/2026-02-28-1430-debrief.md`
   - More files, harder to orient (which to read?)
   - Contradicts the "one per day" decision

**Substrate says:**
- Decision (already made): "one per day, not per session"
- Principle: Session Amnesia — if morning session debrief is overwritten, that comprehension is lost
- `orch debrief` command context: the command knows the current time, can add session markers

**RECOMMENDATION:** Option A — append with timestamped session delimiter.

**Rationale:** A day with 3 sessions has 3 distinct comprehension moments. The morning session might discover a constraint that the afternoon session builds on. Overwriting loses this.

**Implementation shape:**
```markdown
# Debrief: 2026-02-28

## Session 1 (09:15)

### Threads
- Shipped debrief artifact design (orch-go-o8np)

### What Changed
- New constraint: debriefs are orchestrator-only

### In Flight
- 2 agents working on hotspot extraction

### Next
- Review hotspot results, implement orch debrief

---

## Session 2 (15:30)

### Threads
- Completed hotspot review, implemented orch debrief command

### What Changed
- Decision: orient reads last 1 debrief only

### In Flight
- 1 agent finishing test coverage

### Next
- Integration test orch orient + debrief, ship to production
```

**Orient consumption for multi-session days:** Read the LAST session section only. Earlier sessions' "Next" proposals have been consumed by subsequent sessions.

**Trade-off accepted:** File grows within a day. Mitigated by: typical day has 1-3 sessions, each adding ~10-15 lines. A busy day might reach 60 lines — manageable.

**When this would change:** If sessions produce very verbose debriefs, add a size cap per session section.

---

### Fork 5: Staleness and archival

**Question:** Should old debriefs be archived/pruned? What's the useful lookback window?

**Options:**

A. **No archival — debriefs accumulate in .kb/sessions/**
   - Simple, grep-friendly
   - `.kb/sessions/` could grow to 365 files per year
   - But: each file is small (10-60 lines), and `orch orient` only reads the most recent

B. **Monthly archival — move to .kb/sessions/archived/**
   - After 30 days, debriefs move to archived subdirectory
   - Keeps working directory clean
   - Adds maintenance task

C. **Auto-prune after N days**
   - Delete debriefs older than N days
   - Permanent information loss
   - Comprehension that mattered should have been captured in models/decisions

D. **Quarterly consolidation — compress 3 months into 1 summary**
   - Rich but complex
   - Over-engineering for v1

**Substrate says:**
- Principle: Avoid over-engineering — "the right amount of complexity is the minimum needed for the current task"
- System precedent: `.kb/investigations/archived/` pattern exists for archiving old investigations
- Orient reads only last 1 debrief — old debriefs are reference material, not active

**RECOMMENDATION:** Option A (no archival) for v1, with documented path to Option B.

**Rationale:** `orch orient` only reads the most recent debrief. Old debriefs are only useful for manual grep ("what happened on Feb 15?"). At ~365 small files per year and no automated consumption of old files, archival is premature optimization.

**Path to Option B:** If `.kb/sessions/` exceeds 100 files and becomes noisy, add `orch debrief archive --days 30` that moves old files to `.kb/sessions/archived/`. The pattern is established (`.kb/investigations/archived/`).

**Trade-off accepted:** Accumulating files. Mitigated by: files are small, only the latest matters to tooling.

**When this would change:** When `.kb/sessions/` exceeds 100 files OR when a debrief consumption pattern emerges that needs multi-week lookback.

---

## Synthesis: Recommended Design

### Artifact: `.kb/sessions/YYYY-MM-DD-debrief.md`

**One file per day.** Multiple sessions append with timestamped delimiters.

**Template structure:**
```markdown
# Debrief: YYYY-MM-DD

## Session N (HH:MM)

### Threads
- [beads-id] Brief outcome description (1 sentence per thread)

### What Changed
- [constraint|decision|model] Description of durable change

### In Flight
- [beads-id] Agent description, current status
- Open question: [question text]

### Next
- Proposed thread 1 (strategic framing, not to-do)
- Proposed thread 2
```

### Command: `orch debrief`

**Auto-populates:**
- Session timestamp
- Threads: from events.jsonl completions + beads comments for current day
- In Flight: from `orch status` (active agents) + `bd list --status=in_progress`

**Requires orchestrator input (via flags):**
- `--threads "override"` — orchestrator can replace auto-populated threads with comprehension summary
- `--changed "..."` — what changed (constraints, decisions, model updates). **REQUIRED** — the command should warn if omitted
- `--next "..."` — proposed threads for next session. **REQUIRED**

**Behavior:**
- If `.kb/sessions/YYYY-MM-DD-debrief.md` exists, append new session section
- If not, create file with header
- Create `.kb/sessions/` directory if it doesn't exist

### Orient integration: `orch orient` reads latest debrief

**New section in orient output:**

```
Previous session:
   Threads: Shipped debrief design, completed hotspot extraction
   Changed: New constraint — debriefs orchestrator-only
   In flight: 2 agents (orch-go-abc1, orch-go-def2), 1 open question
   Next: Implement debrief command, review hotspot results
```

**Implementation in `pkg/orient/`:**
1. New function `collectRecentDebrief()` — finds most recent `.kb/sessions/*.md` file
2. Parses the LAST session section (for multi-session days)
3. Extracts: Threads, What Changed, In Flight, Next — truncated to orient format
4. New field in `OrientationData`: `RecentDebrief *DebriefSummary`
5. New format function `formatRecentDebrief()` in FormatOrientation

**Constants:**
```go
maxDebriefThreads = 3        // Top 3 thread summaries
maxDebriefChanged = 3        // Top 3 changes
maxDebriefInFlight = 3       // Top 3 in-flight items
truncateDebriefLen = 120     // Slightly longer than model summaries
```

### MEMORY.md relationship

**No change to MEMORY.md.** Distinct artifacts:
- MEMORY.md = tactical agent knowledge (stable patterns, build commands)
- Debrief = daily comprehension (what happened, why it matters)

### Staleness

**No archival for v1.** Files accumulate in `.kb/sessions/`. Path to archival documented if needed (>100 files).

---

## Recommendations

⭐ **RECOMMENDED:** Implement as described above — `orch debrief` command + orient integration.

**Why:** Fills the specific gap between facts (orient) and comprehension (debrief) that exists for interactive orchestrator sessions. Matches existing patterns (auto-populate facts, require human comprehension), respects Session Amnesia, and integrates cleanly with `orch orient`.

**Trade-offs accepted:**
1. Requires orchestrator to call `orch debrief` with flags (vs fully automated) — accepted because comprehension can't be automated
2. Orient reads only last 1 debrief (vs rolling window) — accepted because multi-day context lives in models/decisions
3. No archival (vs monthly cleanup) — accepted because files are small and only latest matters

**Expected outcome:** Orchestrator sessions start with both "what is the state?" AND "what happened yesterday and why it matters?" — enabling Dylan to make informed decisions from the first minute.

## Implementation Phases

**Phase 1: Template and directory structure**
- Create `.kb/sessions/` directory
- Define debrief template in `.orch/templates/DEBRIEF.md`
- Skill: feature-impl

**Phase 2: `orch debrief` command**
- New command in `cmd/orch/debrief_cmd.go`
- Auto-populate from events.jsonl and orch status
- Accept `--threads`, `--changed`, `--next` flags
- Create/append to day's debrief file
- Skill: feature-impl

**Phase 3: Orient integration**
- Add `collectRecentDebrief()` to `pkg/orient/`
- New `DebriefSummary` type in orient types
- Parse last session section from most recent debrief file
- Add "Previous session:" section to orient output
- Update `--json` output to include debrief data
- Skill: feature-impl

**Phase 4: Orchestrator skill update**
- Update session end debrief sequence to reference `orch debrief` command
- Update session start to note orient now includes debrief comprehension
- Skill: feature-impl (skill file edit)

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This design becomes the accepted approach
- Future spawns may try to add alternative session persistence mechanisms

**Suggested blocks keywords:**
- "session debrief"
- "debrief artifact"
- "session persistence"
- "orient comprehension"
- "session end"

## References

- `cmd/orch/orient_cmd.go` — current orient implementation
- `pkg/orient/orient.go` — orient data types and formatting
- Orchestrator skill Session End: lines 365-383
- `.kb/investigations/2026-02-27-design-flow-integrated-knowledge-surfacing.md`
- `.kb/guides/orchestrator-session-management.md`
- `.kb/decisions/2026-02-07-orchestrator-reflection-session-protocol.md`
- Probe: `.kb/models/orchestrator-session-lifecycle/probes/2026-02-28-probe-session-debrief-artifact-design.md`
