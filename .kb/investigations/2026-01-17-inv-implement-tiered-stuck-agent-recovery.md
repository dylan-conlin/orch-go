<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Implement Tiered Stuck Agent Recovery

**Question:** How do we implement the tiered stuck agent recovery mechanism designed in the design session?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-implement-tiered-stuck-17jan-9296
**Phase:** Implementing
**Next Step:** Add recovery tracking infrastructure to daemon
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Stalled Detection Already Exists in Dashboard

**Evidence:** The codebase already has 15-minute stalled detection in `serve_agents.go:332-341` that sets `IsStalled: true` for agents with same phase for 15+ minutes. The dashboard Needs Attention component (`web/src/lib/components/needs-attention/needs-attention.svelte:183-213`) displays stalled agents with orange indicator.

**Source:** `cmd/orch/serve_agents.go:332-341`, `web/src/lib/components/needs-attention/needs-attention.svelte:183-213`, `web/src/lib/stores/agents.ts:326-331`

**Significance:** We don't need to build the "surface in dashboard" infrastructure from scratch - it already exists. We only need to add the recovery loop in the daemon that attempts resume before agents hit the 15-minute stalled threshold.

---

### Finding 2: Resume Command Has All Necessary Infrastructure

**Evidence:** The `cmd/orch/resume.go` file has resume-by-beads-ID, resume-by-workspace, and resume-by-session functions. It generates prompts, sends messages via OpenCode API, and logs resume events. Functions: `runResumeByBeadsID` (lines 140-217), `GenerateResumePrompt` (lines 92-100).

**Source:** `cmd/orch/resume.go:92-100, 140-217`

**Significance:** We can reuse the existing resume infrastructure. The daemon just needs to call the same resume logic that the CLI uses, but with rate limiting to prevent infinite loops.

---

### Finding 3: Daemon Poll Architecture Supports Additional Loops

**Evidence:** Daemon already has completion loop (`pkg/daemon/daemon.go`) that runs alongside spawn loop. Config includes `ReflectEnabled`/`ReflectInterval` (lines 37-43) and `CleanupEnabled`/`CleanupInterval` (lines 49-55) for periodic operations with `ShouldRunX()` and `RunPeriodicX()` pattern (lines 920-1033).

**Source:** `pkg/daemon/daemon.go:37-55, 920-1033`

**Significance:** Recovery can follow the same pattern as reflection and cleanup - periodic check with configurable interval. We need `RecoveryEnabled`, `RecoveryInterval`, and per-agent resume tracking to prevent spamming.

---

### Finding 4: Recovery Infrastructure 95% Complete

**Evidence:** Code review shows `RunPeriodicRecovery()` (lines 1086-1173), `ShouldRunRecovery()` (lines 1063-1074), config fields (lines 69-83), and helper functions (recovery.go) all exist. The `resumeAttempts` map is initialized in `NewWithConfig()` (line 212) for rate limiting. Missing: (1) Call to `RunPeriodicRecovery()` in daemon loop, (2) `resumeAttempts` init in `NewWithPool()`, (3) Recovery status output.

**Source:** `pkg/daemon/daemon.go:69-83, 104-107, 212, 1063-1173`, `pkg/daemon/recovery.go:28-176`

**Significance:** The tiered recovery is already built - we just need to wire it into the main daemon loop. This is a ~10 line addition vs. implementing from scratch.

---

## Synthesis

**Key Insights:**

1. **Recovery Infrastructure Already Exists** - Previous agent implemented the full recovery mechanism (detection, rate limiting, resume attempts) but didn't integrate it into the daemon loop. This reduces implementation to 3 small changes: (1) call RunPeriodicRecovery in loop, (2) fix NewWithPool init, (3) add status output.

2. **Stalled Detection is Independent** - The dashboard already has 15-minute stalled detection (serve_agents.go:332-341) that surfaces in Needs Attention. Recovery tries to prevent agents from hitting the stalled threshold by auto-resuming at 10 minutes. The two systems complement each other: recovery catches recoverable agents early, stalled detection catches unrecoverable ones.

3. **Rate Limiting Prevents Infinite Loops** - The `resumeAttempts` map tracks last resume time per agent (daemon.go:183-185). RunPeriodicRecovery checks this before resuming (lines 1127-1138) to prevent spamming stuck agents. This implements the "1 resume per hour per agent" requirement from the design.

**Answer to Investigation Question:**

The tiered stuck agent recovery is implemented by adding RunPeriodicRecovery to the daemon loop (alongside reflection and cleanup). The recovery loop runs every 5 minutes, detects agents idle >10 minutes without Phase: Complete, attempts resume with 1-hour rate limiting, and relies on existing stalled detection (15min threshold) to surface agents that don't recover. Changes needed: (1) Call RunPeriodicRecovery in cmd/orch/daemon.go around line 302, (2) Initialize resumeAttempts in NewWithPool, (3) Add recovery config output to startup message.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Daemon Recovery Loop with Per-Agent Rate Limiting** - Add recovery loop to daemon that detects stuck agents (idle >10min without Phase: Complete) and attempts resume with 1-hour-per-agent rate limiting

**Why this approach:**
- Reuses existing resume infrastructure (no new code for resume mechanism)
- Follows established daemon pattern (ReflectEnabled/CleanupEnabled precedent)
- Per-agent rate limiting prevents infinite resume loops
- Non-destructive recovery (resume) before escalation (surface in dashboard)

**Trade-offs accepted:**
- Won't recover agents stuck due to context exhaustion or infinite loops
- 15-minute delay before surfacing in dashboard (10min idle + 5min wait after resume fails)
- Additional daemon complexity (new loop, rate tracking)

**Implementation sequence:**
1. Add recovery config and per-agent resume tracker to Daemon struct
2. Create `ShouldAttemptRecovery()` function to detect stuck agents (idle >10min, no Phase: Complete, not recently resumed)
3. Create `RunPeriodicRecovery()` function that calls resume for stuck agents
4. Integrate recovery loop into daemon Run() cycle alongside reflection/cleanup
5. Add tests for recovery detection and rate limiting

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
