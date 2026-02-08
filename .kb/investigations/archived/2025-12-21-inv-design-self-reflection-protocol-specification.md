---
linked_issues:
  - orch-go-ws4z.6
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Self-reflection is a three-layer architecture: `kb reflect` surfaces patterns (synthesis candidates, stale decisions, constraint drift, promotion candidates), chronicle provides temporal narrative views, and reflection checkpoints capture unexplored questions in SYNTHESIS.md.

**Evidence:** Synthesized 6 prior investigations that tested detection mechanisms (grep-based citations), validated signal hierarchies (density > time intervals), and designed the kb reflect command interface (4 types: synthesis/stale/drift/promote).

**Knowledge:** The system develops institutional memory through signal-triggered reflection (not scheduled), content parsing (not indexes), and human-in-loop synthesis (not automation). Session amnesia is the foundational constraint; self-reflection extends amnesia-resilience from sessions to project lifetime.

**Next:** Implement in sequence: (1) SYNTHESIS.md unexplored questions section, (2) kb reflect MVP with synthesis+promote types, (3) kb chronicle command, (4) daemon integration for overnight analysis. Close this epic.

**Confidence:** High (85%) - All component designs validated by prior investigations; end-to-end integration untested.

---

# Design: Self-Reflection Protocol Specification

**Question:** How does the system develop institutional memory that transcends any single session? What are the integration points (hooks, daemon, commands), success metrics for 'system is self-aware', and implementation sequence?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** og-arch-design-self-reflection-21dec
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Framing

### The Core Challenge

The orchestration system addresses session amnesia (single session → next session) through artifacts. But it lacks project-lifetime memory:
- Artifacts accumulate without awareness of their relationships
- No mechanism detects when knowledge is stale, duplicated, or needs synthesis
- Humans must notice patterns manually (cognitive load, inconsistent)
- Decision evolution isn't captured (why did X become Y?)

### Success Criteria

A self-aware system should:
1. Surface synthesis opportunities autonomously ("5 investigations about X with no synthesis")
2. Detect stale decisions (not cited, contradicted by practice)
3. Identify constraint drift (implementation supersedes documented constraint)
4. Enable knowledge promotion (kn → kb → principles)
5. Capture decision evolution narratives (chronicles)

### Constraints

From prior investigations:
- **Content parsing is sufficient** (ws4z.7) - No new indexes or databases needed
- **Density thresholds over time intervals** (ws4z.8) - "3+ investigations" not "weekly review"
- **Human-in-loop synthesis** (ws4z.9) - Automated gathering, human narrative
- **Evidence hierarchy applies** (ws4z.10) - Code is truth; artifacts are claims to verify
- **Artifact-level is minimal change** (4kwt.8) - SYNTHESIS.md section, not new processes

---

## Findings from Prior Investigations

### Finding 1: Four Reflection Types Map to Validated Signals

**Evidence from ws4z.8 Temporal Signals:**

| Signal Discovered | Detection Method | Maps To |
|-------------------|------------------|---------|
| Investigation clustering | `ls .kb/investigations/*{topic}*.md | wc -l` | `--type synthesis` |
| Repeated constraints | `jq '.[] | select(.content | test("topic"))' .kn/` | `--type promote` |
| Implementation contradiction | `rg "constraint-pattern" pkg/` vs constraint | `--type drift` |
| Low/zero citations | `rg -c "artifact-name" .kb/` | `--type stale` |

**Source:** ws4z.8 investigation, lines 35-85

**Significance:** These are the only signals with acceptable value/noise ratio. Other candidates (staleness by age, citation convergence) were rejected as either not observable or too noisy.

---

### Finding 2: Content Parsing Sufficient at Current Scale

**Evidence from ws4z.7 Citations:**
- Inbound link discovery via `rg "artifact-name" .kb/` takes <100ms on 138 files
- 37% of artifacts contain explicit references
- `ref_count` field exists in kn but is always 0 (unused)
- Current scale (172 investigations, 30 kn entries) doesn't justify index

**Source:** ws4z.7 SYNTHESIS.md, lines 30-48

**Significance:** `kb reflect` can be implemented as thin wrappers around grep/rg/jq. No new data structures, no maintenance overhead.

---

### Finding 3: Chronicle is View Over Existing Artifacts

**Evidence from ws4z.9 Chronicle:**
- Registry evolution chronicle was manually authored by orchestrator
- All source data exists (git history, kn entries, investigations, decisions)
- Narrative structure is essential (causation, not just timeline)
- Orchestrator creates, tooling assists

**Source:** ws4z.9 investigation, lines 1-12, 270-292

**Significance:** `kb chronicle "topic"` should gather sources; orchestrator synthesizes narrative. Not a new artifact type—a view command.

---

### Finding 4: Three Signals for Outdated Constraints

**Evidence from ws4z.10 Constraint Validation:**

1. **Implementation supersedes:** Code now does what constraint said was impossible
   - Test: "fire-and-forget" constraint vs pkg/spawn/session.go (provides session ID)
   
2. **Context shift:** Major architectural decision invalidates domain
   - Test: Go rewrite makes Python orch-cli constraints potentially irrelevant
   
3. **Evidence contradiction:** Test shows constraint doesn't hold

**NOT signals:** Age (valid constraints can be old), duplicates (indicate importance, not obsolescence)

**Source:** ws4z.10 investigation, lines 39-100

**Significance:** "Wrong constraint" vs "misapplication" requires testing against current code. Cannot be fully automated—must surface for human review.

---

### Finding 5: Unexplored Questions via Artifact Section

**Evidence from 4kwt.8 Reflection Checkpoints:**
- SYNTHESIS.md already has spawn-follow-up, escalate scaffolding
- Post-synthesis reflection created orch-go-ws4z epic (6 children)
- Value comes from orchestrator reviewing completed work
- Existing workflow (orch review → orch send) sufficient

**Source:** 4kwt.8 investigation, lines 35-95

**Significance:** Add "Unexplored Questions" section to SYNTHESIS.md template—lowest cost change that captures reflection value.

---

### Finding 6: kb reflect Command Design Ready

**Evidence from ws4z.4 Design:**

```bash
kb reflect                     # Run all types, show summary
kb reflect --type synthesis    # Investigations needing consolidation
kb reflect --type stale        # Decisions with low/no citations
kb reflect --type drift        # Constraints contradicted by code
kb reflect --type promote      # kn entries ready for kb promotion
```

Shell script MVP using existing tools (rg, jq, awk). Human-readable output with actionable suggestions.

**Source:** ws4z.4 investigation, lines 180-400 (detection algorithms)

**Significance:** Command interface designed and ready for implementation. Promoted to decision.

---

## Synthesis: The Self-Reflection Protocol

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                  SELF-REFLECTION ARCHITECTURE                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [1] DETECTION LAYER (signal-triggered)                        │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ kb reflect --type {synthesis|stale|drift|promote}       │  │
│  │   └─ Content parsing via rg/grep/jq                     │  │
│  │   └─ Density thresholds (3+ investigations, 2+ dupes)   │  │
│  │   └─ Output: actionable suggestions with file paths     │  │
│  └─────────────────────────────────────────────────────────┘  │
│                           │                                     │
│                           ▼                                     │
│  [2] TEMPORAL NARRATIVE LAYER (on-demand)                      │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ kb chronicle "topic"                                     │  │
│  │   └─ Gathers: git history + kn + investigations + beads │  │
│  │   └─ Output: sorted timeline for orchestrator synthesis  │  │
│  │   └─ Orchestrator writes narrative (human-in-loop)      │  │
│  └─────────────────────────────────────────────────────────┘  │
│                           │                                     │
│                           ▼                                     │
│  [3] CAPTURE LAYER (per-session)                               │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ SYNTHESIS.md: Unexplored Questions section              │  │
│  │   └─ Questions that emerged during session              │  │
│  │   └─ Areas worth exploring further                      │  │
│  │   └─ What remains unclear                               │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Integration Points

| Layer | Component | Integration Point | Trigger |
|-------|-----------|-------------------|---------|
| **Detection** | `kb reflect` | Command (CLI) | On-demand or daemon |
| **Detection** | SessionStart hook | Surfaces if suggestions exist | Every session start |
| **Detection** | `orch daemon run` | Runs overnight analysis | Scheduled/continuous |
| **Narrative** | `kb chronicle` | Command (CLI) | Orchestrator request |
| **Capture** | SYNTHESIS.md section | Agent workflow | Session completion |
| **Validation** | `orch review` | Displays unexplored questions | Before orch complete |

### Data Flow

```
Session N                           Session N+1
    │                                   │
    ▼                                   │
[Agent work]                            │
    │                                   │
    ▼                                   │
[SYNTHESIS.md created]                  │
 - Unexplored questions captured        │
 - kn entries externalized              │
    │                                   │
    ▼                                   │
[orch daemon run]  ────────────────────►│
 - kb reflect detects:                  │
   • 4 investigations on "tmux fallback"│
   • 2 duplicate constraints            │
   • 0-citation decision, 8 days old    │
 - Stores: ~/.orch/reflect-suggestions.json
    │                                   │
    │                      [SessionStart hook]
    │                       - Surfaces: "3 items need reflection review"
    │                                   │
    │                                   ▼
    │                      [Orchestrator reviews]
    │                       - kb chronicle "tmux fallback"
    │                       - Synthesizes narrative
    │                       - kn supersede outdated constraints
    ▼                                   ▼
[Institutional memory evolves] ◄───────┘
```

---

## The Core Question Answered

**How does the system develop institutional memory that transcends any single session?**

Through three complementary mechanisms:

### 1. Signal-Triggered Detection (kb reflect)

The system scans for patterns that indicate knowledge needs attention:
- **Investigation clustering:** "You have 5 investigations about X with no synthesis"
- **Duplicate constraints:** "3 entries about Y - consolidate or promote"
- **Stale decisions:** "Decision Z has zero citations after 7 days"
- **Constraint drift:** "Implementation now contradicts documented constraint"

Detection uses content parsing (grep), not indexes. Triggers on density, not schedules.

### 2. Temporal Narrative Synthesis (kb chronicle)

When patterns surface, the orchestrator creates understanding:
- `kb chronicle "topic"` gathers all temporal data about a topic
- Orchestrator synthesizes narrative (causation, not just timeline)
- Result: understanding of "why did X evolve to Y?"

This is a VIEW over existing artifacts, not a new artifact type.

### 3. Session Capture (SYNTHESIS.md)

Each session captures questions for future sessions:
- "Unexplored Questions" section in SYNTHESIS.md
- `orch review` surfaces these before completion
- Orchestrator decides what to pursue

### Why This Works

The protocol extends amnesia-resilience from sessions to project lifetime:

| Session Amnesia | Project Amnesia |
|-----------------|-----------------|
| SPAWN_CONTEXT.md provides resumption | kb reflect surfaces patterns |
| SYNTHESIS.md captures session delta | kb chronicle captures evolution |
| Phase tracking ensures completion | Unexplored Questions ensure continuity |
| kn externalizes learning | kn → kb promotion formalizes learning |

**Principle applied:** Session Amnesia (foundational). Every pattern in the self-reflection system helps the next Claude (or the orchestrator) understand what the system knows about itself.

---

## Success Metrics for "System Is Self-Aware"

### Observable Behaviors

1. **Synthesis Surfacing**
   - **Metric:** System surfaces "N investigations about X, no synthesis" without human noticing first
   - **Test:** After 4+ investigations on a topic, `kb reflect --type synthesis` lists it

2. **Citation Visibility**
   - **Metric:** Load-bearing artifacts identifiable via citation frequency
   - **Test:** `kb reflect --type stale` finds 0-citation decisions >7 days old

3. **Drift Detection**
   - **Metric:** System detects when decisions no longer match practice
   - **Test:** `kb reflect --type drift` surfaces constraint contradicted by code

4. **Promotion Candidates**
   - **Metric:** Duplicate kn entries trigger consolidation suggestion
   - **Test:** 2+ similar kn entries → `kb reflect --type promote` lists them

5. **Chronicle Capability**
   - **Metric:** `kb chronicle "topic"` produces usable input for synthesis
   - **Test:** Orchestrator can write evolution narrative from chronicle output

### Quantitative Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| **False positive rate** | <20% | User feedback on suggestions |
| **Detection latency** | <5 seconds | `time kb reflect` on 200+ artifacts |
| **Chronicle coverage** | All source types | git + kn + kb + beads in output |
| **Unexplored questions captured** | >50% of sessions | Audit SYNTHESIS.md files |

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

All component designs validated by prior investigations with concrete evidence. The architecture synthesizes tested detection mechanisms (grep-based), validated signals (density thresholds), and proven patterns (SYNTHESIS.md, kb commands). End-to-end integration is the remaining unknown.

**What's certain:**

- ✅ Four reflection types map to actionable signals (tested in ws4z.8)
- ✅ Content parsing is sufficient at current scale (validated in ws4z.7)
- ✅ Chronicle should be view, not artifact type (designed in ws4z.9)
- ✅ Constraint validation requires human-in-loop (proven in ws4z.10)
- ✅ SYNTHESIS.md section is minimal change for reflection capture (designed in 4kwt.8)
- ✅ kb reflect command interface ready (decided in ws4z.4)

**What's uncertain:**

- ⚠️ Exact detection thresholds (3+ investigations? 2+ duplicates?)
- ⚠️ False positive rate in drift detection (semantic matching is imprecise)
- ⚠️ Whether agents will populate Unexplored Questions section consistently
- ⚠️ End-to-end flow through daemon → hook → orchestrator

**What would increase confidence to Very High (95%+):**

- Implement MVP and test against real usage patterns
- Tune thresholds based on false positive/negative rates
- Validate with 5+ orchestrator sessions using the full protocol
- User feedback on suggestion actionability

---

## Implementation Recommendations

### Recommended Approach ⭐

**Layered Implementation with Validation Gates** - Build each layer, validate with real usage, then proceed to next.

**Why this approach:**
- Each layer is independently valuable (partial rollout is useful)
- Thresholds can be tuned at each stage
- Failure of one layer doesn't block others
- Matches existing pattern (kb, kn, orch are separate tools)

**Trade-offs accepted:**
- Slower than all-at-once implementation
- Temporary inconsistency between layers
- Some features delayed (daemon integration is last)

### Implementation Sequence

```
Phase 1: CAPTURE LAYER (Hours)
├── Add Unexplored Questions section to SYNTHESIS.md template
├── Update orch review to display unexplored questions
└── Validate: 5 sessions have questions captured

Phase 2: DETECTION LAYER MVP (1-2 days)
├── Implement kb reflect shell script
├── Start with --type synthesis (simplest detection)
├── Add --type promote (builds on synthesis)
├── Validate: <20% false positive rate

Phase 3: DETECTION LAYER COMPLETE (1-2 days)
├── Add --type stale (citation counting)
├── Add --type drift (constraint validation)
├── Add --json flag for machine output
└── Validate: All 4 types produce actionable output

Phase 4: TEMPORAL LAYER (1-2 days)
├── Implement kb chronicle command
├── Query git + kn + kb + beads for topic
├── Output sorted timeline
└── Validate: Orchestrator can synthesize narrative

Phase 5: INTEGRATION (1-2 days)
├── Add reflection analysis to orch daemon run
├── Store suggestions in ~/.orch/reflect-suggestions.json
├── Add SessionStart hook to surface suggestions
└── Validate: End-to-end flow works
```

### Alternative Approaches Considered

**Option B: All-at-once implementation**
- **Pros:** Faster if design is correct, single integration point
- **Cons:** Risk of systemic failure, harder to debug, no early validation
- **When to use instead:** If team capacity allows full-time focus

**Option C: Daemon-only (no kb reflect command)**
- **Pros:** Fully autonomous, no user action needed
- **Cons:** Loses on-demand capability, harder to debug
- **When to use instead:** After command proves valuable

**Option D: Skip chronicle layer**
- **Pros:** Simpler, fewer tools
- **Cons:** Loses temporal narrative capability
- **When to use instead:** If orchestrator doesn't need evolution understanding

**Rationale for recommendation:** Layered approach provides validation gates. If SYNTHESIS.md section doesn't get used, we know before investing in kb reflect. If kb reflect has high false positives, we tune before daemon integration.

---

### Implementation Details

**File Targets:**

| Layer | Create | Modify |
|-------|--------|--------|
| Capture | - | `.orch/templates/SYNTHESIS.md` |
| Detection | `~/.local/bin/kb-reflect` (or wherever kb lives) | - |
| Temporal | Add to kb script | - |
| Integration | `~/.orch/reflect-suggestions.json` | `pkg/daemon/daemon.go` |

**Unexplored Questions Section (add to SYNTHESIS.md):**

```markdown
---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- [Question 1 - why it's interesting]
- [Question 2 - why it's interesting]

**Areas worth exploring further:**
- [Area 1]
- [Area 2]

**What remains unclear:**
- [Uncertainty 1]
- [Uncertainty 2]

*(If nothing emerged, note: "Straightforward session, no unexplored territory")*
```

**kb reflect Detection Algorithms:**

```bash
# synthesis: Investigation clustering
ls .kb/investigations/*.md | 
  sed 's/.*2025-[0-9-]*-//' | sed 's/\.md$//' |
  sort | uniq -c | sort -rn |
  awk '$1 >= 3 {print "Topic:", $2, "has", $1, "investigations"}'

# promote: Duplicate kn entries
cat .kn/entries.jsonl | jq -r '.content' | 
  sort | uniq -c | sort -rn |
  awk '$1 >= 2 {print "Duplicate:", $0}'

# stale: Zero-citation decisions
for decision in .kb/decisions/*.md; do
  name=$(basename "$decision" .md)
  count=$(rg -c "$name" .kb/investigations/ 2>/dev/null | wc -l)
  age=$(( ($(date +%s) - $(stat -f %m "$decision")) / 86400 ))
  if [ $count -eq 0 ] && [ $age -gt 7 ]; then
    echo "Stale: $decision (0 citations, ${age}d old)"
  fi
done

# drift: Constraint validation (heuristic)
cat .kn/entries.jsonl | jq -r 'select(.type == "constraint") | 
  "\(.id)|\(.content)"' | while IFS='|' read -r id content; do
  patterns=$(echo "$content" | grep -oE '(func|pkg|cmd|\.go|session|registry)' | head -3)
  if [ -n "$patterns" ]; then
    # Surface for human review
    echo "Potential drift: $id - check against current code"
  fi
done
```

**Things to watch out for:**

- ⚠️ macOS vs Linux sed/awk differences (test on both)
- ⚠️ Path handling for cross-project usage
- ⚠️ Drift detection is heuristic—high false positive risk
- ⚠️ Don't train users to ignore suggestions (keep quality high)

**Success Criteria:**

- ✅ SYNTHESIS.md section captured in >50% of sessions
- ✅ kb reflect runs in <5 seconds on 200+ artifacts
- ✅ Each type produces actionable recommendations
- ✅ False positive rate <20%
- ✅ kb chronicle produces usable orchestrator input

---

## References

**Investigations Synthesized:**

| ID | Investigation | Key Finding |
|----|---------------|-------------|
| ws4z.7 | Citation mechanisms | Grep-based sufficient, no index needed |
| ws4z.8 | Temporal signals | Density > time intervals, daemon recommended |
| ws4z.9 | Chronicle artifact | View over existing, not new type |
| ws4z.10 | Constraint validation | Test against code, three signal types |
| 4kwt.8 | Reflection checkpoints | Artifact-level (SYNTHESIS.md) is minimal |
| ws4z.4 | kb reflect design | 4 types, shell script MVP, grep-based |

**Decisions:**
- `.kb/decisions/2025-12-21-kb-reflect-command-interface.md` - Command interface
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - 5+3 artifact types

**Principles Applied:**
- `~/.kb/principles.md` Session Amnesia (foundational constraint)
- `~/.kb/principles.md` Evidence Hierarchy (artifacts are claims, code is truth)
- `~/.kb/principles.md` Surfacing Over Browsing (bring relevant state to agent)

**Related Epics:**
- `orch-go-4kwt` - Amnesia-Resilient Artifact Architecture (what artifacts exist)
- `orch-go-ws4z` - System Self-Reflection (how artifacts become aware of each other)

---

## Self-Review

### Phase-Specific Checks (Architect Skill)

| Phase | Check | Status |
|-------|-------|--------|
| **Problem Framing** | Success criteria defined? | ✅ 5 criteria listed |
| **Exploration** | 2+ approaches compared? | ✅ Layered vs all-at-once vs daemon-only |
| **Synthesis** | Clear recommendation with reasoning? | ✅ Layered implementation with validation gates |
| **Externalization** | Investigation produced? | ✅ This artifact |

### Self-Review Checklist

- [x] All 4 architect phases completed
- [x] Recommendation made with trade-off analysis
- [x] Implementation-ready output (sequence, file targets, algorithms)
- [x] Investigation artifact produced
- [ ] Feature list reviewed (N/A - no .orch/features.json exists)
- [x] Principles cited (Session Amnesia, Evidence Hierarchy)

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn decide "Self-reflection is signal-triggered not time-scheduled" --reason "Density thresholds (3+ investigations) produce actionable signals; time intervals (weekly review) produce noise. Per ws4z.8 investigation."
```

---

## Investigation History

**2025-12-21 17:30:** Investigation started
- Initial question: Design self-reflection protocol specification
- Context: Final synthesis of ws4z epic (6 investigations)

**2025-12-21 17:45:** Read all 6 investigations
- ws4z.7: Citations via grep
- ws4z.8: Temporal signals, daemon recommendation
- ws4z.9: Chronicle as view
- ws4z.10: Constraint validation against code
- 4kwt.8: Reflection checkpoint via SYNTHESIS.md
- ws4z.4: kb reflect command design

**2025-12-21 18:00:** Architecture synthesized
- Three-layer model: Detection, Temporal Narrative, Capture
- Integration points: command, hook, daemon
- Success metrics: 5 observable behaviors

**2025-12-21 18:15:** Implementation sequence specified
- 5 phases with validation gates
- File targets and detection algorithms
- Layered approach with independent value at each stage

**2025-12-21 18:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Coherent self-reflection protocol with validated components and clear implementation path
