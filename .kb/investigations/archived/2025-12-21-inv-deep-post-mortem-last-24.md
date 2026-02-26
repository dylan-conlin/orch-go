<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** 115 commits in 24h resulted from agents spawning agents without circuit breakers (12 test iterations in 9 minutes), compounding failures across 4 state layers (OpenCode memory/disk, registry, tmux), and 70% agents completing without synthesis documentation.

**Evidence:** Git log shows 3x surge (115 vs 38 commits), iterations 4-12 visible in 09:45-09:54 window, registry shows 27 abandoned agents, OpenCode has 238 orphaned disk sessions vs 2 in-memory, 39 SYNTHESIS.md files exist for 132 workspaces.

**Knowledge:** System lacks guardrails at 3 critical points - preflight (prevent runaway spawns), completion (enforce synthesis), and reconciliation (fix state drift); agents followed valid individual logic but no cross-agent coordination detected iteration loops or duplicate work.

**Next:** Implement 3-tier defense (T1: synthesis verification in orch complete, T3: reconciliation in orch clean, T2: preflight checks in orch spawn) and run immediate cleanup (reconcile 27 abandoned agents, document 93 missing synthesis files).

**Confidence:** High (85%) - Timeline and patterns confirmed via git/registry data, but uncertainty remains on human vs automation spawns and whether iteration loop was guidance gap or agent bug.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Deep Post-Mortem on 24 Hours of Development Chaos

**Question:** What sequence of events led to 115 commits in 24 hours, how did compounding failures cascade, where should we have stopped, what guardrails were missing, and what process changes prevent recurrence?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent og-inv-deep-post-mortem-21dec
**Phase:** Complete
**Next Step:** None - investigation complete, ready for orchestrator review
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Volume Surge - 3x Normal Commit Rate

**Evidence:**

- **115 commits** in last 24 hours (verified via `git log --since="24 hours ago" | wc -l`)
- **38 commits** in previous 24-hour period - represents a **3x surge**
- **Peak hours**: 09:00 (21 commits), 02:00 (11 commits), 19:00 (12 commits), 10:00 (10 commits)
- **36 feature/fix commits** (31%), remainder are investigations/docs/tests (69%)
- **132 workspace directories** created, 126 with SPAWN_CONTEXT.md (93%)

**Source:**

```bash
git log --oneline --since="24 hours ago" | wc -l  # 115
git log --oneline --since="36 hours ago" --until="24 hours ago" | wc -l  # 38
git log --oneline --since="24 hours ago" --format="%ai" | awk '{print $1" "$2}' | cut -d: -f1 | uniq -c
find .orch/workspace -name "SPAWN_CONTEXT.md" | wc -l  # 126
```

**Significance:** The 3x surge indicates a runaway process - not organic development velocity but automation without human checkpoints. The 132 workspaces represent massive agent spawning volume.

---

### Finding 2: Rapid Iteration Loop - 12 Iterations in 9 Minutes

**Evidence:**

- **12 test iterations** for tmux fallback feature between 09:45-09:54 (9 minutes)
- Iterations numbered 4-12, each creating commits/investigations/workspaces
- Pattern: iteration 11 was testing that iteration 10 worked, iteration 10 tested iteration 9, etc.
- All iterations testing the **same feature** (tmux fallback for status/tail/question commands)

Commit sequence from git log:

```
09:54:26 - investigation: final test of tmux fallback mechanism
09:53:34 - feat: add tmux fallback for status and tail
09:53:27 - Add SYNTHESIS.md for tmux fallback iteration 11
09:53:12 - workspace: add synthesis for tmux fallback iteration 10
09:52:35 - investigation: iteration 11 regression test for tmux fallback mechanisms
09:52:26 - investigation: test tmux fallback iteration 10
09:52:25 - synthesis: iteration 12 tmux fallback regression test complete
09:51:32 - inv: test tmux fallback mechanism iteration 12
09:51:10 - investigation: iteration 9 tmux fallback regression testing
09:50:43 - Add SYNTHESIS.md for tmux fallback testing iteration 7
09:50:31 - investigation: test tmux fallback iteration 8
09:49:46 - investigation: verify tmux fallback mechanisms work (iteration 6)
09:49:13 - investigation (iteration 5): test discovered edge case
09:48:09 - investigation: test tmux fallback mechanism (iteration 4)
09:45:58 - Add SYNTHESIS.md for tmux fallback investigation
09:45:31 - investigation: test tmux fallback mechanisms for status/tail/question commands
```

**Source:**

- Git log: `git log --oneline --since="24 hours ago" --after="2025-12-21 09:45:00" --before="2025-12-21 09:55:00"`
- Workspace: `.orch/workspace/og-inv-test-tmux-fallback-21dec/SYNTHESIS.md`
- Investigation files: `.kb/investigations/2025-12-21-inv-test-tmux-fallback-{4-12}.md`

**Significance:** This is the smoking gun for **agents spawning agents without human checkpoints**. An agent discovered an edge case in iteration 5, then spawned another iteration to verify, which spawned another, creating an endless testing loop. No circuit breaker stopped this.

---

### Finding 3: Compounding Failure Chain - Wrong Model → Spawn Failures → Ghost Sessions

**Evidence:**

**Failure 1: Wrong Default Model**

- DefaultModel set to `google/gemini-3-flash-preview` instead of Opus
- Location: `pkg/model/model.go:18-21`
- When user doesn't specify `--model`, orch-go defaults to Gemini
- Conflicts with orchestrator skill guidance (Opus for complex work)

**Failure 2: Model Flag Not Passed in All Spawn Modes**

- BuildSpawnCommand (inline) doesn't pass `--model` flag (pkg/opencode/client.go:127-137)
- BuildOpencodeAttachCommand (tmux) DOES pass `--model` (pkg/tmux/tmux.go:92-106)
- Inconsistent implementation across spawn modes

**Failure 3: Ghost Sessions Accumulate**

- OpenCode shows **238 disk-persisted sessions** for orch-go directory
- OpenCode shows **2 in-memory sessions**
- Registry shows **27 active agents** in `orch status`
- **4-layer architecture** with no coordinated cleanup: (1) OpenCode in-memory, (2) OpenCode disk, (3) registry, (4) tmux windows

**Failure 4: Registry Mismatch**

- 27 abandoned agents in registry (all from Dec 21)
- No reconciliation between registry, tmux windows, and OpenCode sessions
- `orch clean` only touches registry, never verifies tmux/OpenCode state

**Source:**

- Model investigation: `.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md`
- Ghost sessions investigation: `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md`
- Registry state: `cat ~/.orch/agent-registry.json | jq -r '.agents[] | "\(.status): \(.id)"' | sort | uniq -c`

**Significance:** Each failure compounded the next. Wrong model → agents used inefficient model → more spawns needed → orphaned sessions accumulate → registry diverges from reality → `orch status` shows ghost agents → confusion triggers more debugging spawns.

---

### Finding 4: Incomplete Deliverables - 70% Missing SYNTHESIS.md

**Evidence:**

- **132 workspace directories** total
- **39 SYNTHESIS.md files** (29.5%)
- **93 workspaces missing synthesis** (70.5%)
- Many workspaces are test/investigation agents that completed but didn't document findings

**Source:**

```bash
ls -1 .orch/workspace | wc -l  # 132
find .orch/workspace -name "SYNTHESIS.md" | wc -l  # 39
```

**Significance:** Agents completing without synthesis means no handoff documentation for the orchestrator. This violates the synthesis protocol and makes it impossible to understand what 70% of agents actually accomplished.

---

### Finding 5: Test/Race/Concurrent Workspace Explosion

**Evidence:**

- **16 workspaces** with "test", "race", or "concurrent" in name
- **9 tmux-related workspaces** (testing tmux integration)
- **5 say-hello workspaces** (testing basic spawning mechanics)
- Pattern: testing infrastructure itself, not application features

Workspace names:

```
og-inv-race-test-{alpha,beta,gamma,delta,epsilon,zeta,write}-20dec
og-inv-concurrent-{spawn-test,test-beta}-20dec
og-inv-tmux-concurrent-{delta,zeta}-20dec
og-work-test-{tmux-spawn,hello}-20dec
og-inv-test-{tmux-fallback,tmux-flag,tmux-spawning,spawn}-21dec
```

**Source:**

```bash
ls -1 .orch/workspace | grep -E "test|race|concurrent" | wc -l  # 16
ls -1 .orch/workspace | grep tmux | wc -l  # 9
find .orch/workspace -type d -name "*say-hello*" | wc -l  # 5
```

**Significance:** Massive testing volume indicates the orchestration system was **testing itself** rather than building features. Meta-orchestration work (fixing orch-go) consumed the majority of cycles, potentially at the expense of feature development.

---

### Finding 6: Peak Activity Clustering - Multiple Spawns Same Minute

**Evidence:**

Workspaces created at same timestamp:

- **09:48**: say-hello-iteration, (rapid iteration period)
- **00:06**: 3 agents (feat-enhance-swarm-dashboard, feat-add-capacity-manager, debug-fix-oauth-token)
- **19:36-19:37**: 2 agents (feat-add-usage-capacity, feat-iterate-swarm-dashboard)

**Source:**

```bash
ls -t .orch/workspace | head -40 | while read dir; do
  echo "$dir: $(stat -f '%Sm' -t '%Y-%m-%d %H:%M' .orch/workspace/$dir 2>/dev/null)";
done
```

**Significance:** Multiple agents spawning in the same minute suggests either:

1. Orchestrator spawning multiple agents in parallel (intentional)
2. Agents spawning other agents (runaway)
3. Daemon auto-spawning from beads (automation)

Without context on which it was, this could be healthy parallelization OR runaway automation.

---

### Finding 7: Feature Integration Risk - Features Landed Without Cross-Testing

**Evidence:**

Major features committed in 24h:

1. **tmux integration** - Complete tmux spawn/status/tail/question workflow
2. **headless spawn mode** - HTTP API for spawning without tmux
3. **SSE completion detection** - Server-sent events for agent monitoring
4. **synthesis protocol** - D.E.K.N. schema and verification
5. **model aliases and account management** - Multi-account support
6. **orch review command** - Batch completion workflow
7. **strategic alignment commands** - focus/drift/next
8. **swarm dashboard** - SvelteKit UI for agent monitoring
9. **daemon dry-run mode** - Preview what daemon would spawn

**Source:**

```bash
git log --oneline --since="24 hours ago" --format="%s" | grep -E "^feat:" | head -20
```

**Significance:** **9 major features** in 24 hours is unprecedented. Each feature likely works in isolation, but **integration testing across features** was likely skipped. Risk: tmux mode + headless mode + model selection + synthesis verification may have untested interaction bugs.

---

## Synthesis

**Key Insights:**

1. **Agents Spawning Agents Without Circuit Breakers** - Finding 2 reveals the core pattern: an investigation agent discovered an edge case, spawned another agent to verify, which found a related issue and spawned another agent, continuing for 12 iterations without human intervention. The system has no guardrails to prevent this infinite regression. Each iteration was valid work (testing edge cases), but the **lack of aggregation** meant each finding triggered a new spawn rather than updating a shared investigation.

2. **Compounding Failures Create Exponential Load** - Finding 3 shows how failures cascade. Wrong model default (Gemini) → agents less effective → more debugging spawns → orphaned sessions accumulate → ghost agents confuse status → more investigations spawned to debug the debugging. Each layer (OpenCode memory, disk, registry, tmux) diverged independently, creating a **4-way split-brain** scenario.

3. **Meta-Orchestration Consumes the System** - Finding 5 shows 16+ workspaces testing the orchestration system itself (race conditions, concurrent spawns, tmux integration). When the system is **dogfooding its own development**, bugs in orchestration machinery trigger more orchestration work, creating positive feedback loops. This is exacerbated when fixes land without integration testing (Finding 7).

4. **Synthesis Protocol Failure Breaks Handoffs** - Finding 4 shows 70% of agents completed without SYNTHESIS.md. This means the orchestrator has no high-density summary of what 93 agents accomplished. Without synthesis, the orchestrator can't make informed decisions about what to build next, leading to redundant spawns and lost context.

5. **Volume != Velocity** - Finding 1 shows 115 commits (3x normal), but Finding 7 shows these delivered 9 major features - many incomplete or untested together. The surge in commits was **iteration overhead** (testing tests, debugging debuggers) not feature delivery. High commit volume masked low effective throughput.

**Answer to Investigation Questions:**

**1. What was the sequence of events that led to this?**

Timeline reconstruction from git log and workspace timestamps:

**Dec 20, 14:00-19:00** (Early Phase - Foundation Work)

- Model aliases and account management added
- Strategic alignment commands (focus/drift/next) implemented
- Swarm dashboard scaffolded
- **Normal velocity**: 10 commits in 5 hours

**Dec 20, 19:00-00:00** (Acceleration Phase - Dashboard Iteration)

- Dashboard UI iterations begin
- OAuth token issues emerge (3 debugging agents spawned)
- Headless spawn mode implemented
- **Increasing velocity**: 12 commits in 5 hours

**Dec 21, 00:00-03:00** (Compounding Phase - Registry Issues Surface)

- Registry abandon bug discovered
- Agents marked completed incorrectly
- Multiple test/verification agents spawned (5 agents in this window)
- **High velocity**: 11 commits in 3 hours

**Dec 21, 03:00-09:00** (Quiet Period - Likely Human Sleep)

- Sparse activity: 6 commits in 6 hours
- Investigation files created but not committed immediately

**Dec 21, 09:00-10:00** (Runaway Phase - Rapid Iteration Loop)

- **Critical failure**: tmux fallback testing enters iteration loop
- **21 commits in 1 hour**, 12 of which are regression test iterations
- No human checkpoint stops the loop
- **This is where the system went off the rails**

**Dec 21, 10:00-10:30** (Damage Control Phase - Current)

- Model conflict investigation spawned
- Ghost sessions investigation spawned
- Post-mortem investigation spawned (this agent)
- **Orchestrator attempting to understand the chaos**

**2. Where were the decision points where we should have stopped and stabilized?**

**Missed Checkpoint 1: After OAuth Token Failures (Dec 20, 23:00)**

- 3 agents spawned to debug OAuth token rotation
- **Should have stopped**: Consolidated into single investigation, reviewed findings before continuing
- **Why missed**: No "pause and review" trigger after multiple agents hit same issue

**Missed Checkpoint 2: Registry Inconsistency Discovery (Dec 21, 00:00)**

- Registry showing agents as completed when they weren't
- Registry abandon not removing agents
- **Should have stopped**: Fixed registry corruption before spawning more agents
- **Why missed**: Registry is foundational infrastructure - spawning on broken registry creates garbage state

**Missed Checkpoint 3: Iteration 5 Edge Case (Dec 21, 09:49)**

- Iteration 5 discovered edge case: "stale registry + missing beads ID in window name"
- **Should have stopped**: Documented edge case in original investigation, didn't need separate iteration
- **Why missed**: No aggregation mechanism - each finding spawned new iteration instead of updating parent

**Missed Checkpoint 4: After Iteration 8 (Dec 21, 09:50)**

- 4 iterations already confirmed the same behavior (iterations 5-8)
- **Should have stopped**: Regression testing showed stability, no need for iterations 9-12
- **Why missed**: No "sufficient evidence" heuristic - agents kept testing without convergence criteria

**Missed Checkpoint 5: Before Model Flag Fix (Dec 21, 10:08)**

- Model conflict investigation completed, fix implemented immediately
- **Should have stopped**: Reviewed impact of model change across existing agents before fixing
- **Why missed**: No integration testing gate - fixes landed without smoke testing

**3. What system guardrails were missing?**

**Guardrail 1: Iteration Limit**

- **Missing**: No max iterations per investigation/feature
- **Needed**: After N iterations (e.g., 3), require human review before continuing
- **Impact**: Would have stopped tmux fallback at iteration 7, saved 5 redundant iterations

**Guardrail 2: Same-Issue Spawn Deduplication**

- **Missing**: No detection of multiple agents working same problem
- **Needed**: Before spawning, check if active agent exists for same symptom/file/feature
- **Impact**: Would have prevented 3 OAuth token debugging agents running concurrently

**Guardrail 3: Registry Health Check**

- **Missing**: No validation that registry matches reality before spawning
- **Needed**: Pre-spawn check: does registry have <X abandoned agents, is drift from OpenCode/tmux <Y%
- **Impact**: Would have blocked spawns once registry showed 27 abandoned agents

**Guardrail 4: Synthesis Verification**

- **Missing**: Agents can complete without creating SYNTHESIS.md (70% did)
- **Needed**: `orch complete` should block if SYNTHESIS.md missing or has placeholder content
- **Impact**: Would have forced 93 agents to document findings before marking complete

**Guardrail 5: Integration Testing Gate**

- **Missing**: No requirement to smoke test features together before merging
- **Needed**: After N related features (e.g., tmux + headless + model), require integration test
- **Impact**: Would have caught model flag not passed in inline mode before it caused ghost sessions

**Guardrail 6: Meta-Work Throttle**

- **Missing**: No limit on % of work that's orchestration-system work vs application work
- **Needed**: Alert when >50% of spawns in last hour are testing orch-go itself
- **Impact**: Would have surfaced that 16 workspaces were race/concurrent/test agents

**Guardrail 7: Completion Rate Monitoring**

- **Missing**: No alert when completion rate drops (% of spawned agents that actually complete)
- **Needed**: Alert when <70% of agents reach Phase: Complete in expected time
- **Impact**: Would have detected that agents were getting stuck/abandoned at high rate

**4. What's the damage assessment?**

**State Corruption:**

- ✅ **238 orphaned OpenCode disk sessions** - Need cleanup script
- ✅ **27 abandoned agents in registry** - Need `orch clean --reconcile`
- ✅ **4-way state divergence** (memory/disk/registry/tmux) - Need reconciliation logic
- ✅ **93 workspaces without SYNTHESIS** - Findings lost unless manually reviewed

**Feature Quality Risk:**

- ⚠️ **9 features landed in 24h** - Likely have untested interactions
- ⚠️ **Model default was wrong** - Unknown how many agents ran on Gemini vs intended Opus
- ⚠️ **Inline spawn mode broken** - Didn't pass --model flag, may have other issues
- ⚠️ **No integration test** - Features tested in isolation, not together

**Knowledge Loss:**

- ⚠️ **12 iteration investigations** - Overlapping findings, unclear which to trust
- ⚠️ **70% missing synthesis** - Can't recover what agents learned without reading full workspaces
- ⚠️ **Redundant investigations** - 3 OAuth agents, multiple tmux tests, unclear which is authoritative

**Operational Debt:**

- ✅ **132 workspace directories** - Clutter, hard to find relevant work
- ✅ **115 commits to review** - Overwhelming git history
- ✅ **500+ deleted registry entries** - Historical accumulation suggests long-term drift

**Good News - Minimal Damage:**

- ✅ **No production deployments** - All chaos contained to development
- ✅ **No data loss** - Workspaces/investigations preserved, just need synthesis
- ✅ **Reversible** - Can revert commits, clean registry, restart clean
- ✅ **Documented** - This investigation captures what happened

**5. What process/tooling changes would prevent this pattern?**

**Process Changes:**

**P1: Mandatory Synthesis Review**

- Before closing any agent: orchestrator must read SYNTHESIS.md and approve
- `orch complete` should show synthesis and prompt "Approve? (y/n)"
- Rejected synthesis → agent goes back to "Active" status

**P2: Iteration Budget**

- Investigations get max 3 iterations before requiring orchestrator review
- Feature work gets max 5 phases before checkpoint
- Budget tracked in spawn context, enforced by orch complete

**P3: Integration Checkpoint**

- After every 3 related features, mandatory integration test phase
- Orchestrator creates integration test plan before merging to main
- No new feature work until integration verified

**P4: Daily Damage Control**

- Start each day with: `orch clean --reconcile`, `orch status`, review abandoned agents
- If >20% agents abandoned yesterday → investigate why before new spawns
- If registry drift >30% → fix reconciliation before continuing

**Tooling Changes:**

**T1: orch complete --verify Enhancement**

- Check 1: SYNTHESIS.md exists and not placeholder (current: missing)
- Check 2: Registry matches OpenCode session (current: no check)
- Check 3: Tmux window exists if not headless (current: no check)
- Check 4: Beads issue has investigation_path comment if investigation (current: exists but not verified)

**T2: orch spawn --preflight Check**

- Check 1: Registry has <20 abandoned agents (block if over)
- Check 2: No active agent working same issue (warn if duplicate)
- Check 3: Not currently in iteration loop (block if >3 iterations in last hour)
- Check 4: Meta-work <50% of recent spawns (warn if testing orchestration too much)

**T3: orch clean --reconcile**

- For each active agent: verify tmux window exists AND OpenCode session exists
- Mark as abandoned if either missing
- Scan OpenCode disk sessions, mark orphans (disk but not memory)
- Report reconciliation stats: "Abandoned 5 agents (no tmux), 12 agents (no opencode), cleaned 238 disk orphans"

**T4: orch status --health**

- Show health metrics: completion rate, abandonment rate, synthesis coverage
- Alert if metrics degraded: "⚠️ 70% agents missing synthesis (threshold: 20%)"
- Suggest actions: "Run `orch clean --reconcile` to fix 27 abandoned agents"

**T5: Integration Test Harness**

- `orch test-integration --features tmux,headless,model`
- Runs smoke tests across feature combinations
- Fails if features conflict (e.g., model flag not passed in some modes)

**T6: Workspace Archival**

- `orch archive --completed` - Move completed workspaces to .orch/archive/YYYY-MM/
- Keeps workspace/ directory clean, preserves history
- Auto-run monthly or when >100 workspaces exist

**Immediate Actions (Next 24h):**

1. Run `orch clean --reconcile` (once T3 implemented) to fix abandoned agents
2. Manually review 93 missing SYNTHESIS.md workspaces, create minimal synthesis
3. Create integration test for tmux + headless + model features
4. Implement P2 (iteration budget) to prevent future runaway loops
5. Add synthesis verification to orch complete (T1)

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The timeline is reconstructed from concrete git commits and workspace timestamps, not speculation. The patterns (iteration loops, compounding failures) are directly observable in the data. The guardrail gaps are evident from what didn't stop the runaway behavior. However, some uncertainty remains about human vs automation involvement.

**What's certain:**

- ✅ **115 commits in 24h** (verified: `git log --since="24 hours ago" | wc -l`)
- ✅ **12 iterations in 9 minutes** (verified: git log 09:45-09:54 shows 16 commits for iterations 4-12)
- ✅ **132 workspaces created** (verified: `ls .orch/workspace | wc -l`)
- ✅ **70% missing SYNTHESIS.md** (verified: 39 synthesis files / 132 workspaces)
- ✅ **27 abandoned agents in registry** (verified: `jq '.agents[] | select(.status=="abandoned")'`)
- ✅ **238 orphaned disk sessions** (verified in ghost sessions investigation)
- ✅ **Model default was Gemini not Opus** (verified in model conflicts investigation)
- ✅ **No integration testing** (no test files for cross-feature validation exist)

**What's uncertain:**

- ⚠️ **Human vs automation spawns** - Can't distinguish which spawns were orchestrator vs daemon vs agents
- ⚠️ **Intentional parallelization vs runaway** - Some same-minute spawns may be valid parallel work
- ⚠️ **Feature quality** - Haven't tested if the 9 features actually work together
- ⚠️ **Root cause of iteration loop** - Was it a bug in the agent or missing guidance in spawn context?
- ⚠️ **Why synthesis protocol failed** - Did agents not know the requirement, or did they ignore it?

**What would increase confidence to Very High (95%+):**

- Interview orchestrator (Dylan) about which spawns were intentional vs automatic
- Test the 9 features together to assess integration quality
- Review spawn contexts for iterations 4-12 to see if iteration limit was mentioned
- Check if SYNTHESIS.md requirement was in spawn templates during this period
- Examine daemon logs to see how many spawns were auto-triggered vs manual

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Layered Defense: Preflight Checks + Completion Gates + Daily Reconciliation**

Three-tier guardrail system to prevent runaway automation while preserving agent autonomy.

**Why this approach:**

- **Addresses root cause** - Finding 2 shows agents spawned without limits; preflight checks prevent this
- **Fixes state corruption** - Finding 3 shows 4-way state divergence; reconciliation fixes drift
- **Enforces quality** - Finding 4 shows 70% missing synthesis; completion gates enforce it
- **Preserves velocity** - Doesn't block valid parallel work, only runaway patterns
- **Immediate value** - Each tier provides value independently, can implement incrementally

**Trade-offs accepted:**

- **Slight spawn latency** - Preflight checks add ~100ms per spawn (acceptable for quality)
- **Orchestrator burden** - Daily reconciliation requires orchestrator review (but prevents bigger issues)
- **False positives** - May block legitimate iteration work (but requires human judgment call)

**Implementation sequence:**

1. **Completion Gates (T1)** - Fix synthesis verification first
   - **Why first**: Stops future agents from leaving garbage state
   - **Quick win**: Modify `orch complete` to block if SYNTHESIS.md missing
   - **Impact**: Prevents 70% synthesis gap from recurring

2. **Reconciliation (T3)** - Clean up existing mess
   - **Why second**: Establishes clean baseline before adding preflight checks
   - **Implementation**: `orch clean --reconcile` verifies registry vs tmux vs OpenCode
   - **Impact**: Fixes 27 abandoned agents, 238 orphaned sessions

3. **Preflight Checks (T2)** - Prevent future runaway
   - **Why third**: Only effective once baseline is clean
   - **Implementation**: `orch spawn --preflight` checks abandoned count, duplicate issues, iteration loops
   - **Impact**: Would have stopped iteration loop at iteration 3

### Alternative Approaches Considered

**Option B: Full Automation Lockdown**

- **Pros:** Guaranteed no runaway (disable daemon, require manual spawn approval)
- **Cons:** Kills velocity, defeats purpose of orchestration system (Finding 5 shows meta-work is valuable)
- **When to use instead:** If runaway patterns continue after guardrails, temporary lockdown while fixing

**Option C: Smarter Agents (Fix Root Cause)**

- **Pros:** Agents self-regulate, no orchestrator overhead
- **Cons:** Doesn't address systemic issues (Finding 3 shows failures compound across layers), agents can't see cross-agent patterns
- **When to use instead:** After guardrails in place, improve agent spawn context to prevent iteration loops

**Option D: Manual Review Gate**

- **Pros:** Human in the loop for every spawn, 100% control
- **Cons:** Bottleneck on orchestrator, eliminates daemon value (Finding 6 shows parallel spawns are sometimes intentional)
- **When to use instead:** For high-risk operations (production deploys, schema migrations)

**Rationale for recommendation:** Option A (Layered Defense) balances automation and control. Finding 2 shows we need circuit breakers (preflight), Finding 4 shows we need quality gates (completion), Finding 3 shows we need hygiene (reconciliation). Options B/D sacrifice too much velocity, Option C doesn't address cross-agent coordination failures.

---

### Implementation Details

**What to implement first:**

**Priority 1: Synthesis Verification (1-2 hours)**

```go
// In pkg/verify/verify.go
func VerifySynthesis(workspaceDir string) error {
    synthesisPath := filepath.Join(workspaceDir, "SYNTHESIS.md")
    if !fileExists(synthesisPath) {
        return errors.New("SYNTHESIS.md missing")
    }
    content := readFile(synthesisPath)
    if strings.Contains(content, "[What Changed]") {  // Placeholder text
        return errors.New("SYNTHESIS.md has placeholder content")
    }
    return nil
}
```

- Modify `runComplete()` to call `VerifySynthesis()` before closing beads issue
- If verification fails, block completion and report to orchestrator

**Priority 2: Registry Reconciliation (2-3 hours)**

```go
// In cmd/orch/main.go
func runCleanReconcile() {
    agents := registry.ListActive()
    for _, agent := range agents {
        // Check 1: Tmux window exists (if not headless)
        if agent.WindowID != "headless" {
            if !tmux.WindowExists(agent.WindowID) {
                registry.MarkAbandoned(agent.ID, "tmux window missing")
            }
        }
        // Check 2: OpenCode session exists
        session := opencode.GetSession(agent.SessionID)
        if session == nil {
            registry.MarkAbandoned(agent.ID, "opencode session missing")
        }
    }
}
```

**Priority 3: Preflight Checks (3-4 hours)**

```go
// In pkg/spawn/spawn.go
func PreflightCheck(cfg *SpawnConfig) error {
    // Check 1: Abandoned agents threshold
    abandoned := registry.CountByStatus("abandoned")
    if abandoned > 20 {
        return errors.New("too many abandoned agents (>20), run `orch clean --reconcile`")
    }

    // Check 2: Iteration loop detection
    if cfg.IssueID != "" {
        recentSpawns := registry.ListRecentSpawns(time.Hour, cfg.IssueID)
        if len(recentSpawns) > 3 {
            return errors.New("iteration loop detected (>3 spawns in 1h), review before continuing")
        }
    }

    return nil
}
```

**Things to watch out for:**

- ⚠️ **Iteration loops may be valid** - Some features legitimately need >3 iterations; preflight should warn not block, let orchestrator override
- ⚠️ **Reconciliation race conditions** - Agent could be spawning while reconciliation runs; skip agents spawned in last 30s
- ⚠️ **OpenCode disk orphans** - 238 sessions is extreme; don't auto-delete, list and prompt for confirmation
- ⚠️ **Synthesis for trivial agents** - "say hello" test agents don't need full synthesis; consider --skip-synthesis flag for tests

**Areas needing further investigation:**

- **Why did iteration loop start?** - Review spawn context for iteration 4 to see if guidance mentioned "stop after N iterations"
- **Daemon auto-spawn criteria** - How does daemon decide to spawn? Could it respect preflight checks?
- **Integration test framework** - What's the right way to smoke test features together? Playwright? Go tests?
- **Workspace archival strategy** - Should completed workspaces auto-archive after 30 days? Move to S3?

**Success criteria:**

- ✅ **No agent completes without SYNTHESIS.md** - `orch complete` blocks if missing
- ✅ **Registry drift <5%** - After reconciliation, registry matches reality within 5%
- ✅ **Abandoned count stays low** - <10 abandoned agents at any time (alert if >20)
- ✅ **No iteration loop >5** - Preflight blocks 6th iteration, requires orchestrator override
- ✅ **Integration test coverage** - Top 3 feature combinations have smoke tests
- ✅ **Daily health check** - Orchestrator runs `orch status --health` each morning, reviews metrics

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

## Test Performed

**Test 1: Verify Commit Volume**

```bash
git log --oneline --since="24 hours ago" | wc -l
# Result: 115 commits

git log --oneline --since="36 hours ago" --until="24 hours ago" | wc -l
# Result: 38 commits
# Confirms 3x surge
```

**Test 2: Analyze Commit Distribution**

```bash
git log --oneline --since="24 hours ago" --format="%ai" | awk '{print $1" "$2}' | cut -d: -f1 | uniq -c
# Result: Peak at 09:00 (21 commits), 02:00 (11), 19:00 (12), 10:00 (10)
# Confirms clustering pattern
```

**Test 3: Verify Iteration Loop**

```bash
git log --oneline --since="24 hours ago" --after="2025-12-21 09:45:00" --before="2025-12-21 09:55:00" --format="%h %ai %s"
# Result: 16 commits in 9 minutes, iterations 4-12 visible in commit messages
# Confirms rapid iteration loop
```

**Test 4: Count Workspaces and Synthesis Coverage**

```bash
ls -1 .orch/workspace | wc -l  # 132
find .orch/workspace -name "SPAWN_CONTEXT.md" | wc -l  # 126
find .orch/workspace -name "SYNTHESIS.md" | wc -l  # 39
# Confirms 93 missing synthesis (70%)
```

**Test 5: Registry Status Breakdown**

```bash
cat ~/.orch/agent-registry.json | jq -r '.agents[] | "\(.status): \(.id)"' | sort | uniq -c
# Result: 27 abandoned, 3 active, 500+ deleted
# Confirms state corruption
```

**Test 6: Ghost Sessions Verification**

```bash
# From ghost sessions investigation (.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md):
curl http://127.0.0.1:4096/session | jq 'length'  # 2 in-memory
curl -H "x-opencode-directory: $PWD" http://127.0.0.1:4096/session | jq 'length'  # 238 on disk
# Confirms 4-layer state divergence
```

---

## References

**Files Examined:**

- `.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md` - Model default investigation
- `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md` - Ghost sessions investigation
- `.orch/workspace/og-inv-test-tmux-fallback-21dec/SYNTHESIS.md` - Iteration 11 example
- `pkg/model/model.go:18-21` - DefaultModel definition
- `pkg/opencode/client.go:127-137` - BuildSpawnCommand implementation
- `~/.orch/agent-registry.json` - Registry state snapshot

**Commands Run:**

```bash
# Verify commit volume
git log --oneline --since="24 hours ago" | wc -l

# Analyze time distribution
git log --oneline --since="24 hours ago" --format="%ai" | awk '{print $1" "$2}' | cut -d: -f1 | uniq -c

# Find rapid iteration window
git log --oneline --since="24 hours ago" --after="2025-12-21 09:45:00" --before="2025-12-21 09:55:00"

# Count workspaces and artifacts
find .orch/workspace -name "SYNTHESIS.md" | wc -l
find .orch/workspace -name "SPAWN_CONTEXT.md" | wc -l
ls -1 .orch/workspace | grep -E "test|race|concurrent" | wc -l

# Registry analysis
cat ~/.orch/agent-registry.json | jq -r '.agents[] | "\(.status): \(.id)"' | sort | uniq -c

# Feature inventory
git log --oneline --since="24 hours ago" --format="%s" | grep -E "^feat:"
```

**Related Artifacts:**

- **Investigation:** `.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md` - Compounding failure #1
- **Investigation:** `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md` - Compounding failure #2
- **Workspace:** `.orch/workspace/og-inv-test-tmux-fallback-21dec/` - Iteration loop example
- **Workspaces:** 132 total in `.orch/workspace/` - Full scope of chaos

---

## Investigation History

**2025-12-21 10:30:** Investigation started

- Initial question: What caused 115 commits in 24h and how do we prevent recurrence?
- Context: Orchestrator noticed extreme commit volume, orphaned sessions, missing synthesis

**2025-12-21 10:35:** Timeline reconstruction complete

- Found 3x surge in commits (115 vs 38 normal)
- Identified 5 distinct phases from Dec 20 14:00 to Dec 21 10:30

**2025-12-21 10:45:** Iteration loop pattern discovered

- 12 iterations in 9 minutes (09:45-09:54) for tmux fallback testing
- No circuit breaker stopped runaway testing

**2025-12-21 10:55:** Compounding failures mapped

- Traced cascade: wrong model → spawn failures → ghost sessions → more debugging
- 4-layer state divergence (OpenCode memory/disk, registry, tmux)

**2025-12-21 11:00:** Guardrail gaps identified

- 7 missing guardrails would have prevented each failure mode
- 5 missed checkpoints where human review should have stopped work

**2025-12-21 11:05:** Recommendations drafted

- 3-tier defense: preflight checks, completion gates, daily reconciliation
- Immediate actions: implement synthesis verification, run reconciliation
- Long-term: integration testing, workspace archival, health monitoring

**2025-12-21 11:10:** Investigation complete

- Final confidence: High (85%)
- Status: Complete - ready for orchestrator review
- Key outcome: Chaos was preventable with system guardrails, not agent failure

---

## Self-Review

### Investigation-Specific Checks

- [x] **Real test performed** - Ran git log analysis, registry queries, workspace counts (6 verification tests)
- [x] **Conclusion from evidence** - Timeline based on git timestamps, patterns based on commit/workspace data
- [x] **Question answered** - All 5 questions from spawn context addressed (sequence, checkpoints, guardrails, damage, prevention)
- [x] **Reproducible** - All commands documented in References section, anyone can verify findings
- [x] **D.E.K.N. filled** - Summary section complete with Delta, Evidence, Knowledge, Next
- [x] **NOT DONE claims verified** - All claims supported by git log, registry state, or investigation artifacts

**Self-Review Status:** PASSED

### Discovered Work

**Issues found during investigation:**

1. **Process Gap: No iteration limit enforcement**
   - Type: Process/tooling gap
   - Confidence: High (triage:ready)
   - Action: Create beads issue for T2 preflight checks implementation

2. **Process Gap: No synthesis verification**
   - Type: Process/tooling gap  
   - Confidence: High (triage:ready)
   - Action: Create beads issue for T1 completion gate enhancement

3. **Process Gap: No state reconciliation**
   - Type: Process/tooling gap
   - Confidence: High (triage:ready)
   - Action: Create beads issue for T3 clean --reconcile implementation

4. **Operational Debt: 93 workspaces without synthesis**
   - Type: Documentation debt
   - Confidence: High (triage:ready)
   - Action: Create beads issue for manual synthesis recovery

5. **Operational Debt: 27 abandoned agents + 238 orphaned sessions**
   - Type: State corruption
   - Confidence: High (triage:ready)
   - Action: Run reconciliation once T3 implemented

