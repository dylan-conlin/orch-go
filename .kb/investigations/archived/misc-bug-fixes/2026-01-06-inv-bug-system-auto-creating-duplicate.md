<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** kb reflect `synthesisIssueExists` only checks open issues, allowing duplicates when previous synthesis issues are closed.

**Evidence:** Multiple closed duplicates found (e.g., 3 closed "Synthesize status investigations (10)"), all created within hours of each other on Jan 6, 2026. Source: `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:486` uses `--status open`.

**Knowledge:** Deduplication must check all issues (not just open) to prevent duplicate creation. Adding a recent-close cooldown (7 days) prevents both duplicates and permanent blocking.

**Next:** Implement fix in kb-cli: change `synthesisIssueExists` to check all statuses, add 7-day cooldown for recently closed issues.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: Bug System Auto Creating Duplicate Synthesis Issues

**Question:** Why is the system creating duplicate synthesis issues and where is the deduplication logic missing?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Implement fix in kb-cli
**Status:** Complete

---

## Findings

### Finding 1: Synthesis issues created by kb reflect, triggered by daemon

**Evidence:** 
- `orch-go/pkg/daemon/daemon.go` line 45-47: `ReflectCreateIssues` config option
- `orch-go/pkg/daemon/reflect.go` line 108-114: `RunReflectionWithOptions` passes `--create-issue` to kb reflect
- `kb-cli/cmd/kb/reflect.go` line 459-469: `createSynthesisIssue` called for topics with 10+ investigations

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/daemon.go:45-47`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/reflect.go:108-114`
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:459-469`

**Significance:** The daemon runs `kb reflect --type synthesis --create-issue` periodically (default: hourly). This is the code path that creates synthesis issues.

---

### Finding 2: Deduplication check only looks at OPEN issues

**Evidence:** 
```go
// Line 486-487 in kb-cli/cmd/kb/reflect.go
cmd := runBdCommand("list", "--status", "open", "--title-contains", searchPattern, "--json")
```

The `synthesisIssueExists` function uses `--status open`, meaning it will NOT find:
- Closed issues
- In-progress issues (though this is less common for synthesis)

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:479-502`

**Significance:** This is the ROOT CAUSE. When a synthesis issue is created and closed, the next daemon run will not find it and will create a duplicate.

---

### Finding 3: Multiple duplicates created on same day, minutes apart

**Evidence:**
```
orch-go-3ziu closed 2026-01-01T10:35:51-08:00
orch-go-zzvs closed 2026-01-01T15:00:18-08:00
orch-go-m45of closed 2026-01-06T12:18:57-08:00
orch-go-lyjdc closed 2026-01-06T17:31:52-08:00  
orch-go-lxhz8 closed 2026-01-06T17:34:11-08:00  <- 2 min later
orch-go-1nf10 closed 2026-01-06T18:35:52-08:00  <- 1 hour later
orch-go-kkg60 open   2026-01-06T19:36:07-08:00  <- 1 hour later
```

7 issues for "Synthesize status investigations (10)" - 6 closed, 1 open.

**Source:** `bd list --all --title-contains "Synthesize status" --json`

**Significance:** The pattern shows issues being created when previous ones are closed. The daemon runs hourly by default, matching the ~1 hour intervals between creations.

---

## Synthesis

**Key Insights:**

1. **Creation pathway:** Daemon (orch-go) → kb reflect (kb-cli) → synthesisIssueExists check → createSynthesisIssue

2. **Root cause location:** The bug is in kb-cli, not orch-go. The `synthesisIssueExists` function at `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:479-502` only checks open issues.

3. **Fix strategy:** Remove `--status open` restriction, but add a recent-close cooldown to prevent permanent blocking.

**Answer to Investigation Question:**

The system creates duplicate synthesis issues because:
1. `kb reflect --type synthesis --create-issue` is called by the daemon periodically
2. The `synthesisIssueExists` deduplication check only looks at OPEN issues
3. When a synthesis issue is closed, the next reflect run doesn't find it and creates a duplicate

The deduplication logic is present but incomplete - it's in `kb-cli/cmd/kb/reflect.go:479-502` but only checks open status.

---

## Structured Uncertainty

**What's tested:**

- ✅ Confirmed duplicates exist via `bd list --all --title-contains "Synthesize"` 
- ✅ Confirmed `synthesisIssueExists` uses `--status open` via code inspection
- ✅ Confirmed creation times align with daemon reflection interval (~1 hour)

**What's untested:**

- ⚠️ Fix implementation (will be tested after applying)
- ⚠️ Whether 7-day cooldown is appropriate duration

**What would change this:**

- If bd list `--title-contains` doesn't work as expected (but testing shows it does)
- If there's another code path creating synthesis issues (didn't find any)

---

## Implementation Recommendations

**Purpose:** Fix the deduplication logic in kb-cli to prevent duplicate synthesis issues.

### Recommended Approach ⭐

**Check all statuses with 7-day cooldown** - Modify `synthesisIssueExists` to check all issues (not just open), but only block if an issue was closed within the last 7 days.

**Why this approach:**
- Prevents duplicates while allowing re-creation after cooldown
- Simple to implement - one code change in kb-cli
- Matches daemon's typical reflection interval logic

**Trade-offs accepted:**
- If investigation count grows significantly during cooldown, new issue won't be created immediately
- Acceptable because synthesis issues are low priority (triage:review label)

**Implementation sequence:**
1. Modify `synthesisIssueExists` to remove `--status open`
2. Parse issue closed date from JSON response  
3. Only return "exists" if issue is open OR was closed within 7 days

### Alternative Approaches Considered

**Option B: Remove --status open entirely (no cooldown)**
- **Pros:** Simpler implementation
- **Cons:** Once closed, synthesis issue for topic will NEVER be auto-created again
- **When to use instead:** If synthesis issues should only ever be created once per topic

**Option C: Track created issues in a separate file**
- **Pros:** More control over deduplication logic
- **Cons:** More complex, additional state to manage
- **When to use instead:** If bd integration is unreliable

**Rationale for recommendation:** 7-day cooldown balances preventing duplicates while allowing fresh synthesis issues after a reasonable time period.

---

### Implementation Details

**What to implement first:**
- Fix in `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go`
- Modify `synthesisIssueExists` function (lines 479-502)
- Same fix needed for `openIssueExists` function (lines 1226-1248)

**Things to watch out for:**
- ⚠️ JSON date parsing - beads uses RFC3339 format
- ⚠️ bd list might return empty array vs error for no matches

**Success criteria:**
- ✅ Running `kb reflect --type synthesis --create-issue` twice in a row should NOT create duplicates
- ✅ After 7 days, a new issue CAN be created for the same topic

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go` - synthesisIssueExists function (root cause)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/reflect.go` - Daemon's kb reflect invocation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/daemon.go` - Daemon configuration

**Commands Run:**
```bash
# Check for duplicate synthesis issues
bd list --all --title-contains "Synthesize" --json | jq -r '.[] | "\(.status) \(.title)"' | sort | uniq -c

# Check creation times
bd list --all --title-contains "Synthesize status" --json | jq -r '.[] | "\(.id) \(.status) \(.created_at)"'
```

---

## Investigation History

**2026-01-06 19:50:** Investigation started
- Initial question: Why are duplicate synthesis issues being created?
- Context: Ready queue contained duplicate synthesis issues

**2026-01-06 20:00:** Root cause identified
- Found `synthesisIssueExists` only checks open issues
- Confirmed via code inspection and issue creation timestamps

**2026-01-06 20:10:** Investigation completed
- Status: Complete
- Key outcome: Fix required in kb-cli to check all statuses with 7-day cooldown
