# Probe: Orchestrator Skill Deployment Sync

**Date:** 2026-02-15  
**Status:** Complete  
**Model:** Orchestrator Session Lifecycle

---

## Question

Does `skillc deploy` correctly sync orchestrator skill source (.skillc/SKILL.md) to both deployment targets (~/.claude/skills/ and ~/.opencode/skill/), ensuring new behavioral protocols (like verifiability-first completion review) reach active orchestrators?

## What I Tested

**Initial state verification:**
```bash
# Count "explain-back" occurrences (marker for verifiability-first protocol)
grep -r "explain-back" ~/orch-knowledge/skills/src/meta/orchestrator/.skillc/
grep -r "explain-back" ~/.claude/skills/meta/orchestrator/
grep -r "explain-back" ~/.opencode/skill/meta/orchestrator/
```

**Deployment:**
```bash
# Running skillc deploy to sync source to deployment targets
cd ~/orch-knowledge/skills/src
skillc deploy --target ~/.claude/skills
skillc deploy --target ~/.opencode/skill
```

**Post-deployment verification:**
```bash
# Check "explain-back" in deployed targets
grep -r "explain-back" ~/.claude/skills/meta/orchestrator/
grep -r "explain-back" ~/.opencode/skill/meta/orchestrator/

# Verify content consistency
diff <(tail -n +13 ~/.claude/skills/meta/orchestrator/SKILL.md) \
     <(tail -n +13 ~/.opencode/skill/meta/orchestrator/SKILL.md)
```

## What I Observed

**Before deployment:**
- Source: 12 matches for "explain-back"
- ~/.claude/skills/: 0 matches
- ~/.opencode/skill/: 0 matches

**After deployment (2026-02-15 11:30):**
- ~/.claude/skills/: 6 matches for "explain-back" ✓
- ~/.opencode/skill/: 6 matches for "explain-back" ✓
- Content identical between both targets (diff returns empty)
- Deployment metadata differs (checksum, path, timestamp) but content matches
- Two-gate completion review protocol confirmed present:
  - Gate 1: Comprehension (Explain-Back)
  - Verifiability-first approach documented

**Checksums:**
- Source (.skillc/SKILL.md): 252b37c8c60a
- Deployed ~/.claude: 14e650f1acab (includes deployment metadata)
- Deployed ~/.opencode: 658d5bd26dd9 (includes deployment metadata)

Note: Deployed checksums differ from source because skillc adds deployment-specific headers ("Deployed to:", full paths, timestamps). Content after metadata is byte-for-byte identical.

## Model Impact

**Confirms claim:** Orchestrator behavioral changes require skillc deploy to reach active orchestrators. Source updates without deployment leave orchestrators running stale protocols.

**Evidence:**
- Agent orch-go-6th updated source on Feb 14 23:24 (checksum 252b37c8c60a)
- Deployed files remained stale (last modified Feb 14 18:43 and Feb 13 14:29)
- Running skillc deploy successfully synced new protocol to both targets
- grep for "explain-back" went from 0 matches → 6 matches in both deployment targets

**Extends model with:**

**New failure mode: Deployment Drift**
- **Symptom:** Source .skillc/SKILL.md updated but deployed SKILL.md unchanged
- **Root cause:** skillc build compiles source locally but doesn't auto-deploy to ~/.claude/skills/ or ~/.opencode/skill/
- **Detection:** grep for protocol markers returns 0 in deployed but >0 in source; checksum mismatch
- **Impact:** Orchestrators load stale protocols and fall back to old behavioral patterns under velocity pressure
- **Fix:** Run `skillc deploy --target ~/.claude/skills && skillc deploy --target ~/.opencode/skill`
- **Prevention:** Add post-compile hook or CI check to verify deployed checksums match source

**Critical invariant:** skillc deploy adds deployment metadata (path, timestamp) which changes checksums. Content verification must compare post-header content, not raw file checksums.
