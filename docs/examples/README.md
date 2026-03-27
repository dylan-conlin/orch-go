# Example Artifacts

Real artifacts from the orch-go knowledge base, cleaned of project-specific IDs and annotated with inline comments explaining metadata fields.

These demonstrate four artifact types used in thread-first knowledge management:

## Artifact Types

### [Briefs](briefs/)
Short narrative summaries of completed work. Three sections — **Frame** (why this matters), **Resolution** (what was found/built), **Tension** (what remains unresolved). Written for a human reader who wasn't in the session.

### [Threads](threads/)
Living documents that track an evolving line of thinking across sessions. Entries are dated. A thread can be `open` (still developing), `resolved` (merged into a decision, model, or parent thread), or `stale`. Threads are the primary thinking artifact — they capture the *development* of understanding, not just the conclusion.

### [Decisions](decisions/)
Architectural or process decisions with context, options considered, and consequences. These are *commitments* — they constrain future work. Decisions reference evidence (investigations, probes, measurements) and can be amended by later decisions.

### [Models](models/)
Synthesized understanding of a domain, built from evidence across multiple investigations and sessions. Models have a validation status (working hypothesis, confirmed, overclaimed) and are updated by *probes* — targeted experiments that test specific claims. A model is never "done" — it evolves as evidence accumulates.

## How These Artifacts Relate

```
Threads (thinking develops)
    |
    v
Decisions (commitments made)     Models (understanding synthesized)
    |                                |
    v                                v
Briefs (work summarized)         Probes (claims tested)
```

Threads generate decisions and feed models. Models are tested by probes. Briefs summarize the work that implements decisions. The knowledge base grows through this cycle, not through top-down planning.
