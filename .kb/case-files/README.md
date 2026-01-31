# Case Files

Case files are manually-curated timelines that tell the story of complex, multi-agent investigations.

## Purpose

When multiple agents work on the same problem over time, contradictions pile up and the big picture gets lost. Case files visualize:

- **Timeline**: All investigations and commits in chronological order
- **Contradictions**: Where agents reached conflicting conclusions
- **Human observations**: What the user actually experienced
- **Evidence trail**: Links to artifacts (investigations, commits, decisions)
- **The failure mode**: What pattern led to the outcome

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

- **coaching-plugin-worker-detection.html** - 19 investigations, 34 commits, 3 weeks, disabled. Canonical example of agents not building on each other's work.

## Future Work

This is a manual spike to test whether case files are useful. If successful, could build:
- Automated case file generation from beads issue history
- "Same topic" detection (naming patterns, issue links)
- Case file schema for metadata
- Dashboard integration

See: `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` for the design session that led to this spike.
