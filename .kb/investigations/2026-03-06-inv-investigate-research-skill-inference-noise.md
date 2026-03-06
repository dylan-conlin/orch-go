## Summary (D.E.K.N.)

**Delta:** Research skill inference has a 100% false-positive rate in daemon spawns (1604 inferences, 0 daemon-spawned research agents; all 6 actual research spawns were manual), caused by overly broad description keywords, unconditional per-poll logging, and test event contamination.

**Evidence:** Analyzed 29,266 `spawn.skill_inferred` events in events.jsonl. Research: 1604 inferences across 26 unique issues, 0 overlap with 6 actual `session.spawned` research events. Description heuristic keywords ("compare", "evaluate") match "Before/After Comparison" (254x), "skillc compare" (178x). Same issue re-inferred ~240x/hour at 15s poll interval.

**Knowledge:** Inference logging fires on every daemon poll cycle regardless of spawn outcome. The signal-to-noise ratio for all skills is 16:1 (29K inferences vs 1.8K spawns), but research at 267:1 is the worst because description heuristic keywords are common English words.

**Next:** Implement two fixes: (1) Move inference logging from `InferSkillFromIssue()` to the spawn execution path (after pipeline approval), (2) Tighten research description keywords to require multi-word intent patterns. Route through architect for the logging change (cross-cutting concern).

**Authority:** architectural - Logging location change affects daemon event pipeline and post-hoc accuracy analysis workflows

---

# Investigation: Investigate Research Skill Inference Noise

**Question:** Why does the research skill show 1513+ inferences vs only 6 actual spawns, and what causes this noise?

**Started:** 2026-03-06
**Updated:** 2026-03-06
**Owner:** investigation agent (orch-go-8e7bd)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Inference events fire on every daemon poll, not on spawn decisions

**Evidence:** The daemon polls every 15 seconds (`DefaultConfig().PollInterval = 15s`). Each poll cycle calls `d.Once()` → `InferSkillFromIssue(issue)` which unconditionally logs a `spawn.skill_inferred` event to events.jsonl. The spawn pipeline (dedup, rate limit, concurrency) runs AFTER this logging. Issue `orch-go-u5o0` was inferred 178 times over 34 hours (avg 704s gap between inferences) — it sat in the queue as the top candidate but was repeatedly rejected by the pipeline.

**Source:** `pkg/daemon/daemon.go:529` (InferSkillFromIssue call), `pkg/daemon/skill_inference.go:252` (logSkillInference call), `pkg/daemonconfig/config.go:181` (15s default), events.jsonl analysis

**Significance:** This is the multiplier that turns 26 unique issues into 1604 events. The inference event was designed for "post-hoc accuracy analysis" but fires ~240x/hour per candidate. The event is meaningless at this volume because it doesn't distinguish "considered but rejected" from "considered and spawned".

---

### Finding 2: Description heuristic keywords are too broad for research skill

**Evidence:** `InferSkillFromDescription()` uses `strings.Contains(lower, keyword)` with generic English words: "compare", "evaluate", "research", "best practice". These match:
- "Before/After **Comparison**" → 254 false inferences (toolshed-153)
- "skillc **compare**" → 178 false inferences (orch-go-u5o0, a feature request about comparison tooling)
- "**Compare** grid core" → 2 false inferences
- "**evaluate** options" context in various descriptions

Of 1604 research inferences, 1045 (65%) came from description heuristic. The remaining 559 (35%) came from explicit `skill:research` labels (which are correct signal but still suffer from repeated logging).

**Source:** `pkg/daemon/skill_inference.go:148-211` (InferSkillFromDescription), events.jsonl grouped analysis

**Significance:** The description heuristic for research uses vocabulary that appears naturally in non-research issues. "Compare" appears in UI feature descriptions, tooling references, and any before/after analysis. The keyword list conflates research-as-topic with research-as-intent.

---

### Finding 3: Test events contaminate production event log

**Evidence:** 47% of all 29,266 inference events come from non-production issue IDs:
- `test-*` IDs: 3,351 (11%) — from unit tests calling `InferSkillFromIssue()`
- `proj-*` IDs: 9,753 (33%) — from test fixtures or manual testing
- Empty IDs: 584 (2%)

For research specifically: test-003 (270x), test-005 (270x), test-006 (270x) = 810 test events (50% of research inferences).

The `logSkillInference()` function at `skill_inference.go:287` creates a real `events.NewDefaultLogger()` which writes to the production `~/.orch/events.jsonl`. Unit tests exercise this same code path.

**Source:** events.jsonl analysis, `pkg/daemon/skill_inference.go:286-299`

**Significance:** Any accuracy analysis of inference quality is impossible without first filtering test events. The test contamination makes the signal-to-noise ratio appear even worse than it actually is.

---

### Finding 4: Zero overlap between daemon research inferences and actual research spawns

**Evidence:** Cross-referencing all `spawn.skill_inferred` events (inferred_skill="research") with `session.spawned` events (skill="research"):
- 26 unique issue IDs inferred as research
- 6 actual research spawns (pw-8951, orch-go-untracked-*, badge-tracker-7, toolshed-173)
- **0 overlap** — none of the 6 spawned research agents came from daemon inference

All 6 research spawns were manual (`orch spawn --skill research` or explicit skill label). The daemon's research inference never led to a single spawn.

**Source:** events.jsonl correlation analysis

**Significance:** The daemon research inference is 100% noise. It produces signal (events) without ever producing value (spawns). This is the clearest evidence that the description heuristic for research is fundamentally broken.

---

## Synthesis

**Key Insights:**

1. **Logging placement is the core multiplier** - Inference logging at the point of inference (every poll) rather than at the point of decision (spawn) creates ~240x amplification per candidate per hour. Moving logging to after pipeline approval would reduce total events from ~29K to ~1.8K.

2. **Description heuristic fails for research because research vocabulary is ambient** - Words like "compare" and "evaluate" are used in all kinds of issues (UI features, tooling, comparisons). Research intent requires multi-word patterns like "evaluate alternatives", "compare approaches", "which should we use", not bare keywords.

3. **The inference/spawn gap reveals daemon queue dynamics** - Issues sit in the queue being re-inferred every 15 seconds because they're the top candidate but can't be spawned (at capacity, rate limited, dedup rejected). The events.jsonl faithfully records every consideration, making it a log of queue pressure rather than decision quality.

**Answer to Investigation Question:**

The 1604:6 ratio is caused by three compounding factors: (1) inference events log on every 15-second poll cycle, not on spawn decisions, creating ~240x amplification; (2) the description heuristic keywords for research ("compare", "evaluate") are common English words that match non-research issues like "Before/After Comparison"; (3) test events contaminate the production log. The actual false-positive count (unique issues) is 26 issues inferred as research vs 0 daemon-spawned research agents — 100% false positive rate for daemon research inference. All 6 real research spawns were manual.

---

## Structured Uncertainty

**What's tested:**

- ✅ 29,266 total inferences vs 1,836 spawns in events.jsonl (verified: direct grep + python analysis)
- ✅ 1,604 research inferences from 26 unique issues, 0 overlap with 6 actual research spawns (verified: cross-reference analysis)
- ✅ Description heuristic matches "Comparison", "compare" in non-research titles (verified: grouped title analysis)
- ✅ Daemon polls every 15s, orch-go-u5o0 inferred 178x over 34h (verified: timestamp analysis)
- ✅ 47% of events from test/proj/empty IDs (verified: prefix analysis)

**What's untested:**

- ⚠️ Moving logging to spawn path would not break accuracy analysis tooling (not tested — need to check if any tool depends on pre-spawn events)
- ⚠️ Tightened keywords would correctly identify actual research issues (not tested with real research issue descriptions)
- ⚠️ The 15s poll interval is the only/main source of amplification (Preview and swarm also call InferSkillFromIssue)

**What would change this:**

- If accuracy analysis tools depend on seeing inferred-but-not-spawned events, moving logging would break a workflow
- If there are research issues where only the single keyword "compare" in description is the correct signal, tightening keywords would create false negatives

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Move inference logging to spawn path | architectural | Cross-cutting change affecting event pipeline, daemon loop, and accuracy analysis |
| Tighten research description keywords | implementation | Isolated keyword list change in InferSkillFromDescription |
| Separate test from production events | implementation | Test infrastructure concern, no production impact |

### Recommended Approach: Two-Phase Fix

**Phase 1 (implementation): Tighten research keywords + add test isolation**

Fix the most obvious noise source without changing the event architecture:

1. Replace bare keywords with multi-word patterns in `InferSkillFromDescription()`:
   - "compare" → "compare options", "compare approaches", "comparison of alternatives"
   - "evaluate" → "evaluate options", "evaluate alternatives"
   - Keep "research" only with intent context: "research best practice", "research approach"
2. Mock the logger in `InferSkillFromIssue` tests to prevent production log pollution

**Phase 2 (architectural): Move inference logging to spawn decision point**

Decouple inference (which happens every poll) from logging (which should happen at decision time):

1. Remove `logSkillInference()` call from `InferSkillFromIssue()`
2. Add logging in the spawn execution path (after pipeline approval), recording both the inference method AND the spawn decision
3. Optionally add a separate `spawn.skill_inferred_and_spawned` event type for clean analysis

**Why this approach:**
- Phase 1 is a quick win that eliminates the worst false positives (no architecture change)
- Phase 2 addresses the structural cause (logging at wrong point in pipeline)
- Either phase independently reduces noise; together they fix it

**Trade-offs accepted:**
- Phase 1 may miss some legitimate research signals (substring matching was overly generous but occasionally correct)
- Phase 2 loses visibility into "what the daemon considered but didn't spawn" — this data has no known consumer

### Alternative Approaches Considered

**Option B: Add dedup/rate-limiting to inference logging**
- **Pros:** Preserves per-poll inference visibility, reduces volume
- **Cons:** Still logs events at the wrong point; adds complexity to logging layer for data that has no consumer
- **When to use instead:** If post-hoc analysis of "considered but rejected" inference is needed

**Option C: Add `spawned: true/false` field to existing event**
- **Pros:** Preserves all data, enables filtering
- **Cons:** Still 29K events, still noisy; analysis requires filtering on every query
- **When to use instead:** If preserving full inference telemetry is a requirement

---

### Implementation Details

**What to implement first:**
- Tighten research keywords in `InferSkillFromDescription()` (5-minute change, immediate noise reduction)
- Mock logger in skill_inference_test.go

**Things to watch out for:**
- ⚠️ `InferSkillFromDescription` is a fallback — it only fires when labels and title don't match. Tightening keywords affects a narrow slice.
- ⚠️ The `swarm.go` command also calls `InferSkillFromIssue` — any logging change needs to account for that path.

**Success criteria:**
- ✅ Research false positives in events.jsonl drops to near-zero for non-research issues
- ✅ No test-* or proj-* events in production events.jsonl
- ✅ Inference-to-spawn ratio drops below 5:1 (from current 16:1)

---

## References

**Files Examined:**
- `pkg/daemon/skill_inference.go` - Core inference logic, all heuristic functions, logging call
- `pkg/daemon/daemon.go:460-660` - Once/OnceExcluding/processIssue showing where InferSkillFromIssue is called
- `pkg/daemon/preview.go` - Preview also calls InferSkillFromIssue
- `pkg/daemonconfig/config.go:178-181` - 15s default poll interval
- `pkg/orch/spawn_inference.go` - Separate inference for `orch work` command (does NOT log events)
- `pkg/events/logger.go:536-548` - SkillInferredData type and LogSkillInferred function
- `cmd/orch/daemon.go:380-630` - Daemon main loop

**Commands Run:**
```bash
# Count events
grep -c 'spawn.skill_inferred' ~/.orch/events.jsonl  # → 29266
grep -c 'session.spawned' ~/.orch/events.jsonl  # → 1836

# Research breakdown
# Python analysis of events.jsonl: research inferences by issue, method, temporal distribution
# Cross-reference between inferred and spawned events

# Temporal analysis of orch-go-u5o0
# 178 research inferences over 34h, avg gap 704s
```

**Related Artifacts:**
- **Events log:** `~/.orch/events.jsonl` - Source data for all analysis
- **Investigation origin:** Feature that added inference tracking: `orch-go-0pr`

---

## Investigation History

**2026-03-06:** Investigation started
- Initial question: Why does research show 1513+ inferences vs 6 actual spawns?
- Context: Post-hoc analysis of inference accuracy revealed extreme noise

**2026-03-06:** Core findings complete
- Identified 4 compounding causes: poll-cycle logging, broad keywords, test contamination, zero overlap
- Quantified: 29,266 total inferences, 1,604 research-specific, 100% false positive rate for daemon research
- Recommended two-phase fix: tighten keywords (implementation) + move logging location (architectural)

**2026-03-06:** Investigation completed
- Status: Complete
- Key outcome: Research inference noise is caused by logging at poll-time (not spawn-time) combined with overly broad description keywords. Zero daemon research spawns have ever resulted from inference — all 6 were manual.
