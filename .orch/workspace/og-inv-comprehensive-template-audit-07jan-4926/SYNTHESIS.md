# Session Synthesis

**Agent:** og-inv-comprehensive-template-audit-07jan-4926
**Issue:** (ad-hoc, no beads tracking)
**Duration:** 2026-01-07 10:00 → 2026-01-07 11:30
**Outcome:** success

---

## TLDR

Comprehensive audit of all templates plus screenshots as artifact type. Text templates (14+ across 4 categories) are well-organized with clear ownership. **Screenshots are a significant gap** - produced by 3 disconnected systems (Playwright MCP, Glass, user-pasted) with NO template, NO storage convention, and NO lifecycle management.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-inv-comprehensive-template-audit-canonical-sources.md` - Comprehensive template + screenshot inventory

### Files Modified
- None (audit-only investigation)

### Commits
- Initial checkpoint commit
- Final investigation commit with screenshot findings (pending)

---

## Evidence (What Was Observed)

### Text Template Locations (Well-Organized)

| Category | Location | Count | Tool |
|----------|----------|-------|------|
| Spawn templates | `pkg/spawn/*.go` | 5 | `orch spawn` |
| Project overrides | `.orch/templates/` | 3 | Reference/override |
| CLAUDE.md templates | `pkg/claudemd/templates/` | 4 | `orch init` |
| kb artifact templates | `kb-cli/cmd/kb/create.go` | 4 | `kb create` |
| Skill components | `orch-knowledge/skills/src/` | ~90 | `skillc deploy` |

### Screenshot Sources (Disconnected)

| Source | Tool | Storage | Lifecycle |
|--------|------|---------|-----------|
| **Playwright MCP** | `browser_take_screenshot` | test-results/ or temp | Ephemeral |
| **Glass tools** | `glass_screenshot` | Base64 in response | Not persisted |
| **User-pasted** | Dylan shares paths | ~/Screenshots/*.png | External, macOS |

### Key Observations
- Verification gate (`pkg/verify/visual.go:82-107`) checks for screenshot MENTIONS in beads comments
- No mechanism to verify actual screenshot files exist
- User screenshots referenced as absolute paths in org/markdown (e.g., `DYLANS_THOUGHTS.org:21,111,117,121,134`)
- Playwright test-results/ directory was empty at time of audit
- Glass returns base64 - never written to disk unless agent explicitly saves

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-comprehensive-template-audit-canonical-sources.md` - Complete inventory with screenshot gap analysis

### Decisions Made
- None yet - screenshot storage needs architectural decision

### Constraints Discovered
- Screenshots cross tool boundaries (no single owner)
- Verification is text-based (comment mentions), not file-based
- User screenshots in ~/Screenshots/ become orphaned references over time
- No way to query "all screenshots for agent X" or "screenshots supporting investigation Y"

### Prior Decision Confirmed
- `.kb/decisions/2025-12-22-template-ownership-model.md` - Accurate for TEXT artifacts, but doesn't cover screenshots

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Screenshot Artifact Storage Decision
**Skill:** architect
**Context:**
```
Screenshots are produced by 3 disconnected systems (Playwright, Glass, user) with no 
canonical storage, referencing, or lifecycle. The verify package checks for screenshot 
MENTIONS but can't verify actual files. Need decision on: (1) Where to store? 
(2) How to reference? (3) Lifecycle/cleanup? (4) Ownership?
```

**Decision Options to Evaluate:**
1. **Per-workspace storage**: `.orch/workspace/{name}/screenshots/` - ties screenshots to agent lifecycle
2. **Per-project storage**: `.orch/screenshots/` - central, but loses agent association
3. **Convention only**: Document "save to workspace" but don't enforce
4. **Status quo**: Keep screenshots as ephemeral verification evidence

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should Glass screenshots be auto-saved to workspace when used for verification?
- Should Playwright MCP outputs be copied to workspace automatically?
- How do we handle user-pasted screenshots (~/Screenshots/)? Copy or link?
- Should screenshot references in markdown be relative or absolute?

**System improvement ideas:**
- `orch screenshot save <path>` - copy external screenshot into current workspace
- Glass tool modification to optionally save to workspace
- Verification gate could check for actual screenshot files, not just mentions

**What remains unclear:**
- Frequency of screenshot retrieval needs (is this actually a pain point?)
- Whether Playwright ever outputs to persistent locations
- Whether snap CLI (mentioned in session-ses_4735.md) is a screenshot tool

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-comprehensive-template-audit-07jan-4926/`
**Investigation:** `.kb/investigations/2026-01-07-inv-comprehensive-template-audit-canonical-sources.md`
**Beads:** (ad-hoc spawn, no beads tracking)
