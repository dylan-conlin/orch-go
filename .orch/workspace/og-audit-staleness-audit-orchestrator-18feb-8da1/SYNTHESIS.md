# Session Synthesis

**Agent:** og-audit-staleness-audit-orchestrator-18feb-8da1
**Issue:** orch-go-1036
**Duration:** 2026-02-18
**Outcome:** success

---

## Plain-Language Summary

Audited the orchestrator skill source (`orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`) against every CLI command it references. Found 13 stale references: 7 actively harmful (teach flags/commands that don't exist and will error), 4 misleading (teach wrong mental model of defaults), and 2 cosmetic. The highest-impact issue is the `--opus` flag (should be `--model opus`) which is recommended for 6 skill types — every orchestrator that follows this guidance gets an error. Second-highest is `orch frontier` in the mandatory session hygiene checkpoint — a command that doesn't exist.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcome: every stale reference was verified by running the actual CLI help command and observing the error or discrepancy.

---

## TLDR

Found 13 stale CLI references in the orchestrator skill: 7 harmful (non-existent commands/flags), 4 misleading (wrong defaults), 2 cosmetic. The `--opus` flag and `orch frontier` command are the most impactful because they appear in high-frequency orchestrator workflows.

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md` - Comprehensive probe with all 13 findings, severity ratings, proposed replacements, and root cause analysis

### Files Modified
- None (audit only — no changes to skill files)

### Commits
- (pending commit of probe file)

---

## Evidence (What Was Observed)

- `orch frontier --help` → exit 1: "unknown command"
- `orch rework --help` → exit 1: "unknown command" (suggests `orch work`)
- `orch reflect --help` → exit 1: "unknown command"
- `orch kb archive-old --help` → no such subcommand (kb has: ask, extract)
- `orch spawn --help` → no `--opus` flag listed; correct flag is `--model opus`
- `orch spawn --help` → default backend is `claude` (tmux-based), not `opencode` (headless)
- `orch clean --help` → no `--stale` or `--untracked` flags
- `bd label --help` → requires subcommand (`bd label add`), not `bd label <id> <label>`
- `pkg/model/model.go:19-22` → DefaultModel is sonnet (confirms skill's model default)
- `orch spawn --help` → `--bypass-triage` is required for manual spawns

### Tests Run
```bash
# Verification was CLI help output, not code tests
orch frontier --help   # exit 1 - confirms non-existence
orch rework --help     # exit 1 - confirms non-existence
orch reflect --help    # exit 1 - confirms non-existence
orch spawn --help      # No --opus flag in flag list
orch clean --help      # No --stale or --untracked flags
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md` - Full staleness audit with proposed replacements

### Decisions Made
- Severity classification: "HARMFUL" = teaches flags/commands that error; "MISLEADING" = teaches wrong defaults; "COSMETIC" = minor inaccuracies

### Constraints Discovered
- The skill's Model Selection table and spawn mode descriptions haven't been updated since the `--backend` flag was introduced
- The Tools & Commands reference list (template line 621-627) contains 4 non-existent commands
- The `--bypass-triage` requirement is inconsistently applied across skill examples

### Externalized via `kb`
- (pending: will run kb quick after self-review)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix orchestrator skill CLI staleness (13 references)
**Skill:** feature-impl
**Context:**
```
The probe at .kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md
documents 13 stale CLI references in the orchestrator skill template at
orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template.
Each finding has the stale text, correct replacement, and template line number.
Priority order: fix HARMFUL items first (findings 1-7), then MISLEADING (8-11).
After editing, run `skillc build` then `skillc deploy` to propagate changes.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the orchestrator skill document the `--backend` flag dimension explicitly? Currently it doesn't mention backends at all.
- Several useful commands exist but aren't referenced in the skill (hotspot, learn, deploy, reconcile, changelog, patterns) — should they be added?
- The worker-base skill uses deprecated `bd comment` syntax — should that be audited separately?

**What remains unclear:**
- Whether `orch frontier` was removed intentionally or was never implemented. No git history checked.
- Whether the opencode backend is still actively used or if claude is now the primary path for all spawn types.

---

## Session Metadata

**Skill:** codebase-audit (organizational dimension)
**Model:** opus
**Workspace:** `.orch/workspace/og-audit-staleness-audit-orchestrator-18feb-8da1/`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md`
**Beads:** `bd show orch-go-1036`
