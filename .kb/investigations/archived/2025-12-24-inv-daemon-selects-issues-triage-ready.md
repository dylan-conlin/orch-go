## Summary (D.E.K.N.)

**Delta:** The bug was already fixed in commit bf383e5 - daemon now uses result.Message from Once() instead of hardcoded text.

**Evidence:** Code analysis shows fix is in place (line 252 uses result.Message). Events log shows orch-go-zsuq.2 was successfully spawned at timestamp 1766614194. Tests pass.

**Knowledge:** The original bug (hardcoded "No spawnable issues found") was fixed. Current "No spawnable issues in queue" message is accurate - it means bd ready returns no issues matching triage:ready label, which is correct.

**Next:** No fix needed - close as already resolved.

**Confidence:** Very High (95%) - Fix verified in code, events log shows successful spawns, tests pass.

---

# Investigation: Daemon Selects Issues Triage Ready

**Question:** Why does daemon select issues with triage:ready but then not spawn them?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent (og-debug-daemon-selects-issues-24dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Bug was already fixed in commit bf383e5

**Evidence:** 
- Commit `bf383e5` message: "fix: daemon loop shows actual Once() result message instead of generic text"
- Line 252 of `cmd/orch/daemon.go` now reads: `fmt.Printf("[%s] %s\n", timestamp, result.Message)`
- No hardcoded "No spawnable issues found" string exists in daemon.go

**Source:** `git log --oneline`, `cmd/orch/daemon.go:248-254`

**Significance:** The original bug reported (misleading message) has been fixed. The daemon now correctly displays the actual reason from `Once()`.

---

### Finding 2: Issue orch-go-zsuq.2 was successfully spawned

**Evidence:**
```json
{"type":"session.spawned","session_id":"ses_4ad96aedcffesHasDlbzxe5KEs",
 "timestamp":1766614194,
 "data":{"beads_id":"orch-go-zsuq.2",...}}
```

**Source:** `~/.orch/events.jsonl`

**Significance:** The issue mentioned in the bug report (orch-go-zsuq.2) was actually spawned successfully. This confirms the daemon is working correctly.

---

### Finding 3: Current "No spawnable issues" is accurate

**Evidence:**
- `bd ready --json` returns 10 issues, none with `triage:ready` label
- Issues with `triage:ready` (e.g., orch-go-zsuq.1-3) have `dependency_count: 1` and are filtered out by `bd ready`
- The message "No spawnable issues in queue" is correct when no issues match criteria

**Source:** 
- `bd ready --json | jq '.[] | select(.labels != null and (.labels | any(. == "triage:ready")))'` - returns empty
- `bd list --json | jq '.[] | select(.id | startswith("orch-go-zsuq"))' ` - shows dependency_count: 1

**Significance:** The current behavior is correct. `bd ready` filters out issues with unmet dependencies, so issues like orch-go-zsuq.2 (which depend on parent epic) won't appear in the daemon's issue list.

---

### Finding 4: Prior investigation exists with same conclusion

**Evidence:**
- Investigation file: `.kb/investigations/2025-12-24-inv-daemon-finds-triage-ready-issues.md`
- Status: Complete
- Conclusion: "Fixed daemon loop to use result.Message for accurate feedback"

**Source:** `.kb/investigations/2025-12-24-inv-daemon-finds-triage-ready-issues.md`

**Significance:** A previous agent already investigated and fixed this issue. This investigation confirms the fix is in place.

---

## Synthesis

**Key Insights:**

1. **Bug already resolved** - The hardcoded message bug was fixed in commit bf383e5. Current code correctly uses `result.Message`.

2. **Successful spawns in events log** - The issue orch-go-zsuq.2 mentioned in the bug report was successfully spawned, proving the daemon works.

3. **bd ready filters by dependencies** - Issues with parent-child dependencies are not returned by `bd ready`, so they won't appear in daemon's issue list.

**Answer to Investigation Question:**

The daemon is working correctly. The original bug (showing "No spawnable issues found" when issues exist but can't be spawned due to capacity) was fixed in commit bf383e5. The daemon now shows accurate messages from `Once()`. The issue orch-go-zsuq.2 was successfully spawned according to the events log. Current "No spawnable issues in queue" messages are accurate when no issues have the triage:ready label in the `bd ready` output.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Multiple lines of evidence confirm the fix is in place and working. Code shows the fix, events show successful spawns, tests pass.

**What's certain:**

- ✅ Fix commit bf383e5 is in the codebase
- ✅ Line 252 uses `result.Message` not hardcoded text
- ✅ orch-go-zsuq.2 was successfully spawned per events log
- ✅ All daemon tests pass

**What's uncertain:**

- ⚠️ Cannot reproduce original bug scenario (fix is already applied)

**What would increase confidence to 100%:**

- Integration test that verifies message accuracy in various scenarios

---

## Implementation Recommendations

**Recommended Approach:** No action needed

The bug is already fixed. No further implementation required.

**Alternative consideration:**

If the intent was to investigate why `bd ready` filters out issues with dependencies, that's a different investigation. Issues with parent-child dependencies won't appear in `bd ready` output, which is intentional beads behavior.

---

## References

**Files Examined:**
- `cmd/orch/daemon.go` - Verified fix at line 252
- `pkg/daemon/daemon.go` - Reviewed Once() message flow
- `~/.orch/events.jsonl` - Checked spawn history
- `.kb/investigations/2025-12-24-inv-daemon-finds-triage-ready-issues.md` - Prior investigation

**Commands Run:**
```bash
# Verify fix is in code
grep -n "No spawnable issues found" cmd/orch/daemon.go  # returns empty

# Check daemon verbose output
./build/orch daemon run --poll-interval 0 -v

# Check events for successful spawn
grep "orch-go-zsuq.2" ~/.orch/events.jsonl

# Check bd ready output
bd ready --json | jq
```

---

## Investigation History

**2025-12-24 14:11:** Investigation started
- Initial question: Why does daemon select issues but not spawn?
- Context: Beads issue orch-go-ugyx

**2025-12-24 14:20:** Found prior investigation
- Discovered `.kb/investigations/2025-12-24-inv-daemon-finds-triage-ready-issues.md`
- Confirmed fix was already applied

**2025-12-24 14:25:** Verified fix in code
- Checked cmd/orch/daemon.go line 252
- Confirmed result.Message is used

**2025-12-24 14:30:** Verified successful spawns
- Found orch-go-zsuq.2 spawn in events log
- Tests pass

**2025-12-24 14:35:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Bug was already fixed, no further action needed
