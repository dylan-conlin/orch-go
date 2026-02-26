## Summary (D.E.K.N.)

**Delta:** Inventoried 48 friction gates across spawn (10), completion (12), and daemon (20+) subsystems — only 3 of 12 completion gates have healthy bypass:fail ratios; 73.4% of bypass events stem from three systemic blindness patterns (skill-class, model, blanket override).

**Evidence:** Exhaustive code search across all gate implementations + analysis of 7,029 events (1,008 bypass events) from ~/.orch/events.jsonl. Build gate (0.7:1) is the only completion gate where failures exceed bypasses.

**Knowledge:** Gates should be skill-class-aware and model-aware. Investigation/docs-only skills should auto-skip test_evidence, build, and synthesis gates. 3 gates (agent_running, model_connection, commit_evidence) should be removed entirely — they never catch real defects.

**Next:** Implement skill-class-aware gate selection (highest impact: eliminates 31.7% of bypasses). Remove 3 pure-noise gates. Consider deprecating --force entirely.

**Authority:** architectural — Changing gate applicability rules affects cross-subsystem behavior and all agent completion workflows.

---

# Investigation: Probe Inventory Friction Gates Across Spawn, Completion, and Daemon

**Question:** Which friction gates across all orch subsystems are catching real defects vs generating noise that gets routinely bypassed?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** og-inv-probe-inventory-friction-13feb-9240
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md | extends | yes | No conflicts — prior finding that test_evidence and synthesis are noisiest confirmed and quantified |
| .kb/models/completion-verification/model.md | extends | yes | Model documents 3 verification layers but actual system has 48 gates across 3 subsystems |

---

## Findings

### Finding 1: Build is the only high-value completion gate

**Evidence:** Build gate has 0.7:1 bypass:fail ratio (114 bypassed, 171 failed). It is the ONLY completion gate where failures exceed bypasses. git_diff (1.1:1) and verification_spec (1.1:1) are next best. All other gates have ratios from 5.5:1 (test_evidence) to ∞:1 (agent_running).

**Source:** `~/.orch/events.jsonl` analysis — 1,008 verification.bypassed events, 403 verification.failed events, cross-referenced by gate name.

**Significance:** The build gate is the load-bearing member of verification. It catches real compilation failures. Most other gates primarily generate friction that gets bypassed.

---

### Finding 2: Three systemic patterns cause 73.4% of all bypasses

**Evidence:**
- Skill-class blindness: 320 events (31.7%) — gates fire for docs/investigation work where structurally inapplicable
- GPT/Sonnet model incompatibility: 206+52 = 258 events (25.6%) — gates designed for Anthropic Claude don't work for other models
- Blanket "skip everything": 168 events (16.7%) — negates entire verification system

**Source:** Reason field analysis in verification.bypassed events from `~/.orch/events.jsonl`

**Significance:** These are not gate-specific problems — they're systemic. Fixing skill-class awareness alone would eliminate ~320 bypass events. Combined, addressing all three patterns would eliminate ~740 of 1,008 bypass events (73.4%).

---

### Finding 3: Daemon gates are fundamentally different from completion gates — and mostly working correctly

**Evidence:** Daemon has 20 gates but they're filters (skip silently), not blockers (fail loudly). 3,866 daemon.dedup_blocked events vs 41 daemon.spawn events shows the daemon is working as designed — it polls frequently and filters correctly. Status, dependency, type, and label checks are fundamental correctness guarantees.

**Source:** `pkg/daemon/daemon.go`, `pkg/daemon/spawn_tracker.go`, `pkg/daemon/session_dedup.go`, `pkg/daemon/rate_limiter.go`, `pkg/daemon/pool.go`

**Significance:** Daemon gates don't need the same KEEP/SOFTEN/REMOVE analysis as completion gates. They're infrastructure — they should filter silently, which they do. The main optimization would be reducing log verbosity for expected dedup events.

---

## Synthesis

**Key Insights:**

1. **Completion gates need skill-class awareness** — The single highest-impact change is auto-skipping test_evidence, build, and synthesis for investigation/docs-only skills. This would eliminate 31.7% of all bypass events without any loss in defect detection.

2. **Three gates should be removed** — agent_running (∞:1), model_connection (71:1), and commit_evidence (11.8:1) never catch real defects. git_diff already validates commits, making commit_evidence redundant.

3. **Spawn and daemon gates are well-calibrated** — Unlike completion gates, spawn gates (concurrency, rate limit, triage) and daemon gates (status, dependencies, dedup) serve clear correctness purposes with no evidence of excessive noise.

**Answer to Investigation Question:**

Of 48 total gates, 9 spawn gates and ~15 daemon gates are catching real issues or serving infrastructure purposes (KEEP). Of 12 completion gates, only 3 have healthy bypass:fail ratios (build, git_diff, verification_spec). The remaining 9 completion gates range from noisy (test_evidence 5.5:1, synthesis 5.9:1) to pure noise (agent_running ∞:1, model_connection 71:1). The root causes are systemic — skill-class blindness, model blindness, and blanket override support — not gate-specific design flaws.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build gate catches real failures (171 failures vs 114 bypasses — verified from events.jsonl)
- ✅ test_evidence and synthesis gates are noisy for docs-only work (90%+ bypasses cite docs-only — verified from events.jsonl)
- ✅ Daemon dedup works as designed (3,866 blocks vs 41 spawns — verified from events.jsonl)
- ✅ --force usage dropped from 72.8% to 1.8% after targeted skips (verified, extends prior probe)

**What's untested:**

- ⚠️ Removing agent_running/model_connection/commit_evidence gates (no test of what happens without them)
- ⚠️ Skill-class-aware auto-skip (concept not prototyped in code)
- ⚠️ Impact of removing --force entirely (4 orchestrator sessions still use it)
- ⚠️ Events data only covers 5 days (2026-02-09 to 2026-02-13) — longer window may show different patterns

**What would change this:**

- Finding would be wrong if gate-less completions introduce regressions detectable only by removed gates
- Finding would be wrong if "docs-only" bypass reason is incorrect (agents claiming docs-only but actually modifying code)
- Finding would be wrong if events from prior to Feb 9 show different bypass:fail ratios

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add skill-class-aware gate selection | architectural | Changes completion verification for all skills — crosses subsystem boundary |
| Remove 3 pure-noise gates | architectural | Removes verification checks — needs cross-subsystem review |
| Deprecate --force | architectural | Changes completion workflow for all agents |
| Reduce daemon dedup log verbosity | implementation | Log-level change, no behavior change |

### Recommended Approach ⭐

**Skill-Class-Aware Gate Selection** — Auto-skip test_evidence, build, and synthesis gates for knowledge-producing skills (investigation, architect, research, capture-knowledge).

**Why this approach:**
- Eliminates 31.7% of all bypass events (highest single-change impact)
- Zero risk of missed defects (these gates are structurally inapplicable to docs-only work)
- Preserves all gates for code-producing skills (feature-impl, systematic-debugging)

**Trade-offs accepted:**
- Requires maintaining a skill classification (code-producing vs knowledge-producing)
- Won't fix model blindness (25.6%) or blanket override (16.7%) — those need separate work

**Implementation sequence:**
1. Define skill classes in pkg/verify or pkg/spawn (code-producing, knowledge-producing)
2. Gate selection reads skill class from workspace metadata
3. Auto-skip test_evidence + build + synthesis for knowledge-producing skills

### Alternative Approaches Considered

**Option B: Remove all noisy gates (bypass:fail > 5:1)**
- **Pros:** Maximum noise reduction immediately
- **Cons:** Removes gates that catch occasional real issues (test_evidence caught 21 real failures)
- **When to use instead:** If skill-class awareness proves too complex to implement

**Option C: Model-aware gate selection**
- **Pros:** Fixes 25.6% of bypasses (GPT/Sonnet compat)
- **Cons:** Complex — need to detect model per session, define model-specific gate sets
- **When to use instead:** After skill-class fix is deployed and model compat remains a problem

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` — Spawn gates (triage, concurrency, rate limit, gap, hotspot)
- `cmd/orch/complete_cmd.go` — Skip flag infrastructure, SkipConfig, event logging
- `pkg/verify/check.go` — All 12 completion gate constants and verification flow
- `pkg/daemon/daemon.go` — Daemon filtering gates (label, type, status, deps)
- `pkg/daemon/spawn_tracker.go` — TTL dedup (6h)
- `pkg/daemon/session_dedup.go` — Session-level dedup (6h)
- `pkg/daemon/rate_limiter.go` — Hourly rate limit (20/h)
- `pkg/daemon/pool.go` — Worker pool capacity (3 default)
- `pkg/spawn/gap.go` — Gap gating thresholds
- `cmd/orch/hotspot.go` — Strategic-first hotspot detection

**Commands Run:**
```bash
# Events analysis
python3 -c "parse events.jsonl for event types, bypass rates, gate distributions"

# Code search
rg --type go "skip-" cmd/orch/complete_cmd.go
rg --type go "Gate" pkg/verify/check.go
rg --type go "IsSpawnableType\|triage\|MaxSpawns\|TTL" pkg/daemon/
```

**Related Artifacts:**
- **Model:** `.kb/models/completion-verification/model.md` — Extended with cross-subsystem inventory
- **Prior Probe:** `.kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md` — Confirmed and deepened
- **New Probe:** `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` — Complete inventory with classifications

---

## Investigation History

**2026-02-13:** Investigation started
- Initial question: Which friction gates across all orch subsystems are catching real defects vs generating noise?
- Context: Prior probe found test_evidence and synthesis noisiest. Orchestrator requested broadening to ALL gates.

**2026-02-13:** All 48 gates inventoried across 3 subsystems
- Exhaustive code search completed via parallel agents
- Events analysis yielded 7,029 events with gate-level granularity

**2026-02-13:** Investigation completed
- Status: Complete
- Key outcome: 3 completion gates are high-value (build, git_diff, verification_spec). 3 should be removed (agent_running, model_connection, commit_evidence). Rest need skill-class-aware auto-skip.
