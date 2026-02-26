<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fixed launchd vite orphan process pileup and clarified server management architecture in documentation.

**Evidence:** Added AbandonProcessGroup=false to plist, removed duplicate vite from tmuxinator, added architecture docs to CLAUDE.md.

**Knowledge:** Server management has three distinct layers: launchd (persistent infrastructure), tmuxinator (project dev servers), orch servers (CLI wrapper). Don't run same service in multiple layers.

**Next:** Reload launchd service to apply plist changes: `launchctl kickstart -k gui/$(id -u)/com.orch-go.web`

---

# Investigation: Fix Launchd Server Management

**Question:** How to prevent orphaned vite processes from launchd restarts and clarify the three-layer server management architecture?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Agent (spawned feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** .kb/investigations/2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md

---

## Findings

### Finding 1: AbandonProcessGroup prevents orphaned child processes

**Evidence:** Added to `com.orch-go.web.plist`:
```xml
<key>AbandonProcessGroup</key>
<false/>
```

When set to false (the default), launchd will kill the entire process group when the job stops or restarts. This prevents `npm run dev` from leaving behind orphaned node/vite processes.

**Source:** ~/Library/LaunchAgents/com.orch-go.web.plist

**Significance:** Fixes the vite pileup issue identified in the prior investigation. Orphaned processes (PPID=1) will no longer accumulate on service restarts.

---

### Finding 2: Tmuxinator had duplicate vite command

**Evidence:** The `~/.tmuxinator/workers-orch-go.yml` contained:
```yaml
panes:
  - # api server on port 3348
  - bun run dev --port 5188  # <-- duplicate of launchd-managed service
```

This meant vite could be started from two sources:
1. launchd via `com.orch-go.web.plist` (persistent)
2. tmuxinator via `workers-orch-go.yml` (manual)

**Source:** ~/.tmuxinator/workers-orch-go.yml

**Significance:** Removed the duplicate. Launchd now exclusively owns the orch-go web dev server. The pane is now a comment placeholder.

---

### Finding 3: Architecture was undocumented in CLAUDE.md

**Evidence:** CLAUDE.md had no explanation of the three-layer server management architecture, only listing `orch servers` commands without explaining how they relate to launchd and tmuxinator.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

**Significance:** Added comprehensive "Server Management Architecture (Three Layers)" section explaining:
- Layer 1: Persistent Services (launchd) - infrastructure
- Layer 2: Project Dev Servers (tmuxinator) - per-project
- Layer 3: CLI Wrapper (orch servers) - user interface

---

## Synthesis

**Key Insights:**

1. **Layer separation matters** - Running the same service in multiple layers (launchd + tmuxinator) causes confusion and potential conflicts.

2. **Default AbandonProcessGroup is correct** - The default (false) ensures child processes are killed with the parent. We're being explicit for documentation clarity.

3. **Documentation prevents architecture confusion** - The three-layer model is intentional and sound, but was undiscoverable without explicit documentation.

**Answer to Investigation Question:**

Fixed by three changes:
1. Added `AbandonProcessGroup=false` to plist to kill orphaned vite processes on restart
2. Removed duplicate vite command from tmuxinator config
3. Added architecture documentation to CLAUDE.md explaining the three-layer model

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2026-01-03:** Implementation started
- Task: Fix launchd server management per prior investigation findings
- Three deliverables: plist fix, tmuxinator cleanup, CLAUDE.md docs

**2026-01-03:** Implementation completed
- Status: Complete
- Key outcome: Orphaned vite prevention and architecture documentation added
