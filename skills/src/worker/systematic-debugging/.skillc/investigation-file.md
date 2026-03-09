## Model Awareness (Probe vs Investigation Routing)

**Before creating any artifact, check SPAWN_CONTEXT.md for model-claim markers.**

### Detection

Find the `### Models (synthesized understanding)` section in SPAWN_CONTEXT.md. Look for injected model-claim markers in model entries:
- `- Summary:`
- `- Critical Invariants:` or `- Constraints:`
- `- Why This Fails:` or `- Failure Modes:`

### If markers are present → Probe Mode

Your debugging findings likely confirm, contradict, or extend an existing model's failure modes.

- Pick the most relevant model from the injected models section
- Create: `.kb/models/{model-name}/probes/{date}-{slug}.md`
- Use template: `.orch/templates/PROBE.md`
- Required sections: `Question`, `What I Tested`, `What I Observed`, `Model Impact`
- Focus the probe on which model claim (especially failure modes) your debugging confirms or contradicts

**Example:** Debugging a spawn failure when the spawn model documents "Failure Mode 2: Header Injection Conflicts" → create a probe testing whether that failure mode explains the current bug.

### If markers are absent → Investigation Mode

Follow standard investigation workflow below.

---

## Investigation File (Optional for Simple Bugs)

Investigation files are **recommended** for complex bugs but **optional** for simple fixes.

### When to Create

**Create when:**
- Multi-step root cause analysis needed
- Multiple hypotheses to test
- Findings should be preserved
- Pattern may recur (for synthesis)

**Skip when:**
- Bug is obvious and localized (typo, wrong variable)
- Fix completes in <15 minutes
- Root cause immediately clear from error
- Commit message can fully document fix

### Create Template (if needed)

```bash
kb create investigation "debug/topic-in-kebab-case" --model <model-name>  # or --orphan
```

**After creating:**
1. Fill Question field with specific bug description
2. Document findings progressively during Phases 1-4
3. Update Confidence and Resolution-Status as you progress
4. Set Resolution-Status when complete (Resolved/Mitigated/Recurring)

### Commits-Only Completion

If skipping investigation file, ensure descriptive commits:
- Include "why" not just "what"
- Example: `fix: handle null session in auth middleware - was causing silent failures when Redis connection dropped`
