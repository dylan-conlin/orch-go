# Investigation Templates

**Purpose:** Record what you tried, what you observed, and whether you tested your hypothesis.

---

## Default: Simple Template

**Use for everything.** The simple template is the default because it focuses on what actually matters: testing before concluding.

```bash
orch create-investigation topic-name
```

Creates: `.orch/investigations/simple/YYYY-MM-DD-topic-name.md`

### Template Structure

```markdown
# [Topic]

**Date:** YYYY-MM-DD
**Status:** Active | Complete

## Question
[What are you trying to figure out?]

## What I tried
- [Action 1]
- [Action 2]

## What I observed
- [Observation 1]
- [Observation 2]

## Test performed
**Test:** [What you did to validate/falsify]
**Result:** [What happened]

## Conclusion
[Only fill if you tested]
```

### The Key Discipline

**You cannot conclude without testing.**

The "Test performed" section is mandatory. If you didn't test your hypothesis, you don't get to fill in "Conclusion." This is the one rule that prevents false conclusions.

---

## What Investigations Are For

1. **Timeline/history** - Record what you tried so the next session doesn't start from zero
2. **Discoverability** - `orch search "proxy"` finds prior work
3. **Amnesia protection** - Some record exists for the next Claude instance

## What Investigations Are NOT For

1. **Confidence calibration** - Self-assessed confidence is meaningless
2. **Synthesis workflows** - Synthesis compounds errors (see deprecation notice)
3. **Elaborate artifacts** - 15-section templates don't prevent wrong conclusions

---

## File Organization

```
.orch/investigations/
  simple/           # Default location
    2025-11-25-auth-flow.md
    2025-11-25-proxy-timeout.md
```

**Naming:** `YYYY-MM-DD-kebab-case-topic.md`

**Search:** `orch search "keywords"` or `rg "pattern" .orch/investigations/`

---

## Legacy Templates (Not Recommended)

These 256-line templates still exist for backward compatibility but are not recommended. They encourage elaborate artifacts over actual testing.

```bash
# Legacy usage (not recommended)
orch create-investigation topic --type systems
orch create-investigation topic --type feasibility
orch create-investigation topic --type audits
orch create-investigation topic --type performance
orch create-investigation topic --type agent-failures
```

**Why deprecated:** Case study showed 5 investigations using these templates all reached wrong conclusions despite "High" and "Very High" confidence. The templates optimized for artifact quality, not truth-finding.

See: `.orch/investigations/systems/2025-11-23-document-codex-hang-epistemic-debt.md`

---

## Key Principle

**Test before concluding.**

No template structure, confidence calibration, or synthesis workflow prevents false conclusions. Only empirical testing does.
