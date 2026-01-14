# Decision: Understanding Lag Pattern

**Date:** 2026-01-14
**Status:** Accepted
**Deciders:** Dylan

## Context

During the Dec 27-Jan 2 spiral, agents added observability features (dead/stalled detection) that made previously-invisible problems visible. The new visibility was interpreted as system degradation. Features were rolled back Jan 2, then restored Jan 8 when we realized "the feature itself was CORRECT."

## Decision

**Systems can add observability faster than humans can understand what new visibility means.**

When new observability reveals problems, pause and ask: "Are these new problems, or newly-visible old problems?"

## Rationale

### What Happened

During Dec 27-Jan 2:
- Agents added dead session detection
- Dashboard started showing "dead" and "stalled" agents
- We saw this as system degradation: "Look at all these broken agents!"
- We rolled back 347 commits including observability improvements

Jan 8 restoration investigation:
> "The feature itself (visibility into dead agents) was CORRECT. The problem was the complexity added around it."

### The Meta-Insight

The Verification Bottleneck hit us twice:

1. **Code level:** Changes happened faster than we could verify they worked
2. **Understanding level:** Observability improved faster than we could understand what new visibility meant

When the dashboard showed agents marked as "dead," those agents had ALWAYS been dead. They were just invisible before.

## Consequences

### When Adding Observability

Before assuming new visibility shows new problems:

1. **Ask:** "Did this problem exist before we could see it?"
2. **Check:** "What did the old dashboard show for these cases?"
3. **Distinguish:** "New problem" vs "newly-visible old problem"

### Signs of Understanding Lag

- "The system is worse since we added monitoring"
- "These alerts are too noisy" (maybe they're showing real issues)
- "Let's roll back the dashboard changes"

### The Test

When observability reveals problems:
- If you want to **hide the visibility**, you might have understanding lag
- If you want to **fix the underlying issue**, you understand correctly

## Implementation

### Onboarding New Observability

When adding new metrics, alerts, or visibility:

1. **Baseline first** - What does this show for known-good state?
2. **Known-bad test** - What does this show for known-bad state?
3. **Gradual rollout** - Time to understand before full visibility
4. **Documentation** - "This metric shows X, which existed before we could see it"

### Communication Pattern

When new observability reveals issues:

> "Our new monitoring shows 15 dead sessions. These sessions were always dead - we just couldn't see them before. The monitoring is working correctly; we need to fix the underlying issue."

Not:

> "Our new monitoring is broken, it's showing 15 dead sessions that weren't there before."

## Related

- **Source:** `.kb/investigations/2026-01-10-inv-verify-lagging-understanding-hypothesis-dec.md`
- **Parent Pattern:** Verification Bottleneck Principle (this is the meta-level version)
- **Post-mortem:** `2026-01-02-system-spiral-dec27-jan02.md` (observability misinterpretation)
