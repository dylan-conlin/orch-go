# Model: Signal-to-Design Loop

**Created:** 2026-02-26
**Status:** Active
**Context:** Discovered while investigating 6 configuration-drift investigations that drove the named-agreements design. The pattern — metadata accumulation → clustering → synthesis → systemic design — was already operating unnamed across multiple systems.

---

## What This Is

A model for how operational friction becomes systemic improvement. The core insight:

> **Individual signals are noise. Clustered signals are design pressure. The difference is metadata that makes signals groupable.**

Every system that learns from its own operation follows this loop. The quality of the learning depends on three things: signal capture (do agents actually record what happened?), clustering resolution (can the system group related signals?), and synthesis authority (does someone act on the cluster?).

This model describes the general loop, catalogs known instances, and identifies where stages are missing — because a loop with a broken stage doesn't learn.

---

## How This Works

### The Five Stages

```
┌─────────────────────────────────────────────────────────────────┐
│  1. SIGNAL CAPTURE                                              │
│     Agents record operational friction as structured metadata   │
│     Key: Must be low-cost, embedded in workflow, not separate   │
│     Anti-pattern: "Fill out the reflection form after"          │
│     Good pattern: Metadata field on artifact you're already     │
│                   creating (defect-class on investigation)      │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. ACCUMULATION                                                │
│     Signals persist across sessions in queryable storage        │
│     Key: Must survive session boundaries (session amnesia)      │
│     Stores: .kb/ artifacts, gap-tracker.json, beads issues,    │
│             quick entries                                       │
│     Retention: Bounded (30-day windows, event caps)             │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. CLUSTERING                                                  │
│     Related signals grouped by shared metadata                  │
│     Key: Requires explicit, machine-readable clustering key     │
│     Methods:                                                    │
│       - Explicit tag (defect-class: configuration-drift)        │
│       - Threshold count (RecurrenceThreshold = 3)               │
│       - Lexical proximity (filename/topic similarity)           │
│     Resolution hierarchy: explicit tag > threshold > lexical    │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. SYNTHESIS                                                   │
│     Clustered signals interpreted as systemic forces            │
│     Key: Requires authority — someone must decide "this         │
│          cluster means something and warrants design"           │
│     Who: Human (Dylan) or orchestrator with human approval      │
│     Output: Design recommendation, not just pattern report      │
│     Anti-pattern: Dashboard that shows clusters but doesn't     │
│                   recommend action                              │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. DESIGN RESPONSE                                             │
│     Systemic change prevents the signal class from recurring    │
│     Key: Response targets the SYSTEM, not individual instances  │
│     Forms: New artifact type (agreements), new gate,            │
│            skill redesign, spawn heuristic change               │
│     Verification: Does the signal class frequency decrease?     │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
   (back to Stage 1 — fewer signals of that class,
    but new signal classes emerge from the changed system)
```

### What Makes Each Stage Work

| Stage | Works When | Breaks When |
|-------|-----------|-------------|
| **Capture** | Embedded in existing workflow; costs nothing extra | Requires separate reflection step; agents skip it |
| **Accumulation** | Survives sessions; queryable; bounded growth | Session-local; grows unbounded; unqueryable |
| **Clustering** | Explicit metadata key; machine-readable | Relies on human to notice pattern; natural language only |
| **Synthesis** | Has authority to recommend design; human in loop | Pure reporting; no one acts on clusters |
| **Design Response** | Targets system, not instances; verifiable | Patches individual instances; no way to measure |

---

## Known Instances

### Instance 1: Defect-Class → Named Agreements

**Signal:** Investigation tagged with `Defect-Class: configuration-drift`
**Accumulation:** `.kb/investigations/` with defect-class field in frontmatter
**Clustering:** `kb reflect` surfaces investigations sharing defect-class
**Synthesis:** Orchestrator + Dylan recognized 6 instances → spawned architect
**Design Response:** Named agreements system (`.kb/agreements/`)
**Stage maturity:** Capture ✅ | Accumulation ✅ | Clustering ⚠️ manual | Synthesis ⚠️ human-initiated | Response ✅

### Instance 2: Context Gaps → Knowledge Entries

**Signal:** `kb context` returns nothing during spawn (GapEvent recorded)
**Accumulation:** `~/.orch/gap-tracker.json` (30-day window, 1000-event cap)
**Clustering:** `FindRecurringGaps()` with RecurrenceThreshold=3
**Synthesis:** `orch learn` generates runnable commands
**Design Response:** Knowledge entries created (`kn decide`, `kn constrain`)
**Stage maturity:** Capture ✅ | Accumulation ✅ | Clustering ✅ automated | Synthesis ✅ automated | Response ⚠️ requires human execution
**Model:** `orch-go/.kb/models/system-learning-loop/model.md`

### Instance 3: Quick Entries → Decisions/Models

**Signal:** `kb quick decide/tried/constrain` during work
**Accumulation:** `.kb/quick/entries.jsonl`
**Clustering:** Manual review; domain keyword proximity
**Synthesis:** Orchestrator reviews at completion touchpoint (continuous maintenance decision)
**Design Response:** Promoted to decision, model update, or principle
**Stage maturity:** Capture ✅ | Accumulation ✅ | Clustering ❌ manual only | Synthesis ⚠️ touchpoint-dependent | Response ✅

### Instance 4: File Accretion → Extraction

**Signal:** Features added to same file (implicit — line count growth)
**Accumulation:** Git history (file size over time)
**Clustering:** Accretion-gravity principle fires when file exceeds threshold
**Synthesis:** Developer/agent recognizes god-object forming
**Design Response:** Package extraction, responsibility split
**Stage maturity:** Capture ⚠️ implicit | Accumulation ✅ | Clustering ⚠️ no automation | Synthesis ⚠️ human-noticed | Response ✅

### Instance 5: Skill-to-Task Fit (Proposed)

**Signal:** Worker self-assessment at session close: `skill_fit: full|partial|poor`
**Accumulation:** Beads comments or dedicated tracker
**Clustering:** Group by skill + fit rating; threshold detection
**Synthesis:** Daemon or orchestrator surfaces: "feature-impl workers report partial fit 4x this month"
**Design Response:** Skill boundary redesign, spawn heuristic update
**Stage maturity:** Capture 🔲 not yet built | Accumulation 🔲 | Clustering 🔲 | Synthesis 🔲 | Response 🔲

---

## Why This Fails

### Failure Mode 1: Capture Friction Kills the Loop

**Symptom:** Agents don't record signals. The loop starves.

**Why it happens:** Signal capture is a separate step from the work. "Fill out the form after" gets skipped under time pressure. AI agents are better than humans here (they "fill out the suggestion box") but only if the capture is embedded in an artifact they're already creating.

**The fix:** Make capture a field on an existing artifact, not a new artifact. Defect-class works because it's a field on investigations agents already write. Skill-fit would work as a field in the beads completion comment agents already post.

**Principle:** Gate Over Remind — if it's not gated, it won't happen consistently.

### Failure Mode 2: Clustering Resolution Too Low

**Symptom:** Signals accumulate but patterns aren't visible. Everything looks unique.

**Why it happens:** Clustering relies on natural language similarity (lexical proximity) instead of explicit metadata. "The daemon dropped a field" and "the skill template wasn't updated" look like different problems until you tag both `configuration-drift`.

**The fix:** Explicit, enumerated clustering keys. The defect-class field works because it's a constrained vocabulary, not free text. Quick entries fail at clustering because they're free-form.

**Principle:** Defect-Class Blindness — individual investigations fixing symptoms without connecting shared root causes.

### Failure Mode 3: Synthesis Without Authority

**Symptom:** Reports show clusters but nothing changes. "Dashboard effect."

**Why it happens:** Synthesis requires someone to decide "this cluster warrants systemic design." If synthesis is fully automated with no authority to create work, it produces reports that sit unread.

**The fix:** Synthesis must either have authority to create architect issues (with human approval gate) or surface to someone who does. The System Learning Loop works because `orch learn` generates *runnable commands*, not just reports.

**Principle:** Understanding Through Engagement — synthesis requires direct orchestrator engagement.

### Failure Mode 4: Design Response Targets Instances, Not System

**Symptom:** Each signal gets a fix but the same class keeps appearing.

**Why it happens:** It's faster to fix the instance than redesign the system. "Fix the template" instead of "create an agreement system that catches all template drift."

**The fix:** When a cluster has 3+ instances, the design response must target the *class*, not the latest instance. Named agreements target configuration-drift as a class, not just the kb-reflect output passthrough as an instance.

---

## Constraints

### What This Model Enables

- **Audit any learning loop:** For each instance, check which stages are present and which are missing
- **Design new loops intentionally:** When you notice recurring friction, ask "what would the 5 stages look like for this signal type?"
- **Prioritize automation:** Stages that are manual (⚠️) are automation candidates; stages that are missing (🔲) need building
- **Explain to newcomers:** "Here's how the system learns from its own operation"

### What This Model Constrains

- **Not all friction deserves a loop.** One-off problems don't need signal capture infrastructure. The threshold is recurrence (3+) not severity
- **Loops have maintenance cost.** Each new signal type needs storage, clustering logic, and synthesis attention. Don't create loops speculatively
- **Human-in-the-loop is intentional, not a gap.** Dylan wants to understand the forces driving systemic changes. Full automation of synthesis is not the goal

### The Audit Question

> For this type of operational friction: which of the 5 stages exists, which is manual, and which is missing? Where is the loop broken?

---

## Integration Points

- **With System Learning Loop** (`orch-go/.kb/models/system-learning-loop/model.md`): Instance 2 of this pattern. The most mature automated instance, covering all 5 stages for context-gap signals.

- **With Continuous Knowledge Maintenance** (`orch-go/.kb/decisions/2026-02-25-continuous-knowledge-maintenance.md`): Defines the touchpoints (completion, spawn, daemon idle) where stages 3-4 can fire. The plumbing that distributes the loop.

- **With Defect-Class Blindness** (`kb/.principlec/src/foundational/defect-class-blindness.md`): The motivating principle. Explains why clustering matters — without it, the same defect class ships repeatedly.

- **With Accretion-Gravity** (`kb/.principlec/src/foundational/accretion-gravity.md`): Instance 4 of this pattern. Describes the physics of signal accumulation in code.

- **With Gate Over Remind** (`kb/.principlec/src/foundational/gate-over-remind.md`): Constrains Stage 1 (capture) — if capture isn't gated, the loop starves.

- **With Named Agreements** (`.kb/investigations/2026-02-26-inv-design-cross-project-drift-detection.md`): The design response from Instance 1. First systemic design driven by explicit defect-class clustering.

---

## Evolution

| Date | Change | Trigger |
|------|--------|---------|
| 2026-02-26 | Created | Interactive session exploring defect-class metadata → reflection loop meta-pattern. Discovered pattern was operating unnamed across 4+ systems. |

---

## Open Questions

1. **Should the daemon auto-create architect issues when clusters cross a threshold, or only surface them for Dylan's approval?** Current preference: surface with evidence, Dylan approves. But the approval step could become a bottleneck if many signal types are active.

2. **What's the right vocabulary for skill-fit signals?** `full|partial|poor` is a start, but the clustering value comes from the *reason* (e.g., "spent 80% investigating"). Should the reason be free-text or also constrained?

3. **Can quick entries be made clusterable without losing their free-form nature?** They're the highest-volume signal but lowest clustering resolution. Adding a `domain:` tag might help, but mandatory tags add capture friction.

4. **How do you measure whether a design response actually reduced signal frequency?** The System Learning Loop has `orch learn effects` but it's manual. Cross-loop measurement ("did agreements reduce configuration-drift investigations?") doesn't exist yet.

5. **Where does the loop operate across projects vs. within a single project?** Context gaps are per-project. Defect-class clustering was cross-project (orch-go investigations drove orch-knowledge design). The model should work at both scales, but the tooling may need different implementations.

6. **Do AI agents reliably self-assess skill fit?** Instance 5 depends on worker self-reports being accurate. First probe when skill-fit capture is built: compare worker `skill_fit` ratings against orchestrator-observable signals (time overruns, unexpected investigation work, scope changes). If self-assessment correlates with external signals, the loop is sound. If not, capture needs corroboration (e.g., orchestrator also rates fit at completion review).
