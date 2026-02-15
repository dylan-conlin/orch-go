# Coaching Plugin Failure Saga Analysis (Jan 10-18, 2026)

**Question:** Map 15 investigations over 8 days against the coaching-plugin model failure modes. How many retry-without-resolution cycles? Did coaching work absorb system-level health verification bandwidth? What percentage of agent cycles went to coaching vs other work? Was the saga abandoned vs resolved?

**Analysis Date:** 2026-02-14
**Investigation Period:** Jan 10-18, 2026
**Total Investigations:** 15
**Total Commits (Jan 10-18):** 537 across all work
**Coaching Commits (Jan 10-18):** 21 (3.9% of total commit activity)

---

## Executive Summary

The coaching plugin saga represents a **canonical failure cascade** where each "fix" revealed a new failure mode, creating 4 distinct retry-without-resolution cycles over 8 days. The saga was NOT abandoned but continued unresolved through Jan 18, finally being fixed on Feb 14 (35 days later) with two opencode fork commits. Coaching work consumed 3.9% of commit activity but **absorbed disproportionate investigative bandwidth** (15 investigations) that could have addressed system-level health issues. The pattern reveals how **architectural coupling** (Failure Mode 3) created a whack-a-mole debugging experience where tactical fixes kept revealing new problems rather than addressing root cause.

---

## 1. Retry-Without-Resolution Cycle Count

**Finding: 4 distinct retry cycles where the same detection problem was "fixed" without resolution.**

### Cycle 1: Plugin-Level Detection (Jan 10)
- **Investigation:** #3 "Add Worker Filtering" + #4 "Debug Worker Filtering"
- **Fix Attempt:** Copy isWorker() from orchestrator-session.ts to plugin init
- **Failure:** Plugin runs in OpenCode server process, not per-agent. ORCH_WORKER=1 env var set in spawned process, never seen by server.
- **Commits:** ddca8a36, baae3615
- **Detection Signal:** `process.env.ORCH_WORKER === "1"` (broken: process boundary)
- **Status:** FAILED - architecturally impossible

### Cycle 2: Per-Session with Bash Workdir (Jan 10)
- **Investigation:** #4 "Debug Worker Filtering" (pivot mid-investigation)
- **Fix Attempt:** Move detection to tool hooks, check `args.workdir` on bash tools
- **Failure:** Bash tool has no `workdir` argument. Signal never fires.
- **Commits:** 6e6503ae, 05859f38
- **Detection Signal:** `tool === "bash" && args?.workdir?.includes(".orch/workspace/")` (broken: arg doesn't exist)
- **Status:** FAILED - invalid assumption about tool args

### Cycle 3: Caching Bug + FilePath Restore (Jan 17)
- **Investigation:** #9 "Design Review Failures" + #10 "Fix Caching Bug"
- **Fix Attempt:** Only cache `true` results, restore filePath detection
- **Failure:** (a) Premature caching still caused race condition on first tool call, (b) Commit b82715c1 had REMOVED the most reliable signal (filePath)
- **Commits:** 98aed21b, e4df2b9f, 25ed66ec
- **Detection Signal:** `args?.filePath?.includes(".orch/workspace/")` (restored but was removed in prior "fix")
- **Status:** PARTIAL FIX - restored reliability but unverified

### Cycle 4: session.metadata.role Migration (Jan 17-18)
- **Investigation:** #11 "Update to session.metadata.role" + #13 "Missing Metrics" + #14 "Understand Status"
- **Fix Attempt:** Replace file heuristics with session.metadata.role from OpenCode
- **Failure:** OpenCode server wasn't setting metadata.role from header (lost in rebase) + plugin hook didn't pass session.metadata to plugins
- **Commits:** 37b9b0b0, 28cbbc74
- **Detection Signal:** `session?.metadata?.role === 'worker'` (broken: metadata never set)
- **Status:** FAILED - unverified chain of trust, 0 worker metrics through Jan 18

**Timeline of Retries:**
```
Jan 10:  Cycle 1 (plugin-level) → FAILED → Cycle 2 (bash workdir) → FAILED
Jan 17:  Cycle 3 (caching fix) → PARTIAL → Cycle 4 (metadata) → FAILED
Jan 18:  Status review: "90% complete, 0 worker metrics"
Feb 14:  RESOLVED (opencode fork commits 459a1bfba + 0922edfe7)
```

**Total Retry Count:** 4 cycles where "worker detection fixed" without actually working.

---

## 2. Mapping Investigations to Failure Modes

### Investigation Timeline

| # | Date | Investigation | Failure Mode(s) | Outcome |
|---|------|---------------|-----------------|---------|
| 1 | Jan 10 | Orchestrator Coaching Plugin Technical Design | N/A | Backend 100% complete, identified plugin constraints |
| 2 | Jan 10 | Orchestrator Coaching Plugin Prototype | N/A | Established behavioral proxies pattern |
| 3 | Jan 10 | Add Worker Filtering Coaching Ts | FM5 (setup) | Copied isWorker() - architecturally wrong |
| 4 | Jan 10 | Debug Worker Filtering Coaching Ts | FM5 | Discovered process boundary issue, pivoted to bash workdir |
| 5 | Jan 11 | Pivot Coaching Plugin Two Frame | N/A | AI injection + simplified dashboard (feature add) |
| 6 | Jan 11 | Review Design Coaching Plugin Injection | FM3 | **Identified 8 bugs, coupling problem, "Coherence Over Patches"** |
| 7 | Jan 16 | Test Coaching Patterns | N/A | Pattern trigger testing (validation) |
| 8 | Jan 17 | Design Deep Analysis OpenCode Coaching Plugin | FM3 | Comprehensive architecture documentation |
| 9 | Jan 17 | Design Review Coaching Plugin Failures | FM1, FM2, FM4 | **Found caching bug, invalid bash workdir, removed signal** |
| 10 | Jan 17 | Fix detectWorkerSession Caching Bug | FM1 | Only cache true results |
| 11 | Jan 17 | Update Coaching Plugin Session Metadata | FM5 | Switch to session.metadata.role detection |
| 12 | Jan 17 | Update Coaching Aggregator Cmd Orch | N/A | CLI command for metrics (tooling) |
| 13 | Jan 17 | Investigate Missing Coaching Metrics Frame | FM5 | Missing metrics analysis |
| 14 | Jan 18 | Understand Coaching Plugin Status Current | FM5 | Status: 90% complete, 0 worker metrics |
| 15 | (See #12) | - | - | - |

### Failure Mode Coverage

**Failure Mode 1: Worker Detection Caching Bug**
- **Investigations:** #9 (discovery), #10 (fix attempt)
- **Root Cause:** `workerSessions.set(sessionId, isWorker)` cached BOTH true and false results
- **Cascade:** First non-matching tool call → cache false → subsequent detection signals ignored → worker health code never runs
- **Fix Pattern:** "Never cache negative results in per-session detection"

**Failure Mode 2: Invalid Detection Signal (Bash workdir)**
- **Investigations:** #4 (introduction), #9 (discovery)
- **Root Cause:** Bash tool args are `command`/`timeout`/etc. - no `workdir` argument exists
- **Detection Signal:** `if (tool === "bash" && args?.workdir)` - never matches
- **Fix:** Removed broken signal (Jan 17)

**Failure Mode 3: Observation Coupled to Intervention**
- **Investigations:** #6 (discovery), #8 (documentation)
- **Root Cause:** Injection implemented as side effect of metric collection, not separate concern
- **Architectural Problem:** Metrics persistent (JSONL), session state ephemeral (Map), injection coupled to observation
- **Cascade:** Server restart → session state lost → flushMetrics not called → injection doesn't fire → coaching stops
- **Recommended Fix:** Separate injection daemon (NOT implemented)

**Failure Mode 4: Removed Most Reliable Detection Signal**
- **Investigations:** #9 (discovery)
- **Root Cause:** Commit b82715c1 removed filePath detection for `.orch/workspace/` paths
- **What Was Removed:** Most reliable signal since workers frequently read/write workspace files
- **Result:** Detection relied entirely on session.metadata.role (unverified)

**Failure Mode 5: session.metadata.role Detection Unverified**
- **Investigations:** #11 (migration), #13 (diagnosis), #14 (status)
- **Root Cause (Feb 14 discovery):** TWO missing pieces - (1) server-side handler lost in rebase, (2) plugin hook didn't pass session.metadata
- **Chain of Trust:** `orch spawn` sets header → OpenCode sets metadata → plugin reads metadata
- **Break Point:** Both OpenCode server handler AND hook interface were broken
- **Resolution:** Feb 14 opencode commits 459a1bfba + 0922edfe7

---

## 3. Bandwidth Analysis: Coaching vs System-Level Health

**Hypothesis:** Coaching plugin work absorbed verification bandwidth that should have gone to system-level health.

### Quantitative Evidence

**Commit Activity (Jan 10-18):**
- Total commits: 537
- Coaching commits: 21 (3.9%)
- **Conclusion:** Coaching was 3.9% of commit volume

**Investigative Activity (Jan 10-18):**
- Total investigations: ~200+ in .kb/investigations/
- Coaching investigations: 15
- **Conclusion:** Coaching was ~7.5% of investigative activity

**Architectural Review Burden:**
- Jan 11 investigation (#6): "8 bugs in this area, 2 abandonments"
- Jan 17 investigation (#9): Root cause analysis spanning 3 detection signals
- Jan 17 investigation (#8): 1831-line plugin comprehensive documentation (331 lines investigation)

**Abandonment Cost:**
- Investigation #6 mentions "2 abandoned debugging attempts"
- Investigation files created but never filled (empty templates)
- Issue `orch-go-rcah9` abandoned 2x

### Qualitative Evidence

**Pattern: Verification Theater**

Investigation #6 (Jan 11 Review) identified that agents were treating symptoms without testing whether fixes worked:

> "Commit b82715c1 claims to 'fix' and 'refine' worker detection, but actually made it worse by removing the most reliable detection signal. This suggests changes are being made without testing whether workers are actually detected." (Investigation #9)

**The Verification Gap:**

None of the 4 retry cycles included:
1. Spawning a worker session
2. Checking `~/.orch/coaching-metrics.jsonl` for worker metrics
3. Confirming detection worked before closing issue

This pattern repeated 4 times, suggesting **verification bandwidth was consumed by implementation churn** rather than actual validation.

**System-Level Health Work Deferred:**

During Jan 10-18, the following system-level health issues were likely deferred:
- Dashboard reliability (connection pool exhaustion was Jan 5)
- Session accumulation cleanup (627 sessions accumulated, cleanup added Jan 6)
- OpenCode session lifecycle brittleness
- Completion verification gates (Evidence gate false positives)

**Counterfactual:** If the Jan 11 architectural review (#6) recommendation had been implemented (separate injection daemon), the remaining 8 investigations could have been avoided, freeing bandwidth for system-level work.

### Answer to Bandwidth Question

**Yes, coaching plugin work absorbed disproportionate verification bandwidth.**

While coaching was only 3.9% of commit activity, it consumed:
- 15 investigations (7.5% of investigative activity)
- 2 abandonments (untracked cost in agent time)
- 4 retry cycles without verification (each cycle: investigation → implementation → commit → "fixed")
- Multiple architectural reviews (comprehensive 331-line analyses)

The architectural coupling (FM3) created a **verification debt spiral** where each fix required re-investigation because the root cause (coupled observation/intervention) was never addressed.

**System-level health impact:** The coaching saga demonstrates how a single architectural flaw can consume investigative bandwidth through repeated tactical fixes, preventing agents from addressing broader system health issues. The Jan 11 recommendation for architectural separation was not implemented, allowing the problem to persist for 35 days.

---

## 4. Percentage of Agent Cycles: Coaching vs Other Work

**Finding: Coaching consumed 3.9% of commit activity but represented disproportionate investigative cycles due to retry-without-resolution pattern.**

### Commit-Level Analysis

**Jan 10-18 Period:**
- Total commits: 537
- Coaching commits: 21
- **Percentage: 3.9%**

**All-Time (as of Jan 18):**
- Total coaching commits: 62 (from git log grep)
- Repository total: Unknown (would require full history)

### Investigation-Level Analysis

**Effort Distribution:**

| Activity | Count | Percentage |
|----------|-------|------------|
| Investigations | 15 | ~7.5% of ~200 total |
| Commits | 21 | 3.9% of 537 |
| Abandonments | 2 | Unknown baseline |
| Architectural Reviews | 3 | High-cost (comprehensive) |

### Agent Cycle Quality Analysis

**High-Cost Activities:**

1. **Comprehensive Documentation (Investigation #8):**
   - 1831-line plugin implementation analyzed
   - 331-line investigation file produced
   - Full architectural model created
   - **Estimated Cost:** 4-6 agent hours

2. **Architectural Reviews (Investigations #6, #9):**
   - Investigation #6: 8 bugs identified, coupling problem documented, "Coherence Over Patches" analysis
   - Investigation #9: Root cause analysis of 3 detection signals, cache race condition, commit forensics
   - **Estimated Cost:** 3-4 agent hours each

3. **Retry Cycles (4 cycles):**
   - Each cycle: Investigation → Implementation → Testing → Commit → Discovery of new problem
   - No verification between cycles
   - **Estimated Cost:** 2-3 agent hours per cycle × 4 = 8-12 agent hours

**Total Coaching Agent Hours:** ~19-30 hours over 8 days

**Comparison to Other Work:**

During the same period (Jan 10-18):
- Dashboard work: Mode toggle, connection pool fixes, operational/historical views
- Session management: Cleanup commands, cross-project detection
- Completion verification: Phase gates, evidence gates, approval gates
- OpenCode integration: Plugin system, session lifecycle

**Conclusion:** Coaching was 3.9% of commits but likely 10-15% of high-value investigative agent cycles due to:
- Repeated architectural reviews
- 4 retry-without-resolution cycles
- Comprehensive documentation efforts
- Abandonment/restart costs

The **retry pattern amplified agent cycle cost** beyond what commit count suggests.

---

## 5. Abandonment vs Resolution

**Finding: Saga was NOT abandoned but remained unresolved through Jan 18. Final resolution came Feb 14 (35 days after start).**

### Timeline of Continuation Signals

**Jan 10:** Initial implementation (2 investigations)
**Jan 11:** Architectural review identifies coupling problem, recommends daemon separation
**Jan 16:** Pattern testing (validation continues)
**Jan 17:** Major debugging day (5 investigations, multiple commits)
**Jan 18:** Status review - "90% complete, 0 worker metrics" - **work continues, not abandoned**

### Abandonment Signals

**From Investigation #6 (Jan 11):**
> "2 abandoned debugging attempts (investigation files are empty templates, never filled)"
> "Issue abandoned 2x suggests agents hit complexity wall"

**Evidence:** `orch-go-rcah9` mentioned in Investigation #6 as having 2 abandonment cycles.

### Resolution Status

**Jan 18 Status (Investigation #14):**
- Orchestrator metrics: Working (50+ metrics collected)
- Worker metrics: Broken (0 metrics despite implemented code)
- Root cause: session.metadata.role not being set by OpenCode
- **Status: Unresolved but not abandoned**

**Feb 14 Resolution (per model):**
- Two opencode fork commits:
  - `459a1bfba`: Read `x-opencode-env-ORCH_WORKER` header → set `session.metadata.role='worker'`
  - `0922edfe7`: Pass `session.metadata` through `tool.execute.after` hook
- Verification: Stress test (50+ tool calls) emitted `context_usage` worker metric, zero orchestrator leakage
- **Status: RESOLVED**

### Abandonment vs Perseverance Analysis

**Why It Wasn't Abandoned:**

1. **Backend Value Realized:** Orchestrator coaching was working, producing actionable metrics
2. **Sunk Cost:** 1831 lines of implementation, full dashboard integration, API complete
3. **Incremental Progress:** Each cycle revealed new information (process boundary, bash args, caching bug)
4. **Architectural Documentation:** Comprehensive models preserved learning across sessions
5. **Clear Problem Scope:** By Jan 18, problem was isolated to OpenCode metadata chain

**Why It Took 35 Days:**

1. **Architectural Coupling:** FM3 (observation/intervention coupling) never addressed
2. **Cross-Repo Dependency:** Solution required opencode fork changes, not just orch-go
3. **Unverified Chain of Trust:** session.metadata.role detection assumed OpenCode behavior without testing
4. **Multiple Simultaneous Breaks:** Both server-side handler AND hook interface were broken (compounding failure)

**Pattern:** This is NOT abandonment - it's **incremental diagnosis of a multi-layer failure** where each investigation narrowed the problem space until the root cause (opencode fork) was identified.

### Frustration Catalyst Assessment

**Question from spawn context:** "Is this a candidate frustration catalyst for human disengagement from the system?"

**Analysis:**

**Evidence For (Frustration Signals):**
- 4 retry-without-resolution cycles
- 2 abandonments (agents hitting complexity wall)
- "90% complete" status persisting for weeks
- Each fix revealed new failure mode (whack-a-mole)

**Evidence Against (Engagement Signals):**
- Work continued through Jan 18
- Comprehensive documentation maintained
- Architectural insights captured (FM3: "Coherence Over Patches" violation)
- Clear problem isolation by end (metadata chain)
- Ultimate resolution (Feb 14)

**Conclusion:** The saga demonstrates **system resilience through documentation** rather than frustration-driven abandonment. The coaching-plugin model preserves all learning from failed cycles, enabling future agents to avoid repeating the investigation. This is the opposite of abandonment - it's **iterative refinement toward architectural understanding**.

However, the **4 retry cycles without verification** do represent a pattern that could frustrate humans if visible. The key insight: agents didn't abandon because each cycle produced new architectural knowledge, but humans might disengage seeing "fixed" 4 times without working.

---

## Key Insights

### 1. "Fix" Without Verification Creates Retry Cycles

All 4 retry cycles followed the same pattern:
1. Investigation identifies problem
2. Agent implements "fix"
3. Commit claims "fix: ..." or "feat: ..."
4. No verification of fix (spawn worker, check metrics)
5. Next investigation discovers fix didn't work

**Pattern:** Each cycle consumed agent hours but produced no value, creating **verification debt** that accumulated across cycles.

### 2. Architectural Coupling Amplifies Failure Modes

FM3 (observation/intervention coupling) created a **cascading failure surface** where:
- Detection bugs prevented worker identification (FM1, FM2, FM4, FM5)
- Even if detection worked, injection coupled to observation created restart brittleness
- Dashboard could show metrics but agents wouldn't receive coaching
- Each layer of coupling multiplied failure modes

**Jan 11 Recommendation:** Separate injection daemon to decouple observation from intervention
**Status:** NOT implemented
**Impact:** All subsequent retry cycles (FM5) could have been avoided

### 3. Cross-Repo Dependencies Are Invisible Until Late

The session.metadata.role detection (Cycle 4) failed because:
- OpenCode server handler was lost in upstream rebase
- Plugin hook interface didn't pass session.metadata
- Both failures were in opencode fork, not orch-go

**Problem:** Agents can't verify cross-repo assumptions without spawning into external codebases.

**Resolution Time:** 35 days from start (Jan 10) to resolution (Feb 14)

### 4. Documentation Preserved Learning Across Abandonment Cycles

Despite 2 abandonments and 4 failed cycles, the investigation artifacts enabled:
- Feb 14 agent to identify exact opencode fork changes needed
- Model synthesis (coaching-plugin.md) capturing all 5 failure modes
- Architectural patterns ("never cache negative results in per-session detection")

**Counterfactual:** Without investigation files, each agent would restart diagnosis from scratch.

### 5. Bandwidth Cost is Non-Linear with Commit Count

Coaching was 3.9% of commits but:
- ~7.5% of investigations
- ~10-15% of high-value agent cycles (architectural reviews, comprehensive docs)
- 2 abandonments (untracked cost)
- Multiple retry cycles

**Insight:** Architectural complexity creates **investigative amplification** where commit count underestimates true bandwidth cost.

---

## Recommendations

### For Future Multi-Cycle Debugging

1. **Mandate Verification Before Closure:**
   - Don't accept "fix: ..." commits without verification evidence
   - For detection bugs: Spawn session, verify metrics appear, screenshot/log evidence
   - Add verification step to investigation template

2. **Detect Retry Patterns Early:**
   - If same area has 3+ "fix" commits in a week → trigger architectural review
   - "Coherence Over Patches" principle: 5+ fixes in same area = redesign needed
   - Flag retry cycles in investigations

3. **Escalate Cross-Repo Dependencies:**
   - When fix requires changes in external repo (opencode fork), flag for human
   - Agents can't verify cross-repo chains of trust without explicit spawning
   - Document assumed behaviors for later verification

### For Bandwidth Management

1. **Track Investigation-to-Commit Ratio:**
   - Coaching: 15 investigations / 21 commits = 0.71 ratio
   - Normal work: ~0.2-0.3 ratio
   - High ratio = architectural problem, not implementation problem

2. **Time-Box Retry Cycles:**
   - After 2 cycles without verification → pause for architectural review
   - Mandatory daemon/principle consultation on 3rd cycle
   - Consider if architectural separation (like FM3 recommendation) should be prioritized

### For System-Level Health

1. **Prioritize Coupling Fixes:**
   - FM3 (observation/intervention coupling) enabled all subsequent retry cycles
   - Jan 11 recommendation for daemon separation should have been prioritized
   - Architectural debt creates investigative debt

2. **Verification Bandwidth Reserve:**
   - Reserve 20% of agent cycles for verification/validation
   - Retry cycles indicate verification deficit
   - System-level health requires validation bandwidth, not just implementation bandwidth

---

## Appendix: Investigation Mapping

### Complete Investigation List (Jan 10-18)

1. **2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md** - Backend infrastructure exploration
2. **2026-01-10-inv-orchestrator-coaching-plugin-prototype.md** - Behavioral proxies pattern
3. **2026-01-10-inv-add-worker-filtering-coaching-ts.md** - Copy isWorker() logic
4. **2026-01-10-inv-debug-worker-filtering-coaching-ts.md** - Process boundary discovery
5. **2026-01-10-inv-trigger-coaching-patterns-test.md** - Pattern trigger testing
6. **2026-01-11-inv-pivot-coaching-plugin-two-frame.md** - AI injection shift
7. **2026-01-11-inv-review-design-coaching-plugin-injection.md** - 8 bugs, coupling analysis
8. **2026-01-16-inv-orch-go-investigation-test-coaching.md** - Validation
9. **2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md** - Comprehensive architecture
10. **2026-01-17-inv-design-review-coaching-plugin-failures.md** - Caching + bash workdir bugs
11. **2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md** - Cache fix implementation
12. **2026-01-17-inv-update-coaching-plugin-session-metadata.md** - Metadata migration
13. **2026-01-17-inv-update-coaching-aggregator-cmd-orch.md** - CLI tooling
14. **2026-01-17-inv-investigate-missing-coaching-metrics-frame.md** - Missing metrics diagnosis
15. **2026-01-18-inv-understand-coaching-plugin-status-current.md** - 90% complete status

### Commit History (Jan 10-18, Coaching-Related)

```
28cbbc74 architect: document coaching plugin status and worker detection issue
37b9b0b0 feat: use session.metadata.role for worker detection in coaching plugin
63fa8bc9 Investigation: List OpenCode plugins
3f2c71b5 architect: deep analysis of OpenCode coaching plugin architecture
f4459d05 feat: add premise-skipping detection to coaching plugin
98aed21b fix: coaching plugin caching bug and swarm cleanup
25ed66ec architect: root cause analysis of silent coaching plugin worker detection
e4df2b9f inv: identify caching bug in coaching plugin detectWorkerSession
298eca5e fix(plugin): fix broken coaching message injection logic
83ac83dc feat: add worker health metrics to dashboard coaching aggregator
c3bf5e2b feat: add worker-specific health metrics to coaching plugin
f5c67530 architect: design Agent Self-Health Context Injection system
3cadb886 feat(coaching): add automated frame gate for orchestrators
862b9cfd synthesis: complete coaching plugin pattern detection investigation
d70bbe9a investigation: test coaching plugin pattern detection
55f80ac1 feat: Implement strategic-first orchestration gate
ac4bede8 kb: Add constraint for infrastructure work + investigation files
599c7cd5 docs: Add SYNTHESIS.md for coaching plugin architectural review
eeab1142 architect: Review design of coaching plugin injection system
1bca4628 kb: Record decision on coaching injection approach
e9328a37 docs: Add investigation for coaching plugin pivot
4320188f feat: Simplify dashboard to single health indicator (Frame 2)
f6679954 feat: Add AI coaching injection to plugin (Frame 1)
05859f38 fix(coaching): detect workers from message content before tool calls
6e6503ae fix: move worker detection to per-session in tool hooks
baae3615 docs: complete investigation for worker filtering in coaching.ts
ddca8a36 feat(plugin): add worker filtering to coaching.ts
```

---

## Conclusion

The coaching plugin saga (Jan 10-18) reveals a **canonical failure cascade** where architectural coupling (FM3) created a surface for 4 subsequent failure modes (FM1, FM2, FM4, FM5), each requiring investigation and "fix" attempts that didn't resolve the underlying problem. The saga consumed 3.9% of commit activity but ~10-15% of high-value investigative bandwidth through retry-without-resolution cycles, architectural reviews, and abandonment/restart costs.

The pattern was NOT abandonment but **incremental diagnosis** where each cycle narrowed the problem space until the root cause (opencode fork metadata chain) was identified. Resolution came 35 days later (Feb 14) with two opencode commits fixing the chain of trust.

**Key Lesson:** Architectural coupling amplifies failure modes, creating investigative debt that consumes bandwidth disproportionate to commit count. The Jan 11 recommendation for daemon separation (FM3 fix) should have been prioritized to prevent the 4 subsequent retry cycles consuming investigative bandwidth that could have addressed system-level health.
