## Summary (D.E.K.N.)

**Delta:** Mid-session comprehension capture via `.kb/threads/` artifact type and `orch thread` command. Threads are the place where forming insight lives — too important to lose, too young to formalize. Debrief becomes finalization (references threads), not creation. Orient surfaces active threads alongside last session insight.

**Evidence:** Current debrief is terminal batch (fights the hand). "What We Learned" auto-populates agent explain-back strings, not Dylan's actual insight. Knowledge placement table has no home for forming threads. Synthesis-as-comprehension (all 3 phases delivered) teaches comprehension at session boundaries but can't capture insight as it crystallizes mid-conversation.

**Knowledge:** Threads fill the gap between ephemeral conversation and formalized knowledge. They carry open questions and accumulate dated entries across sessions. The lifecycle (forming → active → resolved) provides natural promotion paths to models, decisions, and principles. The `orch thread` command is the mid-session capture mechanism — lightweight enough to use during conversation, durable enough to survive session boundaries.

**Next:** 5 implementation issues recommended (see below). Phase 1 (artifact type + CLI) is self-contained and testable. Phase 2 (debrief integration) and Phase 3 (orient integration) depend on Phase 1. Phase 4 (knowledge placement table update) is documentation-only. Phase 5 (orchestrator skill update) teaches thread capture.

**Authority:** architectural — New artifact type, new CLI command, cross-cutting integration with debrief/orient/skill.

---

# Investigation: Design Living Threads System for Mid-Session Comprehension Capture

**Question:** How should the orchestrator capture comprehension as it crystallizes mid-session, rather than reconstructing it at session close?

**Started:** 2026-03-05
**Updated:** 2026-03-17
**Owner:** architect (orch-go-kgxtt)
**Phase:** Complete
**Status:** Complete
**Disposition:** Implemented — all 5 phases delivered. `pkg/thread/`, `cmd/orch/thread_cmd.go`, `.kb/threads/` (27 threads in active use as of 2026-03-17).

**Extends:** `.kb/plans/2026-03-05-synthesis-as-comprehension.md` (Phases 1-3 teach comprehension at session boundaries; this addresses mid-session capture)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/plans/2026-03-05-synthesis-as-comprehension.md` | extends | yes — all 3 phases delivered; this addresses the gap they don't cover (mid-session) | none |
| `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md` | extends | yes — Finding 3 confirms debrief is an event formatter; threads provide the content source | none |
| `~/.kb/models/behavioral-grammars/model.md` | grounds | yes — Claim 3: infrastructure > instruction; `orch thread` provides infrastructure pull toward mid-session capture | none |

---

## Findings

### Finding 1: The debrief "What We Learned" is auto-populated with the wrong content

**Evidence:** `collectDebriefLearned()` in `debrief_cmd.go:234-251` merges three sources:
1. `--learned` flag (explicit, but requires re-articulation at session close)
2. `--changed` flag (backward compat)
3. `CollectWhatWeLearned()` — agent completion reasons from events.jsonl

The third source dominates: it produces lines like "Fixed detectSkillSourceRoot() in skillc to exclude .claude/worktrees/ paths" — factual describe-back from agents, not Dylan's comprehension. The 2026-03-05 debrief has 36 "What We Learned" items, all agent completion reasons.

**Source:** `cmd/orch/debrief_cmd.go:234-251`, `.kb/sessions/2026-03-05-debrief.md:10-57`

**Significance:** The synthesis-as-comprehension work correctly restructured the template and taught Thread→Insight→Position. But the auto-population pipeline floods "What We Learned" with agent factual output, drowning any insight the orchestrator might add via `--learned`. The section needs a different content source — threads provide it.

---

### Finding 2: Comprehension crystallizes mid-conversation, not at session boundaries

**Evidence:** From the task description — "Comprehension is dialogic and continuous. It emerges throughout conversation, not at session boundaries." Dylan's metaphor: the orchestrator skill is a glove, Dylan's behavior is the hand. The debrief glove is shaped for terminal batch processing — it fights the hand.

The `--learned` flag on `orch debrief` asks Dylan to re-articulate what he already said during conversation. This is friction: the insight was already expressed, and reconstructing it at session close loses fidelity.

**Source:** Task description (Dylan's conversation), behavioral grammars Claim 3 (situational pull)

**Significance:** The capture mechanism must work during conversation, not after it. A command like `orch thread append "X"` during the session is lower-friction than `orch debrief --learned "X"` at session close, because the insight is fresh and contextual.

---

### Finding 3: The knowledge placement table has no home for forming threads

**Evidence:** The current placement table (global CLAUDE.md) has homes for:
- Reusable procedure → Skill
- Quick decision → `kb quick decide`
- Significant decision → `.kb/decisions/`
- Rule/constraint → `kb quick constrain`
- Failed approach → `kb quick tried`
- Exploration/analysis → `.kb/investigations/`
- Reusable framework → `.kb/guides/`
- Session context → MEMORY.md

None of these fit "thinking that's too important to lose but too young to formalize." Investigations are point-in-time explorations. Models are mature. Quick entries are atomic facts. Threads are evolving multi-session thinking with open questions.

**Source:** Global CLAUDE.md "Knowledge Placement" table

**Significance:** Without a named home, forming threads either get lost (conversation ephemera) or get forced into inappropriate containers (investigations that aren't investigations, quick entries that aren't quick). The `.kb/threads/` artifact type fills this gap.

---

### Finding 4: Thread file format needs to support accumulation across sessions

**Evidence:** The key property of a thread (vs an investigation or decision) is that it accumulates entries over time. An investigation has a question and findings — it's opened, worked, and closed. A thread may span many sessions, with each session adding an entry as understanding evolves.

Existing artifact types for comparison:
- Investigation: question → findings → synthesis (point-in-time, single owner)
- Decision: context → options → chosen (resolved, rarely updated)
- Quick entry: single fact (atomic, no history)
- Thread (proposed): question/tension → dated entries → eventual promotion

**Source:** Structural analysis of existing artifact types

**Significance:** The file format must support dated entries (showing evolution) and frontmatter for lifecycle state. YAML frontmatter + markdown body with dated sections is the natural format, consistent with other .kb/ artifacts.

---

### Finding 5: Relationship to synthesis-as-comprehension plan

**Evidence:** The synthesis-as-comprehension plan has 3 phases, all implemented:
- Phase 1: Taught T→I→P in skill + restructured template
- Phase 2: Enhanced `orch debrief` with `--learned`, `--quality`, comprehension prompt
- Phase 3: `orch orient` surfaces "Last session insight" from prior debrief

This work addresses comprehension **at session boundaries**. Living threads addresses comprehension **during sessions**. They're complementary:
- Threads capture insight as it crystallizes mid-conversation
- Debrief references threads as evidence of what was learned
- Orient surfaces active threads at session start

**Source:** `.kb/plans/2026-03-05-synthesis-as-comprehension.md`

**Significance:** Living threads is Phase 4 of the comprehension infrastructure, not a replacement. It extends the synthesis-as-comprehension work by providing the content source that the debrief was missing.

---

## Synthesis

**The core insight:** The debrief should finalize comprehension, not create it. Currently `orch debrief` tries to produce comprehension at session close from auto-populated events and an optional `--learned` flag. Living threads invert this: comprehension is captured incrementally during the session, and the debrief references the active threads.

**The flow:**
1. Mid-session: Dylan has an insight → orchestrator runs `orch thread append "thread-name" "entry text"` (or creates new thread with `orch thread new "Thread Title"`)
2. Session close: `orch debrief` scans `.kb/threads/` for recently-updated threads and references them in "What We Learned"
3. Next session: `orch orient` surfaces active threads alongside last session insight
4. Eventually: thread matures → promoted to model, decision, or principle

---

## Design

### 1. Thread Artifact Type

**Location:** `.kb/threads/`

**File format:**
```markdown
---
title: "How enforcement and comprehension relate"
status: active  # forming | active | resolved
started: 2026-03-05
updated: 2026-03-05
promotes-to: ""  # when resolved: "model:behavioral-grammars" or "decision:2026-03-10-..." or ""
---

# How enforcement and comprehension relate

The question: Does teaching comprehension (T→I→P) actually require enforcement infrastructure, or is it a different kind of behavioral constraint?

## 2026-03-05

First session exploring this. The synthesis-as-comprehension investigation revealed that SYNTHESIZE had no infrastructure pull compared to TRIAGE. But the fix was partly teaching (skill text) and partly infrastructure (template restructure, advisory gate). The teaching component worked (+3 in skillc test). Does this mean some behaviors CAN be taught without infrastructure?

Open question: Is the difference between triage and synthesis that triage is a *procedure* (steps to follow) while synthesis is a *cognitive move* (way of thinking)? Procedures need infrastructure; cognitive moves need examples and practice.

## 2026-03-06

After seeing the first debrief with the new template, the "What We Learned" section is still mostly agent completion reasons. The template restructure changed the section name but not the content source. This thread pushed toward living threads as the content source fix.

The distinction is clearer now: procedures (triage routing, label taxonomy) need infrastructure. Cognitive moves (T→I→P synthesis) need capture affordances — a place to put the thinking as it happens.
```

**Key properties:**
- YAML frontmatter with `title`, `status`, `started`, `updated`, `promotes-to`
- Status values: `forming` (just started, may not survive), `active` (being actively developed across sessions), `resolved` (insight has landed, ready for promotion)
- Markdown body with dated H2 sections (one per session that touches the thread)
- Each entry is free-form narrative — not structured like investigation findings
- `promotes-to` is filled when resolved (optional — some threads just resolve without promotion)

**Naming convention:** `YYYY-MM-DD-{slug}.md` using the start date, like investigations.

### 2. Mid-Session Capture: `orch thread` Command

**New CLI command group:**

```bash
# Create a new thread
orch thread new "How enforcement and comprehension relate"
# → Creates .kb/threads/2026-03-05-enforcement-comprehension.md
# → Status: forming

# Append an entry to an existing thread (adds dated section or appends to today's section)
orch thread append enforcement-comprehension "The distinction is clearer now..."
# → Appends to .kb/threads/2026-03-05-enforcement-comprehension.md
# → Updates frontmatter `updated` field

# List active threads
orch thread list
# → Shows title, status, last updated, age

# Promote a thread (mark as resolved, optionally link to target artifact)
orch thread resolve enforcement-comprehension --promotes-to "model:behavioral-grammars"
# → Sets status: resolved, fills promotes-to

# Show a thread
orch thread show enforcement-comprehension
# → Displays the thread content
```

**Design decisions:**

| Decision | Choice | Rationale |
|----------|--------|-----------|
| New command vs extending debrief | New `orch thread` command | Threads are mid-session; debrief is terminal. Different temporal scope requires different affordance. |
| Thread identification | Slug from title (auto-generated) | Full paths are friction; slugs match other kb conventions (investigations, decisions) |
| Append behavior | Adds to today's dated section, or creates new dated section if none exists for today | Supports multiple appends per session without cluttering the file |
| Thread creation timing | Explicit via `orch thread new` | Don't auto-create — the act of naming a thread is part of crystallizing the insight |

**Implementation scope:** ~200-300 lines in `pkg/thread/thread.go` + ~100 lines in `cmd/orch/thread_cmd.go`.

### 3. Debrief Integration

**Change:** `orch debrief` scans `.kb/threads/` for threads updated today and references them in "What We Learned".

**Current flow:**
```
collectDebriefLearned() → --learned flag + --changed flag + agent completion reasons
```

**Proposed flow:**
```
collectDebriefLearned() → --learned flag + active threads updated today + agent completion reasons (demoted to "What Happened")
```

**Key changes:**
1. Move `CollectWhatWeLearned()` (agent completion reasons) from "What We Learned" to "What Happened" — these are event descriptions, not insights
2. Add `collectActiveThreads()` that scans `.kb/threads/` for threads with `updated: today` and includes their latest entry
3. `--learned` flag remains for inline insight that doesn't warrant a thread

**Impact:** The "What We Learned" section shifts from agent describe-back (36 items of factual output) to actual comprehension threads (1-3 items of interpretive insight).

**Code touches:**
- `cmd/orch/debrief_cmd.go`: `collectDebriefLearned()` adds thread scanning, moves `CollectWhatWeLearned()` to `WhatHappened`
- `pkg/debrief/debrief.go`: Add `CollectActiveThreads()` function

### 4. Orient Integration

**Change:** `orch orient` surfaces active threads in a new "Active threads" section.

**Current orient output:**
```
== SESSION ORIENTATION ==

Last 24h: ...
Last session insight: ...
Previous session: ...
Ready to work: ...
Active plans: ...
```

**Proposed orient output:**
```
== SESSION ORIENTATION ==

Last 24h: ...
Last session insight: ...
Active threads:
   - How enforcement and comprehension relate (3 sessions, last: 2026-03-06)
   - Whether daemon capacity should be event-sourced (1 session, forming)
Previous session: ...
Ready to work: ...
Active plans: ...
```

**Code touches:**
- `pkg/orient/orient.go`: Add `ActiveThreads []ThreadSummary` to `OrientationData`, add `formatActiveThreads()`, add to `FormatOrientation()`
- `pkg/orient/threads.go` (new): `ScanActiveThreads()` reads `.kb/threads/`, parses frontmatter, returns non-resolved threads sorted by recency
- `cmd/orch/orient_cmd.go`: Call `ScanActiveThreads()` and populate `data.ActiveThreads`

### 5. Thread Lifecycle

```
forming → active → resolved → [promoted to model/decision/principle/guide]
                              → [dissolved — insight was wrong or unimportant]
```

**State transitions:**
- `forming → active`: Manual via `orch thread activate <slug>` or automatic after 2+ sessions with entries
- `active → resolved`: Manual via `orch thread resolve <slug>` with optional `--promotes-to`
- `resolved → promoted`: The thread file stays as historical record; the target artifact is created/updated separately

**Lifecycle rules:**
- `forming` threads that haven't been touched in 14 days → advisory warning in `orch orient` ("stale forming thread")
- `active` threads with no entry in 30 days → advisory warning
- `resolved` threads are archived (still readable, not surfaced in orient)

### 6. Knowledge Placement Table Update

Add row to the global CLAUDE.md knowledge placement table:

| You have... | Put it in... | Trigger |
|-------------|--------------|---------|
| Forming thread | `.kb/threads/` | "This is too important to lose but too young to formalize" |

Add promotion path:
- `.kb/threads/` → model (when thread reaches mature understanding)
- `.kb/threads/` → decision (when thread resolves into a choice)
- `.kb/threads/` → principle (when thread reveals a universal constraint)

### 7. Orchestrator Skill Update

Add to the orchestrator skill's Completion Lifecycle section, after Session-Level Synthesis:

```markdown
### Thread Capture (Mid-Session)

When insight crystallizes during conversation — not at session close:

- `orch thread new "Thread title"` — name the forming thought
- `orch thread append slug "entry"` — capture the insight while it's fresh
- `orch thread list` — see active threads

Threads are for thinking that's too important to lose but too young to formalize.
Don't wait for the debrief. Capture it now.
```

This is ~6 lines, within the +9 line constraint density budget identified in the synthesis-as-comprehension investigation.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| `.kb/threads/` artifact type | architectural | New artifact category in the knowledge system |
| `orch thread` CLI commands | implementation | New CLI command group within established patterns |
| Debrief integration (content source shift) | architectural | Changes what "What We Learned" means |
| Orient integration (active threads section) | implementation | Adds data source to existing command |
| Knowledge placement table update | documentation | Extends existing table with new row |
| Orchestrator skill thread capture guidance | architectural | Modifies orchestrator policy document |

### Recommended Approach

**Phased implementation with Phase 1 self-contained:**

### Phase 1: Thread Artifact + CLI (self-contained)

**Deliverables:**
- `pkg/thread/thread.go` — Thread type, CRUD operations, frontmatter parsing
- `cmd/orch/thread_cmd.go` — `orch thread new|append|list|show|resolve` commands
- `.kb/threads/` directory convention
- Tests for all thread operations

**Exit criteria:** `orch thread new "test"` creates file, `orch thread append` adds entry, `orch thread list` shows active threads.

### Phase 2: Debrief Integration

**Deliverables:**
- Move `CollectWhatWeLearned()` output from "What We Learned" to "What Happened" in debrief
- Add thread scanning to `collectDebriefLearned()` — threads updated today appear in "What We Learned"
- Update tests

**Depends on:** Phase 1

**Exit criteria:** `orch debrief` shows active threads in "What We Learned" instead of agent completion reasons.

### Phase 3: Orient Integration

**Deliverables:**
- `pkg/orient/threads.go` — `ScanActiveThreads()`, `ThreadSummary` type
- Add "Active threads" section to orient output
- Update tests

**Depends on:** Phase 1

**Exit criteria:** `orch orient` shows active threads between "Last session insight" and "Previous session".

### Phase 4: Knowledge Placement + Skill Update

**Deliverables:**
- Add thread row to global CLAUDE.md knowledge placement table
- Add Thread Capture section to orchestrator skill
- Add promotion paths to table

**Depends on:** Phase 1

**Exit criteria:** Knowledge placement table includes threads; orchestrator skill teaches thread capture.

### Phase 5: Lifecycle Automation (Optional, can defer)

**Deliverables:**
- Advisory warnings for stale forming threads (14 days) and stale active threads (30 days)
- Auto-promote forming → active after 2+ sessions
- Integration with `kb reflect` for thread review suggestions

**Depends on:** Phase 1

**Exit criteria:** `orch orient` warns about stale threads.

---

## Structured Uncertainty

**What's tested:**
- ✅ Debrief "What We Learned" is populated with agent completion reasons, not insight (verified: read debrief_cmd.go and 2026-03-05-debrief.md)
- ✅ No existing artifact type fits forming threads (verified: reviewed knowledge placement table)
- ✅ Synthesis-as-comprehension is complementary, not conflicting (verified: read plan, all 3 phases implemented)
- ✅ Thread file format is consistent with existing .kb/ conventions (verified: compared to investigations, decisions)

**What's untested:**
- ⚠️ Whether Dylan will actually use `orch thread` mid-session (requires observation)
- ⚠️ Whether the friction of `orch thread append` is low enough to capture insight in the moment
- ⚠️ Whether threads accumulate useful content or become another ignored artifact type
- ⚠️ Whether moving agent completion reasons out of "What We Learned" loses valuable information

**What would change this plan:**
- If `orch thread` feels too heavy mid-session → consider keyboard shortcut or implicit capture from conversation patterns
- If threads accumulate but never get resolved/promoted → the lifecycle needs automation or the artifact type needs rethinking
- If agent completion reasons are actually valuable in "What We Learned" → keep them there and add threads alongside, not instead

---

## References

**Files Examined:**
- `cmd/orch/debrief_cmd.go` — Debrief command, collectDebriefLearned pipeline
- `pkg/debrief/debrief.go` — DebriefData struct, RenderDebrief, CollectWhatWeLearned
- `pkg/debrief/quality.go` — Quality heuristics (comprehension detection)
- `cmd/orch/orient_cmd.go` — Orient command, data collection pipeline
- `pkg/orient/orient.go` — OrientationData struct, FormatOrientation
- `pkg/orient/debrief.go` — DebriefSummary, FormatLastSessionInsight, FormatPreviousSession
- `.kb/sessions/TEMPLATE.md` — Current debrief template (still says "What Changed" — stale vs code)
- `.kb/sessions/2026-03-05-debrief.md` — Current debrief showing agent completion reasons as "What We Learned"
- `.kb/plans/2026-03-05-synthesis-as-comprehension.md` — Implemented comprehension plan
- `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md` — Original investigation

**Related Artifacts:**
- **Plan:** `.kb/plans/2026-03-05-synthesis-as-comprehension.md` — this extends Phase 4
- **Model:** `~/.kb/models/behavioral-grammars/model.md` — infrastructure > instruction
- **Model:** `.kb/models/orchestrator-session-lifecycle/model.md` — session lifecycle patterns

---

## Investigation History

**2026-03-05:** Investigation started
- Question: How should the orchestrator capture comprehension mid-session?
- Context: Dylan's glove/hand metaphor — debrief fights the natural shape of comprehension

**2026-03-05:** Design complete
- 5 findings, 5-phase implementation plan
- Key insight: debrief should finalize comprehension (reference threads), not create it
- Thread artifact fills the knowledge placement gap for "forming threads"
