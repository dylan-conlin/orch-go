# Session Handoff — 2026-02-11 (Session 11: Click Freeze Investigation)

## What Happened This Session

### Click freeze investigation — root cause NOT yet identified

The trackpad click freeze happened **4 times** this session (~every 15 minutes). Each time fixed with `sudo killall -HUP WindowServer`. We systematically eliminated suspects:

| Suspect | Eliminated? | How |
|---------|-------------|-----|
| Resource exhaustion (CPU/RAM) | Yes | CPU 74% idle, 12GB RAM free during freeze |
| BetterTouchTool AppleScript crash | Yes | Uninstalled BTT entirely, freeze recurred |
| Hammerspoon, Shortcat, middleClick, Raycast | Yes | All uninstalled, freeze recurred |
| Karabiner mouse rules | Yes | Only keyboard rules in config, no mouse/trackpad |
| yabai focus_follows_mouse | No | Disabled it (`yabai -m config focus_follows_mouse off`), freeze recurred anyway |

**BTT was a red herring.** Its AppleScript crash at 08:47 correlated with one freeze but was not the root cause — the freeze continued after BTT was completely removed.

### What we know
- Trackpad cursor **moves** but **clicks don't register**
- Keyboard still works during freeze
- `sudo killall -HUP WindowServer` fixes it every time
- Recurs within ~15 minutes
- NOT resource-related (CPU/RAM/swap all healthy)
- NOT caused by any of the removed input interceptors
- macOS 15.6.1 (Sequoia), Mac15,7 (M3 Pro)
- Remaining input interceptors: skhd, yabai, Karabiner, borders, sketchybar

### Apps removed this session
- **Uninstalled:** BetterTouchTool, Hammerspoon, Shortcat, middleClick, Raycast
- **Login items removed** for BTT, Hammerspoon, middleClick
- **Kept:** skhd, yabai, borders, sketchybar, Karabiner

### Next steps for click freeze investigation
1. **After reboot:** See if clean boot resolves it — could be accumulated WindowServer corruption from repeated HUPs
2. **If it persists after reboot:** Try disabling yabai entirely (`yabai --stop-service`) — it's the most invasive remaining interceptor
3. **If it persists without yabai:** Try disabling Karabiner — it operates at kernel level via DriverKit
4. **If it persists without any interceptors:** This is a macOS 15.6.1 bug or hardware issue
5. **Manual step:** System Settings > Privacy & Security > Accessibility — remove stale entries for uninstalled apps

### Simplification plan progress

| Phase | What | Status |
|-------|------|--------|
| Phase 0 | Remove dead code | **Done** (53eea7d6) |
| Phase 1 | GateCommitEvidence + startup sweep | **Done** |
| Phase 2 | Gate reclassification + pipeline flatten | **Done** (99a2652d, d5a351a2) |
| Phase 2 | Make core gate failures **block** completion (enforce) | Not started |
| Phase 3 | Simplify CLAUDE.md (579 → ~100 lines) | Not started |
| Phase 4 | Supervised-first shift (daemon opt-in) | Not started |

## Git State

- **Branch:** master, 6 commits ahead of origin (NOT yet pushed)
- **Uncommitted:** `.kb/quick/entries.jsonl`, `.orch/gate-skips.json`, `DYLANS_THOUGHTS.org`, deleted workspace SYNTHESIS.md, `.orch/HANDOFF.md`
- **IMPORTANT:** Push these 6 commits after reboot: `git push`
