# Session Synthesis

**Agent:** og-arch-design-skip-flag-27feb-5dd2
**Issue:** orch-go-f06i
**Duration:** 2026-02-27T~10:00 → 2026-02-27T~10:45
**Outcome:** success

---

## Plain-Language Summary

Designed a system to require reasons whenever safety-override flags are used in orch (--bypass-triage, --force on complete, --force-hotspot, --no-track). Currently these flags are tracked by count only — we know *how often* they're used but not *why*. The design adds a `--reason` flag (min 10 chars) to each, stores the reason in events.jsonl alongside existing event data, and extends `orch stats` with a new "Override Reasons" section that shows frequency tables of reasons per override type. This closes the feedback loop: recurring reasons like "daemon not running" or "docs-only change" reveal systemic issues that can be fixed at the source. The design follows the existing `--skip-reason` pattern from targeted completion gates and also closes a gap where `verification.bypassed` events were being logged but never consumed by stats.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for acceptance criteria.

---

## TLDR

Designed `--reason` flag requirement for 4 safety-override flags, event schema enrichment (3 existing events + 1 new `spawn.hotspot_bypassed`), and `orch stats` Override Reasons section. Follows existing `--skip-reason` pattern. Touches 7 files across spawn, complete, events, and stats.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-design-skip-flag-reason-tracking.md` - Full architect investigation with 5 decision forks navigated
- `.orch/workspace/og-arch-design-skip-flag-27feb-5dd2/SYNTHESIS.md` - This file
- `.orch/workspace/og-arch-design-skip-flag-27feb-5dd2/VERIFICATION_SPEC.yaml` - Acceptance criteria

### Files Modified
- None (design-only session)

---

## Evidence (What Was Observed)

- `--skip-reason` pattern on complete already requires min 10 chars and logs to `verification.bypassed` events (complete_cmd.go:192, logger.go:437-460)
- `--bypass-reason` pattern on spawn already requires justification for `--bypass-verification` (spawn_cmd.go:187, verification.go:21-28)
- `spawn.triage_bypassed` event exists but has only `skill` and `task` fields, no reason (triage.go:26-38)
- `--force-hotspot` has ZERO event emission — only prints to stderr (hotspot.go:87)
- `--no-track` is embedded in `session.spawned` as `no_track:true` but no reason (backends/common.go:144)
- `--force` on complete embeds `forced:true` in `agent.completed` but no reason (complete_cmd.go:1324)
- `verification.bypassed` events are logged but NOT consumed by `orch stats` (confirmed: zero grep hits for "verification.bypassed" in stats_cmd.go)
- Friction-bypass probe (2026-02-09) showed 66% of test_evidence bypasses were "docs-only change" — reason strings DO reveal systemic patterns
- `--force` usage dropped from 72.8% to 16.7% after targeted `--skip-*` flags were introduced

---

## Architectural Choices

### Single `--reason` flag per command vs per-flag reason names
- **What I chose:** Single `--reason` flag that applies to whichever override flag is active
- **What I rejected:** Per-flag names (`--bypass-triage-reason`, `--force-reason`, etc.)
- **Why:** Context is unambiguous (if `--bypass-triage` is set, `--reason` applies to it). Avoids 4 new flag names. Existing `--bypass-reason` and `--skip-reason` continue to work for their respective flags.
- **Risk accepted:** Slight naming inconsistency — three different reason flags (`--reason`, `--skip-reason`, `--bypass-reason`) on spawn command

### Enrich existing events vs create new event types for all
- **What I chose:** Enrich 3 existing events + create 1 new event (`spawn.hotspot_bypassed`)
- **What I rejected:** Creating 4 new dedicated event types
- **Why:** 3 flags already emit events that just lack a reason field — enriching is minimal change. Hotspot has no event at all, so a new type is the only option.
- **Risk accepted:** Mixed approach (enrichment + new type)

### Simple frequency table vs NLP clustering for stats
- **What I chose:** Exact string frequency tables
- **What I rejected:** Fuzzy/NLP clustering of similar reasons
- **Why:** Reasons come from AI orchestrators programmatically — they'll naturally use consistent phrasing. Frequency tables surface patterns immediately. Can evolve if near-duplicate strings appear.
- **Risk accepted:** May miss clusters if reason phrasing varies

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-27-design-skip-flag-reason-tracking.md` - Full design with 5 forks navigated

### Decisions Made
- Decision: Use single `--reason` flag per command because context is unambiguous and avoids flag proliferation
- Decision: Add --reason to deprecated `--force` on complete because it's still at 16.7% usage and reason data will inform when to remove it
- Decision: Scope to 4 specified flags only; additional flags (--force workspace, --skip-gap-gate, etc.) can be added in follow-up

### Constraints Discovered
- `agent.completed` already has a `reason` field (completion reason) — force override reason must use different field name (`force_reason`) to avoid collision
- Stats currently does NOT consume `verification.bypassed` events at all — this is a pre-existing observability gap

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement skip-flag reason tracking (per design orch-go-f06i)
**Skill:** feature-impl
**Context:**
```
Implement --reason flag for 4 override flags per design in .kb/investigations/2026-02-27-design-skip-flag-reason-tracking.md.
7 files to modify: spawn_cmd.go, complete_cmd.go, triage.go, hotspot.go, logger.go, common.go, stats_cmd.go.
Follow existing --skip-reason pattern for validation (min 10 chars). Key nuance: agent.completed already has "reason" field — use "force_reason" for the --force override reason.
```

---

## Unexplored Questions

- **Should `--force` on complete be removed entirely?** Usage is at 16.7% but dropping. The reason data from this design will answer when it's safe to remove.
- **Should stats support time-series analysis of reason shifts?** E.g., "bypass-triage reasons shifted from 'daemon not running' to 'custom context needed' over 4 weeks." Current design is point-in-time frequency only.
- **Should the additional uncovered flags (--force on workspace, --skip-gap-gate, rework --force) also get --reason?** Deferred — can revisit after seeing stats on the 4 primary flags.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-skip-flag-27feb-5dd2/`
**Investigation:** `.kb/investigations/2026-02-27-design-skip-flag-reason-tracking.md`
**Beads:** `bd show orch-go-f06i`
