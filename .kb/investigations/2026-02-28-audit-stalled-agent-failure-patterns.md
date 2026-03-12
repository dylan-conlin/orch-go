# Audit: Stalled Agent Failure Patterns

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** 
**Supersedes:** 
**Superseded-By:** Follow-up work tracked in beads issue orch-go-dfva4


**Date:** 2026-02-28
**Status:** Active
**Beads:** orch-go-0jgj

## D.E.K.N. Summary

- **Delta:** Identified 6 distinct failure modes from 1655 archived workspaces. True stall rate is 4.3% (19/441 manifested agents), not the apparent 56.6%. The dominant category is SYNTHESIS compliance gaps (194 agents), not actual stalls.
- **Evidence:** Full scan of workspace archives, beads phase comments, AGENT_MANIFEST.json metadata. 19 true stalls examined individually.
- **Knowledge:** Non-Anthropic models have 67-87% stall rates vs 44% for Opus. QUESTION deadlock is a structural gap. Prior art confusion wastes expensive re-exploration cycles.
- **Next:** Design orchestrator-level diagnostic responses based on the 6 failure modes (architect task). Immediate wins: phase timeout detection, SYNTHESIS enforcement gate, QUESTION response channel.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---------------|-------------|----------|-----------|
| .kb/investigations/archived/2026-01-09-inv-investigate-actual-failure-distribution-across.md | extends | yes | Prior used event-based analysis; this uses workspace-level scan — more comprehensive |
| .kb/investigations/archived/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md | extends | yes | Prior designed recovery; this audits actual failure patterns to validate assumptions |
| .kb/investigations/archived/2026-01-08-inv-restore-dead-agent-detection-surfacing.md | extends | yes | Prior focused on detection; this categorizes root causes |

## Question

What are the actual failure modes when agents stall without completing, how frequent is each, and what correlations exist with skill type, model, and task complexity?

---

## Finding 1: Dataset Overview

**1655 total archived workspaces** scanned.

| Metric | Count | Percentage |
|--------|-------|------------|
| With SYNTHESIS.md (completed) | 717 | 43.4% |
| Without SYNTHESIS.md (apparent stalls) | 938 | 56.6% |
| Pre-protocol era (no manifest) | 497 | 30.0% |
| Manifested, no SYNTHESIS | 441 | 26.6% |
| Active workspaces (not archived) | 131 | — |

The 56.6% "stall rate" is misleading. Agents from Dec 2024 through mid-Jan 2026 predate the AGENT_MANIFEST system. Of the 441 manifested agents without SYNTHESIS, most are protocol compliance issues, not true stalls.

## Finding 2: Taxonomy of Failure Modes

### Category A: Pre-Protocol Era (497 agents)

Agents spawned before manifest system, phase reporting, and SYNTHESIS requirements existed (Dec 2024 – mid-Jan 2026). These are noise in the dataset, not meaningful stalls.

### Category B: SYNTHESIS Compliance Gap (194 agents — 44% of manifested stalls)

Agents that **successfully completed** their task — reported `Phase: Complete` in beads — but failed to create SYNTHESIS.md.

**Root causes:**
- Pre-SYNTHESIS-requirement skill version (skill didn't require it)
- Agent completed work but died/disconnected before writing SYNTHESIS
- Agent didn't follow session close protocol ordering

**Impact:** Makes the workspace appear "stalled" to any system checking for SYNTHESIS.md as completion evidence. The Priority Cascade in the model correctly handles this (Phase: Complete takes priority over SYNTHESIS.md existence), but workspace-level scans misclassify these as stalls.

**Examples:**
- `og-feat-add-agent-badges-27feb-524e` — Phase: Complete with full work summary, no SYNTHESIS
- `og-feat-add-backlog-cull-26feb-3b49` — Phase: Complete, 5 unit tests passing, no SYNTHESIS
- `og-feat-add-hotspot-advisory-26feb-2684` — Phase: Complete, 6 tests, no SYNTHESIS

### Category C: Silent Failure (228 agents — no phase reported at all)

Agents with manifests that never reported any phase via `bd comment`.

**Breakdown by model/skill:**

| Skill | Model | Count |
|-------|-------|-------|
| feature-impl | Sonnet 4.5 | 133 |
| feature-impl | unknown | 31 |
| investigation | unknown | 16 |
| architect | unknown | 12 |
| investigation | Sonnet 4.5 | 11 |
| architect | Sonnet 4.5 | 11 |
| hello | GPT-5.2-codex | 5 |
| feature-impl | GPT-5.2-codex | 4 |
| systematic-debugging | unknown | 2 |
| hello | GPT-4o | 2 |
| investigation | GPT-5.2-codex | 1 |

**Root causes:**
1. **Pre-phase-reporting skill version** (dominant) — The 133 Sonnet feature-impl agents from Feb 14-17 predate the skill requiring phase reporting. All 15 sampled had 0 beads comments and closed status.
2. **Startup crash** — Agent crashed before producing any output (API error, context exhaustion on spawn context injection)
3. **Model protocol incompatibility** — GPT-4o and hello-skill agents couldn't follow phase protocol

**Key insight:** These agents' beads issues are all status=closed with 0 comments, suggesting they were batch-closed during cleanup without investigation. The work may have been completed through other agents or abandoned.

### Category D: True Phase Stalls (19 agents — 4.3% true stall rate)

The core finding. These agents reported a non-terminal phase and never progressed.

#### D1: Stuck in Implementing (5 agents)

| Workspace | Model | Issue |
|-----------|-------|-------|
| og-feat-add-codex-compatible-17feb-7088 | GPT-4o | orch-go-1019 |
| og-feat-add-orientation-frame-17feb-c605 | GPT-4o | orch-go-1005 |
| og-feat-add-orientation-frame-18feb-a337 | GPT-5.2-codex | orch-go-1005 |
| og-feat-add-orientation-frame-18feb-d0be | GPT-5.2-codex | orch-go-1005 |
| og-feat-phase-staleness-event-17feb-9dcd | GPT-5.2-codex | orch-go-4hrt |

**Pattern:** Agent entered implementing, made some progress, but never reported completion. 3 of these are retries for the same issue (orch-go-1005), all using non-Anthropic models. The issue was eventually closed — presumably by a different agent or manual intervention.

**Correlation:** 100% non-Anthropic models. Zero Opus or Sonnet agents stalled in Implementing.

#### D2: Stuck on QUESTION (5 agents)

| Workspace | Model | Issue |
|-----------|-------|-------|
| og-feat-add-type-model-18feb-1213 | GPT-5.2-codex | orch-go-fq5 |
| og-feat-add-type-model-18feb-28c8 | GPT-5.2-codex | orch-go-fq5 |
| og-feat-add-type-model-18feb-4117 | GPT-5.2-codex | orch-go-fq5 |
| og-feat-add-type-model-18feb-d0a2 | GPT-5.2-codex | orch-go-fq5 |
| og-feat-frontend-show-effective-18feb-b46e | GPT-5.2-codex | orch-go-980.4 |

**Pattern:** Agent explicitly reported `Phase: QUESTION` but no mechanism exists to deliver answers to headless agents. The issue (orch-go-fq5) was spawned 4 times, each time stalling on the same QUESTION. The dependency chain also blocked it (`depends on orch-go-bm9`).

**Structural gap:** `orch send <session-id> "message"` exists but isn't wired into any automated response channel. QUESTION stalls are invisible unless someone manually checks beads comments.

#### D3: Stuck in Planning (4 agents)

| Workspace | Model | Issue |
|-----------|-------|-------|
| og-feat-migrate-remaining-dotfile-18feb-65ea | GPT-5.2-codex | orch-go-p5d |
| og-feat-recover-work-graph-17feb-892a | GPT-5.2-codex | orch-go-988 |
| og-feat-recover-work-graph-17feb-a84e | GPT-5.2-codex | orch-go-988 |
| og-feat-spike-validate-accretion-18feb-5236 | GPT-5.2-codex | orch-go-km3 |

**Pattern:** Agent read the task, started planning, but never transitioned to implementing. Two are retries for the same issue (orch-go-988). All GPT-5.2-codex.

**Hypothesis:** The combination of complex spawn context (often 5000+ tokens) and the model's instruction-following limitations caused the agent to get lost in planning without ever starting work.

#### D4: Stuck on BLOCKED (1 agent)

| Workspace | Model | Issue |
|-----------|-------|-------|
| og-inv-repro-headless-codex-18feb-25b1 | GPT-5.2-codex | orch-go-1054 |

**Pattern:** Agent needed to spawn a headless session to reproduce a bug but hit the concurrency limit (9 active agents, max 5). Reported `CONSTRAINT` and `Phase: BLOCKED`, waited for guidance that never came.

**Structural gap:** No escalation path from BLOCKED to orchestrator attention. The agent correctly surfaced the constraint but the system has no automated response.

#### D5: Stuck in Exploration (1 agent — orch-go-nn43)

| Workspace | Model | Issue |
|-----------|-------|-------|
| og-arch-fix-kb-context-27feb-70f7 | Opus 4.5 | orch-go-nn43 |

**Pattern:** Architect agent was spawned for "Fix kb context query derivation." During Exploration, it discovered that 2 prior agents (og-feat-architect-fix-kb-27feb-fbde, orch-go-amty) had already implemented 2 of the 3 intervention points. The agent reported what it found but stalled trying to determine what remained.

**Root cause:** The spawn context didn't include information about prior agent completions. The agent had to re-discover this by reading code, which created confusion about scope. This is the "Prior Art Confusion" failure mode.

## Finding 3: Model-Specific Stall Rates

| Model | Completed (SYNTHESIS) | No SYNTHESIS | Stall Rate | True Stalls |
|-------|----------------------|--------------|------------|-------------|
| Opus 4.5 | 124 | 100 | 44.6% | 1 (Exploration) |
| Sonnet 4.5 | 85 | 183 | 68.3%* | 0 |
| GPT-5.2-codex | 40 | 83 | 67.5% | 13 |
| GPT-4o | 2 | 14 | 87.5% | 2 |

*Sonnet's high rate is inflated by 133 agents from the pre-phase-reporting era (Feb 14-17). Adjusting for this: ~50/268 = 18.7% is more accurate.

**Key insight:** Non-Anthropic models account for 15/19 true stalls (79%). They struggle with the multi-step worker protocol (phase reporting → implementation → SYNTHESIS → completion). This isn't a capability gap — it's a protocol compliance gap.

## Finding 4: Duplicate Spawn Patterns

| Slug Pattern | Spawns | Issue |
|-------------|--------|-------|
| og-feat-extract-rework-explain-14feb | 10 | Same task retried 10x |
| og-work-say-hello-confirm-09jan | 6 | Hello-world test repeated 6x |
| og-arch-daemon-dedup-failure-15feb | 5 | Same bug spawned 5x in 2 minutes |
| og-feat-test-daemon-pause-15feb | 5 | — |
| og-feat-extract-spawn-flags-16feb | 5 | — |

**Pattern:** Daemon dedup cache is in-memory, doesn't survive restarts. When the daemon restarts, it re-reads `bd ready` and re-spawns issues that already have active agents. This creates 4-9 wasted agent-sessions per occurrence.

## Finding 5: Active Workspace Stalls

Of the 131 active workspaces, many have status=closed beads issues but no SYNTHESIS.md. Most of these completed successfully but weren't cleaned up (archived). They are Category B compliance gaps, not active stalls.

True active stalls (issue still open, agent not progressing):
- **orch-go-nn43** — Architect stuck in Exploration (Prior Art Confusion)
- **orch-go-9uvw** — Debug agent stuck in Planning (just spawned today)

---

## Conclusion

### The 6 Failure Modes (ordered by frequency)

1. **SYNTHESIS Compliance Gap** (194 instances) — Agent completes work but doesn't write SYNTHESIS.md. The most common "stall" isn't a stall at all.

2. **Silent Failure / Pre-Protocol** (228 instances) — Agent never reports any phase. Mostly from pre-protocol era; some are genuine startup crashes.

3. **Model Protocol Incompatibility** (15 of 19 true stalls) — Non-Anthropic models can't reliably follow the multi-step worker protocol. GPT-4o and GPT-5.2-codex agents stall in Implementing, Planning, or QUESTION at dramatically higher rates.

4. **QUESTION Deadlock** (5 instances) — Agent correctly signals it needs input; no automated mechanism delivers answers.

5. **Prior Art Confusion** (1+ instances) — Agent discovers overlapping prior work mid-task, gets confused about remaining scope.

6. **Concurrency Ceiling Stall** (1 instance) — Agent blocked by resource limits with no escalation path.

### Recommendations for Orchestrator-Level Diagnostic Responses

These feed a design task (not implementation here):

#### Immediate Wins (low complexity)

1. **SYNTHESIS.md Enforcement Gate** — `orch complete` should reject close if SYNTHESIS.md is missing for full-tier agents. This eliminates 194 instances of false "stall" classification.

2. **Phase Timeout Detection** — If no new phase is reported within 30 minutes for headless agents, probe the session. Log a diagnostic event. Surface in `orch status` as "unresponsive" rather than "active."

3. **Model Protocol Gating** — Restrict architect and investigation skills to Anthropic models only. Non-Anthropic models have 67-87% stall rates for protocol-heavy skills. Use them for simpler, less protocol-dependent tasks.

#### Medium Complexity

4. **QUESTION Response Channel** — When an agent reports `Phase: QUESTION`, the daemon should:
   - Surface the question in `orch status` prominently
   - Optionally notify the user (desktop notification)
   - Track unanswered questions as a metric

5. **Prior Art Check at Spawn Time** — Before spawning an architect agent, check if prior agents have already completed overlapping work on the same beads issue or similar titles. Inject a "Prior Completions" section into SPAWN_CONTEXT.md.

6. **Persistent Spawn Dedup** — Move dedup cache from in-memory to disk-backed (file lock or beads tag). Survives daemon restarts.

#### Strategic (requires design)

7. **BLOCKED Escalation Path** — When an agent reports `Phase: BLOCKED`, automatically surface to the orchestrator for triage. The current system relies on manual detection.

8. **Stall Pattern Classification** — Build the failure mode taxonomy into the orchestrator's completion review. When abandoning an agent, require selecting a failure mode category. This creates a feedback loop for improving spawn decisions.

---

## Test Performed

Full workspace scan of 1786 directories (1655 archived + 131 active). For each:
- Checked SYNTHESIS.md existence
- Read AGENT_MANIFEST.json for skill and model
- Queried beads for phase comments and issue status
- Identified duplicate spawns by slug pattern matching

This is primary evidence (file system scan + beads API queries), not code review.
