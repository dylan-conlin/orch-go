# Decision: Explore Case Working by Doing It, Not Building For It

**Date:** 2026-02-25
**Status:** accepted
**Deciders:** Dylan
**Follows:** `2026-02-24-career-focus-domain-translation-over-infrastructure.md`

## Decision

Explore the investigation → model → probe pattern by **doing investigation work firsthand** in an accessible field, rather than building tools for investigators in fields where Dylan has no domain knowledge or contacts.

## Context

### The Insight

Building orch-go's knowledge infrastructure revealed a powerful workflow pattern:

1. **Investigations** produce findings (individual fact-gathering)
2. **Synthesis** produces models (externalized understanding)
3. **Probes** test specific claims against models (targeted verification)
4. Cycle repeats, producing increasingly accurate understanding

This pattern maps to "case working" — the core workflow of lawyers, investigators, compliance analysts, journalists, debuggers, and researchers.

### What Was Explored and Rejected

| Idea | Why rejected |
|------|-------------|
| Legal software product | No contacts in legal, no domain knowledge, competing against Harvey AI ($100M+), Relativity, CaseText |
| Open source case-working tool | Still needs users and domain understanding; open source replaces "will they pay?" with "will they adopt?" — equally hard without knowing the user |
| General "case working" platform | Same problem as "general orchestration" — a pattern, not a market. Nobody buys "case working tools"; they buy legal case management or fraud investigation software |

### The Trap Pattern Observed

Each iteration (legal software → open source tool → general platform) was a new container for the same underlying excitement about the pattern. The insight was looking for a home, not responding to a discovered need. This is the shape described in the Visionary Trap — a grand insight connecting disparate fields that feels uniquely yours.

**What's real:** The pattern genuinely exists across domains. The discipline infrastructure (verification, knowledge persistence, session amnesia compensation) is genuinely hard and most AI tools skip it.

**What's not validated:** Whether anyone outside Dylan's specific context experiences this as a pain point worth paying for.

### The Path That Produced Orch-Go's Principles

Orch-go's principles didn't come from theory or from building for an imagined user. They came from **doing the work and hitting walls**. Session amnesia was discovered by running agents. Verification bottleneck was discovered by trusting agent output. Accretion gravity was discovered by watching spawn_cmd.go grow to 2,000 lines.

The same path applies here: do investigation work, discover the real pain, see what tools you reach for that don't exist.

## The Plan

### Fields Where Investigation Work Is Accessible (No Credentials Required)

| Field | Accessibility | How to start |
|-------|--------------|--------------|
| **Investigative journalism** | High — publish on blog, no license needed | Pick something to investigate, publish findings |
| **OSINT** | High — active hobbyist community | Bellingcat-style open source analysis |
| **Bug bounty hunting** | High — HackerOne, Bugcrowd | Investigate systems for vulnerabilities, get paid |
| **Competitive intelligence** | Already doing this | Price Watch at SCS was exactly this |

### What To Watch For While Doing It

- Where does the investigation → model → probe pattern help vs. feel like overhead?
- What tools do you reach for that don't exist?
- Where does AI help and where does it hallucinate in ways that break the workflow?
- What's the difference between your workflow and what a non-technical investigator would do?
- Is the pain in the investigation, the synthesis, the verification, or the organization?

### Success Criteria

- Complete at least one real investigation (not a toy example)
- Identify specific friction points from firsthand experience
- Talk to 5 people who do investigation work professionally
- Determine whether there's a product here or whether this is personal knowledge that makes you better at other things

## Key Principle

The product question is premature. The next step is not building — it's doing. If a product exists here, it will emerge from the same process that produced orch-go: sustained practice revealing real problems.

## Provenance

Emerged from a conversation examining whether orch-go's knowledge infrastructure patterns could become a product. Multiple product framings were tested (legal software, open source tool, general platform) and rejected for lacking domain knowledge and market validation. The "do it yourself" path was identified as the approach most consistent with how orch-go's own insights were discovered.
