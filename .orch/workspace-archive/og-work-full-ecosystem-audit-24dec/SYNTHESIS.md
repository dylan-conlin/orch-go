# Session Synthesis

**Agent:** og-work-full-ecosystem-audit-24dec
**Issue:** orch-go-pthf
**Duration:** 2025-12-24 → 2025-12-24
**Outcome:** success

---

## TLDR

Full ecosystem audit of 8 orch-related repos completed. Recommended consolidation: kb absorbs kn (month 1), orch-go deprecates Python orch-cli (months 2-3), beads gets abstraction layer (month 4), final cleanup (months 5-6). Core finding: 334 orch + 307 beads skill refs = core workflow; kb (123) + kn (108) = merge candidates; agentlog (14 refs) = orphaned.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md` - Detailed investigation with findings
- `.orch/workspace/og-work-full-ecosystem-audit-24dec/SYNTHESIS.md` - This synthesis

### Files Modified
- None

### Commits
- Pending (investigation + synthesis files)

---

## Evidence (What Was Observed)

- **Repo inventory:** 8 repos mapped with LoC, binary sizes, last activity dates
- **Skill reference counts:** orch (334), bd (307), skillc (153), kb (123), kn (108), agentlog (14)
- **Feature parity:** orch-go has ~80% of Python orch-cli features, plus unique features (swarm, port, servers)
- **beads ownership:** External repo (stevey), 28k char CLAUDE.md, different development trajectory
- **kb/kn overlap:** Both have `context` and `search`, promotion path exists

### Tests Run
```bash
# Reference counting across skills
grep -r "orch " ~/.claude/skills/ | wc -l  # 334
grep -r "bd " ~/.claude/skills/ | wc -l    # 307
grep -r "kb " ~/.claude/skills/ | wc -l    # 123
grep -r "kn " ~/.claude/skills/ | wc -l    # 108
grep -r "agentlog" ~/.claude/skills/ | wc -l  # 14

# Line counts
wc -l ~/Documents/personal/orch-go/**/*.go  # 37,550
ls -la ~/Documents/personal/orch-cli/src/orch/*.py | wc -l  # 27,345
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md` - Comprehensive ecosystem analysis

### Decisions Made
- Decision 1: Phased consolidation over 6 months (not big-bang) - preserves stability
- Decision 2: kb should absorb kn (quick entries become kb subcommand)
- Decision 3: beads needs abstraction layer (external dependency risk)
- Decision 4: agentlog should be evaluated for archive vs integrate

### Constraints Discovered
- beads is external - cannot control its API or release schedule
- Python orch-cli has ~6 features not yet in orch-go
- Skill references must be updated when tools move/merge

### Externalized via `kn`
- Not run during this session (findings captured in investigation)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + synthesis)
- [x] Tests passing (N/A - analysis only)
- [x] Investigation file has complete findings
- [x] Ready for `orch complete orch-go-pthf`

### Follow-up Actions (for orchestrator)

**Epic creation recommended:**

```bash
bd create "Epic: Ecosystem Consolidation" --type epic --description "
## Goal
Consolidate 8-repo orch ecosystem to ~4 functional units over 6 months.

## Phases
1. Month 1: kb absorbs kn
2. Month 2-3: orch-go reaches full Python parity, deprecation warnings
3. Month 4: beads abstraction layer in pkg/beads/
4. Month 5-6: Archive Python orch-cli, evaluate agentlog

## Success Criteria
- Single knowledge CLI (kb) instead of two
- orch-go as only orchestration CLI  
- All skill references point to valid tools
- beads interaction via abstraction layer
"
```

**Child issues:**
1. `kb absorb kn` - Add `kb quick` subcommand, migrate entries
2. `orch-go Python parity` - Port synthesis, transcript, friction, stale, lint features
3. `beads abstraction` - Create pkg/beads/client.go interface
4. `agentlog evaluation` - Decide archive or integrate
5. `Python deprecation` - Add warnings, update skill refs, archive

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should skillc merge into orch-go as `orch skill build`? (build tool pattern says maybe not)
- What's the beads API stability commitment from stevey?
- Are there orch-cli users outside Dylan's setup?

**Areas worth exploring further:**
- Test coverage analysis for orch-go (migration risk assessment)
- User interview on kb vs kn preferences
- beads hook system for orch integration

**What remains unclear:**
- agentlog adoption blockers (why only 14 skill refs?)
- Optimal timing for Python deprecation announcement

*(Straightforward investigation with clear evidence)*

---

## Session Metadata

**Skill:** design-session
**Model:** claude-sonnet
**Workspace:** `.orch/workspace/og-work-full-ecosystem-audit-24dec/`
**Investigation:** `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md`
**Beads:** `bd show orch-go-pthf`
