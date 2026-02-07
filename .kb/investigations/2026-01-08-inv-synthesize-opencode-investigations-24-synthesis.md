## Summary (D.E.K.N.)

**Delta:** Synthesized 24 OpenCode investigations into two comprehensive guides; the 8 new investigations (Jan 6-8) focused on plugin system for principle mechanization, distinct from prior synthesis which covered HTTP API.

**Evidence:** Read all 24 investigations, identified plugin-focused evolution arc (Jan 6-8), updated `.kb/guides/opencode.md` with 4 new decisions and 2 new common problems, verified `.kb/guides/opencode-plugins.md` already exists as authoritative plugin reference.

**Knowledge:** OpenCode integration has evolved from "how to spawn/monitor agents" (Dec 2025) to "how to mechanize principles via plugins" (Jan 2026); two guides serve complementary purposes (API vs plugins); cross-project session directory bug needs fixing in spawn_cmd.go.

**Next:** Close - guides updated, one open bug (cross-project session directory) requires separate feature-impl to fix.

**Promote to Decision:** recommend-no (synthesis work consolidating existing knowledge, not new architectural choice)

---

# Investigation: Synthesize OpenCode Investigations (24)

**Question:** What patterns and decisions have emerged from 24 OpenCode investigations, and can they be consolidated into the existing guides?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Worker agent (orch-go-z9wmt)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** `.kb/investigations/2026-01-06-inv-synthesize-opencode-investigations-16-synthesis.md` (extends, doesn't replace)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Clear evolution from API integration to plugin mechanization

**Evidence:** The 24 investigations split into two distinct phases:

**Phase 1 (Dec 19-26, 16 investigations):** Focused on HTTP API, session management, SSE monitoring
- POC, client package, spawn modes, cleanup mechanisms
- Already synthesized into `.kb/guides/opencode.md`

**Phase 2 (Jan 6-8, 8 investigations):** Focused on plugin system for principle mechanization
- Plugin capabilities, event reliability, session compaction, constraint surfacing
- Already synthesized into `.kb/guides/opencode-plugins.md`

**Source:** Investigation date ranges, topic categorization across all 24 files

**Significance:** The OpenCode integration has matured from "how do we integrate?" to "how do we mechanize principles?". The two guides serve complementary purposes and should remain separate.

---

### Finding 2: Four new architectural decisions settled in Jan 6-8 investigations

**Evidence:** New decisions not in prior synthesis:

1. **Plugin system is the bridge for principle mechanization** - Three patterns identified: Gates (blocking), Context Injection (guiding), Observation (learning)
2. **session.idle is deprecated** - Prefer `session.status` event with `status.type === "idle"` check
3. **OpenCode sessions share central storage** - All servers query same `~/.local/share/opencode/storage/`
4. **Question tool is `question`, not `AskUserQuestion`** - Skills corrected to use proper JSON interface

**Source:** 
- `2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md`
- `2026-01-08-inv-test-opencode-plugin-event-reliability.md`
- `2026-01-06-inv-cannot-query-opencode-sessions-other.md`
- `2026-01-07-inv-update-core-skills-opencode-ask.md`

**Significance:** These decisions extend the "settled" list in opencode.md. Future agents should not re-investigate these.

---

### Finding 3: Two new common problems documented

**Evidence:** 

1. **Cross-project sessions show wrong directory** - Bug in `orch spawn --workdir`: sessions get orchestrator's directory instead of target project. Impact: Sessions unfindable via `x-opencode-directory` header filtering.

2. **Session accumulation** - New `orch clean --sessions` command added to delete sessions older than N days. Tested: 461 sessions deleted (627 → 166).

**Source:**
- `2026-01-06-inv-cannot-query-opencode-sessions-other.md`
- `2026-01-06-inv-implement-opencode-session-cleanup-mechanism.md`

**Significance:** These common problems weren't in prior synthesis and need to be documented in opencode.md for troubleshooting.

---

### Finding 4: Plugin guide already comprehensive, no updates needed

**Evidence:** `.kb/guides/opencode-plugins.md` (651 lines) already includes:
- Three plugin patterns with production examples
- Hook selection guide (20+ hooks)
- Worker vs orchestrator detection
- State management across sessions
- Common pitfalls (6 documented)
- Plugin file structure and template

**Source:** Read of `.kb/guides/opencode-plugins.md`

**Significance:** The 2026-01-08 investigations that created the plugin guide did thorough synthesis. No additional consolidation needed for plugin content.

---

## Synthesis

**Key Insights:**

1. **Two complementary guides serve distinct purposes** - `opencode.md` covers HTTP API and orch-go integration; `opencode-plugins.md` covers plugin development for principle mechanization. They should remain separate.

2. **Evolution arc shows maturation** - From exploratory POC (Dec 19) → HTTP integration (Dec 20-26) → principle mechanization via plugins (Jan 6-8). The system is moving up the abstraction ladder.

3. **One bug remains open** - Cross-project session directory bug needs fixing in `cmd/orch/spawn_cmd.go`. This is implementation work, not investigation.

**Answer to Investigation Question:**

The 24 investigations consolidate into two existing guides:
- `.kb/guides/opencode.md` - Updated with 4 new decisions, 2 new common problems, 8 new investigation references
- `.kb/guides/opencode-plugins.md` - Already comprehensive from recent synthesis, no changes needed

No new guide needed. The existing structure is correct.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 24 investigations read and categorized (verified: read each file)
- ✅ opencode.md updated with new decisions and problems (verified: edits applied)
- ✅ opencode-plugins.md is comprehensive (verified: 651 lines covering all patterns)

**What's untested:**

- ⚠️ Cross-project session directory fix (bug identified, not implemented)
- ⚠️ Whether `session.status` is drop-in replacement for `session.idle` (migration not tested)
- ⚠️ Plugin guide usefulness in practice (not validated with new plugin creation)

**What would change this:**

- Finding would be wrong if there are additional investigations not in the listed 24
- Finding would be wrong if opencode-plugins.md has significant gaps (appears complete)
- Synthesis would need updating if cross-project bug fix reveals new patterns

---

## Proposed Actions

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| - | (none) | All investigations still relevant | - |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | issue | "Fix cross-project spawn sets wrong session directory" | Sessions created via `orch spawn --workdir` have orchestrator's directory; fix spawn_cmd.go to pass correct directory | [ ] |

### Promote Actions
| ID | kn-id | To | Reason | Approved |
|----|-------|-------|--------|----------|
| - | (none) | - | Decisions already documented in guides | - |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `.kb/guides/opencode.md` | Added 4 new decisions, 2 common problems, 8 investigation refs | Extend guide with Jan 6-8 findings | [x] (done) |

**Summary:** 1 proposal (1 create)
**High priority:** C1 (cross-project session directory bug)

---

## References

**Files Examined:**
- All 24 `.kb/investigations/*opencode*.md` files
- `.kb/guides/opencode.md` - Updated with new findings
- `.kb/guides/opencode-plugins.md` - Verified comprehensive
- `.kb/investigations/2026-01-06-inv-synthesize-opencode-investigations-16-synthesis.md` - Prior synthesis

**Commands Run:**
```bash
# List all opencode investigations
glob .kb/investigations/*opencode*.md

# Create investigation file
kb create investigation synthesize-opencode-investigations-24-synthesis
```

**Related Artifacts:**
- **Guide:** `.kb/guides/opencode.md` - Updated by this synthesis
- **Guide:** `.kb/guides/opencode-plugins.md` - Comprehensive plugin reference
- **Prior Synthesis:** `.kb/investigations/2026-01-06-inv-synthesize-opencode-investigations-16-synthesis.md`

---

## Investigation History

**2026-01-08 08:00:** Investigation started
- Initial question: Can 24 OpenCode investigations be consolidated?
- Context: kb reflect flagged synthesis opportunity (24 investigations)

**2026-01-08 08:15:** Read all 24 investigations
- Identified two-phase evolution: API (Dec) → Plugins (Jan)
- Found 4 new decisions, 2 new common problems

**2026-01-08 08:30:** Updated opencode.md guide
- Added new decisions to settled list
- Added new common problems section
- Added 8 new investigation references to table

**2026-01-08 08:45:** Investigation completed
- Status: Complete
- Key outcome: Guides updated; one bug (cross-project directory) needs separate issue to fix
