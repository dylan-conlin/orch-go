# Session Synthesis

**Agent:** og-feat-design-config-code-08jan-5f13
**Issue:** orch-go-xzr2q
**Duration:** 2026-01-08 10:32 → 2026-01-08 11:20
**Outcome:** success

---

## TLDR

Designed a config-as-code system to prevent invisible config bugs like the plist flag drift. The solution extends ~/.orch/config.yaml with daemon configuration, adds generation commands (`orch config generate plist`), and drift detection (`orch doctor --config`).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-08-inv-design-config-code-orch-ecosystem.md` - Complete design investigation with D.E.K.N. summary, config inventory, implementation recommendations, and schema specification

### Files Modified
- None (design-only investigation)

### Commits
- To be committed with this synthesis

---

## Evidence (What Was Observed)

- **Plist structure analyzed:** `/Users/dylanconlin/Library/LaunchAgents/com.orch.daemon.plist` contains daemon flags, PATH, working directory - all manually configured and untracked
- **Existing config infrastructure:** `pkg/userconfig/userconfig.go` already manages `~/.orch/config.yaml` with typed Config struct - can be extended
- **6 external config locations identified:** plist, config.yaml, accounts.yaml, plugin symlinks, CLI symlinks, project config
- **Config categories:** Generated (plist, symlinks), Secrets (accounts.yaml), Ephemeral (focus.json, session.json)
- **Root cause of plist bug:** No single source of truth + no drift detection = invisible flag changes

### Tests Run
```bash
# Examined actual system state
ls -la ~/.orch/
cat ~/Library/LaunchAgents/com.orch.daemon.plist
cat ~/.orch/config.yaml
ls -la ~/.bun/bin/
ls -la ~/.config/opencode/plugin/
# All commands successful, config inventory complete
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-design-config-code-orch-ecosystem.md` - Complete design with implementation plan

### Decisions Made
- **Extend config.yaml (not new file):** Existing infrastructure is sufficient, no need for separate system.yaml
- **Generation + drift detection pattern:** Addresses the actual bug class (manual edits causing drift)
- **Keep secrets separate:** accounts.yaml stays as-is, not part of config-as-code

### Constraints Discovered
- Plist has XML escaping requirements for generation
- Launchd reload requires `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`
- Symlinks need careful handling of ~ expansion

### Externalized via `kn`
- Will recommend: `kn decide "Config-as-code via generation + drift detection" --reason "Prevents invisible config bugs like plist flag drift by making expected config explicit and detectable"`

---

## Next (What Should Happen)

**Recommendation:** close (design complete, implementation is separate work)

### If Close
- [x] All deliverables complete (design investigation with implementation plan)
- [x] No tests needed (design-only)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-xzr2q`

### Follow-up Implementation Issues (suggested)

**Phase 1: Add daemon config to config.yaml** (~2 hours)
- Add DaemonConfig struct to pkg/userconfig/userconfig.go
- Fields: poll_interval, max_agents, label, verbose, reflect_issues, working_directory, path

**Phase 2: Plist generation** (~2 hours)
- `orch config generate plist` command
- Template-based XML generation from config.yaml
- Prompt for launchctl reload

**Phase 3: Drift detection** (~1 hour)
- Extend `orch doctor` with `--config` flag
- Compare expected vs actual plist values
- Show actionable fix commands

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should OpenCode plugin deployment also be config-as-code? (Currently mix of symlinks and direct files)
- Is there value in versioning ~/.orch/ as a dotfiles repo? (Alternative approach)

**Areas worth exploring further:**
- Automatic plist regeneration on config.yaml change (file watcher?)
- CI/CD for config changes (when config-as-code in version control)

**What remains unclear:**
- Whether symlink drift actually causes user-visible bugs (assumed but not proven)
- Performance impact of drift detection in orch doctor

---

## Session Metadata

**Skill:** feature-impl (design phase only)
**Model:** Claude Opus
**Workspace:** `.orch/workspace/og-feat-design-config-code-08jan-5f13/`
**Investigation:** `.kb/investigations/2026-01-08-inv-design-config-code-orch-ecosystem.md`
**Beads:** `bd show orch-go-xzr2q`
