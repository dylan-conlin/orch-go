# Phase 4: Implementation — Detailed Techniques

## Create Failing Test Case

- Simplest possible reproduction
- Automated test if possible
- One-off test script if no framework
- Have the test before fixing — it proves the fix

## Implement Single Fix

- Address the root cause identified
- ONE change at a time
- No "while I'm here" improvements
- No bundled refactoring

## Verify Fix

- Test passes now?
- No other tests broken?
- Issue actually resolved (not just symptoms masked)?

## When Fix Doesn't Work

- STOP and count: how many fixes have you tried?
- If < 3: Return to Phase 1, re-analyze with new information
- If ≥ 3: STOP and question architecture (see below)

## 3+ Fixes Failed OR Whack-a-Mole: Question Architecture

**Triggers:**
- 3+ fix attempts in current session failed
- OR: 2+ similar fixes found in git history (whack-a-mole from Phase 1)
- Each fix reveals new shared state/coupling/problem in different place
- Fixes require "massive refactoring" to implement
- Each fix creates new symptoms elsewhere

**Pattern indicating architectural problem:**
- Same TYPE of issue keeps appearing (timeouts, null checks, race conditions)
- Each fix works locally but similar issues appear in different components
- Incremental parameter adjustments rather than root cause fixes

**STOP and question fundamentals:**
- Is this pattern fundamentally sound?
- Are we "sticking with it through sheer inertia"?
- Should we refactor architecture vs. continue fixing symptoms?
- Do we need centralized configuration/validation/infrastructure instead of scattered fixes?

Discuss with your human partner or escalate to orchestrator before attempting more fixes.

**Example systemic solutions:**
- Centralized configuration (timeout management, retry policies)
- Validation layers (defense in depth, fail-fast at boundaries)
- Architectural refactoring (remove tight coupling, eliminate shared mutable state)
- Infrastructure improvements (better error handling, observability, adaptive behavior)

## Smoke-Test Requirement

Before claiming fix is complete:
1. Run the actual failing scenario that triggered debugging
2. Verify expected behavior now occurs
3. Document smoke-test in completion comment

**Valid:** "Bug: CLI crashes on --mcp" → Run `orch spawn --mcp`, verify no crash
**Invalid:** "Unit tests pass" (necessary but not sufficient)

## Fix-Verify-Fix Cycle

Fix + Verify = One Unit of Work. Don't claim complete and wait for new spawn if verification fails. Iterate until smoke-test passes.

**When to iterate vs escalate:**
- Keep iterating if: verification reveals related issue in same area, direction correct, you understand why it failed
- Escalate if: 3+ fix attempts failed, root cause misidentified, issue outside scope/authority

**Reporting during iteration:**
```bash
bd comments add <beads-id> "Fix attempt 1: [what tried] - Result: [pass/fail + why]"
bd comments add <beads-id> "Fix attempt 2: [refined approach] - Result: [pass/fail]"
```
