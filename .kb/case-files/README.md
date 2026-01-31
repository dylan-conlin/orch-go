# Case Files

Case files are diagnosis-first narratives that explain complex, multi-agent investigation failures.

## Purpose

When multiple agents work on the same problem over time, contradictions pile up and the big picture gets lost. Case files answer:

1. **What went wrong?** (verdict first - don't make them scroll)
2. **Where's the proof?** (contradiction section + ground truth)
3. **Why did it keep failing?** (named failure mode with pattern)
4. **What should we do differently?** (lessons + what should have happened)

Case files visualize:

- **The Verdict**: Outcome, root cause, the pattern
- **The Contradiction**: Side-by-side conflicting conclusions (impossible to miss)
- **The Ground Truth**: What the user actually saw (screenshots, quotes)
- **The Timeline**: Compressed by week (not the centerpiece)
- **The Failure Mode**: Named pattern with diagnosis
- **The Evidence Trail**: Links to artifacts
- **What Should Have Happened**: Specific intervention points
- **Lessons For Next Time**: Actionable takeaways

## When to Create a Case File

Create a case file when:
- 5+ investigations on the same topic
- Contradictory conclusions between agents
- Work that "keeps breaking" despite fixes
- Human loses faith in agent conclusions
- Need to understand "what's been tried" before continuing

## How to Use

Case files are standalone HTML files. Open in a browser to view the full timeline with visual styling.

Example: `open .kb/case-files/coaching-plugin-worker-detection.html`

## Existing Case Files

- **coaching-plugin-worker-detection.html** - 19 investigations, 34 commits, 3 weeks, disabled. Diagnosis-first structure reveals "Verification Gap" failure mode: agents self-certified fixes without end-to-end validation while Dylan kept seeing the bug.

## Future Work

This is a manual spike to test whether case files are useful. If successful, could build:
- Automated case file generation from beads issue history
- "Same topic" detection (naming patterns, issue links)
- Case file schema for metadata
- Dashboard integration

See: `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` for the design session that led to this spike.
