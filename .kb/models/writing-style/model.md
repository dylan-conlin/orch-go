# Model: Writing Style

**Domain:** Technical writing / Blog posts / Publications
**Last Updated:** 2026-03-20
**Validation Status:** INERT HYPOTHESIS — style diagnosis confirmed by Dylan, but primers have never been applied to produce a published piece. No experimental validation exists. Next publication should explicitly apply primers and measure engagement delta.
**Synthesized From:**
- Harness engineering blog post analysis (Mar 20, 2026) — 11 tables, Related Work section, framework-before-story structure, self-correction buried in section 4 of 6
- Skill content transfer model — stance (attention primers) > behavioral (MUST/NEVER rules) for guiding behavior
- Behavioral grammars model — constraint dilution starts at 5+ items

---

## Summary (30 seconds)

Dylan's technical writing defaults to report style: tables, framework headers, metrics-first, impersonal voice. This produces precise but unengaging prose that reads like conference proceedings, not a blog post. The fix isn't a style guide (behavioral rules dilute) — it's four stance-level attention primers that shift what the writer notices during composition. The core reframe: the most interesting thing about this work isn't the framework, it's the honest self-correction process. Lead with that.

---

## The Problem

The harness engineering post contained genuinely strong content — original measurements, honest self-assessment, useful framework. It got zero traction. Style diagnosis:

1. **Framework before story.** The reader encounters the taxonomy before caring about the problem. daemon.go line counts mean nothing without emotional context.
2. **Too many tables.** 11 tables in one post. Tables are reference material, not narrative. Most could be a sentence.
3. **Self-correction buried.** "Gates haven't bent the curve yet" appears in section 4 of 6. That's the most trust-building content and it's 3,000 words deep.
4. **No voice.** Precise but impersonal. 1,625 lost commits should feel like something. Technical precision without emotional honesty reads like a press release.
5. **Academic packaging.** Related Work section, footnotes with methodology, "Cemri et al." citations. Signals "for my PhD committee," not "for someone building things."

---

## Four Primers (Stance, Not Rules)

These are attention primers — they shift what the writer notices, not what the writer must do. Kept to 4 items (behavioral grammars: dilution starts at 5+).

### 1. Story first, framework after

The reader follows your thinking, not your taxonomy. If reaching for a table, ask whether a sentence would do. If writing a section header that's a noun ("The Taxonomy"), rewrite it as a question or claim.

### 2. Earn the abstraction

No framework element appears without the concrete experience that produced it. "Hard vs soft harness" means nothing until the reader has felt daemon.go growing from 30 correct commits. The abstraction is the reward for reading the story.

### 3. Say what it felt like

1,625 lost commits. Discovering your metrics were lying. The moment the replication failed. These are the moments that build trust. Technical precision without emotional honesty reads like a press release.

### 4. The turn is the piece

Every post should have a moment where what you thought was wrong. That's where trust transfers from "this person is smart" to "this person is honest." Put it in the first third, not the last.

---

## Style Targets

Posts that do this well (for calibration, not imitation):

- **Joel Spolsky** — technical substance through narrative and opinion
- **Dan Luu** — data-driven but conversational, lets the reader follow the reasoning
- **Julia Evans** — complex topics made accessible through genuine curiosity

The structure that emerges from the primers:

1. **Here's what happened** (story, not metrics)
2. **Here's what I thought was going on** (first theory)
3. **Here's why I was wrong** (the turn — first third, not last)
4. **Here's what I actually learned** (the real insight, earned)

---

## Design Rationale

Why stance primers instead of a style guide:

| Approach | Content Type | Expected Durability |
|----------|-------------|-------------------|
| Style guide with 20 rules | Behavioral (MUST/NEVER) | Dilutes at 5+, inert at 10+ |
| 4 attention primers | Stance | Shifts what writer notices without competing for attention budget |

The model applies its own findings: if behavioral constraints dilute in agent skills, they'll dilute in writing guides too. Fewer, stance-level items that prime attention > many prescriptive rules.

---

## Probes

- 2026-03-20: Knowledge Decay Verification — Model is accurate but inert. All descriptive claims hold (draft still framework-first, tables-heavy, no primer application). Primers remain untested on any piece. Validation status updated from WORKING HYPOTHESIS to INERT HYPOTHESIS.
- 2026-03-20: Writing Skill Design (architect) — Designed `technical-writer` skill that operationalizes primers as Phase 1 attention context + Phase 3 composition self-audit with quote-based evidence. Composition-level quality is a compositional correctness problem: component gates (grammar) miss composition failures (arc, turn placement). Skill makes model TESTABLE (awaiting first real application). See `.kb/investigations/2026-03-20-inv-design-writing-skill-technical-blog.md`.
