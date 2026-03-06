## Self-Review (Mandatory)

After implementing fix, perform self-review before completion.

> **Automated checks** (debug statements, commit format, placeholder data, orphaned files) run at `orch complete` time — no manual check needed.

### Pattern Scope Verification

**If bug was a pattern that could exist elsewhere:**

```bash
rg "bug_pattern"  # Should be 0 or documented exceptions
```

**Skip if:** Bug was truly one-off (typo, unique logic error).

### Debugging-Specific Checks

| Check | If Failed |
|-------|-----------|
| Root cause addressed (not symptom) | Return to Phase 1 |
| No temporary workarounds ("TODO: fix properly") | Complete the fix |
| Regression test exists | Add test |
| Investigation documented | Update file |

### Security

- [ ] No hardcoded secrets
- [ ] No injection vulnerabilities

### Discovered Work

If you found related bugs, tech debt, or strategic unknowns:

```bash
bd create "description" --type bug -l triage:review
bd create "description" --type question -l triage:review
```

**Note "No discovered work" in completion if nothing found.**

### Report via Beads

```bash
bd comments add <beads-id> "Self-review passed - ready for completion"
```
