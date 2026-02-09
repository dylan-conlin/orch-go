---
status: superseded
superseded_by: .kb/decisions/2026-01-28-two-tier-disk-cleanup-infrastructure.md
---

# Decision: Disk Cleanup Strategy for Chrome code_sign_clone

**Date:** 2026-01-26
**Status:** Superseded (2026-01-28)
**Context:** Dylan's disk was filling ~20GB/day, causing Claude Code crashes

## Problem

Disk dropped from 21GB to 333MB in 14 hours. Root cause investigation found:

1. **Chrome code_sign_clone (primary)** - macOS Sequoia creates ~1.2GB clones on every Chrome/Chromium launch for code signature validation. Cleanup only runs on clean exit, so crashes/force-kills leave orphans.

2. **Daemon log verbosity (secondary)** - orch daemon logged 270k lines/day (~17MB), fixed separately in `orch-go-20909`.

3. **Corrupted colima VM** - 24GB with I/O errors, recreated fresh.

## Decision

Multi-layer defense against disk exhaustion:

### Layer 1: Prevent Chrome clones at source

**skhd hotkey** (`cmd+ctrl+e` in `~/.config/skhd/.skhdrc`):
```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --remote-debugging-port=9222 --user-data-dir="$HOME/.chrome-debug-profile" --disable-features=MacAppCodeSignClone &
```

**Environment variable** (in `~/.zshrc`):
```bash
export CHROMIUM_FLAGS="--disable-features=MacAppCodeSignClone"
```

### Layer 2: Hourly cleanup for Playwright/automation

Changed `~/Library/LaunchAgents/com.dylan.disk-cleanup.plist` from daily to **hourly** (runs at :00 each hour).

Cleanup script (`~/bin/disk-cleanup.sh`) includes:
```bash
# Chrome code_sign_clone (macOS Sequoia creates ~1.2GB per launch, unbounded growth)
find /var/folders -type d -name "com.google.Chrome.code_sign_clone" -exec rm -rf {}/* \; 2>/dev/null || true
```

### Layer 3: Additional cleanup rules added

- `~/.claude-backup-*` and `~/.claude.backup.*` (old backups)
- `Google/Chrome.backup.*` directories
- `Google/GoogleUpdater` cache
- `Autodesk/webdeploy/*` older than 7 days

## Why This Approach

1. **Flag alone is insufficient** - Playwright MCP and other automation tools spawn their own Chromium instances that don't inherit the flag.

2. **Cleanup alone is insufficient** - 20GB/day accumulation means even hourly cleanup could leave 1-2GB orphans.

3. **Both together** - Flag prevents most clones, hourly cleanup catches stragglers from automation.

## Alternatives Considered

- **Reboot more often** - Disrupts workflow, not practical
- **Disable Playwright MCP** - Needed for browser automation
- **macOS setting** - None exists for code_sign_clone

## Consequences

- Disk should stabilize with ~15-20GB free
- Hourly cleanup adds minimal system load
- Chrome auto-updates may fail if running (theoretical, not observed)

## References

- Investigation: `.kb/investigations/2026-01-25-inv-investigate-dylan-disk-fills-up.md`
- Research: `.kb/investigations/2026-01-25-inv-research-macos-sequoia-chrome-code-sign-clone.md`
- Chromium bug: https://issues.chromium.org/issues/379125944
- Beads: `orch-go-20909` (daemon verbosity), `orch-go-20911` (Chrome research)
