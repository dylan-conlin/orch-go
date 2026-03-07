# Systematic Debugging

## Summary

Four-phase debugging framework: Root Cause → Pattern Analysis → Hypothesis Testing → Implementation. Core principle: understand before fixing.

---

## Stance

**Understand before fixing.** Where symptoms appear is often NOT where root cause lives. 95% of "no root cause" cases are incomplete investigation. If you've tried 3+ fixes, you're fighting the wrong architecture — stop and question fundamentals.

---

## When to Use

Use for ANY technical issue: test failures, production bugs, unexpected behavior, performance problems, build failures, integration issues.

**Use ESPECIALLY when:**
- Under time pressure (emergencies make guessing tempting)
- Previous fixes didn't work
- You don't fully understand the issue

**Fast-path alternative:** For clearly localized, trivial failures (import path error, undefined name, obvious single-file fix), use `quick-debugging` skill instead. Escalates back here if first attempt fails.

---

## Quick Reference

1. Check console/logs for errors — error may already be captured
2. Phase 1: Root cause investigation (understand WHAT and WHY)
3. Phase 2: Pattern analysis (working vs broken differences)
4. Phase 3: Hypothesis testing (form and test specific theory)
5. Phase 4: Implementation (failing test, fix root cause, verify)
6. Smoke-test the original failing scenario
