---
status: open
type: design
date: 2026-02-27
triggers: ["architect session for orch-go-pgjz, building on design investigation 2026-02-27-design-flow-integrated-knowledge-surfacing.md"]
**Phase:** Complete
---

**Status:** Complete

# Design: Implementation Architecture for Flow-Integrated Knowledge Surfacing

## Design Question

Given the four design threads identified in the flow-integrated knowledge surfacing investigation, which thread should be prototyped first, what is the concrete implementation shape, and how should the threads be sequenced into implementation issues?

## Problem Framing

### Success Criteria
- Clear recommendation for which thread to prototype first, with substrate reasoning
- Concrete implementation shape for the first thread (commands, file changes, data flow)
- Resolution of open question 5 (where does meta-monitoring terminate)
- Sequenced beads issues with dependencies

### Constraints
- Must respect existing orchestrator skill structure (session start, completion review)
- Must respect context window limits (surfaced knowledge must be compact)
- Must respect "Pressure Over Compensation" principle (surface gaps, don't silently fill them)
- Must respect "Verification Bottleneck" (system can't change faster than Dylan can verify)
- Must work with existing kb context infrastructure (don't rebuild)
- Dylan doesn't type CLI commands — the orchestrator calls commands programmatically

### Scope
- IN: Architecture for all four threads, detailed design for first thread, issue sequencing
- OUT: Actual implementation, skill file changes, code writing

## Exploration (Fork Navigation)

### Fork 1: Which Thread First?

**Options:**
- A: Thread 1 — Model Surfacing at Engagement Moments (session start + completion enrichment)
- B: Thread 3 — Calibrated Trust (trust tiers at completion)
- C: Thread 2 — System Self-Awareness (throughput baseline)
- D: Thread 4 — Immersion Without Study (meta-integration)

**Substrate says:**
- **Principle (Verification Bottleneck):** "System can't change faster than a human can verify." Thread 1 directly addresses this — surfacing model context helps Dylan verify faster and more accurately. Thread 3 helps prioritize what to verify. Both serve the bottleneck.
- **Principle (Pressure Over Compensation):** Thread 1's model surfacing creates natural pressure to update stale models. Seeing "Your model of X was last verified 14 days ago" is pressure, not compensation.
- **Model (Orchestrator Session Lifecycle):** Session boundaries are already defined as integration points. Thread 1 extends these existing boundaries rather than creating new infrastructure.
- **Model (Completion Verification):** The completion pipeline already has 4 knowledge touchpoints (probe verdicts, architectural choices, knowledge maintenance, hotspot advisory). Thread 1 enhances the existing pipeline rather than building new infrastructure.
- **Decision (Continuous Knowledge Maintenance):** Already established the pattern of side-effect maintenance at touchpoints. Thread 1 aligns with this pattern — surfacing at existing moments, not creating new moments.

**RECOMMENDATION:** Option A (Thread 1) because:
1. It has the highest-leverage existing integration points (session start protocol, completion pipeline)
2. It's additive to existing infrastructure (extend `orch complete` touchpoints, add `orch orient` command)
3. It enables Thread 3 naturally — model freshness data surfaces at session start, trust signals aggregate at completion
4. Thread 2 (throughput baseline) is partially solved by `orch stats` — needs format adaptation, not new data collection
5. Thread 4 is an emergent property, not separately implementable

**Trade-off accepted:** Deferring calibrated trust means completion review pacing remains manual (orchestrator skill guidance) rather than system-guided. This is acceptable because the current skill guidance for pacing (Light/Medium/Heavy) already works — trust tiers would refine it, not create it.

---

### Fork 2: Session Start — New Command vs. Skill Guidance Only?

**Options:**
- A: New `orch orient` command that produces structured session orientation output
- B: Enhance existing `orch status` or `orch review` to include orientation
- C: Skill-only — update orchestrator skill to instruct running `kb context`, `orch stats`, `bd ready` separately

**Substrate says:**
- **Principle (Session Amnesia):** A dedicated command externalizes the orientation protocol into infrastructure. Skill guidance alone relies on each new orchestrator session remembering to run 3+ commands in sequence. A command is the more reliable path.
- **Decision (User Interaction Model from architect skill):** "CLI flags are for orchestrator to pass programmatically, not Dylan to type." The `orch orient` command is called by the orchestrator AI, which then presents the output to Dylan conversationally.
- **Model (Orchestrator Session Lifecycle):** Session start protocol is 5 steps. Steps 1-2 are conversational. Step 3 is a command (`orch backlog cull --dry-run`). Step 4 uses `bd ready` + `orch status`. Adding `orch orient` as a command that composites the information is consistent with the existing pattern.
- **Principle (Gate Over Remind):** A command that produces structured output is a gate (it either surfaces knowledge or it doesn't). Skill guidance that says "remember to check models" is a reminder (easily skipped).

**RECOMMENDATION:** Option A (new `orch orient` command) because:
1. It composites multiple data sources into a single orchestrator-consumable output
2. It can be referenced in the orchestrator skill's session start protocol as a concrete step
3. It creates infrastructure that Thread 3 (trust calibration) can extend later
4. It's called by the orchestrator AI, presented to Dylan conversationally (matches interaction model)

**Trade-off accepted:** New command means new code to maintain. Mitigated by keeping the command thin — it composes existing data sources (`bd ready`, `orch stats`, `kb context`) rather than implementing new data collection.

---

### Fork 3: Completion Enrichment — New Touchpoint vs. Enhance Existing?

**Options:**
- A: Add a new "model impact" touchpoint after existing knowledge maintenance
- B: Enhance existing knowledge maintenance touchpoint to include model cross-referencing
- C: Enhance probe verdicts touchpoint to also cover non-probe model impacts

**Substrate says:**
- **Model (Completion Verification):** The pipeline has clear separation of concerns: probe verdicts (model evidence), architectural choices (work-level tradeoffs), knowledge maintenance (quick entry lifecycle), hotspot advisory (codebase risk). Each touchpoint has a single responsibility.
- **Principle (Evolve by Distinction):** "When problems recur, ask what are we conflating?" Mixing model-impact analysis into knowledge maintenance would conflate two concerns: quick entry lifecycle and model state assessment.
- **Decision (Continuous Knowledge Maintenance):** "Close the feedback loop at the point of action." Model impact is a different feedback loop than quick entry promotion.

**RECOMMENDATION:** Option A (new "model impact" touchpoint) because:
1. Maintains single-responsibility of existing touchpoints
2. The probe verdicts touchpoint (lines 1031-1039) shows the pattern: find probes, format them, display. A model-impact touchpoint follows the same pattern: find relevant models for this completion, cross-reference with SYNTHESIS.md, display impact.
3. New touchpoint can be independently developed, tested, and disabled without affecting existing flow

**Implementation location:** Between probe verdicts (lines 1031-1039) and architectural choices (lines 1041-1048) in complete_cmd.go. This places model-level impacts before work-level details.

**Trade-off accepted:** Another step in the completion pipeline increases review time. Mitigated by making it informational (2-3 lines) and skipable when no models are affected.

---

### Fork 4: Model Summary Format — Human Sentences vs. Structured Data?

**Options:**
- A: 2-3 sentence plain-language summaries per relevant model
- B: Structured key-value format (model name, last updated, key claims, freshness)
- C: Single-line indicators (model name + freshness traffic light)

**Substrate says:**
- **Principle (Verification Bottleneck):** "Has a human observed this working?" The format must support human comprehension, not just information transfer. Sentences enable comprehension; structured data enables scanning.
- **Model (Orchestrator Session Lifecycle, "Strategic Comprehender" pattern):** The orchestrator's job is understanding. It needs enough context to form questions and probe Dylan's understanding. Sentences support this; single-line indicators don't.
- **Design Investigation (Thread 4: Immersion Without Study):** "Orientation IS model review." The orientation output should feel like a brief, where the orchestrator can say "You believe X about spawn architecture. Is that still true?" Sentences enable this; structured data doesn't.

**RECOMMENDATION:** Hybrid — Option A (sentences) for session start, Option C (indicators) for completion review.

At session start (via `orch orient`): 2-3 sentence model summaries for relevant models. The orchestrator reads these and incorporates them into conversation naturally.

At completion review (model-impact touchpoint): Single-line indicators showing which models are confirmed/extended/contradicted. The orchestrator can dig deeper if needed.

**Rationale:** Session start is an orientation moment — comprehension matters. Completion review is a checkpoint moment — scanning matters. Different moments need different densities.

---

### Fork 5: How to Determine "Relevant" Models?

**Options:**
- A: Match models to `bd ready` issues via keyword extraction (same as spawn kb context)
- B: Match models to `orch focus` areas
- C: Match models by freshness (stale models always surface)
- D: Combination: focus areas + freshness warnings

**Substrate says:**
- **Model (Spawn Architecture, invariant 3):** "KB context uses --global flag — cross-repo constraints are essential." Model relevance should use the same keyword extraction pattern that works for spawn context.
- **Investigation (design-flow-integrated-knowledge-surfacing):** "Models tagged with areas matching `bd ready` issues" is the first candidate.
- **Existing infra:** `pkg/spawn/kbcontext.go` already has keyword extraction (ExtractKeywords), model matching, and freshness detection (`--stale` flag). Reusing this is the lowest-risk path.

**RECOMMENDATION:** Option D (focus areas + freshness) because:
1. Focus areas provide intent-driven relevance (what Dylan is working on)
2. Freshness warnings surface drift regardless of focus (models going stale in areas you're not looking at)
3. Both data sources already exist (`orch focus` for areas, `kb context --stale` for freshness)

**Budget constraint:** At session start, surface at most 3 model summaries (relevant to ready work) + 2 freshness warnings (stale regardless of topic). Total: ~500-800 chars of model context. This fits easily in a conversational orientation without blowing context.

---

## Resolution: Open Question 5 — Where Does Meta-Monitoring Terminate?

**The question:** "How to avoid the meta-trap: building a system to monitor the system that monitors the system. Where does this terminate?"

**Answer:** The termination point is **the orchestrator conversation itself.**

The chain is:

```
Models (externalized understanding)
  → kb context / orch orient (surfacing at moments)
    → Orchestrator conversation (comprehension layer)
      → Dylan's verbal responses (verification signal)
        → Anti-sycophancy constraint (prevents rubber-stamping)
```

This terminates at Dylan's verbal responses because:

1. **The orchestrator IS the comprehension layer.** It's not spawnable, not automatable. Its job is understanding. This is the level where surfaced knowledge becomes actionable — or doesn't.

2. **Dylan's responses ARE the verification signal.** The explain-back gate already tests comprehension. If Dylan consistently misses model claims in explain-back responses, that's a signal that surfacing isn't working — but this is detected through the existing gate, not through a new monitoring system.

3. **The anti-sycophancy constraint prevents the loop from closing silently.** The orchestrator skill's hard constraint ("Don't mirror words as confirmation. If 'yeah that makes sense' without specifics, probe.") means the conversation can't short-circuit into mutual agreement without comprehension.

**What this means for implementation:**

- No "monitoring of the monitor" layer is needed
- The trust calibration system (Thread 3) is a **refinement of the termination point**, not an additional layer — it helps the orchestrator allocate attention more effectively at the point where the chain terminates
- The drift detection signal is: if Dylan's explain-back quality degrades (vague, misses key claims), surface this as a conversation-level observation, not a system-level metric. The orchestrator already has this responsibility via anti-sycophancy

**Substrate:**
- **Principle (Verification Bottleneck):** "System can't change faster than a human can verify." The human IS the termination. Not monitoring the human — being honest about the human's bandwidth.
- **Principle (Pressure Over Compensation):** Don't build a system to compensate for missed surfacing. Let missed surfacing create visible pressure (stale models, un-updated claims) that naturally surfaces in the next orientation.

---

## Synthesis: Implementation Architecture

### Thread Sequencing

```
Thread 1a: orch orient (session start)  ─┐
                                          ├── Thread 3: Trust Calibration
Thread 1b: Model-impact touchpoint (completion) ─┘
                                                    │
Thread 2: Throughput baseline (orch orient enrichment) ←─ independent, can parallel
```

Thread 4 (Immersion Without Study) emerges when 1a + 1b + 3 work together. Not separately implementable.

### Concrete Implementation: Thread 1a — `orch orient`

**New command:** `orch orient`

**Purpose:** Produce structured session orientation for the orchestrator to present to Dylan conversationally.

**Data sources (composited):**
1. `bd ready` — issues available to work on (already exists)
2. `orch stats --days 1 --json` — recent throughput (already exists, needs `--json` output)
3. `kb context "{focus areas}" --format json` — relevant models and constraints (already exists)
4. Model freshness check — `Last Updated` dates from `.kb/models/*/model.md` headers (new logic, lightweight)

**Output format (for orchestrator consumption):**

```
== SESSION ORIENTATION ==

📊 Since last session:
   Completions: 4 | Abandonments: 1 | In-progress: 3
   Avg duration: 38 min

🎯 Ready to work (from bd ready):
   [3 issues with titles and priorities]

🧠 Relevant models (matching ready work):
   - Spawn Architecture (updated 2d ago): [2-sentence summary]
   - Completion Verification (updated 1d ago): [2-sentence summary]

⚠️ Stale models:
   - Coaching Plugin (updated 33d ago, no recent probes)

🔥 Focus areas: [from orch focus, if set]
```

**Files to create/modify:**
- `cmd/orch/orient_cmd.go` — New command (~150-200 lines)
- `pkg/orient/orient.go` — Orientation data collection and formatting (~200-250 lines)
- `pkg/orient/model_freshness.go` — Model freshness scanning (~100 lines)

**Dependencies:** None on existing code changes. Uses existing `bd`, `orch stats`, `kb context` as data sources.

**Orchestrator skill change (separate issue):** Add step 3.5 to session start protocol: "Run `orch orient` and present orientation to Dylan."

### Concrete Implementation: Thread 1b — Model-Impact Touchpoint

**Enhancement to completion pipeline.**

**New touchpoint in `complete_cmd.go`:** Between probe verdicts (line 1039) and architectural choices (line 1041).

**Logic:**
1. Extract topics from SYNTHESIS.md plain-language summary
2. Run `kb context "{topics}" --format json` to find relevant models
3. Cross-reference: which models' claims are confirmed/extended/contradicted by this completion?
4. Display 1-3 line summary: "Model impact: [model] — [impact type]"

**Heuristics for impact detection:**
- If agent created a probe → already handled by probe verdicts touchpoint (skip)
- If SYNTHESIS.md mentions model names or key domain terms → surface those models
- If agent modified files that appear in model's "Primary Evidence" → flag potential impact

**Files to create/modify:**
- `cmd/orch/complete_model_impact.go` — New touchpoint (~150 lines)
- Modify `cmd/orch/complete_cmd.go` — Wire in new touchpoint (5-10 lines)

**Dependencies:** Should come after `orch orient` because model freshness scanning logic can be shared.

### Future Threads (not in first prototype)

**Thread 2 (Throughput Baseline):** Integrate into `orch orient` output. Needs `orch stats --json` for programmatic consumption. Low effort, can be part of the `orch orient` implementation.

**Thread 3 (Calibrated Trust):** After Thread 1b, extend the completion pipeline with a trust tier calculation:
- Green: V0-V1 work, relevant model probed <3 days ago, single-file scope, tests pass
- Yellow: V2 work, model partially stale, multi-file changes
- Red: V3 work, no model exists, contradicts model claims, architectural change

Display trust tier at the beginning of completion review. Orchestrator uses it to set pacing (light/medium/heavy review). Advisory, not a gate.

**Files:** `cmd/orch/complete_trust.go` (~200 lines), `pkg/trust/score.go` (~150 lines)

---

## Implementation-Ready Output

### File Targets

| File | Action | Size Est |
|------|--------|----------|
| `cmd/orch/orient_cmd.go` | Create | ~200 lines |
| `pkg/orient/orient.go` | Create | ~250 lines |
| `pkg/orient/model_freshness.go` | Create | ~100 lines |
| `cmd/orch/complete_model_impact.go` | Create | ~150 lines |
| `cmd/orch/complete_cmd.go` | Modify (5-10 lines) | Wire in new touchpoint |

### Acceptance Criteria

1. `orch orient` produces structured orientation output with throughput, ready issues, relevant models, and freshness warnings
2. `orch complete` surfaces model-impact lines between probe verdicts and architectural choices
3. Neither command blocks on failure — both are informational/advisory
4. Model summaries at session start are ≤3 models, ≤3 sentences each
5. Model impact at completion is ≤3 lines per affected model
6. Orchestrator skill session start protocol references `orch orient` (separate issue)

### Out of Scope
- Trust calibration tiers (Thread 3 — follow-on)
- `orch stats --json` output format (convenience, can wrap existing text output initially)
- Model freshness threshold tuning (start with 7 days stale, 14 days warning)
- Automated model updates based on agent work (manual orchestrator action)

## Recommendations

⭐ **RECOMMENDED:** Thread 1 (Model Surfacing) first, split into two parallel implementation tracks:

**Track A: `orch orient` command** — New session start orientation
- **Why:** No programmatic hook exists for session start orientation today. This is the biggest gap.
- **Trade-off:** New command means new maintenance. Acceptable because it composites existing data sources.
- **Expected outcome:** Session start becomes "here's what you believe, here's what's ready, here's what's stale" instead of "what should we do?"

**Track B: Model-impact touchpoint** — Completion review enrichment
- **Why:** Completion review has 4 touchpoints but none answer "how does this change your model of X?" across the full corpus.
- **Trade-off:** Slightly longer completion reviews. Acceptable because information is 1-3 lines and skipable.
- **Expected outcome:** Completion review includes "this confirms/extends/contradicts your model of [domain]"

**Alternative: Thread 3 first (Calibrated Trust)**
- **Pros:** Most impactful for completion review efficiency
- **Cons:** Requires trust scoring infrastructure that doesn't exist yet; Thread 1 provides model freshness data that Thread 3 needs
- **When to choose:** If completion review is the primary pain point (currently it's session start orientation)

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the knowledge surfacing gap identified in the original investigation
- Future spawns on knowledge system improvements should respect this architecture

**Suggested blocks keywords:**
- "knowledge surfacing"
- "model surfacing"
- "session orientation"
- "orch orient"
- "completion model impact"
- "trust calibration"
