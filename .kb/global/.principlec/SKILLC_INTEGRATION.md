# skillc Integration Requirements

**Status:** Pending implementation
**Created:** 2026-01-21
**Context:** principlec built, need skillc to support principle includes

---

## The Requirement

Skills should be able to include specific principles at compile time, eliminating inline summaries that drift from source.

### Manifest Schema Addition

```yaml
# In skill.yaml
includes:
  principles:
    - session-amnesia
    - evidence-hierarchy
    - premise-before-solution
```

### What skillc Should Do

1. Read `includes.principles` from skill.yaml
2. For each principle name, find the file in `~/.kb/.principlec/src/{category}/`
3. Embed the content into compiled SKILL.md under a `## Principles` section
4. If principle not found, warn (don't fail build)

---

## Principle File Locations

```
~/.kb/.principlec/src/
├── foundational/
│   ├── provenance.md
│   ├── session-amnesia.md
│   ├── self-describing-artifacts.md
│   ├── progressive-disclosure.md
│   ├── surfacing-over-browsing.md
│   ├── evidence-hierarchy.md
│   ├── gate-over-remind.md
│   ├── capture-at-context.md
│   ├── track-actions-not-just-state.md
│   ├── pain-as-signal.md
│   ├── infrastructure-over-instruction.md
│   ├── asymmetric-velocity.md
│   ├── verification-bottleneck.md
│   ├── understanding-lag.md
│   ├── coherence-over-patches.md
│   ├── perspective-is-structural.md
│   ├── authority-is-scoping.md
│   ├── escalation-is-information-flow.md
│   └── friction-is-signal.md
├── system-design/
│   ├── local-first.md
│   ├── compose-over-monolith.md
│   ├── graceful-degradation.md
│   ├── share-patterns-not-tools.md
│   └── observation-infrastructure.md
└── meta/
    ├── evolve-by-distinction.md
    ├── reflection-before-action.md
    ├── pressure-over-compensation.md
    ├── understanding-through-engagement.md
    ├── premise-before-solution.md
    └── strategic-first-orchestration.md
```

---

## Example Compiled Output

Given this skill.yaml:
```yaml
name: architect
includes:
  principles:
    - session-amnesia
    - evidence-hierarchy
```

skillc would produce SKILL.md with:
```markdown
## Principles

### Session Amnesia

Every pattern in this system compensates for Claude having no memory between sessions.

**The test:** "Will this help the next Claude resume without memory?"
...

---

### Evidence Hierarchy

Code is truth. Artifacts are hypotheses.
...

---
```

---

## Skills to Update After Implementation

These skills currently have inline principle summaries that should become includes:

| Skill | Current | After |
|-------|---------|-------|
| orchestrator | "Principles Quick Reference" table | `includes.principles: [...]` |
| architect | Lines 31-44 inline list | `includes.principles: [session-amnesia, evidence-hierarchy, premise-before-solution]` |
| meta-orchestrator | Scattered references | `includes.principles: [perspective-is-structural, escalation-is-information-flow, pressure-over-compensation]` |
| design-session | References principles.md | `includes.principles: [...]` |

---

## Implementation Notes

- Principle lookup: Search all category directories (foundational, system-design, meta) for `{name}.md`
- No category required in manifest - skillc finds the right directory
- Provenance table NOT included in skill includes (stays in full principles.md only)
- Compile-time embedding ensures no drift between source and skill content
