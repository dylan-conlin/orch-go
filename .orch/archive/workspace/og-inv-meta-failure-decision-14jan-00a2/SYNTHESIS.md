# Session Synthesis

**Agent:** og-inv-meta-failure-decision-14jan-00a2
**Issue:** orch-go-4nxsx
**Duration:** 2026-01-14 (start) → 2026-01-14 (complete)
**Outcome:** success

---

## TLDR

Investigated meta-failure pattern of decision documentation gaps using Jan 7 follow-orchestrator case. Identified four systemic failures: empty template accumulation (10+ unfilled templates archived), missing feedback loops (107 recommend-no vs ~10 recommend-yes, none acted upon), model staleness (Evolution sections not updated), and tooling-process mismatch (kb reflect doesn't check investigation promotion flags). Recommended kb reflect --type investigation-promotion extension plus immediate cleanup actions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md` - Meta-failure analysis with 5 findings, recommendations, and D.E.K.N. summary

### Files Modified
None - investigation only, no code changes

### Commits
- `4764863a` - investigation: meta-failure-decision-documentation-gap - initial checkpoint
- `81e1d826` - investigation: meta-failure-decision-documentation-gap - findings 1-2
- `5e84efd2` - investigation: meta-failure-decision-documentation-gap - finding 3 systemic pattern
- `0c2d2006` - investigation: meta-failure-decision-documentation-gap - finding 4 promotion gap
- `2b9f8b20` - investigation: meta-failure-decision-documentation-gap - implementation details
- `7f8a7d51` - investigation: meta-failure-decision-documentation-gap - references and history
- `8cbb335c` - investigation: meta-failure-decision-documentation-gap - D.E.K.N. summary and status complete

---

## Evidence (What Was Observed)

### Finding 1: Empty Templates Archived
- Searched `.kb/investigations/archived/` with placeholder counting
- Found 10+ investigations with 36-89 placeholders (unfilled templates)
- Example: `2026-01-07-inv-implement-follow-orchestrator-dashboard-filtering.md` has 86 placeholders, 225 lines

### Finding 2: Model Evolution Gaps
- Read `.kb/models/dashboard-architecture.md` Evolution section (lines 202-260)
- Jan 7 entry mentions "Two-Mode Design" but not follow-orchestrator beads feature
- Complete investigation exists: `2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md`
- Investigation added cross-project beads support via project_dir parameter (architectural)

### Finding 3: Systemic Pattern
- Counted 3 archived investigations from Jan 7 alone
- Found 10+ total across Dec 19 - Jan 7 timeframe
- Pattern: agent death/restart → new file creation → archive instead of delete

### Finding 4: Missing Feedback Loop
- Grepped for "Promote to Decision" flags across all investigations
- 107 with "recommend-no", ~10 with "recommend-yes", 0 with "unclear"
- Examples of recommend-yes without decisions created:
  - synthesis completion recognition pattern
  - spoofing-based auth pattern

### Finding 5: Tooling-Process Mismatch
- Ran `kb reflect --type promote` → "No promote opportunities found"
- kb reflect --help shows it searches "kn entries" not investigation files
- Investigation template field has no automated check

### Tests Run
```bash
# Count placeholder-heavy archived investigations
for f in .kb/investigations/archived/*.md; do
  placeholders=$(grep -c '\[.*\]' "$f" 2>/dev/null || echo 0)
  if [ "$placeholders" -gt 30 ]; then echo "$f"; fi
done
# Result: 10+ files identified

# Count promotion flags
grep -r "Promote to Decision: recommend-no" .kb/investigations/ | wc -l
# Result: 107

grep -r "Promote to Decision: recommend-yes" .kb/investigations/ | wc -l
# Result: ~10

# Test kb reflect
kb reflect --type promote
# Result: No promote opportunities found
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md` - Analysis of decision documentation gaps with four root causes identified

### Decisions Made
- Decision 1: Recommend kb reflect --type investigation-promotion extension because it makes the "Promote to Decision" field actionable instead of performative
- Decision 2: Recommend immediate cleanup of empty templates because archive directory has accumulated noise
- Decision 3: Flag for promotion to decision ("recommend-yes") because this establishes pattern that investigation promotion requires tooling support

### Constraints Discovered
- "Promote to Decision" field in investigation template is performative without workflow tooling - no automated check reads these flags
- kb reflect --type promote only searches kb quick entries, not investigation files
- Agent death/restart pattern causes template proliferation but root cause unknown

### Externalized via `kb quick`
Not yet - will run before completion:
- `kb quick constrain "Investigation promotion fields need tooling support" --reason "Performative documentation without workflow integration fails"`
- `kb quick tried "Investigation template Promote to Decision field" --failed "No tooling reads the flags, becoming noise"`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Immediate Actions (Cleanup)
- [ ] Clean up 10+ empty templates from `.kb/investigations/archived/`
- [ ] Update dashboard-architecture.md Evolution section with Jan 7 follow-orchestrator entry
- [ ] Create beads issues for ~10 investigations with "recommend-yes" flags

### Follow-up Spawns
- [ ] Spawn feature-impl to add `kb reflect --type investigation-promotion` to kb-cli
- [ ] Spawn investigation to understand why agents die mid-investigation and create duplicate files (root cause of empty templates)
- [ ] Spawn investigation to establish criteria for "recommend-yes" vs "recommend-no" (currently seems arbitrary - 107 vs 10)

### If Escalate
Not needed - clear recommendations available

---

## Unexplored Questions

**Questions that emerged during investigation:**
- Why do agents die mid-investigation and create new files instead of resuming? (10+ occurrences suggests systemic issue)
- Should model Evolution sections be auto-updated from investigation completions, or remain manual curation?
- What criteria should determine "recommend-yes" vs "recommend-no"? (107 recommend-no suggests possible over-use)
- Are there other promotion workflows besides kb reflect that we missed?
- Should the "Promote to Decision" field be removed entirely in favor of kb quick decide during investigation?

**System improvement ideas:**
- Immediate checkpoint protocol worked well - investigation survived session without loss
- Progressive documentation pattern should be reinforced in investigation skill
- kb reflect pattern (--type flag) is effective for surfacing hygiene items
- Need forcing function for model Evolution section updates (similar to "Promote to Decision" field)

---

## Session Metadata

**Investigation file:** `.kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md`
**Commits:** 7 total (initial checkpoint through D.E.K.N. summary)
**Key references:**
- Jan 7 follow-orchestrator investigation (complete)
- Jan 7 empty template (archived)
- dashboard-architecture.md model (Evolution section)
- orchestrator skill (promotion workflow references)
