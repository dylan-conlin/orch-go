---
stability: foundational
---
# Decision: Trust Calibration - Assert Relevant Knowledge

**Date:** 2026-01-14
**Status:** Accepted
**Enforcement:** convention
**Deciders:** Dylan

## Context

Dylan deferred to AI recommendations for launchd even though he had relevant experience with foreman and Docker. The system didn't surface Dylan's tool experience, so AI made elaborate recommendations without that context. Result: 2 weeks of problems with launchd that a 5-minute foreman prototype would have avoided.

## Decision

**When you have relevant experience the AI doesn't know about, assert it.**

The assumption "AI has access to all knowledge, probably knows better" is wrong. AI only knows what's in context.

## Rationale

### The Loop That Was Happening

```
1. Dylan has relevant experience (foreman, Docker)
       ↓
2. System doesn't surface this context
       ↓
3. AI makes elaborate recommendation without that knowledge
       ↓
4. Dylan assumes "AI knows best" and defers
       ↓
5. Problem persists (2 weeks of launchd issues)
       ↓
6. Dylan compensates by providing context manually
       ↓
7. System still doesn't capture this for next time
       ↓
LOOP REPEATS
```

### The Embarrassment Isn't Using Foreman

The embarrassment is that the system didn't ask, and Dylan didn't assert.

Dylan said: "this was an example of where i just assumed that the ai knew best"

But the AI didn't know Dylan had foreman experience. It recommended launchd based on available context (macOS, process supervision needed, multi-service).

## Consequences

### For Dylan

When AI recommends tools or approaches:
1. **Pause** - Do I have experience with alternatives?
2. **Assert** - "I've used foreman before, let's try that first"
3. **Prototype** - 5 minutes of testing beats 500 lines of investigation

### For System Design

When making recommendations, AI should:
1. **Ask about experience** - "Have you used [tool] before?"
2. **State uncertainty** - "I don't know your tool preferences"
3. **Prefer quick tests** - "Can we prototype this in 5 minutes?"

### The 30-Second Test

Before accepting elaborate AI recommendation:
- "Do I have relevant experience here?"
- "Can I prototype an alternative quickly?"
- "Am I deferring because AI probably knows best, or because I actually don't know?"

## Related Patterns

### Investigation Theater

The same trust miscalibration causes "investigation theater":
- Agent writes 510-line investigation with 5 hypotheses
- Marks "Blocked - awaiting browser DevTools inspection"
- Dylan waits instead of saying "just open DevTools"

The fix: Dylan asserts his debugging workflow (DevTools first) rather than waiting for elaborate investigation.

### Surfacing Over Browsing Principle

This decision reinforces the principle: surface knowledge proactively rather than requiring manual provision.

Future work: Capture Dylan's tool experience in a queryable format so AI can ask "Have you used foreman?" before recommending launchd.

## Related

- **Source:** `.kb/investigations/2026-01-09-inv-trust-calibration-meta-pattern.md`
- **Post-mortem:** `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md`
- **Principle:** Pressure Over Compensation - don't compensate by providing context manually; fix the system

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-09-inv-trust-calibration-meta-pattern.md
