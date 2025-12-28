# Session Synthesis

**Agent:** og-inv-knowledge-fragmentation-433-28dec
**Issue:** orch-go-s03z
**Duration:** 2025-12-28 ~14:50 → ~15:30
**Outcome:** success

---

## TLDR

Goal was to determine if 500 investigations in 7 days represent wasteful rediscovery. Finding: investigations are NOT duplicative - they explore distinct aspects. The real problem is knowledge graph disconnection (zero lineage references, zero kn citations). Recommend improving knowledge linkage rather than running synthesis passes.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md` - Complete investigation with findings and recommendations

### Files Modified
- None

### Commits
- (Pending) Investigation file creation

---

## Evidence (What Was Observed)

- 500 total investigations, 402 from last 7 days (`find .kb/investigations -name "*.md" -mtime -7 | wc -l`)
- Dashboard topic: 24 investigations, each addressing distinct aspect (SSE events, progressive disclosure, agent panel, account name, status mismatch, etc.)
- Headless topic: 14 investigations, each addressing distinct aspect (scoping, implementation, debugging, token explosion, making default, etc.)
- Zero investigations contain "Supersedes", "Prior", or "kn-" references (`rg` searches returned 0)
- 308 kn decisions, 68 kn constraints, 9 kb formal decisions
- `kb reflect --type synthesis` groups by keyword ("add" = 41) not semantic topic

### Tests Run
```bash
# Investigation count
find .kb/investigations -name "*.md" | wc -l
# Result: 500

# Lineage reference check
rg -l "Supersedes:|Extracted-From:" .kb/investigations/*.md | wc -l
# Result: 0

# Topic-specific counts
find .kb/investigations -name "*dashboard*" -mtime -7 | wc -l  # 24
find .kb/investigations -name "*headless*" -mtime -7 | wc -l   # 14

# kb chronicle for evolution check
kb chronicle "dashboard" | head -50  # Shows distinct evolution, not repetition
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md` - Investigation proving fragmentation is linkage problem, not duplication

### Decisions Made
- Volume reflects development pace, not waste - because each investigation addresses distinct aspect
- Synthesis passes are wrong solution - because kb reflect groups by keyword, not semantic topic
- Focus on linkage improvement - because knowledge exists but is disconnected

### Constraints Discovered
- kb reflect keyword grouping is unhelpful for synthesis - groups "add" investigations that are unrelated
- Spawn context uses generic keyword extraction - doesn't surface related prior investigations
- Investigation template has lineage fields but they're never used - 0 Supersedes/Extracted-From references

### Externalized via `kn`
- (To run) `kn decide "Knowledge fragmentation is linkage problem not duplication" --reason "500 investigations are distinct; 0 have lineage references; kb reflect groups by keyword"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created with full analysis)
- [x] Tests passing (rg, find, kb chronicle all verified claims)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-s03z`

### Discovered Work (create issues for these)
1. **kb reflect semantic clustering** - Replace keyword grouping with topic clustering (dashboard, headless, daemon)
2. **Spawn context related investigations** - Include 2-3 most recent investigations matching task topic
3. **Lineage reminder in spawn template** - Add prompt: "Check if any investigation might be superseded by your work"

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What's the appropriate decision promotion rate? (Currently 1.8% - is that too low or appropriate?)
- Should kn decisions be searchable during spawn context generation? (Currently they're not surfaced as "related investigations")

**Areas worth exploring further:**
- Whether semantic clustering would actually improve synthesis suggestions (needs A/B testing)
- Whether better context injection would reduce investigation creation rate (needs longitudinal study)

**What remains unclear:**
- Whether agents are actively ignoring the "IMPORTANT: Reference investigations for prior findings" instruction in spawn context
- Whether the lineage fields (Supersedes/Extracted-From) are too hidden in the template

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-knowledge-fragmentation-433-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md`
**Beads:** `bd show orch-go-s03z`
