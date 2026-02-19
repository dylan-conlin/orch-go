# Probe: Skillc Pipeline Audit

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-18
**Status:** Complete

---

## Question

How does the skillc pipeline (source -> compile -> deploy -> agent load) actually route orchestrator skill content, and where do stale copies diverge from the load path?

---

## What I Tested

```bash
pwd
which skillc
file /Users/dylanconlin/bin/skillc
stat -f "%Sm %N" /Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/.opencode/skill/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/.opencode/skill/SKILL.md \
  /Users/dylanconlin/.claude/skills/src/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/.claude/skills/skills/src/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/Documents/personal/orch-cli/skills/orchestrator/SKILL.md \
  /Users/dylanconlin/.config/opencode/agent/orchestrator.md
```

Reviewed skillc deploy and compiler behavior in:
- `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go`
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go`

Reviewed orch-go spawn load path in:
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/orch/extraction.go`

---

## What I Observed

- `skills.DefaultLoader()` hardcodes `~/.claude/skills`, and spawn loads skill content once at spawn time via `LoadSkillWithDependencies()`.
- `skillc deploy` uses `relPath = filepath.Rel(absSourcePath, baseDir)`; if deploy is run from inside a skill directory, output lands at `target/SKILL.md` instead of `target/meta/orchestrator/SKILL.md`.
- Multiple orchestrator SKILL.md copies exist on disk with different timestamps; the canonical orch-go load target is `~/.claude/skills/meta/orchestrator/SKILL.md`.

---

## Model Impact

- [x] **Extends** model with: deploy-root-relative pathing explains why skill fixes land in `~/.opencode/skill/SKILL.md` when run from inside a skill directory, and why orch-go only uses `~/.claude/skills/**/SKILL.md` at spawn time.
- [ ] **Confirms** invariant: [which one]
- [ ] **Contradicts** invariant: [which one] — [what's actually true]

---

## Notes

[Any additional context, caveats, or follow-up questions]
