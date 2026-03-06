## Summary (D.E.K.N.)

**Delta:** Designed a friction capture protocol for worker-base exit that adds one bd comment per friction item before Phase: Complete, with structured categories (bug/gap/ceremony/tooling/none), consumed by orch complete as an advisory.

**Evidence:** 5 worker friction reports from price-watch surfaced 12 distinct friction items (3 bugs, 4 gaps, 3 ceremony, 2 tooling). Principles analysis: Capture at Context favors pre-Phase:Complete placement; Legibility Over Compliance favors advisory over gate; Gate Over Remind resolved via progressive enforcement (advisory → soft gate → hard gate).

**Knowledge:** Friction and discovered work are distinct categories with permitted overlap. Friction = "the system made my work harder" (subjective experience). Discovered work = "I found a bug/debt" (objective artifact). Some items are both.

**Next:** Implement in 3 phases: (1) worker-base skill update + SYNTHESIS.md template section, (2) orch complete advisory + events.jsonl accumulation, (3) cross-session synthesis view.

**Authority:** architectural - Touches worker-base skill (all agents), orch complete pipeline, events system, and SYNTHESIS.md template. Cross-boundary by nature.

---

# Investigation: Design Friction Capture Protocol for Worker-Base Exit

**Question:** How should friction experienced by worker agents be captured, stored, and consumed so that system-level patterns are detected instead of rationalized away?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — ready for implementation
**Status:** Complete

**Patches-Decision:** N/A (new protocol, no existing decision)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-02-13-inv-probe-inventory-friction-gates-across.md` | extends | Yes — 48 gates, bypass ratios confirmed | None |
| `2026-02-28-design-orientation-frame-gate-friction-biggest.md` | extends | Yes — V0-V3 migration debt confirmed | None |
| `archived/2026-01-16-inv-design-skill-execution-bridge-friction.md` | supersedes | Yes — `orch friction` command was designed and later removed | This design takes a different approach (protocol, not standalone command) |

---

## Findings

### Finding 1: Raw Friction Data Clusters into 4 Actionable Categories

**Evidence:** 5 worker friction reports from price-watch (2026-03-05) surfaced 12 distinct items:

| Category | Items | Examples | Consumption Pattern |
|----------|-------|----------|-------------------|
| **bug** (3) | Beads dir resolution from nested repos, git add hook false positive on .kb/ paths, stop hook Phase:Complete false positive | Fix via issue → feature-impl | Issue creation → fix |
| **gap** (4) | SYNTHESIS.md ephemeral/gitignored, VERIFICATION_SPEC.yaml concurrent overwrite, kb context 20% signal, duplicate formatDuration | Architect review → design decision | Route to architect |
| **ceremony** (3) | Ceremony-to-code ratio (12 lines, 3 min impl, process took longer), investigation template mismatch for design work, bd dep add gate learning curve | Pattern tracking → periodic meta-review | Accumulate, threshold-trigger review |
| **tooling** (2) | bd sync error noise, /exit not a skill | UX issue → fix | Issue creation → fix |

**Source:** Raw friction reports in SPAWN_CONTEXT.md for this task

**Significance:** Categories require different orchestrator responses. Bugs and tooling issues → create issues. Gaps → route to architect. Ceremony → accumulate for periodic meta-review. A flat list loses this routing information.

---

### Finding 2: Principles Constrain the Design Space to a Narrow Band

**Evidence:** Six principles apply, and their tensions define the design:

| Principle | Says | Implication |
|-----------|------|-------------|
| **Capture at Context** | Capture when context is fresh, not reconstructed | Friction step goes BEFORE Phase: Complete |
| **Gate Over Remind** | Gates beat reminders; gates must be passable | "none" must be valid (one tool call) |
| **Legibility Over Compliance** | Ceremony that hides failures is worse than no ceremony | Start as advisory, not hard gate |
| **Friction is Signal** | Recurring friction > novel friction; capture immediately | Must accumulate for pattern detection |
| **Infrastructure Over Instruction** | Enforce via tooling, not documentation | Eventually move to gate; start with advisory + skill update |
| **Redundancy boundary** | ≤4 co-resident norms get attention, 5+ dilutes | Worker-base already has ~4 exit steps; adding friction as step 5 is at the threshold |

**Source:** `~/.kb/principles.md` — Capture at Context (L195), Gate Over Remind (L162), Legibility Over Compliance (L701), Friction is Signal (L578), Infrastructure Over Instruction (L290), Redundancy is Load-Bearing (L666)

**Significance:** The principles converge on: lightweight structured comment, before Phase: Complete, starting as advisory with progressive enforcement. They explicitly reject: heavy forms, post-Phase:Complete placement, hard gates from day one, and free-form-only capture.

---

### Finding 3: Friction and Discovered Work Are Distinct but Overlapping

**Evidence:** The existing discovered work protocol captures bugs/tech debt/enhancements found *in the code being worked on*. Friction captures the agent's *subjective experience of system impedance*.

| | Discovered Work | Friction |
|---|----------------|----------|
| **What** | Bugs, tech debt, enhancements in code | System impedance experienced during work |
| **Example** | "Found race condition in token refresh" | "bd sync output is noisy, can't tell success from failure" |
| **Action** | Create beads issue | Report for pattern detection |
| **Granularity** | Issue-worthy | May be too small/ambient for an issue |
| **Overlap** | Some items are both | git hook false positive is a discovered bug AND friction experienced |

**Source:** `skills/src/shared/worker-base/.skillc/discovered-work.md`, raw friction reports

**Significance:** The two protocols serve different purposes and should remain separate. Friction captures the "sigh, this again" moments too ambient for individual issues but too valuable to lose. Allow overlap — reporting the same item as both friction and discovered work is fine (different purposes: signal vs tracking).

---

### Finding 4: The Prior `orch friction` Command Failed Because It Was Standalone

**Evidence:** `orch friction` was designed in Jan 2026 (`archived/2026-01-16-inv-design-skill-execution-bridge-friction.md`) as a Skill-Execution Bridge with 4 components (Skill Parser, Context Annotator, Failure Analyzer, Friction Reporter). It was removed by Mar 2026 skill audit.

**Source:** `.kb/sessions/2026-03-05-debrief.md` line 17 (removal noted), archived investigation

**Significance:** A standalone friction command requires the agent to proactively invoke it — a reminder, not a gate. The protocol should be embedded in the exit sequence (structural placement) rather than offered as an optional command.

---

### Finding 5: Orch Complete Advisory Pipeline Has a Natural Extension Point

**Evidence:** `runCompletionAdvisories` in `cmd/orch/complete_pipeline.go` already processes 7 advisory types in sequence:
1. SYNTHESIS TLDR (scan tier)
2. Discovered work disposition
3. Probe verdicts
4. Architectural choices
5. Knowledge maintenance
6. Explain-back gate
7. Verification checklist + hotspot/model-impact/synthesis advisories

Each advisory surfaces information for the orchestrator. A friction advisory fits naturally between probe verdicts and architectural choices (position 4, after discovered work since friction may reference same items).

**Source:** `cmd/orch/complete_pipeline.go:199-419`

**Significance:** No new infrastructure needed for orchestrator consumption. Parse friction comments from beads thread, display as advisory section. The pattern already exists.

---

### Finding 6: The Ceremony Trap Is the Primary Risk

**Evidence:** Worker 1 explicitly reported ceremony-to-code ratio as friction: "12 line fix, 3 min implementation, process took significantly longer." Worker-base already has ~4 exit steps (VERIFICATION_SPEC, Phase: Complete, SYNTHESIS.md, commit/.kb check, /exit). Adding friction as step 5 is at the redundancy boundary threshold (≤4 norms = attention, 5+ = dilution begins).

**Source:** Raw friction reports, Redundancy is Load-Bearing principle (boundary condition at L688)

**Significance:** If friction capture itself becomes the #1 reported friction item, we've failed. The meta-signal must be tracked. This is why the design must be: one comment, structured format, "none" is valid. Total cost for frictionless sessions: ~10 seconds (one `bd comment` call).

---

## Synthesis

**Key Insights:**

1. **Placement before Phase: Complete is non-negotiable** — Capture at Context says context decays from "full fidelity" (in the moment) to "reconstructed, rationalized" (end of session). The friction step must fire while the agent still has emotional context about what was annoying, before they switch into completion-mode rationalization.

2. **Progressive enforcement resolves the Gate vs. Legibility tension** — Start as advisory in orch complete (Legibility: orchestrator can see whether friction was reported). If agents skip it >50% of the time with genuine friction, promote to soft gate (warning). If that fails, promote to hard gate at V0. This avoids adding ceremony prematurely while preserving the path to enforcement.

3. **Categories enable differential routing** — Bugs and tooling issues create issues. Gaps route to architect. Ceremony patterns accumulate for threshold-triggered meta-reviews. Without categories, the orchestrator must re-classify every item manually.

4. **Friction capture and discovered work are complementary, not redundant** — Discovered work tracks concrete artifacts (issues). Friction tracks subjective experience (signals). The overlap is intentional — git hook false positive is both a bug to fix and friction to track as a pattern.

**Answer to Investigation Question:**

Friction should be captured via structured beads comments (`Friction: <category>: <description>`) before Phase: Complete, stored as the primary artifact in the beads thread with a supplementary SYNTHESIS.md section for full-tier spawns, consumed by orch complete as an advisory that parses and surfaces friction for orchestrator triage, and accumulated in events.jsonl for cross-session pattern detection. The protocol avoids empty ceremony by accepting "Friction: none" as a valid one-word answer and starting as advisory (not gate) with progressive enforcement.

---

## Structured Uncertainty

**What's tested:**

- ✅ Raw friction data clusters into 4 categories with distinct consumption patterns (verified: 12 items from 5 workers manually classified)
- ✅ Orch complete advisory pipeline has extension point for friction consumption (verified: read `complete_pipeline.go`, identified insertion point)
- ✅ Existing `orch friction` standalone command approach was tried and removed (verified: debrief + archived investigation)
- ✅ Principles converge on lightweight structured comment before Phase: Complete (verified: 6 principles analyzed with specific line references)

**What's untested:**

- ⚠️ Agent compliance rate with friction reporting (not tested — need real deployment data)
- ⚠️ Quality of end-of-session friction recall vs. mid-session capture (hypothesis: end-of-session is "good enough" for most items, but ceremony complaints will be under-reported because agents rationalize them away)
- ⚠️ Cross-session pattern detection effectiveness (not tested — need accumulation volume)
- ⚠️ Whether 5 exit steps causes dilution per Redundancy boundary (hypothesis: yes for trivial tasks, no for substantial tasks)

**What would change this:**

- If agents consistently report "Friction: none" when orchestrator review reveals they clearly experienced friction → promote to gate
- If friction reports are consistently low-quality (single word, no detail) → consider mid-session capture triggers
- If ceremony complaints dominate friction reports → the protocol is too heavy, simplify
- If bugs account for >60% of friction → auto-create path becomes justified

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add friction step to worker-base exit protocol | architectural | Touches all worker skills via dependency |
| Add SYNTHESIS.md friction section | architectural | Template change affects all full-tier spawns |
| Add friction advisory to orch complete | architectural | Pipeline change, cross-boundary |
| Add friction events to events.jsonl | implementation | Extends existing event system, single scope |
| Progressive enforcement (advisory → gate) | strategic | When to promote is a judgment call about system maturity |

### Recommended Approach ⭐

**Structured Friction Comment Protocol** — Add a single bd comment step before Phase: Complete with structured categories, consumed by orch complete as an advisory.

**Why this approach:**
- Minimal ceremony: one comment, "none" is valid, ~10 seconds for frictionless sessions
- Machine-parseable: regex `Friction: (\w+)(?:: (.+))?` enables automated routing
- Follows Capture at Context: fires before Phase: Complete while context is fresh
- Follows Legibility Over Compliance: advisory surfaces info without forcing compliance
- Natural extension of existing advisory pipeline in orch complete

**Trade-offs accepted:**
- End-of-session capture is reconstructed, not observed (Capture at Context tension). Accepted because mid-session capture would add invasive ceremony.
- Advisory (not gate) means some agents will skip it. Accepted because progressive enforcement provides the escalation path.
- Categories are coarse (4 types). Accepted because finer granularity adds ceremony; coarse categories can be refined by orchestrator.

**Implementation sequence:**

#### Phase 1: Protocol Definition (worker-base skill + SYNTHESIS.md template)

**1a. Worker-base exit protocol update** (`skills/src/shared/worker-base/.skillc/completion.md`)

Add new step between VERIFICATION_SPEC.yaml and Phase: Complete:

```
{{if ne .Tier "light"}}
2. **Report friction** experienced during this session:
   ```bash
   # For each friction item (one comment per item):
   bd comments add {{.BeadsID}} "Friction: bug: beads dir resolution fails from nested repos"
   bd comments add {{.BeadsID}} "Friction: ceremony: 12 line fix took 30min due to process overhead"

   # If no friction experienced:
   bd comments add {{.BeadsID}} "Friction: none"
   ```
   Categories: `bug` (broken behavior), `gap` (missing capability), `ceremony` (disproportionate process), `tooling` (tool UX issues), `none`
{{end}}
```

**Light-tier exemption:** Light-tier spawns skip friction reporting. These are trivial tasks where the ceremony-to-value ratio of friction capture is worst. If this creates a blind spot, revisit.

**1b. SYNTHESIS.md template update** (`.orch/templates/SYNTHESIS.md`)

Add optional "Friction" section after "Unexplored Questions":

```markdown
## Friction

**System friction experienced during this session:**
- [Category]: [Description] — [Impact: time cost, tool calls wasted, workaround used]

*(If no friction, write: "No friction — smooth session")*
```

**1c. Discovered work section update** (`skills/src/shared/worker-base/.skillc/discovered-work.md`)

Add clarification to distinguish friction from discovered work:

```markdown
**Friction vs. Discovered Work:**
- **Discovered work** = "I found a bug/debt in the code" → Create issue via `bd create`
- **Friction** = "The system made my work harder" → Report via `Friction:` comment
- Overlap is fine — report both if applicable
```

#### Phase 2: Orchestrator Consumption (orch complete + events)

**2a. Friction advisory in orch complete** (`cmd/orch/complete_pipeline.go`)

Add `surfaceFrictionAdvisory()` to `runCompletionAdvisories()`, after probe verdicts:

```go
// Surface friction reports for orchestrator review
if target.BeadsID != "" && !target.IsOrchestratorSession {
    frictionItems := verify.ParseFrictionComments(target.BeadsID, target.BeadsProjectDir)
    if len(frictionItems) > 0 {
        fmt.Print(verify.FormatFrictionAdvisory(frictionItems))
    } else if !isLightReview {
        fmt.Println("\n⚠️  No friction reported (expected Friction: comment)")
    }
}
```

**2b. Friction comment parser** (`pkg/verify/friction.go`)

```go
type FrictionItem struct {
    Category    string // bug, gap, ceremony, tooling, none
    Description string
    BeadsID     string
    Timestamp   time.Time
}

func ParseFrictionComments(beadsID, projectDir string) []FrictionItem {
    // Parse bd comments matching: Friction: (\w+)(?:: (.+))?
}

func FormatFrictionAdvisory(items []FrictionItem) string {
    // Format for orchestrator display during orch complete
    // Group by category, highlight bugs and gaps
}
```

**2c. Event accumulation** (`pkg/events/logger.go`)

Add `friction.reported` event type:

```go
// Emitted during orch complete when friction is present
{
    "type": "friction.reported",
    "beads_id": "orch-go-xxxxx",
    "category": "bug",
    "description": "beads dir resolution fails from nested repos",
    "skill": "feature-impl",
    "timestamp": "2026-03-05T17:00:00Z"
}
```

This enables cross-session pattern detection via `grep friction.reported ~/.orch/events.jsonl | jq .category | sort | uniq -c`.

#### Phase 3: Cross-Session Synthesis

**3a. Friction summary command** (future, not immediate)

```bash
orch friction summary          # Last 30 days, grouped by category
orch friction summary --bugs   # Bugs only, with recurrence count
orch friction patterns         # Recurring items (3+ reports)
```

This replaces the removed `orch friction` command with a consumption-focused tool instead of a capture-focused one.

**3b. Progressive enforcement trigger**

Track friction reporting rate in events.jsonl. If non-light completions report friction <50% of the time after 30 days, consider promoting to soft gate (warning in orch complete but non-blocking).

### Alternative Approaches Considered

**Option B: Mid-Session Capture (inline friction triggers)**
- **Pros:** Captures friction at full fidelity per Capture at Context; no reconstruction needed
- **Cons:** Invasive — requires agents to be friction-aware throughout work; adds cognitive load to every session; would be the ceremony agents complain about
- **When to use instead:** If end-of-session reports are consistently low-quality (short, vague, reconstructed)

**Option C: Hard Gate from Day One**
- **Pros:** Guaranteed compliance; no agents skip it
- **Cons:** Violates Legibility Over Compliance (adding ceremony before measuring its value); breaks all in-flight work on deployment; adds one more gate to an already-heavy V0 set
- **When to use instead:** If advisory data shows agents skip friction reporting >50% of the time with genuine friction present

**Option D: Auto-Create Issues from Friction Reports**
- **Pros:** Zero orchestrator overhead; bugs get tracked immediately
- **Cons:** Duplicate detection is hard; ceremony items aren't bugs; flooding issue tracker with ambient signals defeats triage; loses orchestrator judgment about what's already known
- **When to use instead:** If bugs account for >60% of friction reports and duplication rate is <10%

**Rationale for recommendation:** Option A (structured comment + advisory) is the minimal viable protocol that satisfies all 6 constraining principles. It's lightweight enough to avoid the ceremony trap, structured enough for machine consumption, and positioned correctly in the exit flow per Capture at Context. The progressive enforcement path means we pay the cost of gates only when evidence justifies them.

---

### Implementation Details

**What to implement first:**
1. Worker-base skill update (Phase 1a) — changes agent behavior immediately
2. SYNTHESIS.md template update (Phase 1b) — provides narrative friction context
3. Friction parser + advisory (Phase 2a, 2b) — enables orchestrator consumption
4. Event accumulation (Phase 2c) — enables cross-session patterns

**Things to watch out for:**
- ⚠️ Light-tier exemption creates a blind spot — trivial tasks may have the highest ceremony friction but are exempt from reporting it
- ⚠️ "Friction: none" could become a muscle-memory response that agents use without reflecting. Monitor for sessions where orch complete reveals clear friction but agent reported "none"
- ⚠️ The `Friction:` prefix must not collide with other beads comment patterns (Phase:, CROSS_REPO_ISSUE:, etc.). Verified: no current collision
- ⚠️ Defect class 5 (Contradictory Authority Signals): Friction comment says "bug" but discovered work says "not a bug" — orchestrator must reconcile

**Areas needing further investigation:**
- Optimal threshold for progressive enforcement promotion (50% skip rate? 30%?)
- Whether light-tier exemption should be revisited after initial data
- Cross-project friction aggregation (friction in price-watch about orch-go tooling — how does it route?)

**Success criteria:**
- ✅ >70% of non-light completions include friction comments within 30 days
- ✅ At least 3 bugs discovered via friction reports that would not have been captured via discovered work
- ✅ Friction capture itself is NOT the #1 reported friction item (the meta-signal check)
- ✅ Orchestrator uses friction data to drive at least 1 systemic improvement per month

---

## References

**Files Examined:**
- `skills/src/shared/worker-base/.skillc/completion.md` — Current exit protocol (template with tier conditionals)
- `skills/src/shared/worker-base/.skillc/discovered-work.md` — Existing discovered work protocol
- `cmd/orch/complete_pipeline.go` — Advisory pipeline in orch complete
- `pkg/verify/check.go` — Verification gate constants
- `~/.kb/principles.md` — 6 constraining principles identified
- `.kb/models/completion-verification/model.md` — V0-V3 gate levels
- `.kb/sessions/2026-03-05-debrief.md` — Confirmation `orch friction` was removed

**Commands Run:**
```bash
bd comments add orch-go-sw58d "Phase: Planning - Reading worker-base exit protocol, existing friction data, and orch complete flow"
kb create investigation design-friction-capture-protocol-worker
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-13-inv-probe-inventory-friction-gates-across.md` — Friction gate inventory (bypass ratios)
- **Investigation:** `.kb/investigations/2026-02-28-design-orientation-frame-gate-friction-biggest.md` — V0-V3 migration debt
- **Investigation:** `.kb/investigations/archived/2026-01-16-inv-design-skill-execution-bridge-friction.md` — Prior `orch friction` design (superseded)
- **Model:** `.kb/models/completion-verification/model.md` — Verification architecture

---

## Investigation History

**2026-03-05 17:00:** Investigation started
- Initial question: How should friction experienced by worker agents be captured during exit?
- Context: 5 price-watch workers reported friction when asked post-session; this signal is currently lost

**2026-03-05 17:30:** Context gathered
- Read worker-base exit protocol, orch complete pipeline, 6 constraining principles
- Identified 6 decision forks with substrate consultation

**2026-03-05 18:00:** Investigation completed
- Status: Complete
- Key outcome: Structured friction comment protocol with progressive enforcement, consuming via orch complete advisory
