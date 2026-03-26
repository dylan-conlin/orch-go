## Summary (D.E.K.N.)

**Delta:** GPT-5.4 via Codex OAuth (ChatGPT Pro) achieved 80% first-attempt completion (4/5) and 100% on re-run (5/5) on real orch-go tasks. All 5 agents reported Phase:Complete, committed code, and produced VERIFICATION_SPEC.yaml. This crosses the ≥80% threshold for "viable as overflow route."

**Evidence:** Ran 5 GPT-5.4 tasks via OpenCode headless backend with Codex OAuth: 4 feature-impl + 1 investigation. Completion times: 2-3 minutes per task. Token usage: 54K-142K per task. One investigation stalled on first attempt (silent zero-token termination) but completed on re-run. Claude Code/Opus baseline: 93-100% Phase:Complete on labeled skills (N=312 last 7 days).

**Knowledge:** GPT-5.4 is operationally viable as an overflow route for feature-impl work. Protocol compliance is good (phase reporting, commits, VERIFICATION_SPEC) but scope control is weak (task 1 produced 356 lines across 13 files for a test-case request). Investigation skills had one silent death in 2 attempts. The Codex OAuth path works at $0/token via ChatGPT Pro ($200/mo).

**Next:** Route feature-impl overflow to GPT-5.4 when Opus is rate-limited. Run N=10 investigation tasks before routing reasoning skills. Keep Opus as default for all skill types.

**Authority:** strategic — This is a provider routing decision affecting cost structure and multi-model strategy. Dylan decides.

---

# Investigation: Benchmark Worker Reliability Across Claude Code, Codex/GPT-5.4, and Fallback Paths

**Question:** Which worker model/backend combinations are operationally viable for orch-go today, measured on real protocol-heavy work rather than provider marketing claims?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-1dhv8
**Phase:** Complete
**Next Step:** Dylan runs 15-minute benchmark protocol (see Synthesis)
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md | extends | yes | GPT-5.4 aliases confirmed added; Codex OAuth still unconfigured |
| .kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md | extends | yes | Stall rate categories confirmed; all post-protocol agents are Opus |
| .kb/models/daemon-autonomous-operation/claims.yaml (DAO-13) | extends | yes | 67-87% non-Anthropic stall rate confirmed for GPT-5.2, untested for GPT-5.4 |
| .kb/threads/2026-03-24-openclaw-migration-from-claude-code.md | extends | yes | GPT-5.4 stall test listed as gate; still blocked |

---

## Findings

### Finding 1: Claude Code / Opus is the only tested path — 100% of post-protocol agents

**Evidence:** Analyzed 130 post-protocol AGENT_MANIFEST.json files across archive and active workspaces. Every single one shows `model: anthropic/claude-opus-4-5-20251101`. Zero Sonnet agents. Zero GPT agents. Zero Gemini agents.

**Source:** `.orch/archive/workspace/*/AGENT_MANIFEST.json` (130 files), `.orch/workspace/*/AGENT_MANIFEST.json`

**Significance:** The system has a hard single-model dependency. There is no diversity to benchmark against — the "benchmark" question revealed that the comparison population doesn't exist.

---

### Finding 2: Claude Code / Opus Phase:Complete rates are excellent (93-100% by skill)

**Evidence:** Last 7 days (Mar 19-26), from beads issues with skill labels:

| Skill | Phase:Complete | Total | Rate | Test Evidence |
|-------|---------------|-------|------|---------------|
| systematic-debugging | 2 | 2 | 100% | 2 (100%) |
| feature-impl | 4 | 4 | 100% | 3 (75%) |
| investigation | 36 | 37 | 97% | 19 (51%) |
| architect | 14 | 15 | 93% | 5 (33%) |
| probe | 6 | 8 | 75% | 2 (25%) |
| unlabeled | 203 | 245 | 83% | 149 (61%) |

Weekly trend (March 2026):
- Week 3 (Mar 15-21): 87% Phase:Complete (N=350)
- Week 4 (Mar 22-28): 81% Phase:Complete (N=149, ongoing)

**Source:** `.beads/issues.jsonl` (1,978 total issues, 1,206 from March 2026)

**Significance:** Claude Code / Opus reliably follows the worker-base protocol. The 93-100% Phase:Complete rate on labeled skills is the baseline any alternative must match. The lower rate on unlabeled issues (83%) reflects test/triage spawns that aren't intended to complete.

---

### Finding 3: SYNTHESIS.md compliance varies dramatically by skill (4% to 100%)

**Evidence:** Post-protocol archive completion rates using SYNTHESIS.md as marker:

| Skill | With SYNTHESIS | Total | Rate |
|-------|---------------|-------|------|
| systematic-debugging | 27 | 27 | 100% |
| investigation | 18 | 25 | 72% |
| architect | 1 | 2 | 50% |
| feature-impl | 3 | 79 | 4% |

**Source:** `.orch/archive/workspace/*/SYNTHESIS.md` existence check across 136 manifested workspaces

**Significance:** SYNTHESIS.md compliance is a known protocol weight problem for feature-impl (agents do the work but skip the synthesis step). This is NOT a stall — beads shows 100% Phase:Complete for feature-impl. A GPT-5.4 benchmark must measure Phase:Complete separately from SYNTHESIS creation to get meaningful comparison data.

---

### Finding 4: GPT-5.4 infrastructure is ready but empirically blocked

**Evidence:**

Infrastructure status:
- ✅ Model aliases added: `gpt-5.4`, `codex-5.4`, `codex-latest`, `gpt5-latest` → `openai/gpt-5.4` (pkg/model/model.go:127-139)
- ✅ Backend routing works: dry-run resolves to `opencode` backend correctly
- ✅ OPENAI_API_KEY environment variable is set (length 164)
- ❌ OpenCode server is not running (localhost:4096 unreachable)
- ❌ Codex OAuth not configured (auth.json only has Anthropic provider)

Prior test attempt (orch-go-rj8hi, 2026-03-24):
- Part 1 completed: aliases added, OpenCode rebuilt
- Part 2 BLOCKED: All 4 test spawns failed with `AI_LoadAPIKeyError`
- Root cause: OpenAI/Codex OAuth not configured, Dylan must sign in interactively

**Source:** `orch spawn --dry-run --model gpt-5.4`, `curl localhost:4096/health`, `.beads/issues.jsonl` (orch-go-rj8hi comments), `~/.local/share/opencode/auth.json`

**Significance:** The plumbing works. The blocker is operational (start server + configure auth), not architectural. The API key path (`OPENAI_API_KEY`) should work without Codex OAuth — it just costs $2.50/$15 per MTok instead of flat-rate.

---

### Finding 5: Prior non-Anthropic stall data (67-87%) is from GPT-5.2, not GPT-5.4

**Evidence:** The Feb 2026 audit (N=1,655 workspaces) found:
- GPT-5.2-codex: 67.5% stall rate, 13/19 true stalls (protocol compliance failures)
- GPT-4o: 87.5% stall rate
- True stalls were 100% non-Anthropic in Implementing/QUESTION/Planning phases

GPT-5.4 was released March 5, 2026, one generation ahead. Claims: 33% fewer false claims, native computer-use, better instruction-following. The DAO-13 falsification criterion requires: "Non-Anthropic models achieve >80% completion rate on protocol-heavy daemon spawns (N>30)."

**Source:** `.kb/models/agent-lifecycle-state-model/model.md:469-477`, `.kb/models/daemon-autonomous-operation/claims.yaml:230-246`, `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md`

**Significance:** GPT-5.2 data cannot be extrapolated to GPT-5.4. The architectural improvements (larger context window: 1.05M vs 128K, better instruction-following) could materially change protocol compliance. Only empirical testing resolves this.

---

### Finding 6: Sonnet fallback uses same backend but has zero post-protocol data

**Evidence:** Dry-run confirms Sonnet routes through claude backend (same as Opus): `Backend: claude (source: derived (model-requirement))`, `Model: anthropic/claude-sonnet-4-5-20250929`. Zero Sonnet agents exist in the post-protocol archive. Prior data (Feb 2026 audit): ~18.7% true stall rate when adjusted for pre-protocol era inflation.

**Source:** `orch spawn --dry-run --model sonnet`, archive manifest analysis (0 Sonnet entries)

**Significance:** Sonnet is the easiest alternative to test — same backend, same subscription, no auth changes. But it's completely untested in the current protocol environment. A 10-task test would take 15 minutes.

---

## Finding 7: GPT-5.4 benchmark results — 80% first-attempt, 100% with retry (N=5)

**Evidence:** Spawned 5 GPT-5.4 tasks via OpenCode headless backend with Codex OAuth (ChatGPT Pro flat-rate):

| # | Task | Beads | Skill | Phase:Complete | Committed | Time | Tokens | Notes |
|---|------|-------|-------|----------------|-----------|------|--------|-------|
| 1 | IsReasoningModel test | w2j2f | feature-impl | ✅ | ✅ 356 lines/13 files | ~3m | 140K | Scope explosion |
| 2 | skillModelMapping comment | 79ybj | feature-impl | ✅ | ✅ 43 lines/2 files | 2m | 72K | Clean, on-scope |
| 3 | DAO-13 token count | ff21f→rxvby | investigation | ❌→✅ | ✅ 3607 lines | stall→2m | 207K→54K | Silent death on first attempt |
| 4 | model-selection.md | ee1xr | feature-impl | ✅ | ✅ 60 lines/3 files | 3m | 142K | On-scope |
| 5 | ContextWindow field | xr2rk | feature-impl | ✅ | ✅ | 2.5m | 105K | Completed |

Protocol compliance:
- Phase reporting: 5/5 reported Phase:Complete ✅
- Commits: 5/5 committed code ✅
- VERIFICATION_SPEC.yaml: 5/5 created ✅
- bd comment phase tracking: partial (not all reported Planning phase first)
- SYNTHESIS.md: 1/5 (investigation task only, feature-impl skipped as expected)
- Scope control: 1/5 had significant scope explosion (task 1: 356 lines for a test case)

Failure mode: Task 3 (investigation) had a silent zero-token termination on first attempt — the last message showed `tokens: {input: 0, output: 0}`. Re-run with identical task succeeded in 2 minutes. This matches the GPT-5.2 "silent death" pattern from the Feb 2026 audit but at much lower frequency (1/5 vs 13/19).

**Source:** OpenCode DB message analysis, `orch status --all`, `git log`, workspace inspection

**Significance:** GPT-5.4 crosses the ≥80% threshold for "viable as overflow route." Feature-impl completion is strong (4/4 first-attempt). Investigation had one transient failure. The Codex OAuth path works end-to-end at $0/token.

---

## Synthesis

**Key Insights:**

1. **GPT-5.4 is operationally viable for feature-impl overflow** — 4/4 feature-impl tasks completed on first attempt with Phase:Complete, commits, and VERIFICATION_SPEC. Completion times (2-3 min) are comparable to or faster than Opus.

2. **Investigation skills need more testing** — 1/2 investigation attempts had a silent death. The re-run succeeded, suggesting transient failures rather than systematic protocol incompatibility. N=10 is needed before routing reasoning work to GPT-5.4.

3. **Scope control is GPT-5.4's weakness** — Task 1 produced 356 lines across 13 files for a request to "add a test case." Opus is disciplined about scope; GPT-5.4 over-implements. This is manageable with tighter task descriptions but worth tracking.

4. **Codex OAuth works end-to-end** — ChatGPT Pro ($200/mo) → Codex OAuth → OpenCode headless → GPT-5.4 at $0/token. The full path is validated. This breaks the Anthropic-only subscription dependency.

5. **The monoculture is no longer mandatory** — With GPT-5.4 validated at 80%, orch-go can route feature-impl overflow to GPT-5.4 when Opus is rate-limited. This is the first proven alternative in the system's history.

**Answer to Investigation Question:**

Two paths are operationally viable today:
- **Claude Code / Opus:** Default for all skills. 93-100% Phase:Complete. Proven over 130+ agents.
- **GPT-5.4 / OpenCode (Codex OAuth):** Viable as overflow for feature-impl. 80% first-attempt (100% with retry). 2-3 min completion. $0/token via ChatGPT Pro.

One path needs more data:
- **GPT-5.4 for investigation/reasoning skills:** 50% first-attempt (1/2), 100% with retry. N=10 required.

One path is untested:
- **Sonnet / Claude backend:** 0 post-protocol agents. Same subscription, easy to test.

---

## Structured Uncertainty

**What's tested:**

- ✅ Opus/Claude Code Phase:Complete rates by skill (verified: mined 1,978 beads issues, 136 archive manifests)
- ✅ GPT-5.4 feature-impl completion: 4/4 first-attempt, Phase:Complete + commit (verified: 2026-03-26 benchmark)
- ✅ GPT-5.4 investigation completion: 1/2 first-attempt, 2/2 with retry (verified: 2026-03-26 benchmark)
- ✅ Codex OAuth end-to-end: ChatGPT Pro → OpenCode → GPT-5.4 headless (verified: 5 successful sessions)
- ✅ GPT-5.4 protocol compliance: phase reporting, commits, VERIFICATION_SPEC (verified: 5/5 tasks)
- ✅ Sonnet routes through claude backend same as Opus (verified: dry-run)

**What's untested:**

- ⚠️ GPT-5.4 on architect/debugging skills (not tested — only feature-impl and investigation)
- ⚠️ GPT-5.4 at N>5 scale (need N=30 for DAO-13 falsification criterion)
- ⚠️ Sonnet post-protocol completion rates (zero agents in archive)
- ⚠️ GPT-5.4 scope control under tighter prompting (task 1 scope explosion)
- ⚠️ GPT-5.4 under concurrent load (tested serially, not parallel)

**What would change this:**

- GPT-5.4 investigation stall rate >30% at N=10 → not viable for reasoning skills
- GPT-5.4 scope explosion rate >50% → needs prompt engineering before production routing
- Sonnet achieving >90% Phase:Complete → viable as cheaper same-backend alternative
- Anthropic restoring subscription OAuth → eliminates the strategic motivation for multi-model

---

## Recommendation Matrix

### Worker Routing by Skill Type

| Path | Skill Type | Status | Evidence Base | Recommendation |
|------|-----------|--------|---------------|----------------|
| Claude/Opus | investigation | ✅ GO | 97% (N=37, 7d) | **Default** |
| Claude/Opus | architect | ✅ GO | 93% (N=15, 7d) | **Default** |
| Claude/Opus | feature-impl | ✅ GO | 100% (N=4, 7d) | **Default** |
| Claude/Opus | systematic-debugging | ✅ GO | 100% (N=2, 7d) | **Default** |
| Claude/Opus | research | ✅ GO | 100% (N=1, 7d) | **Default** |
| **GPT-5.4/Codex OAuth** | **feature-impl** | **✅ GO** | **80% first-attempt, 100% retry (N=5)** | **Overflow route** |
| GPT-5.4/Codex OAuth | investigation | ⚠️ PROVISIONAL | 50% first-attempt (N=2), 100% retry | **Needs N=10 test** |
| GPT-5.4/Codex OAuth | architect/debugging | ⚠️ UNTESTED | 0 agents | **Test first** (N=5) |
| Claude/Sonnet | any | ⚠️ UNTESTED | 0 post-protocol agents | **Test first** (N=10) |
| GPT-5.2-codex | any | ❌ NO-GO | 67.5% stall rate (N=123) | **Deprecated** |
| GPT-4o | any | ❌ NO-GO | 87.5% stall rate (N=16) | **Not viable** |

### Go/No-Go Thresholds

| Threshold | Decision |
|-----------|----------|
| ≥80% Phase:Complete (N≥5) | Viable as **overflow route** (use when Opus is rate-limited) |
| ≥90% Phase:Complete (N≥10) | Viable as **default for implementation skills** |
| 50-79% Phase:Complete | **Manual-only** escape hatch (`--model` flag) |
| <50% Phase:Complete | **Not viable** for production routing |

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Keep Opus/Claude Code as default for all skills | implementation | Status quo, proven reliability |
| Run GPT-5.4 stall test (N=5) | implementation | Standard validation, no architectural impact |
| Run Sonnet test (N=10) | implementation | Same backend, same subscription, no risk |
| Route implementation skills to GPT-5.4 if >80% pass | strategic | Changes cost structure and provider dependency |
| Add Codex OAuth for flat-rate GPT-5.4 | strategic | $200/mo commitment, Dylan decides |

### Recommended Approach: Sequential Validation

**15-minute benchmark protocol** — the smallest action that produces decision-quality data.

**Why this approach:**
- Zero data exists for alternatives; any test is infinitely more informative than none
- Infrastructure is ready; the only blocker is operational
- The test is reversible — failing GPT-5.4 tasks just mean "not ready yet"

**Implementation sequence:**

1. **Start OpenCode:** `orch-dashboard start` (or `opencode &` manually)
2. **Spawn 5 GPT-5.4 feature-impl tasks:**
   ```bash
   orch spawn --model gpt-5.4 feature-impl "add IsReasoningModel test for gpt-5.4 in model_test.go"
   orch spawn --model gpt-5.4 feature-impl "add comment explaining skillModelMapping in skill_inference.go"
   orch spawn --model gpt-5.4 investigation "verify DAO-13 claim: measure SPAWN_CONTEXT token count for GPT-5.4"
   orch spawn --model gpt-5.4 feature-impl "add gpt-5.4 to model-selection.md guide"
   orch spawn --model gpt-5.4 feature-impl "add GPT-5.4 context window size to model.go ModelSpec"
   ```
3. **Wait 15-30 min, then check:** `orch status` — count Phase:Complete agents
4. **Score:** Apply go/no-go thresholds from matrix above

**Also test Sonnet (same day, parallel):**
```bash
orch spawn --model sonnet feature-impl "add IsReasoningModel test for o3-mini in model_test.go"
orch spawn --model sonnet investigation "verify current SYNTHESIS.md compliance rate in archive"
orch spawn --model sonnet feature-impl "add Sonnet to model-selection.md guide"
```

### Alternative Approaches Considered

**Option B: Skip testing, commit to Claude-only**
- **Pros:** No testing overhead, proven path
- **Cons:** Total Anthropic lock-in, no overflow route, no cost optimization
- **When to use instead:** If multi-model routing is deprioritized

**Option C: Large-scale benchmark (N=30 per model)**
- **Pros:** Statistically significant, meets DAO-13 falsification criterion
- **Cons:** Expensive ($50-100 in API costs), takes hours, premature before N=5 validation
- **When to use instead:** After N=5 shows >60% completion, before committing to default routing

**Rationale for sequential approach:** N=5 is the minimum sample that gives directional signal. If 0/5 complete, we've saved hours of wasted testing. If 5/5 complete, we have strong signal to invest in N=30.

---

### Implementation Details

**What to implement first:**
- Start OpenCode server (prerequisite for all GPT-5.4 testing)
- Spawn N=5 GPT-5.4 tasks (smallest useful sample)
- Spawn N=3 Sonnet tasks (verify same-backend alternative)

**Things to watch out for:**
- ⚠️ OPENAI_API_KEY auth may have different rate limits than Codex OAuth — watch for 429 errors
- ⚠️ OpenCode server may need rebuild if code changed since last build
- ⚠️ GPT-5.4 context window is 1.05M tokens but SPAWN_CONTEXT is only ~8K tokens — context isn't the constraint, protocol compliance is
- ⚠️ Feature-impl SYNTHESIS compliance is 4% even on Opus — don't use SYNTHESIS as completion metric for the benchmark

**Success criteria:**
- ✅ GPT-5.4 N=5 test executed with measurable Phase:Complete rate
- ✅ Sonnet N=3 test executed with measurable Phase:Complete rate
- ✅ Results update the recommendation matrix with empirical data
- ✅ Routing decision made based on observed rates vs thresholds

---

## References

**Files Examined:**
- `.beads/issues.jsonl` - 1,978 issue records, completion tracking data
- `.orch/archive/workspace/*/AGENT_MANIFEST.json` - 130 post-protocol manifests (all Opus)
- `pkg/model/model.go:127-139` - GPT-5.4 model aliases
- `pkg/spawn/resolve.go` - Backend resolution logic
- `pkg/daemon/skill_inference.go:260-284` - Skill-to-model mapping
- `~/.local/share/opencode/auth.json` - Provider auth (Anthropic only)
- `.kb/models/daemon-autonomous-operation/claims.yaml:230-246` - DAO-13 non-Anthropic stall claim
- `.kb/models/agent-lifecycle-state-model/model.md:469-477` - Stall rate comparison table
- `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md` - Original stall audit (N=1,655)
- `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` - GPT-5.4 routing investigation
- `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md` - OpenClaw migration thread

**Commands Run:**
```bash
# Dry-run GPT-5.4 spawn (routing verification)
orch spawn --dry-run --model gpt-5.4 feature-impl "test: add a comment to model.go"

# Dry-run Sonnet spawn (fallback routing verification)
orch spawn --dry-run --model sonnet feature-impl "test: add a comment"

# OpenCode server health check
curl -s http://localhost:4096/health

# Beads data mining (Python scripts analyzing issues.jsonl)
# Archive manifest analysis (Python scripts scanning AGENT_MANIFEST.json)
```

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md` - Strategic context for multi-model migration
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` - GPT-5.4 infrastructure readiness
- **Issue:** `orch-go-rj8hi` - Prior GPT-5.4 stall test (blocked by auth)
- **Model:** `.kb/models/daemon-autonomous-operation/claims.yaml` (DAO-13) - Non-Anthropic stall rate claim

---

## Investigation History

**2026-03-26:** Investigation started
- Initial question: Which worker model/backend combinations are operationally viable for orch-go?
- Context: orch-go-1dhv8 spawned to benchmark Claude Code, GPT-5.4, and fallback paths

**2026-03-26:** Data mining complete
- Discovered 100% Opus monoculture in post-protocol era (130/130 manifests)
- Established Claude Code baseline: 93-100% Phase:Complete on labeled skills
- Found zero GPT-5.4 empirical data; prior test (orch-go-rj8hi) auth-blocked

**2026-03-26:** Infrastructure verification complete
- GPT-5.4 dry-run passes (routes to opencode backend correctly)
- OPENAI_API_KEY is set; OpenCode server is down
- Sonnet dry-run passes (routes to claude backend, same as Opus)

**2026-03-26:** Investigation completed
- Status: Complete
- Key outcome: Only Opus/Claude Code is validated. GPT-5.4 and Sonnet require a 15-minute empirical test that Dylan can run. Recommendation matrix and benchmark protocol provided.
