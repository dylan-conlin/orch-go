# Investigation Examples

**When to use:** Consult when writing findings, D.E.K.N. summary, or when stuck on how to structure evidence.

## D.E.K.N. Summary Examples

**Good D.E.K.N.:**
```markdown
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation (tactical fix within existing patterns, no architectural impact)
```

**Guidelines:**
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)

---

## Common Test Failures

**"Logical verification" is not a test.**

Wrong:
```markdown
## Test performed
**Test:** Reviewed the code logic
**Result:** The implementation looks correct
```

Right:
```markdown
## Test performed
**Test:** Ran `time orch spawn investigation "test"` 5 times
**Result:** Average 6.2s, breakdown: 70ms orch overhead, 5.5s Claude startup
```

**Speculation is not a conclusion.**

Wrong:
```markdown
## Conclusion
Based on the code structure, the issue is likely X.
```

Right:
```markdown
## Conclusion
The test confirmed X is the cause. When I changed Y, the behavior changed to Z.
```

---

## Evidence Hierarchy Examples

**Primary evidence (this IS the evidence):**
- Actual code: `grep "function" src/*.ts`
- Test output: `go test -v ./pkg/...`
- Observed behavior: "Clicked button, saw error in console"

**Secondary evidence (claims to verify):**
- Workspace saying "feature X NOT DONE" - verify by searching codebase
- Investigation claiming "Y doesn't exist" - search before concluding
- Decision doc stating "we chose Z" - verify Z is actually implemented

**The failure mode:** An agent reads a workspace claiming "feature X NOT DONE" and reports that as a finding without checking if feature X actually exists in the code.
