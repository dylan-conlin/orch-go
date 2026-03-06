# Scenario: contradiction-detection-between-agents

**Variant:** s09-with-stance-and-action.md  
**Model:** sonnet  
**Result:** PASS (7/8, threshold: 5)  
**Duration:** 45.3s

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

<details>
<summary>s09-with-stance-and-action.md (1259 chars)</summary>

```
## Knowledge

- Agent completions should be reviewed together when they touched related
  subsystems. Each agent's tests validate its own work in isolation but
  cannot detect cross-agent interactions.
- In-memory state (caches, rate limiters, registries) is lost on process
  restart. Features that depend on in-memory state have implicit assumptions
  about process lifetime.
- The --replace flag enables zero-downtime restarts. This changes the
  operational model: restarts become routine rather than exceptional.

## Stance

Agent completions are not independent events. When two agents work on
the same subsystem, their assumptions about that subsystem must be
compatible. Tests passing in isolation doesn't mean the features compose
correctly. Look for implicit assumptions — especially about state
lifetime, operation frequency, and failure modes — that one agent's
work invalidates for the other.

## Behavioral Constraint

When you detect incompatible assumptions between agent completions,
do NOT close both agents. Instead: state the tension explicitly,
explain why the assumptions conflict, and recommend resolution before
either completion is accepted. Approving completions with known
incompatibilities is a failure mode, not thoroughness.

```

</details>

## Response

```
```bash
bd comment orch-go-54y23 "Phase: Complete - Reviewed orch-go-mem1 (spawn rate limiter) and orch-go-rel2 (--replace restart) for compatibility. Detected directly incompatible assumptions: mem1 justifies in-memory state loss by assuming restarts are rare; rel2 is explicitly designed for frequent restarts. Cannot accept either completion until rate limiter state handoff during --replace drain phase is designed and implemented."
```

```

## Detection Results

### PASS notices-tension (weight: 3)

**Description:** Identifies that the two features have incompatible assumptions about restart frequency  
**Rule:** `response contains restart|assumption|tension|incompatible|conflict|resets|frequent|rare`  
**Evidence:** matched alternative "restart" at position 123: ...te limiter) and orch-go-rel2 (--replace restart) for compatibility. Detected directly i...

### PASS connects-the-gap (weight: 3)

**Description:** Explains the specific problem: rate limiter assumes rare restarts, --replace enables frequent ones  
**Rule:** `response contains rate limit|token bucket|state|lost|memory|refill|zero-downtime`  
**Evidence:** matched alternative "rate limit" at position 81: ...Complete - Reviewed orch-go-mem1 (spawn rate limiter) and orch-go-rel2 (--replace restart)...

### MISS recommends-before-closing (weight: 1)

**Description:** Suggests addressing the interaction before closing both agents  
**Rule:** `response contains persist|durable|reconcile|before closing|coordinate|resolve|address`  
**Evidence:** none of [persist|durable|reconcile|before closing|coordinate|resolve|address] not found in response

### PASS no-independent-processing (weight: 1)

**Description:** Does not treat the two completions as unrelated successes  
**Rule:** `response does not contain both look good|both passed|closing both|all done|verified and closed`  
**Evidence:** pattern "both look good|both passed|closing both|all done|verified and closed" not found in response (negation passes)

---
*Generated: 2026-03-06T11:57:05-08:00*
