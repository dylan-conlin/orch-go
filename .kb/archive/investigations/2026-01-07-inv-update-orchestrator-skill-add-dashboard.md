## Summary (D.E.K.N.)

**Delta:** Added dashboard troubleshooting protocol to orchestrator skill with systematic flow and quick decision tree.

**Evidence:** Protocol deployed via `skillc deploy`, verified in `~/.claude/skills/meta/orchestrator/SKILL.md` with grep.

**Knowledge:** Orchestrator skill uses `.skillc/SKILL.md.template` as source; `skillc deploy` compiles all skills recursively.

**Next:** Close - implementation complete, committed in orch-knowledge repo.

**Promote to Decision:** recommend-no (procedural documentation addition, not architectural)

---

# Investigation: Update Orchestrator Skill Add Dashboard

**Question:** How to add dashboard troubleshooting protocol to orchestrator skill?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** og-feat-update-orchestrator-skill-07jan-4b3d
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Skill source location

**Evidence:** The orchestrator skill source is at `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`

**Source:** File listing and read operation

**Significance:** Must edit the template file, not the deployed SKILL.md which gets overwritten on deploy.

---

### Finding 2: Skill deployment process

**Evidence:** `skillc deploy --target ~/.claude/skills/ ~/orch-knowledge/skills/src/` compiles and deploys all skills

**Source:** Command execution - deployed 17/17 .skillc directories

**Significance:** Single command handles all skill compilation and deployment.

---

### Finding 3: Best placement location

**Evidence:** "Monitoring and Window Layout" section (lines 464-483) discusses dashboard at localhost:5188

**Source:** SKILL.md.template content analysis

**Significance:** Placing troubleshooting section immediately after monitoring section provides logical flow.

---

## Synthesis

**Key Insights:**

1. **Follow existing patterns** - The skill already has decision tree tables (see "Quick Decision Trees") - used same format for troubleshooting section.

2. **Flow diagram format** - Used the → with ↓ format that appears elsewhere in the skill for step sequences.

**Answer to Investigation Question:**

Added the dashboard troubleshooting protocol by editing `SKILL.md.template`, adding a new "## Dashboard Troubleshooting" section after "Monitoring and Window Layout" with a flow diagram and quick decision tree, then deploying via `skillc deploy`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Section appears in deployed skill (verified: `grep -A 30 "## Dashboard Troubleshooting" ~/.claude/skills/meta/orchestrator/SKILL.md`)
- ✅ Skill compiles without errors (verified: `skillc deploy` succeeded with 17/17 skills)

**What's untested:**

- ⚠️ Whether orchestrators actually follow this protocol when dashboard is slow (not tested in production use)

**What would change this:**

- If orchestrators report the protocol is incomplete or missing steps, would need to iterate

---

## References

**Files Examined:**
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template` - Source file to edit
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Deployed target to verify

**Commands Run:**
```bash
# Deploy all skills
skillc deploy --target ~/.claude/skills/ ~/orch-knowledge/skills/src/

# Verify deployment
grep -A 30 "## Dashboard Troubleshooting" ~/.claude/skills/meta/orchestrator/SKILL.md

# Commit changes
cd ~/orch-knowledge && git add skills/src/meta/orchestrator/.skillc/SKILL.md.template && git commit -m "feat: add dashboard troubleshooting protocol..."
```

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: Add dashboard troubleshooting protocol per spawn context
- Context: Protocol provided in SPAWN_CONTEXT.md

**2026-01-07:** Implementation completed
- Added section to SKILL.md.template
- Deployed via skillc
- Committed to orch-knowledge repo
