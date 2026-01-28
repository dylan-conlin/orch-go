# Two-Tier Disk Cleanup Infrastructure

**Date:** 2026-01-28
**Status:** Accepted
**Context:** Disk filling up daily despite hourly cleanup script

## Problem

Disk was filling up 2-3GB/hour during active work, exhausting space despite hourly cleanup. On 2026-01-28, available space dropped from 9.8GB (9am) to 3.3GB (2pm) in 5 hours.

## Root Causes Identified

1. **Go build temp files** - 665MB in 45 directories; `go clean -cache` doesn't clean `/var/folders/*/T/go-build*`
2. **Docker volumes in colima** - Unused volumes/images accumulating (9.2GB recovered on first aggressive cleanup)
3. **Chrome code_sign_clone** - macOS Sequoia bug creates ~1.2GB per Chrome launch, unbounded growth
4. **Claude project artifacts** - node_modules and session .jsonl files in `~/.claude/projects/` (1.3GB, never cleaned)
5. **Wallpaper caches** - 2.1GB in `/var/folders/*/C/com.apple.wallpaper.caches/`

## Decision

Implement two-tier disk cleanup:

| Tier | Job | Frequency | Threshold | Purpose |
|------|-----|-----------|-----------|---------|
| Normal | `com.dylan.disk-cleanup` | Hourly | Always runs | Routine maintenance |
| Aggressive | `com.dylan.disk-threshold` | Every 5 min | <10GB free | Emergency recovery |

### Normal Cleanup (hourly)
- Docker build cache (>48h old)
- Go cache (`go clean -cache`)
- Go build temps (>60 min old)
- Old session files in Claude projects (>7 days, >5MB)
- node_modules in Claude projects
- Package manager caches (npm, bun, uv >30 days)
- Orch/OpenCode logs
- Chrome code_sign_clone

### Aggressive Cleanup (threshold-triggered)
All of the above, plus:
- ALL go-build temps (not just old)
- ALL wallpaper caches
- Docker system prune with volumes (`-af --volumes`)
- Playwright browser cache (full wipe)
- Yarn cache (full wipe)

## Alternatives Considered

1. **Single aggressive cleanup hourly** - Rejected: too disruptive (wipes Playwright cache every hour, slows rebuilds)
2. **Higher threshold (15GB)** - Rejected: 460GB disk with heavy Docker use means 10GB is reasonable buffer
3. **Lower frequency threshold check (15 min)** - Rejected: 2-3GB/hour bleed rate means 5 min catches issues before critical

## Trade-offs

- **Aggressive Docker prune deletes volumes** - May need to repopulate dev databases after threshold cleanup
- **Playwright cache wipe** - First test run after cleanup downloads browsers (~1GB, ~2 min)
- **5-min polling** - Minimal overhead (script exits immediately if above threshold)

## Files

- `/Users/dylanconlin/bin/disk-cleanup.sh` - Normal cleanup
- `/Users/dylanconlin/bin/disk-threshold-cleanup.sh` - Aggressive cleanup
- `~/Library/LaunchAgents/com.dylan.disk-cleanup.plist` - Hourly trigger
- `~/Library/LaunchAgents/com.dylan.disk-threshold.plist` - 5-min threshold check

## Logs

- `~/.orch/disk-cleanup.log` - Normal cleanup history
- `~/.orch/disk-threshold.log` - Threshold alerts and aggressive cleanup history

## Verification

After implementation, recovered 5.5GB (4.5GB → 10GB available). Threshold monitor will prevent future exhaustion.
