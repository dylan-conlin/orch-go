## Summary (D.E.K.N.)

**Delta:** Verification theater in orch-go follows a consistent pattern: tests verify that isolated functions return correct values with mocked inputs, but never verify that the integrated system produces correct outcomes. This gap allowed the hotspot acceleration detector to be "fixed" twice (both times with passing tests) while the daemon continued creating false positive issues — 97 agent-hours wasted.

**Evidence:** (1) All 8 trigger detector tests use mocks that hardcode the detector's source data — the test proves `Detect()` maps struct fields correctly, not that the detector identifies real hotspots. (2) All spawn gate tests verify gates return nil error, but every gate is advisory (never blocks) — the tests prove gates don't crash, not that they prevent bad spawns. (3) Accretion gate had 100% bypass rate over 2 weeks with all tests passing. (4) Completion verification checks `info.Size() > 0` for SYNTHESIS.md — literally any non-empty file passes. (5) Test evidence gate uses regex pattern matching on beads comments — agents can satisfy it by pasting test output without running tests.

**Knowledge:** The verification system has three layers of theater: (a) mock-isolated unit tests that prove function mechanics but not system behavior, (b) advisory gates that signal but never block (by design), and (c) artifact-existence checks that verify ceremony completion, not value creation.

**Next:** Architectural decision needed — this is structural, not a bug to fix. The core question is whether to add integration tests that verify end-to-end daemon behavior, or accept the current advisory model and invest in post-hoc measurement instead.

**Authority:** strategic — Deciding what "verified" means for an autonomous agent system is a value judgment about where to invest effort.

---

# Investigation: Verification Theater in orch-go

**Question:** Where do tests, gates, and checks create an appearance of quality without actually ensuring it?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** orch-go investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md | extends | yes | None — confirms 100% bypass rate |
| orch-go-9aicv (hotspot net growth fix) | deepens | yes | Tests passed but daemon still created false positives |
| orch-go-6jk8g (hotspot detector disable) | deepens | yes | Final resolution was to disable detector entirely, not fix it |

---

## Findings

### Finding 1: Trigger Detector Tests Verify Mock Plumbing, Not Detection Accuracy

**Evidence:** All 8 detector tests in `trigger_detectors_phase2_test.go` use mock source objects that hardcode return values. For example, `TestHotspotAccelerationDetector_FindsFastGrowing` (line 111-141) injects a mock that returns `{Path: "pkg/daemon/ooda.go", NetGrowth: 350}` and verifies the detector maps it to a `TriggerSuggestion` with the right fields. The test proves the Detect() function's field-mapping logic works.

What it does NOT test: whether the real `defaultHotspotAccelerationSource` (in `trigger_detectors_phase2.go`) correctly identifies fast-growing files from actual git history. The real source runs `git diff --numstat` and parses output — the complexity and bugs live there, not in the struct mapping.

The churn false-positive test (`TestHotspotAccelerationDetector_ChurnNotFlaggedAsGrowth`, line 392-413) is particularly theatrical: it injects a mock that returns `nil` (no files) and asserts 0 suggestions. The test comment says "Net growth approach: stats_cmd.go went from 900 to 302 lines... not returned by source" — but the test doesn't verify the source actually does this filtering. It verifies that when the source returns nothing, the detector returns nothing.

**Source:** `pkg/daemon/trigger_detectors_phase2_test.go:111-141`, `pkg/daemon/trigger_detectors_phase2_test.go:392-413`

**Significance:** This is the exact gap that caused the hotspot detector to be "fixed" twice. Fix #1 (orch-go-mbhzv) and fix #2 (orch-go-9aicv) both passed all unit tests because they modified the source implementation, but the tests don't exercise the source. The detector was eventually disabled entirely (orch-go-6jk8g) after 97 agent-hours of false positive investigations.

---

### Finding 2: Every Spawn Gate Is Advisory — Tests Verify That Nothing Blocks

**Evidence:** All spawn gates in `pkg/spawn/gates/` are warning-only:

- `CheckHotspot()` (hotspot.go:41): Comment says "Advisory only — emits warnings and events but never blocks." Test `TestCheckHotspot_Critical_NeverBlocks` (hotspot_test.go:65-85) verifies `err == nil` for critical hotspots with blocking skills.
- `CheckAgreements()` (agreements.go:70): Comment says "This is a WARNING-ONLY gate (Phase 3) — it never blocks spawn." Test `TestCheckAgreements_WithErrorFailures_StillWarning` (agreements_test.go:88-115) verifies even error-severity failures don't block.
- `CheckOpenQuestions()` (question.go:58): Comment says "This is a WARNING-ONLY gate — it never blocks spawn." All tests verify nil error return.

The gate tests are honest about what they verify: "does the gate crash?" and "does it return the right data structure?" But the test names and structure create the appearance of testing that gates enforce quality. `TestCheckHotspot_Critical_NeverBlocks` sounds like it's testing an important safety property, but it's testing that the gate is deliberately toothless.

**Source:** `pkg/spawn/gates/hotspot.go:39-41`, `pkg/spawn/gates/hotspot_test.go:65-85`, `pkg/spawn/gates/agreements.go:67-69`, `pkg/spawn/gates/agreements_test.go:88-115`

**Significance:** The accretion gate decision document (2026-03-17) provides the measured evidence: 55 gate firings, 2 blocks, both bypassed instantly = 100% bypass rate. Tests "passed" throughout this entire period. The tests were never designed to detect that the gates weren't enforcing anything — they verify the mechanism works, not that it has an effect.

---

### Finding 3: Completion Verification Checks Artifact Existence, Not Value

**Evidence:** The verification pipeline (`pkg/verify/check.go`) runs up to 14 gates organized by level (V0-V3), but the foundational checks are existence checks:

- **SYNTHESIS.md gate** (check.go:52-65): `VerifySynthesis()` checks `info.Size() > 0`. A SYNTHESIS.md containing "placeholder" or "TODO" passes. There is no content quality check.
- **Phase Complete gate** (check.go:523-550): Checks if a beads comment contains "Phase: Complete" as a string. An agent can report "Phase: Complete - nothing done" and pass.
- **Session Handoff validation** (check.go:99-138): Checks that TLDR has ≥20 characters and Outcome is one of {success, partial, blocked, failed}. An agent writing "This session was a complete waste of time" (42 chars) with "Outcome: success" passes.
- **Test evidence gate** (test_evidence.go:89-120): Uses regex matching on beads comment text. An agent that pastes `ok  pkg/test  0.123s` into a comment — without running any test — satisfies this gate.

**Source:** `pkg/verify/check.go:52-65`, `pkg/verify/check.go:523-550`, `pkg/verify/check.go:99-138`, `pkg/verify/test_evidence.go:89-120`

**Significance:** The verification system verifies ceremony completion, not value creation. "Did the agent produce the right artifacts?" is answerable by code. "Did the agent's work produce value?" is not — but the verification system's architecture implies it answers the second question when it only answers the first.

---

### Finding 4: The Skip System Makes All Gates Formally Bypassable

**Evidence:** `pkg/verify/skip.go` defines `SkipConfig` with 14 boolean flags, one for each gate. Every gate in the system can be skipped with `--skip-{gate} --skip-reason "..."`. The minimum reason length is 10 characters. An agent can skip any gate by providing a 10-character string.

Additionally, the verification level system (`pkg/verify/level.go`) means most gates don't run at all for lower-tier spawns:
- V0: Only Phase Complete
- V1: Adds 6 artifact gates
- V2: Adds synthesis, test evidence, git diff, build, accretion
- V3: Adds visual verification, explain-back

A V0 spawn is verified only by a string match on "Phase: Complete" in beads comments.

**Source:** `pkg/verify/skip.go:1-128`, `pkg/verify/level.go:1-88`

**Significance:** The skip system and level system together mean that "verification passed" covers a spectrum from "agent typed a Phase: Complete comment" (V0) to "build passed, tests ran, synthesis exists, accretion checked" (V2+). But the completion event just records `Passed: true` — the consumer has no visibility into what was actually verified.

---

### Finding 5: The Trigger-to-Daemon Integration Gap

**Evidence:** The daemon wires detectors via `DefaultTriggerDetectors()` in `trigger_detectors.go:27-38`, which creates production detector instances with real sources. The OODA loop calls `RunPeriodicTriggerScan()` in `daemon_periodic.go:181`. There are zero integration tests that exercise this path end-to-end.

The unit tests verify:
- Detectors map source output to suggestions correctly (mocked source)
- `RunPeriodicTriggerScan` processes suggestions through budget and dedup gates (mocked service)
- Budget enforcement works (mocked counts)
- Scheduler gating works (mocked time)

What's never tested:
- Real source implementations produce correct data from real filesystem/git state
- Real `TriggerScanService` creates correct beads issues
- A file actually growing rapidly triggers an issue creation
- A contradiction probe actually triggers an issue creation
- The complete path from git state → source → detector → orchestrator → beads

**Source:** `pkg/daemon/trigger_detectors.go:27-38`, `cmd/orch/daemon_periodic.go:181`, `pkg/daemon/trigger_test.go`, `pkg/daemon/trigger_detectors_phase2_test.go`

**Significance:** This is the gap that made the hotspot detector fixable-on-paper but broken-in-practice. Each component was tested in isolation with mocks, so tests passed. But the real bug was in how the source implementation interacted with actual git history (extraction churn creating false positives). No test exercises that interaction.

---

## Synthesis

**Key Insights:**

1. **Mock-Boundary Theater** — When every test replaces the I/O boundary with a mock, the tests verify the code between boundaries (pure logic, struct mapping), but the bugs live AT the boundaries (git parsing, filesystem scanning, beads API interaction). The hotspot detector's pure logic was always correct — it was the git diff parsing that produced false positives.

2. **Advisory-Gate Paradox** — The system evolved every gate from blocking to advisory based on measured evidence (100% bypass rate). This is rational. But it means the entire gate layer now produces signals that nothing acts on except the daemon (via events). The tests verify gates emit correct signals, but don't verify anything consumes those signals effectively. The gates are verified plumbing with no verified consumer.

3. **Ceremony-as-Proxy** — Completion verification uses artifact existence as a proxy for work quality. This worked early (when the question was "did the agent engage at all?") but becomes theater at scale. A system with 14 verification gates that checks `file.Size() > 0` and `string.Contains("Phase: Complete")` has the ceremony of thorough verification without the substance.

**Answer to Investigation Question:**

Verification theater in orch-go manifests in three interconnected layers:

1. **Unit tests that verify mocks, not behavior:** Tests pass regardless of whether the system actually works because bugs live in the I/O boundaries that tests replace with mocks. This is the direct cause of the 97-agent-hour hotspot false positive waste.

2. **Gates that signal but never block:** Every spawn gate is advisory-only, verified by tests that confirm gates don't block. The tests are accurate — the gates genuinely don't block — but the verification creates false confidence that quality is being enforced.

3. **Artifact checks that verify ceremony, not value:** Completion verification checks if outputs exist and contain minimum text, not whether the outputs are correct or useful. This is the deepest form of theater because it's hardest to fix — verifying output quality requires judgment, not code.

---

## Structured Uncertainty

**What's tested:**

- ✅ All spawn gates are advisory — confirmed by reading every gate's source code and test (hotspot.go, agreements.go, question.go)
- ✅ Accretion gate had 100% bypass rate — confirmed by `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md`
- ✅ Hotspot detector was disabled after tests passed twice — confirmed by git log showing 4 fix commits followed by disable commit
- ✅ SYNTHESIS.md check is `info.Size() > 0` — confirmed by reading `check.go:52-65`
- ✅ Test evidence gate uses regex on comments — confirmed by reading `test_evidence.go:89-120`

**What's untested:**

- ⚠️ Whether the other active detectors (recurring_bugs, investigation_orphans, thread_staleness, model_contradictions, knowledge_decay, skill_performance_drift) also have false positive issues — their real sources haven't been audited
- ⚠️ Whether the advisory gate signals actually drive daemon behavior in practice (the accretion decision says they do for extraction cascades, but no measurement for other gates)
- ⚠️ Whether the test evidence regex can be gamed by agents pasting test output without running tests — plausible but not observed in practice

**What would change this:**

- If integration tests existed that exercise the real source implementations against real filesystem/git state, the mock-boundary theater finding would be partially mitigated
- If gate signal consumption were measured (how often does a gate warning lead to a behavioral change?), the advisory-gate finding might show gates have value despite not blocking
- If SYNTHESIS.md content quality were scored and correlated with agent outcomes, the ceremony-as-proxy finding might show that existence is sufficient (or not)

---

## Catalog of Verification Theater Instances

### Severity 1: CRITICAL (≥50 agent-hours wasted)

| # | Signal Produced | Reality Masked | What Would Actually Verify | Harm |
|---|----------------|----------------|---------------------------|------|
| 1 | "Hotspot detector tests pass" (8 tests green) | Detector creates false positive issues from extraction churn | Integration test: create a file via extraction, run real source, verify no issue created | 97 agent-hours on false positive investigations across 3 fix attempts |

### Severity 2: HIGH (false confidence in quality enforcement)

| # | Signal Produced | Reality Masked | What Would Actually Verify | Harm |
|---|----------------|----------------|---------------------------|------|
| 2 | "Accretion gate test passes" (6 test cases) | Gate had 100% bypass rate over 2 weeks | Measurement: track bypass rate as a metric, alert when >90% | Months of believing accretion was enforced when it wasn't |
| 3 | "Spawn gate tests pass" (all gates) | Every gate is advisory-only, never prevents a bad spawn | Either: make gates blocking and test enforcement, or relabel tests as "gate-doesn't-crash tests" | False confidence that spawn quality is gated |
| 4 | "Verification passed" for completion | V0 verification = "Phase: Complete string found in comment" | Report verification level alongside pass/fail; don't let V0 create same confidence signal as V2 | Completion events imply thorough verification when actual check may be minimal |

### Severity 3: MODERATE (ceremony without substance)

| # | Signal Produced | Reality Masked | What Would Actually Verify | Harm |
|---|----------------|----------------|---------------------------|------|
| 5 | "SYNTHESIS.md exists" (gate passes) | SYNTHESIS.md could contain "TODO" or gibberish | Content quality heuristic: min word count, no placeholder patterns, section headers present | Agents produce empty syntheses that pass verification |
| 6 | "Test evidence found" (gate passes) | Regex matches pasted output, not actual test execution | Compare test output timestamp with agent session window; or require test output in structured format from tool call results | Agents could satisfy gate without running tests |
| 7 | "Trigger scan: created N issues" | Issues may be false positives (all 6 active detectors use mocked sources in tests) | Per-detector false positive rate measurement in production | Daemon creates issues that waste agent time investigating non-problems |

### Severity 4: LOW (design-intentional but unlabeled)

| # | Signal Produced | Reality Masked | What Would Actually Verify | Harm |
|---|----------------|----------------|---------------------------|------|
| 8 | "All 14 gates can be skipped" with 10-char reason | Skip system makes verification formally optional | Log and aggregate skip frequency; surface patterns to orchestrator | Erosion of verification value when agents learn they can skip everything |

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add integration tests for trigger detector sources | architectural | Crosses component boundaries (detector ↔ source ↔ filesystem) |
| Differentiate verification level in completion events | implementation | Tactical change to event data, single-scope |
| Measure false positive rates for active detectors | implementation | Instrumentation within existing daemon |
| Decide whether to invest in content quality verification | strategic | Value judgment about verification ROI vs. post-hoc measurement |

### Recommended Approach ⭐

**Instrument-Then-Decide** — Add measurement of verification theater before trying to fix it.

**Why this approach:**
- The accretion gate advisory decision showed measurement-first works: 2-week probe revealed 100% bypass, leading to rational conversion from blocking to advisory
- Attempting to "fix" theater without measuring which instances actually cause harm risks adding more ceremony (more theater)
- The hotspot detector case shows that adding tests without integration coverage just moves the theater — the "fix" passes tests while the bug persists

**Trade-offs accepted:**
- Some verification theater continues in the short term
- Investment in measurement infrastructure before direct fixes

**Implementation sequence:**
1. **Add verification-level to completion events** (V0/V1/V2/V3) — enables measurement of what "verified" actually means across the population
2. **Add per-detector false positive tracking** — when a daemon-triggered issue is closed without action, record it as potential FP; surface FP rates in `orch stats`
3. **Add integration test for one detector** (e.g., model_contradictions, since it scans the local filesystem) — establish the pattern, then extend to others
4. **Content quality heuristics for SYNTHESIS.md** — word count, placeholder detection, section coverage — as warnings, not gates, to measure before enforcing

### Alternative Approaches Considered

**Option B: Fix all tests to use real sources**
- **Pros:** Directly addresses mock-boundary theater
- **Cons:** Some sources run git commands and need real repos; test setup complexity would be high; risk of brittle tests that break on unrelated git state changes
- **When to use instead:** If a specific detector needs to be re-enabled (e.g., hotspot acceleration)

**Option C: Make gates blocking again with better bypass tracking**
- **Pros:** "Gate Over Remind" principle; blocking has theoretical deterrent value
- **Cons:** Accretion gate measurement already showed 100% bypass = zero deterrent; reimplementing blocking without changing agent behavior is more theater
- **When to use instead:** If a new agent population is introduced that doesn't know the bypass patterns

---

## References

**Files Examined:**
- `pkg/daemon/trigger_detectors_phase2_test.go` — All Phase 2 detector tests (mock-based)
- `pkg/daemon/trigger_detectors_phase2.go` — Phase 2 detector implementations
- `pkg/daemon/trigger.go` — Trigger scan orchestration (RunPeriodicTriggerScan)
- `pkg/daemon/trigger_service.go` — Production TriggerScanService
- `pkg/daemon/trigger_test.go` — Trigger scan integration tests (all mocked)
- `pkg/spawn/gates/hotspot.go` + `hotspot_test.go` — Advisory hotspot gate
- `pkg/spawn/gates/agreements.go` + `agreements_test.go` — Advisory agreements gate
- `pkg/spawn/gates/question.go` + `question_test.go` — Advisory open questions gate
- `pkg/verify/check.go` — Full verification pipeline (14 gates, V0-V3 levels)
- `pkg/verify/accretion.go` + `accretion_test.go` — Accretion verification (advisory)
- `pkg/verify/level.go` — Verification level system
- `pkg/verify/skip.go` — Gate skip configuration
- `pkg/verify/test_evidence.go` — Test evidence regex matching
- `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` — 100% bypass evidence

**Commands Run:**
```bash
# Verify hotspot detector was disabled after multiple fix attempts
git log --oneline -15 --all -- pkg/daemon/trigger_detectors.go pkg/daemon/trigger_detectors_phase2.go

# Verify hotspot detector is commented out in production
grep -n "hotspot_acceleration\|HotspotAcceleration" pkg/daemon/trigger_detectors.go
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` — Measured evidence of gate theater (100% bypass rate)
- **Commits:** bb3187123 (disable), c0fab4e5d (fix #2), 65d249045 (fix #1) — Hotspot detector fix-then-disable trajectory

---

## Investigation History

**2026-03-17:** Investigation started
- Initial question: Where do tests, gates, and checks create an appearance of quality without actually ensuring it?
- Context: Hotspot acceleration detector "fixed" twice with passing tests but 97 agent-hours wasted on false positive investigations

**2026-03-17:** All source files and tests examined across three areas (trigger detectors, spawn gates, completion verification)
- Found consistent pattern: tests verify mock plumbing, gates are advisory-only, completion checks artifact existence

**2026-03-17:** Investigation completed
- Status: Complete
- Key outcome: Verification theater is structural (mock-boundary isolation, advisory-only gates, existence-as-proxy) — requires measurement before fixes to avoid adding more theater
