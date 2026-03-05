## Summary (D.E.K.N.)

**Delta:** The synthesis-as-comprehension plan solved the teaching and format problems (what to synthesize, how to present it) but not the timing problem — comprehension emerges mid-session in dialogue, yet the only capture mechanism (`orch debrief --learned`) runs at session end, forcing lossy reconstruction. A new `orch thread` command + `.kb/threads/` artifact type fills this gap.

**Evidence:** Read current debrief pipeline (`debrief_cmd.go`, `debrief.go`, `quality.go`): `CollectWhatWeLearned()` auto-populates from `agent.completed` event reasons (factual summaries, not insight). Read orient pipeline (`orient_cmd.go`, `debrief.go`): `FormatLastSessionInsight` reads debrief's "What We Learned" section but has no thread-level continuity. Verified knowledge placement table has no home between `kb quick` (ephemeral) and investigations (formal, point-in-time). Confirmed no `.kb/threads/` directory exists.

**Knowledge:** Living threads require three things: (1) a capture command that runs when insight crystallizes (`orch thread "name" "entry"`), (2) a durable artifact that accumulates entries across sessions (`.kb/threads/{slug}.md`), and (3) integration points that surface threads at orient and reference them at debrief. This extends the synthesis-as-comprehension plan's teaching (Thread→Insight→Position cognitive moves) with the infrastructure to actually capture the Thread and Position at the moment they emerge.

**Next:** Implement in 3 phases: (1) Core thread package + CLI commands, (2) Debrief integration (populate "What We Learned" from today's thread entries), (3) Orient integration (surface active threads at session start). Skill text update adding thread capture to the orchestrator's SYNTHESIZE moves.

**Authority:** architectural — New artifact type, new CLI command family, changes to debrief/orient pipelines, skill text update. Cross-component coordination across `pkg/thread`, `pkg/debrief`, `pkg/orient`, and orchestrator skill.

---

# Investigation: Design Living Threads System for Mid-Session Comprehension Capture

**Question:** How should the orchestration system capture comprehension when it crystallizes mid-session (not at debrief time), accumulate it across sessions, and integrate it with the existing debrief/orient lifecycle?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-kgxtt)
**Phase:** Complete
**Next Step:** None — proceed to implementation issues
**Status:** Complete

**Patches-Decision:** N/A (new design, may produce decision)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md` | extends | yes — confirmed synthesis-as-comprehension solves teaching/format but not timing | none |
| `.kb/plans/2026-03-05-synthesis-as-comprehension.md` | extends | yes — all 3 phases implemented; this fills the gap they couldn't | none |
| `~/.kb/models/behavioral-grammars/model.md` | grounds | yes — Claim 3 (infrastructure > instruction) justifies dedicated command over norm | none |
| `.kb/investigations/2026-02-27-design-flow-integrated-knowledge-surfacing.md` | confirms | yes — "capture at engagement moments" aligns with mid-session thread capture | none |
| `.kb/models/orchestrator-session-lifecycle/model.md` | extends | yes — adds thread continuity to session lifecycle model | none |

---

## Findings

### Finding 1: The timing gap — comprehension captured too late

**Evidence:** The current capture pipeline has exactly one entry point for orchestrator insight: `orch debrief --learned "text"`. This runs at session end (Session End Protocol step 1). But reading the 2026-03-05 debrief's "What We Learned" section reveals 28 items — all from `agent.completed` event reasons (`CollectWhatWeLearned()` at `debrief.go:208-235`). These are agent explain-back strings like "Added --dry-run flag to orch spawn" and "Fixed tagBeadsAgent/untagBeadsAgent to use Config.ProjectDir" — factual summaries of what agents did, not Dylan's insight or the orchestrator's synthesis.

The `--learned` flag was designed to fix this, but it requires the orchestrator to reconstruct insight that was expressed hours earlier during conversation. This is the glove/hand mismatch: comprehension is continuous and dialogic, capture is terminal and batch.

**Source:** `cmd/orch/debrief_cmd.go:109-111` (collectDebriefLearned merges --learned + --changed + event reasons), `pkg/debrief/debrief.go:208-235` (CollectWhatWeLearned extracts from agent.completed events), `.kb/sessions/2026-03-05-debrief.md` (28 items, all agent completion reasons)

**Significance:** The synthesis-as-comprehension work (phases 1-3) correctly identified WHAT to capture (comprehension using Thread→Insight→Position) and HOW to format it (restructured debrief template, quality advisory). But it didn't solve WHEN to capture it. Teaching the orchestrator to synthesize at debrief time still requires batch reconstruction. The missing infrastructure is a capture mechanism at the moment of crystallization.

---

### Finding 2: No artifact home for evolving thinking

**Evidence:** The knowledge placement table (global CLAUDE.md) has these homes for knowledge:

| Artifact | Character | Lifecycle |
|----------|-----------|-----------|
| `kb quick` entries | Ephemeral, working memory | Promoted or superseded at touchpoints |
| Investigations | Point-in-time, answer a question | Active → Complete |
| Models | Mature, queryable claims with evidence | Updated via probes |
| Decisions | Accepted recommendations | Accepted → may be superseded |
| Plans | Decision navigation + phased execution | Active → Implemented |
| Guides | Reusable frameworks | Evolved over time |

None of these fits "thinking that's too important to lose but too young to formalize." `kb quick` entries are individual observations without accumulation. Investigations are structured around answering a specific question. Models require confirmed claims with evidence. What's missing is an artifact that:
- Accumulates entries over multiple sessions (not point-in-time)
- Carries open questions and forming connections (not conclusions)
- Has a lifecycle toward promotion (not standalone indefinitely)
- Is lightweight enough to capture in conversation flow (not formal)

**Source:** Global CLAUDE.md knowledge placement table, `.kb/` directory structure (has `decisions/`, `investigations/`, `models/`, `plans/`, `guides/`, `sessions/`, `specifications/` — no `threads/`)

**Significance:** The absence of this artifact type means forming comprehension falls into one of two failure modes: (1) captured as `kb quick` entries which are isolated observations without narrative coherence, or (2) lost entirely because the insight doesn't fit any existing format. Threads fill this gap as multi-session accumulating narrative artifacts.

---

### Finding 3: Orient has the integration surface but not thread data

**Evidence:** `orch orient` (`orient_cmd.go:58-111`) already surfaces:
1. Throughput metrics (events.jsonl)
2. Previous session summary (latest debrief)
3. Last session insight (FormatLastSessionInsight from debrief's "What We Learned")
4. Ready work (bd ready)
5. Active plans (.kb/plans/)
6. Model freshness (.kb/models/)
7. Focus goal

The `OrientationData` struct (`orient.go:35-43`) is extensible — adding an `ActiveThreads []ThreadSummary` field and a `formatActiveThreads()` formatter follows the established pattern exactly. The orient output renders sections sequentially, and "Active threads" would sit naturally between "Last session insight" and "Previous session" — surfacing multi-session thinking threads alongside the last session's comprehension.

**Source:** `pkg/orient/orient.go:35-43` (OrientationData struct), `pkg/orient/orient.go:91-121` (FormatOrientation sections), `cmd/orch/orient_cmd.go:58-111` (runOrient data collection)

**Significance:** Orient is the right integration point because it runs at session start — the moment when thread continuity matters most. The infrastructure is ready to consume thread data; only the data source is missing.

---

### Finding 4: Debrief's "What We Learned" should aggregate, not originate

**Evidence:** Currently, `collectDebriefLearned()` (`debrief_cmd.go:234-251`) merges three sources:
1. `--learned` flag (explicit insight)
2. `--changed` flag (backward compat)
3. `CollectWhatWeLearned()` from agent.completed events

The agent completion reasons dominate because they're auto-populated while `--learned` requires manual input at session end. But if threads captured insight mid-session, debrief could aggregate from a fourth source: thread entries dated today. This makes debrief a finalization step (collect thread entries + events + manual additions) rather than a creation step (reconstruct insight from memory).

The quality heuristic (`quality.go:41-97`) would still apply — detecting summary-shaped output in "What We Learned" regardless of source. Thread entries that are themselves summary-shaped would be flagged. This preserves the advisory gate's value.

**Source:** `cmd/orch/debrief_cmd.go:234-251` (collectDebriefLearned), `pkg/debrief/quality.go:41-97` (CheckQuality)

**Significance:** This is the key behavioral shift: debrief becomes FINALIZATION (reading back what was captured) rather than CREATION (writing insight from scratch). The orchestrator's cognitive load at session end drops because insight was already captured when it crystallized.

---

### Finding 5: Thread lifecycle maps to existing promotion paths

**Evidence:** The knowledge placement table already defines promotion paths:
- `kb quick` entries → model updates, decisions, skills
- Investigation → Decision (when recommendation accepted)
- Investigation → Plan (when multi-phase execution needed)
- Investigation → Guide (when reusable pattern emerges)

Thread promotion follows the same pattern:
- Thread entries that confirm a repeating pattern → Model update (or new model)
- Thread that resolves with a specific choice → Decision record
- Thread that establishes universal constraint → Principle
- Thread that produces reusable procedure → Skill

The resolution mechanism needs to record WHERE the thread was promoted, both for traceability and to stop surfacing resolved threads in orient.

**Source:** Global CLAUDE.md knowledge placement table (promotion paths section), `~/.kb/principles.md` (principles have explicit criteria for promotion)

**Significance:** Threads don't create a new promotion paradigm — they use existing ones. The only new lifecycle state needed is `resolved` with a pointer to where the thread's thinking landed. This keeps the system coherent (Principle: Coherence Over Patches).

---

## Synthesis

**Key Insights:**

1. **The timing problem is distinct from the teaching problem.** Synthesis-as-comprehension taught HOW (Thread→Insight→Position). Living threads solve WHEN (at crystallization, not at debrief). These are complementary, not competing. The cognitive moves taught in the skill become the vocabulary for thread entries.

2. **Threads are the missing artifact type between ephemeral and formal.** `kb quick` entries are isolated observations. Investigations answer specific questions. Threads accumulate forming understanding across sessions — they're narrative, not structured. They fill the gap in the knowledge placement table for "thinking too important to lose but too young to formalize."

3. **Debrief shifts from creation to finalization.** When threads capture insight mid-session, the debrief's "What We Learned" section aggregates existing thread entries instead of requiring from-scratch reconstruction. This eliminates the glove/hand mismatch: the hand (continuous comprehension) shapes the glove (incremental thread entries), not the other way around.

4. **Orient surfaces thread continuity, not just session continuity.** "Last session insight" from the debrief gives one session of context. Active threads give multi-session thinking continuity. Orient already has the infrastructure to consume this — it just needs the data source.

**Answer to Investigation Question:**

The system should capture comprehension via a new `orch thread "name" "entry"` command that creates/appends to `.kb/threads/{slug}.md` files with dated entries. Threads are multi-session accumulating artifacts with `open`/`resolved` lifecycle states. `orch debrief` aggregates today's thread entries into "What We Learned" (finalization, not creation). `orch orient` surfaces active threads alongside last session insight. When a thread crystallizes into a formal artifact (model, decision, principle), `orch thread resolve "name" --to "path"` marks it resolved and stops surfacing. The orchestrator skill's SYNTHESIZE section gets a thread capture instruction: when insight crystallizes in conversation, call `orch thread` immediately rather than waiting for debrief.

---

## Design Specification

### Thread Artifact Format

**Location:** `.kb/threads/{slug}.md`

```markdown
---
title: [The question, tension, or forming insight]
status: open
created: 2026-03-05
updated: 2026-03-05
resolved_to: ""
---

# Thread: [title]

## 2026-03-05

[Entry text — insight, connection, question, forming understanding]

## 2026-03-06

[Subsequent entry — accumulates dated sections]
```

**Frontmatter fields:**
- `title`: Human-readable description of the thinking thread
- `status`: `open` (surfaced at orient, accumulated) or `resolved` (no longer surfaced)
- `created`/`updated`: Date stamps for staleness filtering
- `resolved_to`: Path to the artifact where thinking landed (model, decision, principle, etc.)

### `orch thread` Command Family

**`orch thread "name" "entry text"`** — Create-or-append (primary command)
- If `.kb/threads/{name}.md` doesn't exist, creates with frontmatter + first entry
- If it exists, appends new dated section (or appends to today's section if one exists)
- Updates `updated` in frontmatter
- Prints confirmation: `Thread updated: .kb/threads/{name}.md (3 entries)`

**`orch thread list`** — List threads
- Shows all threads with status, last update date, and latest entry preview
- Flags stale threads (not updated in 7+ days while still open)
- Output: `{name} ({status}, updated {date}) — {first line of latest entry}`

**`orch thread show "name"`** — Display thread content
- Prints full thread content (cat-style)
- Useful for orient or when orchestrator needs to review a thread

**`orch thread resolve "name" --to "path"`** — Mark resolved
- Sets `status: resolved`, `resolved_to: path` in frontmatter
- Prints: `Thread resolved: enforcement-gaps → .kb/models/enforcement-architecture/model.md`
- Resolved threads are excluded from orient and debrief aggregation

### Debrief Integration

**Change to `collectDebriefLearned()` (`debrief_cmd.go`):**

Add a fourth source: scan `.kb/threads/*.md` for entries dated today.

```
Sources (in priority order):
1. --learned flag (explicit orchestrator insight)
2. --changed flag (backward compat)
3. Thread entries from today (mid-session comprehension)
4. Agent completion reasons (factual summaries — demoted)
```

Thread entries are prefixed with thread name for context: `[enforcement-gaps] Decidability graphs and behavioral grammars are two halves of the same problem`.

Agent completion reasons remain but are effectively demoted — when threads exist, they'll be more insightful because they capture the orchestrator's actual synthesis rather than agent self-report.

### Orient Integration

**Add to `OrientationData` struct:**
```go
ActiveThreads []ThreadSummary `json:"active_threads,omitempty"`
```

**`ThreadSummary` type:**
```go
type ThreadSummary struct {
    Name        string `json:"name"`
    Title       string `json:"title"`
    Updated     string `json:"updated"`
    LatestEntry string `json:"latest_entry"`
}
```

**Placement in orient output:** After "Last session insight" and before "Previous session":
```
Active threads:
   enforcement-gaps (updated 2026-03-05): Decidability graphs and behavioral grammars are...
   comprehension-timing (updated 2026-03-04): The glove/hand mismatch suggests capture...
```

**Staleness filter:** Only show open threads updated within last 3 sessions (compare thread `updated` date to 3 most recent debrief dates). Stale threads are omitted from orient output but not auto-resolved.

### Orchestrator Skill Update

**Add to SYNTHESIZE definition (after Thread→Insight→Position):**

```markdown
**Capture threads live:** When insight crystallizes in conversation — a connection, a forming question, a shift in understanding — capture it immediately:
`orch thread "thread-name" "the insight or forming question"`
Don't wait for debrief. Debrief aggregates; threads capture.
```

**Add to knowledge placement table reference (if skill contains one):**

```markdown
| Forming insight | `orch thread "name" "text"` | "This connects to..." / "The real question is..." |
```

### New Package: `pkg/thread/`

**Files:**
- `thread.go` — Core operations: Create, Append, List, Resolve, Show, ParseThread, TodaysEntries
- `thread_test.go` — Unit tests

**Key functions:**
- `CreateOrAppend(threadsDir, name, entry string) error` — Creates file if missing, appends dated entry if exists
- `List(threadsDir string) ([]ThreadSummary, error)` — Lists all threads with metadata
- `Resolve(threadsDir, name, resolvedTo string) error` — Updates frontmatter status + resolved_to
- `TodaysEntries(threadsDir string, date string) ([]ThreadEntry, error)` — Returns entries dated today for debrief
- `ParseThread(content string) (*Thread, error)` — Parses thread file into structured data
- `ActiveThreads(threadsDir string, maxAge int) ([]ThreadSummary, error)` — Filters by staleness

---

## Structured Uncertainty

**What's tested:**

- ✅ Current "What We Learned" is auto-populated from agent.completed events, not orchestrator insight (verified: read `debrief.go:208-235` + `debrief_cmd.go:234-251` + actual 2026-03-05 debrief with 28 agent-reported items)
- ✅ No `.kb/threads/` directory exists (verified: `ls .kb/threads/` returns "no such file or directory")
- ✅ Knowledge placement table has no artifact type for accumulating forming insight (verified: read global CLAUDE.md table, confirmed gap between `kb quick` and investigations)
- ✅ Orient infrastructure is extensible for new data sources (verified: read `OrientationData` struct and `FormatOrientation` — adding a field + formatter follows established pattern)
- ✅ No naming conflict with existing orch commands (verified: grep for `rootCmd.AddCommand` shows 31 commands, none named "thread")

**What's untested:**

- ⚠️ Whether orchestrators will actually call `orch thread` mid-conversation (requires behavioral observation over 3+ sessions)
- ⚠️ Whether thread entries produce better "What We Learned" content than agent completion reasons (requires comparison of debrief quality before/after)
- ⚠️ Whether the create-or-append semantics feel natural for the orchestrator's workflow (requires usage observation)
- ⚠️ Whether staleness filtering (3-session window) is the right threshold for orient (requires calibration)
- ⚠️ Whether threads accumulate without resolution (Class 3: Stale Artifact Accumulation — mitigation is staleness filtering + `orch thread list` showing stale threads)

**What would change this:**

- If orchestrators don't call `orch thread` mid-session despite skill text teaching, the capture mechanism needs to be more automatic — possibly extracting insight from conversation patterns rather than requiring explicit command
- If thread entries are too terse to be useful in "What We Learned", the debrief integration should format them differently (add thread title as context, concatenate related entries)
- If threads proliferate without resolution, add advisory warning to session-end protocol: "You have N threads older than 14 days — consider resolving or updating"

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Thread artifact type + `.kb/threads/` | architectural | New artifact type extending knowledge placement taxonomy |
| `orch thread` command family | implementation | CLI extension within established Cobra patterns |
| Debrief integration (aggregate from threads) | implementation | Extends existing `collectDebriefLearned` within established patterns |
| Orient integration (surface active threads) | implementation | Adds data source to existing orient pipeline within established patterns |
| Orchestrator skill text update | architectural | Modifies orchestrator policy document affecting all orchestrator behavior |
| Knowledge placement table update | architectural | Cross-project taxonomy change in global CLAUDE.md |

### Recommended Approach ⭐

**Living threads with create-or-append semantics** — A new `orch thread` command captures insight at crystallization time, `.kb/threads/{slug}.md` files accumulate entries across sessions, debrief aggregates from threads instead of reconstructing, orient surfaces active threads for continuity.

**Why this approach:**
- Follows Infrastructure Over Instruction (P11): dedicated command provides gravitational pull toward capture, not just a norm
- Follows Capture at Context (P8): insight captured when it crystallizes, not reconstructed later
- Extends (not replaces) synthesis-as-comprehension: Thread→Insight→Position cognitive moves become the vocabulary for thread entries
- Matches existing patterns: command family (like `orch plan`), artifact directory (like `.kb/plans/`), orient integration (like active plans), debrief integration (like completion reasons)

**Trade-offs accepted:**
- Requires orchestrators to explicitly call `orch thread` — accepted because infrastructure pull (command exists, skill teaches it, debrief aggregates from it) creates affordance without forcing automation. If orchestrators don't call it, the failure mode is identical to current state (no worse), and we get data on whether the affordance is sufficient or automation is needed.
- Threads can accumulate without resolution (Defect Class 3: Stale Artifact Accumulation) — accepted because staleness filtering in orient limits noise, and `orch thread list` makes stale threads visible. Advisory warning at session end can be added if needed.
- One more artifact type in the knowledge taxonomy — accepted because the gap it fills (accumulating forming insight) is real and currently causes loss. The promotion paths to existing artifact types are clear.

**Implementation sequence:**

1. **Phase 1: Core thread package + CLI** — `pkg/thread/` with Create/Append/List/Resolve/Show + `cmd/orch/thread_cmd.go`. This is foundational — all other phases depend on thread files existing.
2. **Phase 2: Debrief integration** — Modify `collectDebriefLearned()` to scan threads for today's entries. Makes debrief aggregate from threads immediately.
3. **Phase 3: Orient integration** — Add `ActiveThreads` to `OrientationData`, `FormatActiveThreads` formatter, thread collection in `orient_cmd.go`. Surfaces threads at session start.
4. **Phase 4: Skill text update** — Add thread capture instruction to orchestrator skill's SYNTHESIZE section. Teaches orchestrators to use the command.

### Alternative Approaches Considered

**Option B: Incremental `--learned` on `orch debrief`**
- **Pros:** No new artifact type. Reuses existing debrief infrastructure.
- **Cons:** Conflates session-level and thread-level capture. `--learned` is session-scoped (one debrief per day); threads are multi-session. Doesn't provide accumulation across sessions. Doesn't surface in orient.
- **When to use instead:** If threads prove too heavyweight and a simpler single-session capture is sufficient.

**Option C: Orchestrator writes directly to `.kb/threads/` files**
- **Pros:** No new CLI command needed. Orchestrator uses Read/Write tools.
- **Cons:** Error-prone (orchestrator must know the format), no validation, no frontmatter management, no integration with debrief/orient. File operations don't trigger events for tracking.
- **When to use instead:** Never — CLI commands provide validation, consistency, and integration that raw file writes cannot.

**Option D: Event-based capture (`orch emit comprehension.captured`)**
- **Pros:** Fits existing event model. Events can be aggregated by debrief.
- **Cons:** Events are ephemeral (events.jsonl grows forever, no cross-session continuity). No accumulating artifact. No orient surfacing. Adds complexity without solving the multi-session continuity problem.
- **When to use instead:** If thread capture events would be useful for analytics, emit events as a side-effect of `orch thread` (not as the primary mechanism).

**Rationale for recommendation:** Option A (living threads) is the only approach that solves all three aspects of the timing problem: capture at crystallization, accumulate across sessions, and surface for continuity. The alternatives solve at most one aspect each.

---

### Implementation Details

**Specific file targets:**

| File | Change | Lines (est.) |
|------|--------|-------------|
| `pkg/thread/thread.go` | New: thread operations | ~200 |
| `pkg/thread/thread_test.go` | New: unit tests | ~200 |
| `cmd/orch/thread_cmd.go` | New: CLI commands | ~150 |
| `cmd/orch/main.go` | Add: `rootCmd.AddCommand(threadCmd)` | 1 |
| `cmd/orch/debrief_cmd.go` | Modify: `collectDebriefLearned` adds thread source | ~20 |
| `pkg/debrief/debrief.go` | Add: `CollectThreadLearned()` or similar | ~30 |
| `pkg/orient/orient.go` | Add: `ActiveThreads` field + `formatActiveThreads` | ~30 |
| `pkg/orient/threads.go` | New: thread scanning for orient | ~80 |
| `cmd/orch/orient_cmd.go` | Add: thread collection step | ~10 |
| `skills/src/meta/orchestrator/.skillc/SKILL.md.template` | Modify: add thread capture to SYNTHESIZE | ~5 |

**Total estimated new code:** ~730 lines (new packages ~550, modifications ~180)

**Things to watch out for:**
- ⚠️ Thread entries appended to today's date section must handle the case where the file already has today's date heading — append to existing section, don't create duplicate heading
- ⚠️ Frontmatter parsing must be robust to handle YAML edge cases (colons in title, empty resolved_to)
- ⚠️ Staleness filtering in orient requires knowing the last 3 session dates — parse from `.kb/sessions/*-debrief.md` filenames
- ⚠️ The skill text addition must net-add minimally — orchestrator skill is at constraint density limits per behavioral grammars model. The thread capture instruction replaces nothing; it adds ~3 lines. Monitor skillc test scores.
- ⚠️ Defect Class 3 exposure: threads that are never resolved accumulate indefinitely. Mitigation: staleness filter in orient + `orch thread list` showing age. Future: advisory warning at session end.

**Success criteria:**
- ✅ `orch thread "enforcement-gaps" "insight text"` creates/appends to `.kb/threads/enforcement-gaps.md`
- ✅ `orch thread list` shows all threads with status, last update, and preview
- ✅ `orch thread resolve "enforcement-gaps" --to ".kb/models/..."` marks resolved and stops surfacing
- ✅ `orch debrief` "What We Learned" includes thread entries from today (prefixed with thread name)
- ✅ `orch orient` shows "Active threads" section with open, non-stale threads
- ✅ skillc test scores for orchestrator skill don't degrade after thread capture instruction added
- ✅ Thread entries appear in at least 1 of first 3 sessions after deployment (behavioral adoption)

---

## References

**Files Examined:**
- `cmd/orch/debrief_cmd.go` — Debrief command implementation, `--learned` flag, collectDebriefLearned pipeline
- `pkg/debrief/debrief.go` — DebriefData struct, RenderDebrief, CollectWhatWeLearned (agent event reasons)
- `pkg/debrief/quality.go` — Quality heuristic (summary detection), ComprehensionPrompt
- `cmd/orch/orient_cmd.go` — Orient command, data collection pipeline (7 sources)
- `pkg/orient/orient.go` — OrientationData struct, FormatOrientation (section rendering)
- `pkg/orient/debrief.go` — FormatLastSessionInsight, ParseDebriefSummary, FindLatestDebrief
- `cmd/orch/main.go` — Command registration (31 commands, no thread conflict)
- `skills/src/meta/orchestrator/.skillc/SKILL.md.template` — SYNTHESIZE definition, Session-Level Synthesis
- `.kb/sessions/2026-03-05-debrief.md` — Current debrief showing 28 agent-reported "What We Learned" items
- `.kb/sessions/TEMPLATE.md` — Debrief template (still old format with "What Changed")
- `.kb/plans/2026-03-05-synthesis-as-comprehension.md` — Predecessor plan (all 3 phases implemented)
- `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md` — Synthesis-as-comprehension investigation
- Global CLAUDE.md — Knowledge placement table, promotion paths
- `~/.kb/principles.md` — Foundational principles (Infrastructure Over Instruction, Capture at Context, Session Amnesia)
- `~/.kb/models/behavioral-grammars/model.md` — Claims 3, 5, 7

**Related Artifacts:**
- **Plan:** `.kb/plans/2026-03-05-synthesis-as-comprehension.md` — The plan this extends (teaching + format)
- **Investigation:** `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md` — Foundation for comprehension infrastructure
- **Model:** `~/.kb/models/behavioral-grammars/model.md` — Theoretical grounding (infrastructure > instruction)
- **Model:** `.kb/models/orchestrator-session-lifecycle/model.md` — Session lifecycle this extends with thread continuity
- **Investigation:** `.kb/investigations/2026-02-27-design-flow-integrated-knowledge-surfacing.md` — "Capture at engagement moments" principle

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: How should the system capture comprehension mid-session rather than requiring batch reconstruction at debrief?
- Context: Dylan identified glove/hand mismatch — comprehension is continuous/dialogic, debrief is terminal/batch. Synthesis-as-comprehension (phases 1-3) solved teaching and format but not timing.

**2026-03-05:** Evidence gathering complete
- Read debrief pipeline (debrief_cmd.go, debrief.go, quality.go): confirmed auto-population from agent events, not orchestrator insight
- Read orient pipeline (orient_cmd.go, orient.go, debrief.go): confirmed extensible infrastructure, no thread data source
- Verified knowledge placement gap: no artifact between kb quick (ephemeral) and investigations (formal)
- Confirmed no naming conflict (31 commands, no "thread")

**2026-03-05:** Design complete with 6 forks navigated
- Fork 1: Capture mechanism → `orch thread` command (infrastructure pull via dedicated command)
- Fork 2: Artifact format → Structured frontmatter + dated entries (human-readable + machine-parseable)
- Fork 3: Lifecycle states → `open`/`resolved` (minimal states that change behavior)
- Fork 4: Debrief integration → Aggregate from today's thread entries (finalization, not creation)
- Fork 5: Orient integration → "Active threads" section with staleness filtering
- Fork 6: Create vs append UX → Create-on-first-use semantics (`orch thread "name" "entry"`)

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: Living threads system extends synthesis-as-comprehension with mid-session capture, multi-session accumulation, and debrief/orient integration
