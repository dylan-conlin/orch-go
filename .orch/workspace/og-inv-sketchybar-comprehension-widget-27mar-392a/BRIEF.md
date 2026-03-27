# Brief: orch-go-qloo8

## Frame

You glanced at the menu bar and saw comprehension: 0. Good — nothing to review. Except the Claude Code hook in your terminal said 5 unread. The daemon had been SIGKILL'd hours earlier, and the status file it writes froze at its last value. You had no ambient signal that work was piling up.

## Resolution

The fix is three lines of bash and ten lines of Lua. The event provider now checks whether the daemon PID is actually alive (`kill -0`), not just whether the status file exists. When the process is dead, it queries beads directly for the live comprehension count instead of trusting the frozen file. The widget shows "dead C:54" in red — you know the daemon is down AND you know how much is waiting.

The surprising part: the provider already had staleness detection via `last_poll` age, but it has a 2-minute blind spot after SIGKILL. The file looks fresh because the daemon just wrote it before dying. PID liveness catches it on the very next poll cycle — no blind spot. The bd fallback adds ~500ms per 10s poll, but only when the daemon is confirmed dead. Normal operation is unchanged.

## Tension

This fixes the display, not the crash. The daemon was killed with SIGKILL/-9 — the question of why (OOM? launchd watchdog? something else?) is still open. And the popup has the same stale-data problem — it reads daemon-status.json directly from Lua, not via the event provider, so clicking the widget when daemon is dead would still show stale details. The bar label is now trustworthy; the popup is not.
