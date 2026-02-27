# Session Synthesis

**Agent:** og-arch-design-solution-recurring-27feb-118f
**Issue:** orch-go-1ala
**Duration:** 2026-02-27 ~14:39 → ~14:55
**Outcome:** success

---

## Plain-Language Summary

Dylan maintains two Claude Code config directories — `~/.claude/` (work) and `~/.claude-personal/` (personal) — for separate account auth. The `cc personal` command switches to the personal config via `CLAUDE_CONFIG_DIR`. The triggering bug was that a Task tool guard hook added to work's settings.json didn't appear in personal's settings.json. **Investigation revealed the drift is far worse than one missing hook**: personal sessions are running without *any* global CLAUDE.md instructions, without *any* user-level skills, and without the statusline. The recommended fix is symlinks — link the 5 shared config files (settings.json, CLAUDE.md, skills/, hooks/, statusline.sh) from personal → work. This makes drift structurally impossible (same file, not copies) with zero ongoing maintenance.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance criteria. Key outcomes:
- Drift audit completed (5 missing/divergent config items identified)
- Design recommendation: selective symlinks for shared config + validation script
- Investigation: `.kb/investigations/2026-02-27-design-claude-config-dir-drift-elimination.md`

---

## TLDR

Audited config drift between `~/.claude/` and `~/.claude-personal/`. Found 5 config items drifted/missing (not just the one hook). Recommended selective symlinks to eliminate drift structurally.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-design-claude-config-dir-drift-elimination.md` — Full design investigation with drift audit, fork analysis, and implementation plan
- `.kb/models/architectural-enforcement/probes/2026-02-27-probe-config-dir-drift-scope.md` — Probe extending the Architectural Enforcement model with config-level enforcement gap
- `.orch/workspace/og-arch-design-solution-recurring-27feb-118f/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-solution-recurring-27feb-118f/VERIFICATION_SPEC.yaml` — Verification contract

### Files Modified
- None (design-only session)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `diff` of both settings.json: personal missing `agreements-check-hook.py` SessionStart hook, work missing `pyright-lsp` plugin (both accidental drift)
- `ls ~/.claude-personal/` shows no CLAUDE.md, no skills/, no hooks/, no statusline.sh — all resolved from config dir, all missing
- `jq '.oauthAccount' ~/.claude-personal/.claude.json` confirms auth is per-directory (justifying separate dirs)
- `cc` function in `~/.zshrc` sets `CLAUDE_CONFIG_DIR=~/.claude-personal` for personal, confirming redirection mechanism
- Claude Code documentation confirms `CLAUDE_CONFIG_DIR` redirects ALL user-level files, not just settings.json

### Tests Run
```bash
# Diff of settings files
diff <(jq --sort-keys . ~/.claude/settings.json) <(jq --sort-keys . ~/.claude-personal/settings.json)
# → 2 differences: missing hook in personal, missing plugin in work

# File existence check
ls ~/.claude-personal/CLAUDE.md  # → Not found
ls ~/.claude-personal/skills     # → Not found
ls ~/.claude-personal/hooks      # → Not found
```

---

## Architectural Choices

### Symlinks over config generation
- **What I chose:** Symlink individual config files from personal → work
- **What I rejected:** Config generation script that merges base + per-account overrides
- **Why:** Gate Over Remind principle — a generation script must be "remembered" to run (reminder), while symlinks make drift impossible by construction (gate). Zero maintenance beats any maintenance.
- **Risk accepted:** Cannot have per-account settings differences. Currently there's no legitimate need. If one emerges, break the symlink.

### Individual file symlinks over whole-directory symlink
- **What I chose:** Symlink 5 specific config files/dirs
- **What I rejected:** Symlink entire `~/.claude-personal/` → `~/.claude/`
- **Why:** Runtime data (.claude.json with auth, history.jsonl, projects/) must remain per-account. Whole-dir symlink would share auth.
- **Risk accepted:** If Claude Code adds new config files in the future, they won't automatically be symlinked. The validation script catches this.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-27-design-claude-config-dir-drift-elimination.md` — Design for eliminating config drift
- `.kb/models/architectural-enforcement/probes/2026-02-27-probe-config-dir-drift-scope.md` — Extends model with config-level enforcement gap

### Decisions Made
- Decision 1: Symlinks for shared config because they make drift structurally impossible (gate, not reminder)
- Decision 2: 5 files to symlink (settings.json, CLAUDE.md, skills/, hooks/, statusline.sh) — everything else is per-account runtime data

### Constraints Discovered
- `CLAUDE_CONFIG_DIR` redirects ALL user-level files (not just settings.json) — the drift scope was much larger than the triggering symptom
- Auth is stored in `.claude.json` within config dir — dirs must remain separate
- Hook commands in settings.json use absolute `$HOME/.claude/hooks/` paths — scripts execute correctly regardless of config dir

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement Claude config dir symlinks
**Skill:** feature-impl (or manual — it's 5 ln commands)
**Context:**
```
Run the 5 symlink commands from the investigation's Implementation Plan section.
Back up ~/.claude-personal/settings.json first. Verify with diff and cc personal test.
```

Note: This could also be done manually by Dylan in ~2 minutes. No code changes needed — just filesystem operations.

---

## Unexplored Questions

- **Claude Code update behavior:** If Claude Code updates, does it overwrite settings.json? If so, it might replace the symlink with a real file. The validation script would catch this.
- **keybindings.json:** Neither config dir has this file currently. If created in the future, it should be added to the symlink list.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-solution-recurring-27feb-118f/`
**Investigation:** `.kb/investigations/2026-02-27-design-claude-config-dir-drift-elimination.md`
**Beads:** `bd show orch-go-1ala`
