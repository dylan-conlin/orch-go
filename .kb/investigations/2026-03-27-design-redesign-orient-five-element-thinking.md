## Summary (D.E.K.N.)

**Delta:** Orient was rendering 15+ sections including operational noise; redesigned to render only 5 elements (threads, briefs, tensions, ready work, focus) — everything else moved to `orch health`.

**Evidence:** All 80+ orient and cmd tests pass. `orch orient` output drops from ~100 lines to ~40 lines. `orch health` shows all operational sections.

**Knowledge:** The thinking surface / ops dashboard split maps cleanly onto the existing FormatOrientation function — no data model changes needed beyond adding RecentBrief, just routing sections to different renderers.

**Next:** No further implementation needed. Shape and resistance (elements 4-5) remain as orchestrator judgment per decision constraint.

**Authority:** implementation - Implements an existing architectural decision (kb-c85a86)

---

# Investigation: Redesign Orient as Five-Element Thinking Surface

**Question:** How should orient be restructured to render only the thinking surface (threads, briefs, tensions) while moving operational sections to orch health?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** dashboard-architecture

## Findings

### Finding 1: Clean rendering split is possible without breaking backward compatibility

**Evidence:** OrientationData struct retains all fields. FormatOrientation renders thinking surface; FormatHealth renders operational sections. JSON output (`--json`) returns full data for backward compat.

**Source:** `pkg/orient/orient.go:115-147` (struct), `pkg/orient/orient.go:226-320` (renderers)

**Significance:** No API-breaking changes. The `--hook` path gets the concise thinking surface; `--json` consumers see everything.

### Finding 2: Brief scanning reuses existing infrastructure

**Evidence:** Created `pkg/orient/briefs.go` with `ScanRecentBriefs` that reads `.kb/briefs/`, extracts first sentence of Frame as title, checks tension presence, and applies read state from `~/.orch/briefs-read-state.json` (same file as web UI).

**Source:** `pkg/orient/briefs.go`, `cmd/orch/orient_cmd.go:loadBriefReadStateForOrient()`

**Significance:** Briefs surface with unread count and tension markers without duplicating compose package logic.

### Finding 3: Operational sections move cleanly to health command

**Evidence:** `orch health` now shows: harness health score (existing) + operational metrics (throughput, changelog, models, daemon health, divergence, adoption drift, explore candidates, reflection suggestions).

**Source:** `cmd/orch/health_cmd.go:collectAndFormatOperationalHealth()`

**Significance:** All operational data has a home. No information was lost in the split.

## Structured Uncertainty

**What's tested:**

- ✅ FormatOrientation renders only threads, briefs, tensions, ready work, plans, focus (verified: 80+ tests)
- ✅ FormatHealth renders all operational sections (verified: tests for each section)
- ✅ Brief scanning parses titles, detects tension, respects read state (verified: 8 unit tests)
- ✅ orch orient output is concise (~40 lines vs ~100 lines before)

**What's untested:**

- ⚠️ Session-start hook performance impact with brief scanning (should be minimal — directory read + file parse)
- ⚠️ Tension filtering by thread relevance (deferred — claim edges already have keyword-based filtering)

**What would change this:**

- If orient output exceeds 80 lines with many active threads + briefs, may need tighter limits
- If brief read state diverges between web UI and orient, may need shared read-state package
