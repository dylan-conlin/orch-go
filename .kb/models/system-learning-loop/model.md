# Model: System Learning Loop

**Domain:** Observability / Learning System / Gap Management
**Last Updated:** 2026-03-06
**Synthesized From:** 6 investigations (Dec 2025 - Dec 2026) into learning loop implementation, gap tracking, command generation, and validation

---

## Summary (30 seconds)

The System Learning Loop is the third layer of the Pressure Visibility System that automatically converts recurring context gaps into actionable improvements. It tracks gaps during spawns, identifies patterns using RecurrenceThreshold=3, and suggests specific actions (kn entries, beads issues, investigations). The system uses shell-aware command parsing to generate runnable commands with proper quoted string handling, and ensures minimum length requirements for downstream tools (kn requires 20+ chars). This creates a closed feedback loop: gaps → patterns → suggestions → improvements → fewer gaps.

---

## Core Mechanism

### The Learning Loop Flow

```
Agent spawn encounters gap (kb context returns nothing)
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. GAP RECORDING                                               │
│     recordGapForLearning() captures:                            │
│     - Query (what was searched)                                 │
│     - GapType (no_context, no_constraints, no_decisions)        │
│     - Skill (which skill encountered gap)                       │
│     - Task (what was being done)                                │
│     - Timestamp, Severity, ContextQuality                       │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. PERSISTENT STORAGE                                          │
│     GapTracker appends to ~/.orch/gap-tracker.json              │
│     - 30-day retention window                                   │
│     - 1000-event cap                                            │
│     - Survives sessions                                         │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. PATTERN DETECTION (RecurrenceThreshold = 3)                 │
│     FindRecurringGaps() groups by normalized query:             │
│     - 1 occurrence = noise                                      │
│     - 2 occurrences = coincidence                               │
│     - 3+ occurrences = pattern worth addressing                 │
│     - Excludes resolved events                                  │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. SUGGESTION GENERATION                                       │
│     determineSuggestion() maps gap type → action:               │
│     - no_context → kn decide (add foundational knowledge)       │
│     - no_constraints → kn constrain (add rules)                 │
│     - no_decisions → bd create (track pattern via issue)        │
│     - default → orch spawn investigation (explore)              │
│     Generates runnable shell commands with quoted args          │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. ACTION & RESOLUTION                                         │
│     orch learn act N → executes suggested command               │
│     orch learn resolve → marks pattern as addressed             │
│     RecordResolution() marks ALL matching events (not just one) │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

**GapEvent structure:**
```go
type GapEvent struct {
    Timestamp      time.Time
    Query          string           // What was searched
    GapType        GapType          // no_context, no_constraints, etc.
    Severity       GapSeverity      // low, medium, high
    Skill          string           // Which skill encountered gap
    Task           string           // Task description
    ContextQuality ContextQuality   // none, partial, sufficient
    Resolution     string           // How gap was resolved
    ResolutionDetails string        // Additional resolution context
}
```

**Command generation:**
- `generateReasonFromGaps()` extracts skills, tasks, occurrence counts from gap events
- Produces contextual reasons like: `"Used by: investigation, feature-impl. Occurred 5 times. Tasks: analyze auth flow; add db migration"`
- Ensures 20+ character minimum (kn requirement) by prepending `"Recurring gap for topic: {query}"` when needed
- Shell-aware quoting for complex strings containing colons, commas, periods

**Generated command patterns:**
1. `kn decide "query" --reason "reason"` - for no_context gaps
2. `kn constrain "query" --reason "reason"` - for no_constraints gaps
3. `bd create "title" -d "description"` - for no_decisions gaps
4. `orch spawn investigation "task"` - for default/sparse gaps

**orch learn command suite:**
```bash
orch learn                # Show suggestions (recurring patterns)
orch learn patterns       # Analyze by topic
orch learn skills         # Gap rates by skill
orch learn effects        # Improvement effectiveness
orch learn act N          # Execute suggestion N
orch learn resolve        # Mark gap as resolved manually
```

### State Transitions

**Normal learning cycle:**

```
Gap encountered during spawn
    ↓
Recorded to ~/.orch/gap-tracker.json
    ↓
3rd occurrence triggers pattern detection
    ↓
Suggestion generated (with runnable command)
    ↓
User runs: orch learn act N
    ↓
Command executed (kn decide, bd create, etc.)
    ↓
ALL matching events marked resolved
    ↓
Gap no longer appears in future suggestions
```

**Resolution tracking:**

```
User takes action outside orch learn (e.g., manual kn entry)
    ↓
User runs: orch learn resolve
    ↓
Selects resolution type:
  - added_knowledge (created kn entry)
  - created_issue (tracked in beads)
  - investigated (explored and documented)
  - wont_fix (intentional gap)
  - custom (other resolution)
    ↓
ALL matching events marked resolved
    ↓
Gap excluded from future FindRecurringGaps calls
```

### Critical Invariants

1. **RecurrenceThreshold = 3** - Pattern detection balances noise (1) vs signal (3+)
2. **All matching events must be marked resolved** - Not just the most recent one
3. **FindRecurringGaps excludes resolved events** - Prevents resolved gaps from reappearing
4. **Shell-aware command parsing required** - Quoted strings with spaces must be preserved
5. **Minimum 20-character reasons** - kn decide/constrain requirement enforced at generation time
6. **30-day retention window** - Gap events older than 30 days are pruned
7. **Gap recording happens after gating** - Captures all gaps whether spawn proceeds or not

---

## Why This Fails

### Failure Mode 1: Resolved Gaps Keep Appearing

**Symptom:** `orch learn` shows same gap after running `orch learn act` to resolve it

**Root cause:** Two bugs combined:
1. `RecordResolution` only marked the most recent event (used `break` after first match)
2. `FindRecurringGaps` counted all events without filtering by Resolution field

**Why it happens:**
- Gap occurs 5 times → user resolves → only 1 event marked
- Next `FindRecurringGaps` call counts 4 unresolved events → still above threshold (3)
- Same pattern keeps appearing despite resolution

**Impact:**
- User frustration ("I already fixed this!")
- Loss of trust in learning system
- Duplicate work

**Fix:**
- `RecordResolution` now marks ALL matching events (removed `break`)
- `FindRecurringGaps` filters out resolved events before counting

**Source:** `.kb/investigations/2025-12-25-inv-orch-learn-resolved-gaps-still.md`

---

### Failure Mode 2: Generated Commands Fail Due to Broken Quoting

**Symptom:** `orch learn act N` generates command that fails when executed

**Root cause:** Using `strings.Fields()` to parse commands - splits on whitespace without respecting quotes

**Why it happens:**
- Command: `kn decide "auth" --reason "Used by: investigation. Occurred 5 times"`
- `strings.Fields` splits into: `["kn", "decide", "\"auth\"", "--reason", "\"Used", "by:", "investigation.", ...]`
- Shell receives mangled arguments, command fails

**Impact:**
- Learning loop broken - suggestions can't be executed
- User must manually reconstruct commands
- Reduces value of automated suggestions

**Fix:**
- Added `ParseShellCommand()` with shell-aware quote handling
- Respects double and single quotes as argument delimiters
- Added `ValidateCommand()` to catch malformed commands before execution

**Source:** `.kb/investigations/2025-12-26-inv-orch-learn-act-commands-should.md`

---

### Failure Mode 3: Short Reasons Fail kn Validation

**Symptom:** `orch learn act` generates kn command that fails with "reason too short" error

**Root cause:** `generateReasonFromGaps` produced "Occurred N times" (16 chars) when gap events lacked skill/task metadata

**Why it happens:**
- Gap events with sparse metadata → only occurrence count available
- "Occurred 3 times" = 16 characters
- kn requires 20+ characters for `--reason` flag
- Command fails validation at execution time

**Impact:**
- Generated commands unusable
- User must manually edit reason strings
- Learning loop broken for sparse gaps

**Fix:**
- Added `MinReasonLength = 20` constant
- When base reason < 20 chars, prepend `"Recurring gap for topic: {query}"`
- Adds meaningful context (the query) rather than arbitrary padding
- Added validation in `validateKnCommand` as defense in depth

**Source:** `.kb/investigations/2025-12-26-inv-orch-learn-act-generates-truncated.md`

---

### Failure Mode 4: Placeholder Reasons Made Commands Useless

**Symptom:** Generated commands had `--reason "TODO: document decision"` requiring manual editing

**Root cause:** `determineSuggestion` used hardcoded placeholder strings instead of extracting context from gap events

**Why it happens:**
- Original implementation: `--reason "TODO: document decision"`
- Gap events contain rich context (skill, task, query) but it wasn't extracted
- Commands technically valid but semantically empty

**Impact:**
- Manual editing required after every `orch learn act`
- Defeats purpose of automated suggestions
- Users stop using the feature

**Fix:**
- Added `generateReasonFromGaps()` function
- Extracts skills (unique list), tasks (up to 3, truncated to 40 chars), occurrence count
- Formats as: `"Used by: {skills}. Occurred N times. Tasks: {tasks}"`
- Produces actionable, contextual reasons

**Source:** `.kb/investigations/2025-12-25-inv-fix-orch-learn-act-generate.md`

---

## Evolution History

**Phase 1: Initial Implementation (2025-12-25)**
- Created GapTracker, GapEvent structures
- Implemented RecordGap, FindRecurringGaps
- Added `orch learn` command suite
- Established RecurrenceThreshold=3 pattern
- Source: `2025-12-25-inv-system-learning-loop-convert-gaps.md`

**Phase 2: Reason Generation (2025-12-25)**
- Replaced placeholder TODOs with contextual reasons
- Added `generateReasonFromGaps()` extracting skills, tasks, counts
- Source: `2025-12-25-inv-fix-orch-learn-act-generate.md`

**Phase 3: Resolution Tracking (2025-12-25)**
- Fixed RecordResolution to mark ALL matching events
- Fixed FindRecurringGaps to exclude resolved events
- Prevented resolved gaps from reappearing
- Source: `2025-12-25-inv-orch-learn-resolved-gaps-still.md`

**Phase 4: Shell Parsing (2025-12-26)**
- Added ParseShellCommand for proper quote handling
- Added ValidateCommand for pre-execution checks
- Enabled complex quoted strings in commands
- Source: `2025-12-26-inv-orch-learn-act-commands-should.md`

**Phase 5: Minimum Length Enforcement (2025-12-26)**
- Added MinReasonLength constant (20 chars)
- Modified generateReasonFromGaps to pad short reasons
- Added validation in validateKnCommand
- Source: `2025-12-26-inv-orch-learn-act-generates-truncated.md`

---

## Design Decisions

### RecurrenceThreshold = 3

**Decision:** Use 3 as the minimum occurrence count to trigger suggestions

**Rationale:**
- 1 occurrence = random noise (user might have made a typo)
- 2 occurrences = coincidence (could be two unrelated tasks)
- 3+ occurrences = pattern worth addressing (established recurrence)

**Trade-offs:**
- Too low (1-2): Spam with trivial gaps, overwhelming suggestions
- Too high (5+): Miss real patterns, delayed feedback
- 3 is industry-standard heuristic for pattern detection

**Alternative considered:** Dynamic threshold based on time window
**Why rejected:** Added complexity without clear benefit; 3 works well in practice

### Shell-Aware Parsing Over Simple Split

**Decision:** Implement custom ParseShellCommand instead of using strings.Fields

**Rationale:**
- Generated commands contain quoted arguments with spaces, colons, punctuation
- `strings.Fields` breaks on whitespace, doesn't respect quotes
- Shell-aware parsing required for commands to be runnable

**Trade-offs:**
- More complex implementation (quote state tracking)
- Must handle edge cases (unclosed quotes, nested quotes)
- Worth it: commands must actually work when executed

**Alternative considered:** Escape spaces instead of quoting
**Why rejected:** Less readable, doesn't solve punctuation issues

### Minimum Length Padding with Query Context

**Decision:** When reason < 20 chars, prepend "Recurring gap for topic: {query}"

**Rationale:**
- Adds meaningful context (the query) rather than arbitrary padding
- Query always available in generateReasonFromGaps
- Produces human-readable reasons that explain what triggered the gap

**Trade-offs:**
- Slightly longer reason strings when padding needed
- Format change for short reasons (but improvement in clarity)

**Alternative considered:** Pad with generic text like "Pattern detected: {reason}"
**Why rejected:** Less informative; query provides actual context

### 30-Day Retention Window

**Decision:** Prune gap events older than 30 days

**Rationale:**
- Balances learning from history vs stale data
- Gaps older than 30 days likely no longer relevant (project context shifts)
- Keeps gap-tracker.json manageable size

**Trade-offs:**
- Lose long-term pattern visibility
- Could miss slowly-emerging patterns (once per month)

**Alternative considered:** Infinite retention with resolution-based pruning
**Why rejected:** File growth unbounded; 30 days sufficient for active projects

---

## Related Systems

**Upstream (feeds into learning loop):**
- Gap Detection (Layer 1 of Pressure Visibility) - Detects gaps during spawn
- KB Context System - Determines what gaps exist
- Spawn Flow - Integration point for gap recording

**Downstream (consumes learning loop output):**
- kn (knowledge CLI) - Receives suggested decide/constrain commands
- Beads (issue tracker) - Receives suggested create commands
- Investigation System - Receives suggested spawn investigation commands

**Peer Systems:**
- Failure Surfacing (Layer 2 of Pressure Visibility) - Surfaces test/build failures
- Completion Verification - Gates on proper knowledge externalization

---

## References

**Literature (legibility/human factors — incorporated 2026-03-01):**
- Bainbridge (1983) — "Ironies of Automation" — skill degradation, passive monitoring risk
- Endsley (1995) — SA-1/2/3 framework — the three levels the loop must support
- Parasuraman, Sheridan & Wickens (2000) — filtering information before presenting to operators
- Hollnagel & Woods (2005-2006) — joint cognitive systems, adaptation at boundaries
- Chen et al. (2014/2018) — SAT/DSAT agent transparency model
- ISA-101.01-2015 — High-Performance HMI: grayscale default, color for abnormality
- Scott (1998) via Rao — legibility trap: oversimplification produces false confidence

### Merged Probes

| Probe | Date | Verdict | Key Finding |
|-------|------|---------|-------------|
| `probes/2026-03-01-probe-legibility-literature-review-bainbridge-forward.md` | 2026-03-01 | Confirms + Extends | Core architecture confirmed by 40 years of human factors research. Three gaps identified: SA-3 projection absent, honest legibility risk (Scott's trap), joint cognitive system framing neglected. Pace-layered transparency needed. Bainbridge Irony #3 is live — autonomous daemon degrades Dylan's intervention readiness. |

---

## Observability

**What you can observe:**

```bash
# Current suggestions
orch learn

# Pattern analysis by topic
orch learn patterns

# Gap rates by skill (which skills hit most gaps?)
orch learn skills

# Improvement effectiveness (did actions reduce gaps?)
orch learn effects

# Raw gap data
cat ~/.orch/gap-tracker.json | jq '.Events | length'

# Unresolved gaps
cat ~/.orch/gap-tracker.json | jq '.Events[] | select(.Resolution == "")'

# Gap trends over time
cat ~/.orch/gap-tracker.json | jq '.Events | group_by(.Query) | map({query: .[0].Query, count: length})'
```

**What you cannot observe:**
- Whether users act on suggestions (no acceptance rate tracking)
- Whether improvements actually reduced gaps (only manual checking via `orch learn effects`)
- Gap patterns across multiple projects (tracker is per-project)

---

## Future Directions

**Considered for future implementation:**

1. **Cross-project gap aggregation** - Detect patterns across all Dylan's projects, not just one
2. **Automated action execution** - Daemon could auto-run low-risk suggestions (with approval)
3. **Machine learning for threshold tuning** - Adjust RecurrenceThreshold based on actual false positive rates
4. **Gap prediction** - Suggest knowledge additions before gaps occur (based on task description)
5. **Suggestion acceptance tracking** - Measure which suggestions users actually execute

**Intentionally deferred:**
- Semantic query matching (kb context uses keyword matching; semantic would require LLM-based RAG)
- Natural language suggestion generation (current format is structured/actionable, not prose)

---

## Legibility & Human Factors Assessment (2026-03-01)

A literature review against 40 years of human factors research (Bainbridge 1983 → Chen SAT/DSAT 2018) produced the following alignment assessment:

### What the literature confirms

- **Gaps → patterns → suggestions → improvements maps to a continuous SA-2 mechanism** (Endsley situation awareness level 2: comprehension, not just perception). The 40-year literature strongly supports closed feedback loops for maintaining supervisory awareness in automated systems.
- **RecurrenceThreshold=3 is sound.** Parasuraman et al.'s taxonomy validates information filtering before surfacing to operators — showing every gap (threshold=1) creates alarm fatigue, a well-documented SCADA/HMI failure mode.
- **"Pain as signal" architectural principle is validated.** Bainbridge's core finding — automation that hides problems creates worse outcomes — directly validates injecting friction into agent streams rather than silently logging.

### Extensions identified by the literature

1. **SA-3 gap (projection is absent).** The learning loop provides SA-1 (what gaps exist) and SA-2 (why they recur), but has no SA-3 capability — it cannot project which gaps are likely to emerge before they occur. The "Future Directions" gap prediction item is not optional per the literature; it's the third leg of situation awareness. Without projection, the supervisor can only react, not anticipate.

2. **"Honest legibility" risk.** The model doesn't address Scott's legibility trap (James C. Scott, *Seeing Like a State*). Learning loop metrics (gap rates by skill, improvement effectiveness) could create false confidence if gap recording is inconsistent or metrics are gamed. Legibility tools become dangerous when the simplified view diverges from reality.

3. **Joint cognitive system framing.** The current architecture puts Dylan in a reactive position (gaps accumulate → patterns surface → human acts). The human factors literature frames the interesting unit as the *human + system together*, with the human as an active participant, not just a consumer of automated suggestions. `orch learn resolve` partially addresses this but the overall architecture privileges automated detection.

4. **Pace-layered transparency gap.** All suggestions surface at the same urgency level. ISA-101 / High-Performance HMI strongly recommends tiered presentation: normal gaps quiet, critical/blocking gaps loud, detail on demand. Current `orch learn` output is flat.

### Broader system implications (from the literature review)

- Dashboard should follow High-Performance HMI: quiet for normal progress, color/alerts for anomalies and blocked work.
- Agent spawn visibility should answer all three SA levels: what agents are running (SA-1), why they were spawned and what they depend on (SA-2), what's likely to complete/block next (SA-3).
- **Bainbridge's Irony #3 is live in this system:** when the daemon handles everything autonomously, Dylan's manual intervention skill degrades. The daemon's autonomous spawning amplifies this risk. The design should ensure Dylan periodically engages with raw system behavior, not just orchestrator-mediated summaries.
