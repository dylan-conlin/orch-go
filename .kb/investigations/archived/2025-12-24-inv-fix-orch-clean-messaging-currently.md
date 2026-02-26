<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch clean` messaging was misleading - said "Cleaned X agents" but never deleted anything from OpenCode or workspaces.

**Evidence:** Code inspection showed workspaces are always preserved (line 2735 comment), output said "Cleaned" when only listing.

**Knowledge:** The command's actual purpose is to LIST completed workspaces and OPTIONALLY clean resources (tmux windows, phantom windows, orphaned disk sessions) - not to delete workspaces.

**Next:** None - fix implemented and tested.

**Confidence:** Very High (95%) - direct code changes tested with manual verification.

---

# Investigation: Fix Orch Clean Messaging

**Question:** Why does `orch clean` say "Cleaned X agents" when it doesn't actually delete anything from OpenCode?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Messaging was misleading

**Evidence:** The output said "Cleaned X workspaces" but the code at line 2735-2736 has a comment: "We don't delete the workspace directory itself - Workspaces are kept for investigation reference"

**Source:** `cmd/orch/main.go:2735-2736`, `cmd/orch/main.go:2891`

**Significance:** Users would expect "Cleaned" to mean something was actually deleted, but in reality nothing was modified unless explicit flags were passed.

---

### Finding 2: Default behavior is report-only

**Evidence:** Without `--windows`, `--phantoms`, or `--verify-opencode` flags, the command only scans and lists completed workspaces. No cleanup actions are taken.

**Source:** `cmd/orch/main.go:2773-2826`

**Significance:** The command's actual behavior is to LIST completed workspaces, with optional cleanup actions behind flags.

---

### Finding 3: OpenCode cleanup requires explicit flag

**Evidence:** OpenCode disk sessions are only cleaned when `--verify-opencode` is passed, and tmux windows only when `--windows` is passed.

**Source:** `cmd/orch/main.go:2828-2844`

**Significance:** Default behavior is completely read-only, which is good but needs to be clearly communicated.

---

## Synthesis

**Key Insights:**

1. **Report vs. action mismatch** - The command's default behavior is to report/list, but the messaging ("Cleaned X") implied action was taken.

2. **Workspaces are intentionally preserved** - This is by design for investigation reference, but not documented in help text.

3. **Cleanup is opt-in** - All cleanup actions require explicit flags, which is a good safety pattern.

**Answer to Investigation Question:**

The messaging was simply wrong - it said "Cleaned" when it should have said "Found" or "Listed". The fix updates:
1. Help text to clearly state default is report-only
2. Output to say "Completed workspaces:" instead of "Cleaning:"
3. Final message to explain what flags do cleanup instead of misleading "Cleaned X"

---

## Implementation Recommendations

### Recommended Approach ⭐

**Update messaging to match actual behavior** - Change help text and output to clearly describe what the command does.

**Why this approach:**
- Minimal code changes
- Preserves existing safe behavior
- Accurately communicates to users

**Implementation sequence:**
1. Updated command help text (Short and Long descriptions)
2. Updated runtime output messages
3. Added explanatory note at end when no cleanup flags used

---

## References

**Files Examined:**
- `cmd/orch/main.go:2519-2544` - cleanCmd definition
- `cmd/orch/main.go:2773-2902` - runClean function
- `cmd/orch/clean_test.go` - existing tests

**Commands Run:**
```bash
# Build and test
go build ./cmd/orch/
go test ./cmd/orch/ -v

# Manual verification
go run ./cmd/orch clean --help
go run ./cmd/orch clean
```

---

## Investigation History

**2025-12-24:** Investigation started
- Initial question: Why does `orch clean` say "Cleaned X agents" when it doesn't actually delete anything?
- Context: User reported misleading messaging

**2025-12-24:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Fixed messaging to accurately describe report-only default behavior with optional cleanup actions behind flags.
