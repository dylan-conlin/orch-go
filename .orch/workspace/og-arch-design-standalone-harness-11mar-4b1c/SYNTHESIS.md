# Session Synthesis

**Agent:** og-arch-design-standalone-harness-11mar-4b1c
**Issue:** orch-go-sb13k
**Duration:** 2026-03-11T17:00Z → 2026-03-11T19:00Z
**Outcome:** success

---

## TLDR

Designed a standalone `harness` CLI extracted from orch-go that gives blog post readers a working tool for Day 1 multi-agent governance. The extraction is clean: 3 packages (`pkg/verify/accretion*.go`, `pkg/control/`, standalone functions from `harness_init.go`) have zero orch-go dependencies and can be copied with only import path changes, plus ~800 lines of fresh code for CLI scaffolding, templates, and the report command.

## Plain-Language Summary

The harness engineering blog post describes how to govern multi-agent codebases, but readers finish it with nothing to try. This design solves that by extracting the governance tools already built inside orch-go into a standalone binary called `harness` with three commands: `harness init` (scaffolds deny rules, pre-commit growth gates, control plane locks, and a CLAUDE.md governance template), `harness check` (scans for bloated files manually), and `harness report` (shows file growth velocity and gate firing history). The key finding is that orch-go already has a clean standalone mode in `harness_init.go` — the extraction is lifting this tested code path into its own binary, not reimplementing it. Three core packages have zero orch-go imports and can be copied directly. The CLI targets anyone running 10+ AI agents on a shared codebase, installable via `go install`, with no dependency on orch, beads, or any orchestration infrastructure.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- Investigation file complete with extraction map, repo structure, command specs, and 7 decomposed implementation issues
- 5 of 7 blog post Day 1 items implementable standalone (2 require issue tracker, explicitly out of scope)
- Zero orch-go imports in extracted packages (verified by reading imports)

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-11-inv-design-standalone-harness-cli-extracted.md` — Full investigation with findings, extraction map, repo structure, command specs, and decomposed implementation issues

### Files Modified
- None (architect session — design only, no code changes)

---

## Architectural Choices

### Separate repo vs monorepo subpackage
- **What I chose:** New repo `github.com/dylan-conlin/harness`
- **What I rejected:** Subpackage within orch-go
- **Why:** Blog post constraint requires zero orch-go dependency; readers need `go install` from a standalone repo
- **Risk accepted:** Maintaining two copies of accretion logic (orch-go and harness)

### Extract + adapt vs rewrite
- **What I chose:** Copy 3 zero-dependency packages, write ~800 lines fresh
- **What I rejected:** Full rewrite (would lose production-tested logic)
- **Why:** The standalone mode in harness_init.go already proves the extraction boundary works
- **Risk accepted:** Extracted code may drift from orch-go over time

### CLI flags vs config file
- **What I chose:** CLI flags only for MVP
- **What I rejected:** `.harness.yaml` config file
- **Why:** Blog post readers need 30-minute setup, not configuration. Flags are simpler.
- **Risk accepted:** Power users can't persist preferences without wrapping in a script

### Simplified events vs full extraction
- **What I chose:** Write fresh with 4 event types
- **What I rejected:** Extract full `pkg/events/logger.go` (40+ event types)
- **Why:** 36 of 40+ event types are orch-specific; extracting would bring dead code
- **Risk accepted:** Losing event type compatibility with orch-go (acceptable — they're separate tools)

---

## Knowledge (What Was Learned)

### Decisions Made
- Standalone harness CLI should be a new repo with 3 commands (init/check/report)
- Extract from 3 zero-dependency packages, write fresh where orch-coupled
- No config file in MVP, CLI flags only
- Cursor/Windsurf support deferred to post-MVP

### Constraints Discovered
- Control plane lock (chflags uchg) is macOS-only; Linux needs chattr +i (requires root)
- 2 of 7 Day 1 items (self-close gate, event emission) require issue tracking, out of standalone scope
- Claude Code settings.json location may vary by platform

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Create harness repo and implement 3 commands
**Skill:** feature-impl (7 issues decomposed in investigation)
**Context:**
```
Investigation at .kb/investigations/2026-03-11-inv-design-standalone-harness-cli-extracted.md
has full extraction map, repo structure, command specs, and 7 decomposed implementation issues.
Start with Issue 1 (repo scaffolding), then Issues 2-3 (extract packages), then Issues 4-6 (commands).
Issue 7 is integration testing.
```

---

## Unexplored Questions

- Whether 800-line threshold is right for non-Go codebases (Python, TypeScript may have different norms)
- How to handle Cursor/Windsurf settings format (different from Claude Code)
- Whether to add `harness eject` to remove all governance artifacts cleanly
- Whether `harness watch` (continuous monitoring daemon) is useful post-MVP

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-standalone-harness-11mar-4b1c/`
**Investigation:** `.kb/investigations/2026-03-11-inv-design-standalone-harness-cli-extracted.md`
**Beads:** `bd show orch-go-sb13k`
