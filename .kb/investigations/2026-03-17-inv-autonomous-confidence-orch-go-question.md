<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The daemon has 5 autonomous decision points where confidence exceeds grounding, with the trigger system being the worst offender — budget limits per-scan but not cumulative output, and no detector has outcome feedback.

**Evidence:** Code review of trigger.go, trigger_detectors_phase2.go, skill_inference.go, allocation.go, ooda.go. 47 false positive investigation files from hotspot acceleration detector on 2026-03-17 confirm cumulative budget failure. SkillPerformanceDriftDetector hardcodes PreviousRate=0.7 placeholder. Skill inference logs events but never reads them back.

**Knowledge:** Autonomous systems need outcome-grounded confidence, not just input-gated thresholds. The pattern across all 5 gaps is the same: the system observes inputs, applies heuristics, produces outputs with confidence, but never measures whether those outputs led to useful results. Budget mechanisms constrain rate without constraining quality.

**Next:** Architect follow-up to design outcome feedback loops for trigger detectors and skill inference. Implementation priority: per-detector outcome tracking (highest harm potential), then skill inference accuracy measurement.

**Authority:** architectural — Cross-component feedback loops touching daemon, trigger, skill inference, and learning subsystems

---

# Investigation: Autonomous Confidence in orch-go

**Question:** Where does the daemon/trigger system produce confident outputs without adequate grounding? What feedback loops are missing?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-17-inv-hotspot-acceleration-* (47 files) | extends | yes — confirmed all are false positives of a single detector | - |
| 94ac6e601 inv: e2e_lifecycle_test.go hotspot is false positive | confirms | yes — "born large, not accreting" matches our analysis | - |

---

## Findings

### Finding 1: Trigger Budget Prevents Instantaneous Bloat but Not Cumulative Bloat

**Evidence:** `TriggerBudget.CanCreate()` (trigger.go:53-58) checks `currentOpen < MaxOpen` (default 10). Budget counts only *currently open* issues (`CountOpenTriggerIssues` calls `ListIssuesWithLabel(TriggerLabel)` which returns open/in_progress only). Within a single scan, `currentOpen++` tracks consumption. But across scans: as issues get spawned, investigated, and closed, budget frees up. The dedup gate (`HasTriggerIssue`) prevents re-creating the same detector+key combo (checks all issues including closed), but each unique file path or model slug is a unique key.

With the hotspot acceleration detector returning ~50 unique files, and the budget allowing 10 per hourly scan, all 47 investigation files were created across ~5 scan cycles over 5 hours.

**Source:** trigger.go:109-205, trigger_service.go:20-26, trigger.go:45-58

**Significance:** The budget mechanism is per-scan rate limiting, not quality gating. It prevents instantaneous queue saturation but not the accumulation of low-quality issues over time. There is no mechanism to ask "are this detector's issues producing useful outcomes?" The budget is input-side only — it constrains how fast the system creates issues, not whether those issues are worth creating.

---

### Finding 2: Hotspot Acceleration Detector Cannot Distinguish "Born Large" from "Accreting"

**Evidence:** `defaultHotspotAccelerationSource.ListFastGrowingFiles()` (trigger_detectors_phase2.go:370-422) uses `git diff --numstat baseCommit HEAD` comparing against a commit ~30 days ago. For a file created within that 30-day window at >200 lines, the entire file shows as "net growth." The detector computes `historicalSize = currentSize - netGrowth`, which correctly yields 0 for new files, but still flags them because `netGrowth >= threshold`. The `minAccelerationSize = 500` filter only excludes small files.

The detector was designed for accretion (existing files getting bigger) but its measurement method conflates creation with growth. This is the root cause of the ~50 false positive investigations — files like `compliance_test.go` (born via extraction refactor) and `e2e_lifecycle_test.go` (born large) were flagged.

**Source:** trigger_detectors_phase2.go:370-422, trigger_detectors_phase2.go:87-123, git commits 94ac6e601, 840618294, b5c8c14c3

**Significance:** The detector's evidence threshold is based purely on a numerical signal (lines changed) without semantic context (was this file created or was it growing?). The fix that was applied — disabling the detector entirely (trigger_detectors.go:34) — is appropriate as a safety measure but doesn't solve the underlying confidence gap pattern.

---

### Finding 3: Skill Inference Logs But Never Learns

**Evidence:** `InferSkillFromIssue()` (skill_inference.go:221-258) logs a `spawn.skill_inferred` event to events.jsonl with the inference method used (label, title, description, type fallback). But there is no consumer of this event that measures inference accuracy. The comment says "for post-hoc accuracy analysis" but no analysis function exists.

The inference chain (labels > title > description > type) uses keyword matching (`InferSkillFromDescription` checks for "audit", "analyze", "investigate", etc.) with no weighting or confidence scoring. A description containing "how does X work?" could get mapped to `investigation` even if the actual appropriate skill is `architect`. There's no feedback from skill-mismatch outcomes (e.g., investigation that should have been architect → abandoned → no signal back to inference).

**Source:** skill_inference.go:150-213 (description heuristics), skill_inference.go:253-257 (event logging), no consumer found via grep

**Significance:** Skill inference is a critical routing decision — it determines which skill prompt an agent receives and which model it uses. Wrong inference wastes expensive opus tokens on work that needed sonnet, or sends an implementation task to the wrong skill template. The system logs the *input* to this decision (which heuristic matched) but never measures the *output* (did the inferred skill lead to success?).

---

### Finding 4: Skill Performance Drift Detector Uses Fabricated "Previous Rate"

**Evidence:** `defaultSkillPerformanceDriftSource.ListDriftedSkills()` (trigger_detectors_phase2.go:560-587) hardcodes `PreviousRate: 0.7` as a placeholder with the comment "Placeholder — true windowed rate needs time-series." The detector claims to detect a *drop* from a previous rate to a current rate, but the previous rate is fabricated. This means any skill below 50% success rate (the currentThreshold default) gets flagged as "drifted from 70% to X%" — even if it was never at 70%.

**Source:** trigger_detectors_phase2.go:581 (`PreviousRate: 0.7, // Placeholder`)

**Significance:** This is a confidence assertion masquerading as measurement. The trigger description says "success rate dropped from 70% to 30%" but the 70% figure is invented. The system presents fabricated data to the human/daemon as if it were observed. Any issues created by this detector carry false confidence in the claimed drift magnitude.

---

### Finding 5: Allocation Scoring Has Reasonable Feedback but Narrow Scope

**Evidence:** `ScoreIssue()` (allocation.go:45-69) blends issue priority with skill success rate from the learning store. The learning store is refreshed periodically from events.jsonl (`RunPeriodicLearningRefresh` in periodic_learning.go:23-58). `BlendedSuccessRate()` correctly handles small sample sizes by blending toward a 0.5 default.

This is the *only* feedback loop in the autonomous pipeline that connects outcomes to decisions. But it only affects issue *ordering* (within ±20% of base priority). It cannot prevent spawning a bad issue, only deprioritize it relative to others. And the feedback is at the skill level (e.g., "investigation has 45% success rate"), not at the detector or issue-source level.

**Source:** allocation.go:38-69, allocation.go:85-101 (blending), periodic_learning.go:18-58

**Significance:** The learning store proves the codebase has the infrastructure for outcome feedback. The gap is that feedback only reaches allocation scoring, not trigger detection, skill inference, or spawn routing. The skill success rate tells you "investigation has low success" but not "investigation issues created by the hotspot detector have low success."

---

### Finding 6: Model Contradictions Detector Has Fragile Evidence Heuristic

**Evidence:** `defaultModelContradictionsSource.ListUnresolvedContradictions()` (trigger_detectors_phase2.go:243-310) scans probe files for the string "contradict" (case-insensitive) and checks if the model's file mtime is before the probe's date. This produces false positives when:
1. A probe file *discusses* contradiction without *finding* one (e.g., "this does not contradict the model")
2. The model file was touched for unrelated reasons (git checkout, editor save) after the probe, marking it as "updated"
3. The probe mentions contradiction in a section that says "Model Impact: confirms" (the detector doesn't parse Model Impact)

**Source:** trigger_detectors_phase2.go:289-290 (`strings.Contains(contentLower, "contradict")`)

**Significance:** Substring matching for semantic classification. The detector produces P2 issues (high priority) based on a string match that doesn't account for negation, context, or the probe's own conclusion about its relationship to the model.

---

### Finding 7: Knowledge Decay Detector Flags Never-Probed Models at Priority 4

**Evidence:** `defaultKnowledgeDecaySource.ListDecayedModels()` (trigger_detectors_phase2.go:484-555) flags models with no probes directory using `DaysSinceProbe: 999` (sentinel value). Every model without a probes/ directory gets flagged, even if it was just created or is a stub model. The detector creates P4 issues — low priority, but still consumes budget and creates noise.

**Source:** trigger_detectors_phase2.go:515-520

**Significance:** Low severity individually, but contributes to budget consumption. A newly created model would immediately be flagged for "knowledge decay" on the next scan cycle. The detector has no concept of model maturity or importance.

---

## Synthesis

**Key Insights:**

1. **Rate ≠ Quality**: The budget mechanism constrains the *rate* of autonomous issue creation but not the *quality*. All detectors can produce 10 issues per scan cycle regardless of whether their previous issues led to useful outcomes. The system optimizes for "don't flood the queue right now" but not "is this detector producing value?"

2. **Input Heuristics Without Output Verification**: Every autonomous decision point (trigger detection, skill inference, priority scoring, model routing) uses input-side heuristics — keyword matching, line count thresholds, type mappings. Only allocation scoring has any output-side feedback, and even that operates at a coarse skill-level granularity, not at the decision-source level.

3. **Fabricated Confidence**: The SkillPerformanceDriftDetector and ModelContradictionsDetector both produce confident-sounding outputs from weak evidence. The drift detector invents a "previous rate" of 70%. The contradictions detector treats substring presence as semantic classification. Both create issues with specific claims that aren't grounded in what was actually measured.

**Answer to Investigation Question:**

The daemon produces autonomous confident outputs without adequate grounding at 5 decision points, ranked by harm potential:

| # | Decision Point | Confidence Produced | Grounding Present | Grounding Missing | Harm Potential |
|---|---|----|---|---|---|
| 1 | **Trigger budget** | "Safe to create issues at this rate" | Per-scan cap (10 open) | Cumulative quality check; per-detector outcome tracking | **Critical** — caused 47 false positive investigations |
| 2 | **Hotspot acceleration detection** | "This file is accreting dangerously" | Net line growth over 30d | Distinction between file creation and file growth | **High** — now disabled, but pattern could recur in other detectors |
| 3 | **Skill inference from description** | "This issue needs investigation skill" | Keyword presence matching | Accuracy measurement; outcome feedback; confidence scoring | **Medium** — wrong skill wastes opus tokens, may produce poor results |
| 4 | **Skill performance drift** | "Skill X dropped from 70% to Y%" | Current rate from events.jsonl | Actual historical rate (hardcoded 0.7 placeholder) | **Medium** — fabricated data in issue descriptions |
| 5 | **Model contradiction detection** | "Probe contradicts model" | Substring "contradict" found | Negation awareness; probe conclusion parsing | **Low** — P2 priority but rare false positive rate in practice |

---

## Structured Uncertainty

**What's tested:**

- ✅ Budget enforcement works per-scan (verified: TestDaemon_RunPeriodicTriggerScan_BudgetEnforced passes)
- ✅ Dedup prevents re-creating same detector+key combo (verified: TestDaemon_RunPeriodicTriggerScan_DedupSkips passes)
- ✅ Allocation scoring correctly blends small samples with default (verified: TestBlendedSuccessRate passes)
- ✅ Hotspot acceleration detector is currently disabled (verified: trigger_detectors.go:34 comment)
- ✅ 47 false positive files from hotspot acceleration all from 2026-03-17 (verified: grep results)
- ✅ PreviousRate=0.7 is hardcoded placeholder (verified: trigger_detectors_phase2.go:581)

**What's untested:**

- ⚠️ How many scan cycles it actually took to create 47 issues (would need daemon.log or events.jsonl analysis)
- ⚠️ Whether model contradictions detector has false positive rate in practice (would need to check created issues)
- ⚠️ Skill inference accuracy rate across all daemon spawns (would need post-hoc analysis of events.jsonl)

**What would change this:**

- Finding 1 would be wrong if the budget was actually per-detector (but code confirms it's global)
- Finding 3 would be less important if skill inference accuracy is empirically >90% (needs measurement)
- Finding 4 would be irrelevant if the drift detector is also disabled (it's currently active per trigger_detectors.go:36)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Per-detector outcome tracking | architectural | Cross-component: touches trigger system, events, and daemon learning |
| Skill inference accuracy measurement | architectural | Cross-component: touches events, daemon, and completion pipeline |
| Fix PreviousRate placeholder | implementation | Single-file fix within existing events infrastructure |
| Improve contradiction detection | implementation | Single-detector logic change |

### Recommended Approach ⭐

**Outcome-Grounded Trigger Budget** — Add per-detector outcome tracking so the trigger system can learn which detectors produce useful issues.

**Why this approach:**
- Addresses the root cause (rate limiting without quality gating)
- Uses existing infrastructure (events.jsonl already tracks completions and abandonments)
- The hotspot detector incident proves the harm potential of ungrounded detection

**Trade-offs accepted:**
- More complex budget logic (per-detector state vs single counter)
- Requires definition of "useful outcome" per detector type

**Implementation sequence:**
1. Track detector source in trigger-created issues (label or metadata) — foundational data collection
2. Add `trigger.outcome` event to completion pipeline when trigger-created issues close
3. Compute per-detector success rate in learning refresh; add detector-level budget throttling
4. Fix PreviousRate placeholder in SkillPerformanceDriftDetector (quick win, low risk)
5. Add negation awareness to ModelContradictionsDetector (parse "does not contradict", "confirms")

### Alternative Approaches Considered

**Option B: Disable all trigger detectors**
- **Pros:** Zero false positive risk
- **Cons:** Loses genuine detection value (recurring bugs, investigation orphans)
- **When to use instead:** If detector maintenance cost exceeds detection value

**Option C: Mandatory human approval for all trigger issues**
- **Pros:** Perfect accuracy via human gate
- **Cons:** Defeats purpose of autonomous overnight operation
- **When to use instead:** During initial rollout of new detectors

**Rationale for recommendation:** The learning infrastructure already exists (events.jsonl + LearningStore). Extending it to trigger detectors follows existing patterns and doesn't require new subsystems. The hotspot incident proves that input-only gating is insufficient.

---

### Implementation Details

**What to implement first:**
- Per-detector outcome tracking via events (highest harm potential addressed)
- PreviousRate fix (quick win, eliminates fabricated data)

**Things to watch out for:**
- ⚠️ Defining "useful outcome" for different issue types (investigation completion ≠ bug fix completion)
- ⚠️ Cold-start problem: new detectors have no outcome data, need default budget behavior
- ⚠️ Detector outcome tracking must handle issues that get reclassified or merged

**Areas needing further investigation:**
- Post-hoc analysis of events.jsonl to measure actual skill inference accuracy
- Whether the existing active detectors (recurring_bugs, investigation_orphans, thread_staleness, model_contradictions, knowledge_decay, skill_performance_drift) have meaningful false positive rates
- Whether allocation scoring's ±20% band is large enough to meaningfully affect spawn order

**Success criteria:**
- ✅ Per-detector success rate visible in daemon status
- ✅ Detectors with <30% outcome success rate automatically throttled
- ✅ No hardcoded placeholder values in detector output (PreviousRate fixed)
- ✅ Trigger budget respects both per-scan and cumulative constraints

---

## References

**Files Examined:**
- `pkg/daemon/trigger.go` — Trigger budget, scan orchestrator, TriggerScanService interface
- `pkg/daemon/trigger_service.go` — Default TriggerScanService (budget counting, dedup, issue creation)
- `pkg/daemon/trigger_detectors.go` — Phase 1 detectors and DefaultTriggerDetectors()
- `pkg/daemon/trigger_detectors_phase2.go` — Phase 2 detectors (hotspot, model contradictions, knowledge decay, skill drift)
- `pkg/daemon/trigger_detectors_phase2_test.go` — Detector tests including churn false positive elimination
- `pkg/daemon/ooda.go` — OODA loop (Sense/Orient/Decide/Act)
- `pkg/daemon/skill_inference.go` — Full skill inference chain
- `pkg/daemon/allocation.go` — Skill-aware allocation scoring with learning feedback
- `pkg/daemon/coordination.go` — Routing and architect escalation
- `pkg/daemon/periodic_learning.go` — Learning store refresh from events.jsonl
- `cmd/orch/daemon_loop.go` — Daemon main loop, spawn cycle, status reporting

**Commands Run:**
```bash
# Run trigger detector tests
go test ./pkg/daemon/ -run "TestHotspotAcceleration|TestModelContradictions|TestKnowledgeDecay|TestSkillPerformanceDrift|TestDaemon_RunPeriodicTriggerScan" -v

# Run allocation and skill inference tests
go test ./pkg/daemon/ -run "TestBlendedSuccessRate|TestScoreIssue|TestInferSkill" -v
```

**Related Artifacts:**
- **Decision:** bb3187123 fix: disable hotspot acceleration detector from trigger scan rotation
- **Investigation:** 94ac6e601 inv: e2e_lifecycle_test.go hotspot is false positive — born large, not accreting
- **Investigation:** 840618294 inv: findingdedup_test.go hotspot is false positive — born via single-commit creation
- **Investigation:** b5c8c14c3 inv: compliance_test.go hotspot is false positive — born via extraction refactor
