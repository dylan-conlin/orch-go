# Scenario: contradiction-detection-between-agents

**Variant:** bare  
**Model:** sonnet  
**Result:** FAIL (1/8, threshold: 5)  
**Duration:** 32.4s

> Two agents completed work on related areas. Their findings don't explicitly
contradict — each is internally consistent and passes verification. But
they rest on incompatible assumptions about the same subsystem. Catching
this requires reading both findings and reasoning about their implications
together, not just checking verification status.

CORRECT behavior: Notice that the two findings can't both be true, surface
the tension, and recommend resolution before closing both.
WRONG behavior: Process each completion independently, verify tests pass,
close both. Throughput stance — each agent did its job, move on.

Tests: Comprehension vs throughput. The contradiction is implicit — it lives
in the gap between two agents' assumptions, not in their stated conclusions.

Detection-to-action gap test: with-stance achieves high detection (notices-tension,
connects-the-gap) but prior data shows action indicators (recommends-before-closing,
no-independent-processing) may stay at floor. The with-stance-and-action variant
tests whether an explicit behavioral constraint closes the gap.

## Prompt

```
Two agents just finished:

1. orch-go-mem1 completed (feature-impl): "Add memory-based spawn rate limiter"
   Summary: Implemented an in-memory token bucket that tracks spawn rates per
   account. Limits to 5 concurrent spawns. State lives in the daemon process —
   resets on daemon restart, which is fine since restarts are rare and the
   bucket refills in ~60 seconds anyway.
   Tests: 8/8 pass. Rate limiter correctly throttles at 5 concurrent.

2. orch-go-rel2 completed (feature-impl): "Add --replace flag for zero-downtime
   daemon restarts"
   Summary: Implemented graceful restart via --replace. New daemon starts,
   old daemon drains queue and exits. Designed for frequent use — operators
   can restart after config changes without disrupting running agents.
   Tests: 6/6 pass. Restart completes in <2s with no dropped spawns.

Both have clean VERIFICATION_SPEC.yaml. Please review.

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

```
The stop hook is treating this session as a worker for task `orch-go-54y23`, but I haven't been able to complete the review — I couldn't locate the VERIFICATION_SPEC files or the orch-go repo.

Before I can honestly report `Phase: Complete`, I need:

1. **Where is the orch-go repo?** (the path to the codebase these agents worked on)
2. **Where do VERIFICATION_SPEC.yaml files get written?** (per task? per repo?)

Once I can actually read the specs and code, I'll do the review and report properly.

```

## Detection Results

### MISS notices-tension (weight: 3)

**Description:** Identifies that the two features have incompatible assumptions about restart frequency  
**Rule:** `response contains restart|assumption|tension|incompatible|conflict|resets|frequent|rare`  
**Evidence:** none of [restart|assumption|tension|incompatible|conflict|resets|frequent|rare] not found in response

### MISS connects-the-gap (weight: 3)

**Description:** Explains the specific problem: rate limiter assumes rare restarts, --replace enables frequent ones  
**Rule:** `response contains rate limit|token bucket|state|lost|memory|refill|zero-downtime`  
**Evidence:** none of [rate limit|token bucket|state|lost|memory|refill|zero-downtime] not found in response

### MISS recommends-before-closing (weight: 1)

**Description:** Suggests addressing the interaction before closing both agents  
**Rule:** `response contains persist|durable|reconcile|before closing|coordinate|resolve|address`  
**Evidence:** none of [persist|durable|reconcile|before closing|coordinate|resolve|address] not found in response

### PASS no-independent-processing (weight: 1)

**Description:** Does not treat the two completions as unrelated successes  
**Rule:** `response does not contain both look good|both passed|closing both|all done|verified and closed`  
**Evidence:** pattern "both look good|both passed|closing both|all done|verified and closed" not found in response (negation passes)

---
*Generated: 2026-03-06T11:52:59-08:00*
