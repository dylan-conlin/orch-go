## Summary (D.E.K.N.)

**Delta:** Added `orch automation list` and `orch automation check` documentation to orchestrator skill and created comprehensive orch-commands-reference.md.

**Evidence:** Verified changes via grep showing "Automation:" line at line 1338 in deployed skill; docs/orch-commands-reference.md created.

**Knowledge:** Automation commands audit custom launchd agents matching com.dylan.*, com.user.*, com.orch.*, com.cdd.* patterns.

**Next:** Close - documentation complete.

**Promote to Decision:** recommend-no (documentation update, not architectural)

---

# Investigation: Document Orch Automation Commands Orchestrator

**Question:** Where should orch automation commands be documented in the orchestrator skill?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Orch Commands section structure in orchestrator skill

**Evidence:** The "Orch Commands (Essential)" section (lines 1297-1317 in SKILL.md.template) groups commands by category: Lifecycle, Monitoring, Agent Management, Session, Strategic, Beads, Knowledge, Health, Servers.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template:1297-1319`

**Significance:** Automation commands fit naturally as a new category after Servers, following the established pattern.

---

### Finding 2: orch automation command capabilities

**Evidence:** From `orch automation --help`:
- `orch automation list` - Shows all custom launchd agents with status, exit code, and schedule
- `orch automation check` - Health audit that flags failures, exits non-zero when issues found
- Scans `~/Library/LaunchAgents/` for agents matching: `com.dylan.*`, `com.user.*`, `com.orch.*`, `com.cdd.*`

**Source:** `orch automation --help`, `orch automation list --help`, `orch automation check --help`

**Significance:** Commands provide visibility into background services critical for orchestration system health.

---

### Finding 3: Missing docs/orch-commands-reference.md

**Evidence:** SKILL.md references `docs/orch-commands-reference.md` at lines 1275 and 1325, but the file did not exist. Only auto-generated CLI docs in `docs/cli/` were present.

**Source:** Glob search of `docs/**/*.md`, file read attempts

**Significance:** Created comprehensive reference file to fulfill existing documentation references.

---

## Synthesis

**Key Insights:**

1. **Consistent categorization** - Added Automation category following Servers, maintaining the established pattern of grouping related commands.

2. **Cross-referenced documentation** - Both orchestrator skill and orch-commands-reference.md now document these commands for different use cases.

3. **Scripting integration** - Documented the exit code behavior of `orch automation check` for integration with monitoring scripts.

**Answer to Investigation Question:**

Automation commands belong in the "Orch Commands (Essential)" section as a new category line. Added:
```
**Automation:** `orch automation list` (show launchd agents with status/exit code/schedule) | `orch automation check` (health audit, exits non-zero on issues)
```

---

## Structured Uncertainty

**What's tested:**

- ✅ Automation line appears in deployed skill (verified: grep shows line 1338)
- ✅ skillc build succeeds (verified: compiled with 15214 tokens)
- ✅ docs/orch-commands-reference.md created with automation section

**What's untested:**

- ⚠️ Token budget warning (101.4% of 15000) may need attention in future

**What would change this:**

- Finding would need revision if orchestrator skill structure changes significantly

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template` - Skill source to edit
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Deployed skill location

**Commands Run:**
```bash
# Check orch automation capabilities
orch automation --help
orch automation list --help
orch automation check --help

# Build and deploy skill
cd ~/orch-knowledge/skills/src/meta/orchestrator && skillc build
cp ~/orch-knowledge/skills/src/meta/orchestrator/SKILL.md ~/.claude/skills/meta/orchestrator/SKILL.md

# Verify deployment
grep -n "Automation:" ~/.claude/skills/meta/orchestrator/SKILL.md
```

---

## Investigation History

**2026-01-19 15:38:** Investigation started
- Initial question: Where to document orch automation commands
- Context: Spawned task to document new automation commands

**2026-01-19 15:42:** Implementation complete
- Added Automation line to orchestrator skill template
- Created docs/orch-commands-reference.md
- Rebuilt and deployed skill

**2026-01-19 15:43:** Investigation completed
- Status: Complete
- Key outcome: Automation commands documented in skill and reference doc
