# Session Synthesis

**Agent:** og-inv-questioning-inherited-constraints-21dec
**Issue:** orch-go-ws4z.10
**Duration:** 2025-12-21 16:30 → 2025-12-21 17:20
**Outcome:** success

---

## TLDR

Investigated when/how to question inherited constraints. Found three clear signals for outdated constraints: implementation supersession (code contradicts constraint), context shift (architectural decisions invalidate domain), and duplicate creation (signals importance not obsolescence). Constraints don't need expiration dates—they need signal-triggered validation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md` - Investigation artifact

### Files Modified
- None

### Commits
- Investigation file to be committed

---

## Evidence (What Was Observed)

- Constraint kn-34d52f "fire-and-forget - no session ID capture" (Dec 19) is outdated—`pkg/spawn/session.go` (Dec 21) now provides `WriteSessionID` and `ReadSessionID`
- 5 duplicate tmux fallback constraints created in 3 minutes (09:49-09:52)—not obsolescence, just discovery failure
- Context shift decision kn-2e08c6 "orch-go is primary CLI, orch-cli (Python) is reference/fallback" potentially invalidates Python-era constraints
- `ref_count` field in kn entries is always 0—citation tracking infrastructure unused

### Tests Run
```bash
# Validated fire-and-forget constraint is outdated
rg "session.*id|sessionID|session_id" pkg/spawn/ --type go
# Found: WriteSessionID, ReadSessionID functions in session.go

# Analyzed constraint duplicates
cat .kn/entries.jsonl | jq -s '[.[] | select(.content | test("tmux fallback"; "i"))]'
# Found: 5 entries in 3 minutes, same content
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md` - Full investigation with signals, validation methods, recommendations

### Decisions Made
- Constraints validated by testing against code, not by age
- Expiration dates rejected—signal-triggered review is better approach
- "Wrong vs misapplied" requires testing: if code contradicts → wrong; if code matches but issues → misapplied

### Constraints Discovered
- Evidence hierarchy applies: constraints are claims, code is truth
- Duplicates indicate importance worth consolidating, not obsolescence

### Externalized via `kn`
- `kn decide "Constraint validity tested by implementation, not age" --reason "..."` - kn-9c641f

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (investigation tested constraint against code)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-ws4z.10`

### Potential Follow-up (Optional)
**Issue:** Add constraint validation to orch reflect
**Skill:** feature-impl
**Context:**
```
Implementation of signal detection for outdated constraints. Should find constraints 
where code contradicts claim. Uses rg patterns extracted from constraint content.
See investigation for signal definitions and validation method.
```

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-questioning-inherited-constraints-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md`
**Beads:** `bd show orch-go-ws4z.10`
