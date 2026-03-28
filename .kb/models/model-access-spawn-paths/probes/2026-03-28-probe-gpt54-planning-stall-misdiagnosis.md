# Probe: GPT-5.4 Planning Stall Misdiagnosis — Silent Deaths, Not Phase Stalls

**Model:** model-access-spawn-paths
**Date:** 2026-03-28
**Status:** Complete
**claim:** N/A (bug report claim, not formal model claim)
**verdict:** contradicts

---

## Question

The bug report claims "GPT-5.4 agents consistently stall in Planning phase" based on two instances (orch-go-95x1b, orch-go-mlqje). Does the evidence support this diagnosis, and what's the actual GPT-5.4 failure pattern in production?

---

## What I Tested

### 1. Verified orch-go-95x1b (Prior Art Check) via OpenCode API

```bash
curl -s http://127.0.0.1:4096/session/ses_2cb05e1e9ffeHLuCMgpTJy99Ei | python3 ...
# metadata.model: "openai/gpt-5.4", spawn_mode: "headless"
# additions: 0, deletions: 0, files: 0

curl -s http://127.0.0.1:4096/session/ses_2cb05e1e9ffeHLuCMgpTJy99Ei/message | python3 ...
# Messages: 11, model confirmed gpt-5.4 via providerID/modelID
# Assistant messages from 1774710300 to 1774710333 (~33 seconds of active work)
```

Cross-referenced with beads comments:
- Phase: Planning at 08:05
- DISCOVERED: governance file at 08:05
- Phase: BLOCKED at 08:05

### 2. Verified orch-go-mlqje (Kenning trademark search)

```bash
curl -s http://127.0.0.1:4096/session | python3 -c "... search for mlqje ..."
# NO OpenCode session found for orch-go-mlqje

bd show orch-go-mlqje
# Labels: skill:research, effort:small
# Phase: Planning at 14:33, then nothing
```

Checked model routing rules in `pkg/daemon/skill_inference.go:285`:
```go
"research": CapabilityDeepReasoning,
```
And `RouteModel()` at line 327:
```go
case CapabilityDeepReasoning:
    return ModelRoute{Model: "opus", Reason: ...}
```

Research skill → `CapabilityDeepReasoning` → routes to `opus`, NOT `gpt-5.4`.

### 3. Full GPT-5.4 production session analysis

```bash
curl -s http://127.0.0.1:4096/session | python3 -c "..."
# Fetched all 24 GPT-5.4 sessions + message counts + content analysis
```

### 4. Checked stall clustering by date

Analyzed session creation timestamps and grouped by date to detect infrastructure-related clustering.

---

## What I Observed

### Finding 1: orch-go-95x1b was NOT stalled

The agent produced 11 messages over ~33 seconds of active work. It correctly:
1. Read the task and SPAWN_CONTEXT.md
2. Identified the governance-protected worker-base path
3. Reported Phase: BLOCKED with a clear reason
4. Continued providing useful context about the required changes

The "UNRESPONSIVE BLOCKED" label in `orch status` reflects the BLOCKED state, not a stall. The agent functioned correctly — its task was blocked by a real constraint.

### Finding 2: orch-go-mlqje was likely NOT GPT-5.4

No OpenCode session exists for this agent. The `skill:research` label routes through `CapabilityDeepReasoning` → `opus` in the daemon model routing. The agent was likely spawned on Claude backend with Opus, not OpenCode with GPT-5.4.

The `daemon:verification-failed` label and missing workspace suggest the spawn either failed during setup or the Claude/tmux session died early. This is a different failure class than GPT-5.4 silent death.

### Finding 3: GPT-5.4 production failure rate is 29%, not 5%

| Classification | Count | Percentage |
|----------------|-------|------------|
| Functional (real assistant output) | 17 | 71% |
| Silent death (≤2 messages, 0-char assistant) | 5 | 21% |
| Retried death (≤4 messages, multiple 0-char attempts) | 2 | 8% |
| **Total failures** | **7** | **29%** |

Silent death signature: User sends prompt (~572-593 chars), assistant "responds" with 0 chars, 0 tool calls, completes in <10ms. The assistant message is structurally present but empty.

### Finding 4: Failures cluster dramatically by date

| Date | Functional | Failed | Failure Rate | N |
|------|-----------|--------|--------------|---|
| 03-24 | 0 | 4 | 100% | 4 |
| 03-26 | 14 | 3 | 18% | 17 |
| 03-27 | 1 | 0 | 0% | 1 |
| 03-28 | 2 | 0 | 0% | 2 |

Mar 24: All 4 sessions created within 7 seconds (batch spawn), all failed — suggests OpenCode server or API issue during that window.
Mar 26: 3 failures among 17 sessions — some during an early batch (08:27), most of the day's sessions succeeded.
Mar 27-28: 3/3 sessions functional.

### Finding 5: Controlled test vs production divergence

The N=21/95% stat comes from controlled tests (Mar 26-27 benchmarks) using direct API calls with careful model override. Production spawns go through the full `DispatchSpawn` → `runSpawnHeadless` pipeline with concurrent agents, server state, and real infrastructure. The 29% production failure rate vs 5% controlled rate suggests the failures are infrastructure-triggered, not model-capability issues.

---

## Model Impact

- [x] **Contradicts** the reported symptom: Neither cited agent is evidence of "GPT-5.4 Planning stalls." orch-go-95x1b worked correctly (governance block). orch-go-mlqje was likely routed to opus via skill capability routing, not GPT-5.4.

- [x] **Extends** model with: **Failure Mode 10: GPT-5.4 Zero-Token Silent Death in Production (Mar 2026)**
  - Symptom: Session created, prompt accepted, assistant "responds" with 0 text, 0 tools, completes in <10ms
  - Production rate: 29% (7/24 sessions) but clusters on specific dates/moments
  - Not a model capability issue — controlled tests show 95% success (N=21)
  - Likely infrastructure-triggered: OpenCode server instability, API rate limits, or concurrent spawn overload
  - Matches the known "zero-token silent death" pattern from `.kb/investigations/2026-03-26-inv-design-retry-strategy-gpt-silent.md`
  - Fix path: The fingerprinted one-shot retry from the retry strategy investigation would address this

- [x] **Extends** model with: **CLAUDE.md stat needs update**
  - Current: "GPT-5.4 is significantly better (95% completion, N=21)"
  - Should add: "Production sessions show ~29% zero-token silent death rate, clustering on infrastructure events. Controlled tests (N=21) show 95% but don't reflect concurrent production conditions."

---

## Notes

### Recommendations

1. **Update CLAUDE.md GPT-5.4 reliability stat** to reflect production conditions
2. **Implement the fingerprinted retry** from inv-design-retry-strategy-gpt-silent — this would recover most silent deaths
3. **Investigate Mar 24 batch failure** — all 4 sessions created at 08:54 failed; check if OpenCode server had issues at that time
4. **Add model verification to bug reports** — before filing "GPT-5.4 stall" issues, verify via `bd show` + OpenCode API that the agent was actually GPT-5.4

### What this investigation did NOT test

- Whether the OpenCode server was restarting/unstable during the failure windows
- Whether the silent deaths correlate with specific concurrent agent counts
- Whether the zero-token responses are from OpenAI API errors or OpenCode session handling bugs
- The actual spawn mechanism for orch-go-mlqje (no workspace or OpenCode session to examine)
