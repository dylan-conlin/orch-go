# Session Synthesis

**Agent:** og-inv-audit-stalled-agent-28feb-4fcb
**Issue:** orch-go-0jgj
**Duration:** 2026-02-28 11:27 → 2026-02-28 ~12:00
**Outcome:** success

---

## Plain-Language Summary

Audited 1655 archived workspaces and 131 active workspaces for agents that never reached completion. The headline "56.6% stall rate" is misleading — the true stall rate is 4.3% (19 agents). The dominant category is agents that completed their work but didn't write SYNTHESIS.md (194 agents, a protocol compliance gap). Of the 19 true stalls, 15 involved non-Anthropic models (GPT-4o, GPT-5.2-codex) that couldn't follow the multi-step worker protocol. The investigation produced a 6-mode failure taxonomy with actionable recommendations: SYNTHESIS enforcement gate, phase timeout detection, model protocol gating, QUESTION response channel, prior art dedup at spawn time, and persistent spawn dedup.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- Investigation file covers 10+ stalled instances with evidence-backed taxonomy: `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md`
- Probe file extends agent-lifecycle-state-model with 6 new failure modes: `.kb/models/agent-lifecycle-state-model/probes/2026-02-28-probe-stalled-agent-failure-pattern-audit.md`

---

## TLDR

Scanned 1786 workspaces for stalled agents. Found 6 distinct failure modes: SYNTHESIS compliance gap (194), silent failure (228), model protocol incompatibility (15), QUESTION deadlock (5), prior art confusion (1), concurrency ceiling stall (1). True stall rate is 4.3%, not 56.6%. Non-Anthropic models account for 79% of true stalls. Recommendations: enforce SYNTHESIS at completion, add phase timeouts, gate protocol-heavy skills to Anthropic models.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md` — Full investigation with 6-mode taxonomy, frequency counts, correlations, and recommendations
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-28-probe-stalled-agent-failure-pattern-audit.md` — Probe extending model with 5 new failure modes beyond "Agent Went Idle But Not Complete"
- `.orch/workspace/og-inv-audit-stalled-agent-28feb-4fcb/SYNTHESIS.md` — This file

### Files Modified
- None (read-only investigation)

### Commits
- (pending — will commit before completing)

---

## Evidence (What Was Observed)

- 1655 archived workspaces scanned: 717 with SYNTHESIS.md, 938 without
- 441 post-protocol-era agents without SYNTHESIS.md examined individually
- 194 agents reported Phase: Complete but no SYNTHESIS.md (compliance gap, not stall)
- 228 agents never reported any phase (133 from pre-phase-reporting Sonnet era, Feb 14-17)
- 19 agents stalled in non-terminal phases: Implementing (5), QUESTION (5), Planning (4), BLOCKED (1), Exploration (1)
- 15 of 19 true stalls used non-Anthropic models (GPT-4o: 87.5% stall rate, GPT-5.2-codex: 67.5%)
- Duplicate spawn storms: up to 10 retries for same slug, 5 agents for same beads ID in 2 minutes
- orch-go-nn43 (the triggering case): Opus architect stuck in Exploration after discovering prior agents already completed overlapping work

### Tests Run
```bash
# No code tests — this is a read-only investigation
# Evidence gathered via:
find .orch/workspace/archived/og-*/ -maxdepth 1 -name "SYNTHESIS.md" | wc -l  # 717
ls -d .orch/workspace/archived/og-*/ | wc -l  # 1655
bd show <beads-id> | grep "Phase:"  # For ~441 agents
```

---

## Architectural Choices

No architectural choices — this was a read-only audit investigation. Recommendations are captured for a follow-up architect task.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md` — Taxonomy of 6 failure modes with evidence
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-28-probe-stalled-agent-failure-pattern-audit.md` — Model extension

### Constraints Discovered
- Non-Anthropic models can't reliably follow multi-step worker protocols (phase reporting, SYNTHESIS, completion flow)
- QUESTION deadlock is structural — no automated response channel exists for headless agents
- Spawn dedup cache is in-memory, doesn't survive daemon restarts, causing duplicate spawn storms

### Externalized via `kb quick`
- (will externalize key findings via kb quick before completing)

---

## Next (What Should Happen)

**Recommendation:** close + spawn-follow-up

### If Close
- [x] All deliverables complete (investigation + probe)
- [x] Tests passing (N/A — read-only investigation)
- [x] Investigation file has findings and conclusion
- [x] Ready for `orch complete orch-go-0jgj`

### If Spawn Follow-up
**Issue:** Design orchestrator-level diagnostic responses for stalled agent failure modes
**Skill:** architect
**Context:**
```
Investigation orch-go-0jgj identified 6 failure modes for stalled agents. The 3 highest-impact
actionable responses are: (1) SYNTHESIS.md enforcement gate in orch complete, (2) phase timeout
detection for headless agents, (3) QUESTION response channel. See investigation at
.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md for full taxonomy.
```

---

## Unexplored Questions

- What is the cost (in tokens/dollars) of the 194 SYNTHESIS compliance gap agents? They completed work but the knowledge isn't externalized.
- Do true stalls correlate with spawn context size? Hypothesis: larger spawn contexts increase stall probability for non-Opus models.
- The 133 silent Sonnet agents from Feb 14-17: did any of them produce useful work that was captured through other channels?
- Should failed/stalled agents automatically create beads issues for their failure mode, enabling trend analysis?

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-audit-stalled-agent-28feb-4fcb/`
**Investigation:** `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md`
**Probe:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-28-probe-stalled-agent-failure-pattern-audit.md`
**Beads:** `bd show orch-go-0jgj`
