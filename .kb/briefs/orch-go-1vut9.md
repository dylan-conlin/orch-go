# Brief: orch-go-1vut9

## Frame

Every time `orch serve` restarted — auto-rebuild, `orch-dashboard restart`, `make install` — every brief in the dashboard flipped back to unread. The mark-as-read state lived in a Go map that vanished with the process.

## Resolution

The fix is exactly what you'd expect: a JSON file at `~/.orch/briefs-read-state.json`, loaded at startup, written on each mark-as-read. Atomic writes (temp file + rename) so a crash mid-write can't corrupt it. The map keys already included the project directory, so cross-project isolation came for free — no schema changes needed.

The one thing worth noting: I chose user-level storage (`~/.orch/`) over project-level (`.orch/`) because the server process serves multiple projects from one instance. The keys handle the scoping; the file just needs to live somewhere stable.

## Tension

This is purely UI convenience state — `orch complete` remains the real comprehension gate. But there's a subtle question: as the dashboard accumulates more of these "UI-only" persistence needs, does `~/.orch/` become a shadow state store that drifts from the authoritative sources? Right now it's one file for one purpose, which is fine. Worth watching if more follow.
